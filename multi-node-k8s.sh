#!/bin/sh

# TODO: disable debugging
set -x

# enable additional sources for yum
# (epel for ansible and golang)
yum -y install epel-release

# Install additional packages
yum -y install \
	qemu-kvm \
	qemu-kvm-tools \
	qemu-img \
	libvirt \
	ansible \
	golang

# install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
mv kubectl /usr/bin/kubectl
chmod +x /usr/bin/kubectl

# Vagrant needs libvirtd running
systemctl start libvirtd

# Log the virsh capabilites so that we know the
# environment in case something goes wrong.
virsh capabilities

# TODO: this is not the right way to install vagrant on CentOS, but SCL only
# provides vagrant 1.x and we need >= 2.2
yum -y install https://releases.hashicorp.com/vagrant/2.2.7/vagrant_2.2.7_x86_64.rpm
yum -y install gcc libvirt-devel

vagrant plugin install vagrant-libvirt

# setup the kubernes cluster
git clone https://github.com/galexrt/k8s-vagrant-multi-node
cd k8s-vagrant-multi-node

make preflight
make up -j4 BOX_OS=centos VAGRANT_DEFAULT_PROVIDER=libvirt MASTER_CPUS=4 MASTER_MEMORY_SIZE_GB=12 NODE_COUNT=3 NODE_MEMORY_SIZE_GB=4 DISK_COUNT=4 DISK_SIZE_GB=50
make versions
make status
