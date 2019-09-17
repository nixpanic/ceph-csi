#!/bin/sh

vagrant up
# add ~/go/bin to the path
# shellcheck disable=SC2016
echo 'export PATH=${PATH}:~/go/bin' | vagrant ssh -c 'cat >> ~/.bashrc'
( cd ../.. ; git archive --format=tar --prefix=go/src/github.com/ceph/ceph-csi/ HEAD ) | vagrant ssh -c 'tar x'
vagrant ssh -c 'cd go/src/github.com/ceph/ceph-csi && ./scripts/centos-ci/install-deps.sh && source /opt/rh/rh-ruby26/enable && make test'
RET=$?

if [ -z "${CEPHCSI_KEEP_VM}" ]
then
	vagrant destroy --force
fi

exit ${RET}
