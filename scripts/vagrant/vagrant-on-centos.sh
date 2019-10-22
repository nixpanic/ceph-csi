#!/bin/bash
#
# - install Vagrant
# - clone the git repo and checkout the branch to test
# - run ./scripts/vagrant/make-test.sh to start a VM and run the tests
#

# if anything fails, we'll abort
set -e

# TODO: disable debugging
set -x

# we get the code from git
yum -y install git

# enable the SCL repository for Vagrant
yum -y install centos-release-scl

# install Vagrant with QEMU
#
# WARNING: adding sclo-vagrant1-vagrant on the "yum install" command makes it
#          work fine. Without sclo-vagrant1-vagrant the following error occurs
#          and starting the VMs fails:
#
#    Call to virDomainCreateWithFlags failed: the CPU is incompatible with host
#    CPU: Host CPU does not provide required features: svm
#
yum -y install qemu-kvm sclo-vagrant1-vagrant sclo-vagrant1-vagrant-libvirt \
               qemu-kvm-tools qemu-img

# Vagrant needs libvirtd running
systemctl start libvirtd

# Log the virsh capabilites so that we know the
# environment in case something goes wrong.
virsh capabilities

# We'll run this from a temporary directory
WORKDIR=$(mktemp -d)
pushd "${WORKDIR}"

# TODO: update branch and repo location
git clone -b vagrant/make-test https://github.com/nixpanic/ceph-csi.git
pushd ceph-csi

# by default we clone the master branch, but maybe this was triggered through a PR?
# shellcheck disable=SC2154
if [ -n "${ghprbPullId}" ]
then
	git fetch origin "pull/${ghprbPullId}/head:pr_${ghprbPullId}"
	git checkout "pr_${ghprbPullId}"

	# Now rebase on top of master
	git rebase master
	if [ $? -ne 0 ] ; then
	    echo "Unable to automatically merge master. Please rebase your patch"
	    exit 1
	fi
fi

# set the current working directory so that the script find the Vagrantfile
pushd scripts/vagrant
set +e
scl enable sclo-vagrant1 ./make-tests.sh
RET=$?

# cleanup, can fail
scl enable sclo-vagrant1 vagrant destroy
popd # scripts/vagrant
popd # ceph-csi
popd # ${WORKDIR}
rm -rf "${WORKDIR}"

exit ${RET}
