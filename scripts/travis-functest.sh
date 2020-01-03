#!/bin/bash
set -e

# This script will be used by travis to run functional test
# against different kuberentes version
export KUBE_VERSION=$1

sudo mkdir -p /opt/minikube /opt/kube
sudo ln -s /opt/minikube /root/.minikube
sudo ln -s /opt/kube /root/.kube

sudo bash scripts/minikube.sh up
sudo scripts/minikube.sh deploy-rook
# pull docker images to speed up e2e
sudo scripts/minikube.sh cephcsi
sudo scripts/minikube.sh k8s-sidecar
[ -d "$HOME"/.minikube ] && sudo chown -R ${USER}: "$HOME"/.minikube /usr/local/bin/kubectl
sudo chown -R ${USER} /opt/minikube /opt/kube
# functional tests

go test github.com/ceph/ceph-csi/e2e --deploy-timeout=10 -timeout=30m -v

sudo scripts/minikube.sh clean
