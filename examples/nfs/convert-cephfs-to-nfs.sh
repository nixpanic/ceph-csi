#!/bin/sh

set -e

PVC=${1}
[ -n "${PVC}" ] || { echo "PVC is empty" ; exit 1 ; }

PV=$(oc get pvc/${PVC} -ojsonpath='{.spec.volumeName}')

SIZE=$(oc get pv/${PV} -ojsonpath='{.spec.capacity.storage}')
FS=$(oc get pv/${PV} -ojsonpath='{.spec.csi.volumeAttributes.fsName}')
SUBVOL=$(oc get pv/${PV} -ojsonpath='{.spec.csi.volumeAttributes.subvolumeName}')
SUBVOL_PATH=$(oc get pv/${PV} -ojsonpath='{.spec.csi.volumeAttributes.subvolumePath}')
EXPORT=$(oc get pv/${PV} -ojsonpath='{.spec.csi.volumeHandle}')

NFS=$(oc -n openshift-storage get cephnfs -ojsonpath='{.items[0].metadata.name}')
SERVER=$(oc -n openshift-storage get service -l ceph_nfs=my-nfs -ojsonpath='{.items[0].metadata.name}').openshift-storage.svc.cluster.local

oc -n openshift-storage rsh $(oc -n openshift-storage get pods -l app=rook-ceph-tools -o jsonpath='{.items[0].metadata.name}{"\n"}') << EOS
ceph nfs export create cephfs ${FS} ${NFS} /${EXPORT} ${SUBVOL_PATH}
EOS

oc create -f - << EOT
apiVersion: v1
kind: PersistentVolume
metadata:
  name: nfs-${PV}
spec:
  storageClassName: ceph-nfs
  accessModes:
  - ReadWriteMany
  capacity:
    storage: ${SIZE}
  csi:
    driver: openshift-storage.nfs.csi.ceph.com
    volumeAttributes:
      clusterID: openshift-storage
      fsName: ${FS}
      subvolumeName: ${SUBVOL}
      subvolumePath: ${SUBVOL_PATH}
      server: ${SERVER}
      share: /${EXPORT}
    volumeHandle: nfs-${EXPORT}
  persistentVolumeReclaimPolicy: Retain
  volumeMode: Filesystem
EOT

oc create -f - << EOT
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nfs-${PVC}
spec:
  storageClassName: ceph-nfs
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: ${SIZE}
EOT
