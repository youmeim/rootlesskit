package tmpfssymlink

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsExcludedFile(t *testing.T) {
	assert.DeepEqual(t, isExcludedFile("/etc/passwd", []string{"/etc/passwd", "/etc/groups", "/etc/subuid", "/etc/subgid"}), true)
	assert.DeepEqual(t, isExcludedFile("/etc/passwd", []string{"/etc/groups", "/etc/subuid", "/etc/subgid"}), false)
}

func TestCopyUpNothing(t *testing.T) {
	d := NewChildDriver(nil)
	_, err := d.CopyUp(nil)
	assert.NilError(t, err)
}

func TestCopyUpEtc(t *testing.T) {
	d := NewChildDriver(nil)
	_, err := d.CopyUp([]string{"/etc"})
	assert.NilError(t, err)
}

func TestCopyUpEtcWithoutAA(t *testing.T) {
	excludedDir := []string{"/etc/apparmor", "/etc/apparmor.d"}
	d := NewChildDriver(excludedDir)
	_, err := d.CopyUp([]string{"/etc"})
	assert.NilError(t, err)
	dirs, err := os.ReadDir("/etc")
	for _, d := range dirs {
		for _, e := range excludedDir {
			assert.Assert(t, "/etc/"+d.Name() != e, "no")
		}
	}
}
