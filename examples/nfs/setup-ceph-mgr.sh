#!/bin/sh

set -e -x

# enable the Rook toolbox and wait for it to be ready
oc -n openshift-storage patch ocsinitializations.ocs.openshift.io/ocsinit --type=merge --patch '{"spec": {"enableCephTools": true }}'
oc -n openshift-storage wait pods -l app=rook-ceph-tools --for condition=Ready --timeout=90s

# configure Ceph MGR
# the nfs-ganesha pool needs to exist before creating exports
oc -n openshift-storage rsh $(oc -n openshift-storage get pods -l app=rook-ceph-tools -o jsonpath='{.items[0].metadata.name}{"\n"}') << EOS
ceph osd pool stats nfs-ganesha || ceph osd pool create nfs-ganesha
ceph mgr module enable rook
ceph mgr module enable nfs
ceph orch set backend rook
EOS

oc -n openshift-storage create -f - << EOY
apiVersion: ceph.rook.io/v1
kind: CephNFS
metadata:
  name: my-nfs
  namespace: openshift-storage
spec:
  # For Ceph v15, the rados block is required. It is ignored for Ceph v16.
  rados:
    # fixed value for Ceph v16
    pool: nfs-ganesha
    # RADOS namespace where NFS client recovery data is stored in the pool.
    # fixed value for Ceph v16: the name of this CephNFS object
    namespace: my-nfs

  # Settings for the NFS server
  server:
    # the number of active NFS servers
    active: 1
EOY

# Example commands to run in the toolbox
#ceph nfs export create cephfs ocs-storagecluster-cephfilesystem my-nfs /0001-0011-openshift-storage-0000000000000001-0cc9b7f8-8fd2-11ec-9982-0a580a800210 /volumes/csi/csi-vol-0cc9b7f8-8fd2-11ec-9982-0a580a800210/9221c93a-827c-49f3-becb-63444178b1a0
#cat /tmp/conf-nfs.my-nfs
#EXPORT {
#    FSAL {
#        name = "CEPH";
#        filesystem = "ocs-storagecluster-cephfilesystem";
#
#        # "nfs-ganesha.my-nfs.a" is used for fetching the configuration, it
#        # does not have permissions on the filesystem
#        #user_id = "nfs-ganesha.my-nfs.a";
#
#        # "admin" ceredentials fetched through the Rook toolbox
#        user_id = "admin";
#        secret_access_key = "AQDcBQ5iBHv3DBAAbYjLgj/HnOoOHvXcSUkUhg==";
#    }
#    export_id = 123;
#
#    # path is the "subvolumePath" from the CephFS PV
#    path = "/volumes/csi/csi-vol-0cc9b7f8-8fd2-11ec-9982-0a580a800210/9221c93a-827c-49f3-becb-63444178b1a0";
#
#    # pseudo is the path on the NFS-server
#    pseudo = "/0001-0011-openshift-storage-0000000000000001-0cc9b7f8-8fd2-11ec-9982-0a580a800210";
#    access_type = "RW";
#    squash = "none";
#    attr_expiration_time = 0;
#    security_label = true;
#    protocols = 4;
#    transports = "TCP";
#}

#rados put -p ocs-storagecluster-cephfilesystem-data0 -N nfs-ganesha conf-nfs.my-nfs /tmp/conf-nfs.my-nfs

# mount -t nfs4 rook-ceph-nfs-my-nfs-a.openshift-storage.svc.cluster.local:/0001-0011-openshift-storage-0000000000000001-0cc9b7f8-8fd2-11ec-9982-0a580a800210 /mnt
