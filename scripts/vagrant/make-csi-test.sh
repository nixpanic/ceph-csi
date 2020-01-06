#!/bin/sh

vagrant up
# add ~/go/bin to the path
# shellcheck disable=SC2016
cat << EOF | vagrant ssh -c 'sudo tee -a /etc/environment'
VM_DRIVER=none
#CONTAINER_RUNTIME=cri-o
KUBECONFIG=/opt/kube/config
EOF

( cd ../.. ; git archive --format=tar --prefix=go/src/github.com/ceph/ceph-csi/ HEAD ) | vagrant ssh -c 'tar x'
vagrant ssh -c 'cd go/src/github.com/ceph/ceph-csi && ./scripts/vagrant/prepare-vm.sh'
#vagrant ssh -c 'cd go/src/github.com/ceph/ceph-csi && ./scripts/vagrant/minikube-cn.sh'
vagrant ssh -c 'cd go/src/github.com/ceph/ceph-csi && VM_DRIVER=none ./scripts/travis-functest.sh'
RET=$?

if [ -z "${CEPHCSI_KEEP_VM}" ]
then
	vagrant destroy --force
fi

exit ${RET}
