#!/bin/bash
source $(realpath $(dirname $0))/common.inc.sh
function benchmark::iperf3::slirp4netns() {
	INFO "[benchmark:iperf3] slirp4netns ($@)"
	set -x
	$ROOTLESSKIT --net=slirp4netns $@ -- $IPERF3C 10.0.2.2
	set +x
}

function benchmark::iperf3::vpnkit() {
	INFO "[benchmark:iperf3] vpnkit ($@)"
	set -x
	$ROOTLESSKIT --net=vpnkit $@ -- $IPERF3C 192.168.65.2
	set +x
}

function benchmark::iperf3::rootful_veth() {
	INFO "[benchmark:iperf3] rootful_veth ($@) for reference"
	# only --mtu=MTU is supposed as $@
	mtu=$(echo $@ | sed -e s/--mtu=//g)
	set -x
	sudo ip netns add foo
	sudo ip link add foo_veth0 type veth peer name foo_veth1
	sudo ip link set foo_veth1 netns foo
	sudo ip addr add 10.0.42.1/24 dev foo_veth0
	sudo ip -netns foo addr add 10.0.42.2/24 dev foo_veth1
	sudo ip link set dev foo_veth0 mtu $mtu
	sudo ip -netns foo link set dev foo_veth1 mtu $mtu
	sudo ip link set foo_veth0 up
	sudo ip -netns foo link set foo_veth1 up
	sudo ip netns exec foo $IPERF3C 10.0.42.1
	sudo ip link del foo_veth0
	sudo ip netns del foo
	set +x
}

if [ $# -lt 2 ]; then
	ERROR "Usage: $0 NETDRIVER MTU [FLAGS...]"
	exit 1
fi
net=$1
mtu=$2
shift 2
flags=$@
INFO "net=${net} mtu=${mtu} flags=$@"

iperf3 -s >/dev/null &
iperf3pid=$!
function cleanup() {
	kill $iperf3pid
}
trap cleanup EXIT
benchmark::iperf3::$net --mtu=$mtu $flags
