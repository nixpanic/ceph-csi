#!/bin/sh
#
# - install minikube
# - install Ceph Nano

set -e
set -x

# testing... seems needed for minikube :-(
sudo setenforce 0

# recommended hostname for minikube
sudo hostnamectl set-hostname minikube

# CRI-O is preferred, but not directly available on CentOS-7
sudo yum -y install docker

# minikube/kubelet runs with cgroupfs
sed 's/native.cgroupdriver=systemd/native.cgroupdriver=cgroupfs/' /usr/lib/systemd/system/docker.service | sudo tee /etc/systemd/system/docker.service
sudo systemctl daemon-reload
sudo systemctl enable docker
sudo systemctl start docker

echo 1 | sudo tee /proc/sys/net/bridge/bridge-nf-call-iptables > /dev/null

# From https://minikube.sigs.k8s.io/docs/start/linux/
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 \
	&& sudo install minikube-linux-amd64 /usr/local/bin/minikube

# "./cn kube" is broken with version 2.3.1, fix is not released yet
#curl -L https://github.com/ceph/cn/releases/download/v2.3.1/cn-v2.3.1-linux-amd64 -o cn && chmod +x cn

sudo yum -y install \
	make \
	gcc \
	epel-release \
	; # empty line for 'git blame'

sudo yum -y install \
	golang \
	; # empty line for 'git blame'

export GOPATH=~/go
export PATH=$PATH:~/go/bin

go get github.com/golang/dep/cmd/dep
mkdir -p $GOPATH/src/github.com/ceph
cd $GOPATH/src/github.com/ceph
git clone https://github.com/ceph/cn
cd cn
make

###
### Installation of tools finished, start deployment
###

# when CRI-O is used, pass --container-runtime=cri-o
sudo /usr/local/bin/minikube start --vm-driver=none
#sudo cp -r /root/.kube /root/.minikube $HOME
#sudo chown -R $USER $HOME/.kube $HOME/.minikube

alias kubectl='sudo /usr/local/bin/minikube kubectl -- '
# show the version, might dump some non-yaml to stdout
./cn version
./cn kube > cn.yaml
sed -i 's/memory: 512M/memory: 1024M/g' cn.yaml
kubectl apply -f cn.yaml
