#!/bin/bash
set -e

# This script will be used by travis to run functional test
# against different kuberentes version
export KUBE_VERSION=$1

sudo scripts/minikube.sh up
sudo scripts/minikube.sh deploy-rook
# pull docker images to speed up e2e
sudo scripts/minikube.sh cephcsi
sudo scripts/minikube.sh k8s-sidecar

# download kubectl and setup access for local user
MK_KUBE_VERSION=$(sudo /usr/local/bin/minikube kubectl version -- --client -o yaml | awk '/gitVersion:/{print $2}')
sudo cp /root/.minikube/cache/linux/"${MK_KUBE_VERSION}"/kubectl /usr/local/bin/
sudo cp -r /root/.minikube /opt/minikube
sed "s|/root/.minikube/|/opt/minikube/|g" -i /opt/kube/config
sudo chown "${USER}:${GROUP}" -R /opt/minikube /opt/kube
kubectl version

# functional tests
USE_SUDO=""
#[ "${VM_DRIVER}" = "none" ] && USE_SUDO="sudo"
${USE_SUDO} go test github.com/ceph/ceph-csi/e2e --deploy-timeout=10 -timeout=30m -v

sudo scripts/minikube.sh clean
