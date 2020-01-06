#!/bin/bash
set -e

# This script will be used by travis to run functional test
# against different kuberentes version
export KUBE_VERSION=$1

scripts/minikube.sh up
scripts/minikube.sh deploy-rook
# pull docker images to speed up e2e
scripts/minikube.sh cephcsi
scripts/minikube.sh k8s-sidecar
# functional tests
go test github.com/ceph/ceph-csi/e2e --deploy-timeout=10 -timeout=30m -v

scripts/minikube.sh clean
