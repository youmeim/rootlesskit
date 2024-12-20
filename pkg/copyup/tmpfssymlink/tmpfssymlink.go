package tmpfssymlink

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"

	"github.com/rootless-containers/rootlesskit/v2/pkg/copyup"
)

func NewChildDriver() copyup.ChildDriver {
	return &childDriver{}
}

func NewChildDriverWithExclusion(excludedFiles []string) copyup.ChildDriver {
	return &childDriver{
		excludedFiles,
	}
}

type childDriver struct {
	excludedFiles []string
}

func isExcludedFile(path string, dirs []string) bool {
	for _, fExcluded := range dirs {
		if path == fExcluded {
			_, _ = fmt.Fprintf(os.Stderr, "matched: %s", path)
			return true
		}
	}
	return false
}

func (d *childDriver) CopyUp(dirs []string) ([]string, error) {
	// we create bind0 outside of StateDir so as to allow
	// copying up /run with stateDir=/run/user/1001/rootlesskit/default.
	bind0, err := os.MkdirTemp("/tmp", "rootlesskit-b")
	if err != nil {
		return nil, fmt.Errorf("creating bind0 directory under /tmp: %w", err)
	}
	defer os.RemoveAll(bind0)
	var copied []string
	for _, dir := range dirs {
		dir := filepath.Clean(dir)
		if dir == "/tmp" {
			// TODO: we can support copy-up /tmp by changing bind0TempDir
			return copied, errors.New("/tmp cannot be copied up")
		}

		if err := unix.Mount(dir, bind0, "", uintptr(unix.MS_BIND|unix.MS_REC), ""); err != nil {
			return copied, fmt.Errorf("failed to create bind mount on %s: %w", dir, err)
		}

		if err := unix.Mount("none", dir, "tmpfs", 0, ""); err != nil {
			return copied, fmt.Errorf("failed to mount tmpfs on %s: %w", dir, err)
		}

		bind1, err := os.MkdirTemp(dir, ".ro")
		if err != nil {
			return copied, fmt.Errorf("creating a directory under %s: %w", dir, err)
		}
		if err := unix.Mount(bind0, bind1, "", uintptr(unix.MS_MOVE), ""); err != nil {
			return copied, fmt.Errorf("failed to move mount point from %s to %s: %w", bind0, bind1, err)
		}

		files, err := os.ReadDir(bind1)
		if err != nil {
			return copied, fmt.Errorf("reading dir %s: %w", bind1, err)
		}
		for _, f := range files {
			fFull := filepath.Join(bind1, f.Name())
			var symlinkSrc string
			if f.Type()&os.ModeSymlink != 0 {
				symlinkSrc, err = os.Readlink(fFull)
				if err != nil {
					return copied, fmt.Errorf("reading dir %s: %w", fFull, err)
				}
			} else {
				symlinkSrc = filepath.Join(filepath.Base(bind1), f.Name())
			}
			symlinkDst := filepath.Join(dir, f.Name())
			if len(d.excludedFiles) > 0 && isExcludedFile(symlinkDst, d.excludedFiles) {
				continue
			}
			// `mount` may create extra `/etc/mtab` after mounting empty tmpfs on /etc
			// https://github.com/rootless-containers/rootlesskit/issues/45
			if err = os.RemoveAll(symlinkDst); err != nil {
				return copied, fmt.Errorf("removing %s: %w", symlinkDst, err)
			}
			if err := os.Symlink(symlinkSrc, symlinkDst); err != nil {
				return copied, fmt.Errorf("symlinking %s to %s: %w", symlinkSrc, symlinkDst, err)
			}
		}
		copied = append(copied, dir)
	}
	return copied, nil
}
