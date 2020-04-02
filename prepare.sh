#!/bin/sh

set -x

yum -y install git podman

mkdir -p /opt/build/go/src/github.com/ceph/
git clone --single-branch --branch=master https://github.com/ceph/ceph-csi /opt/build/go/src/github.com/ceph/ceph-csi
