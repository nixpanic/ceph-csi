# Current Status and Notes about NVMe-oF Testing

This branch is based on [Gadi's `nvmeof/create_driver`
branch](https://github.com/gadididi/ceph-csi/tree/nvmeof/create_driver).

There are additional commits that cleanup things, make it deployable on
OpenShift in the `openshift-storage` Namespace (like ODF) and add an example
StorageClass with a PersistentVolumeClaim.

A list of commits on top of Gadi's branch can be constructed on [this GitHub
compare
page](https://github.com/gadididi/ceph-csi/compare/nvmeof/create_driver...nixpanic:ceph-csi:nvmeof/testing).

## Testing Steps

### Deploy the NVMe-oF Provisioner

```
$ kubectl -n openshift-storage create -f deploy/nvmeof/kubernetes/csi-provisioner-rbac.yaml
$ kubectl -n openshift-storage create -f deploy/nvmeof/kubernetes/csi-nvmeofplugin-provisioner.yaml
```

This creates a ServiceAccount (with appropriate permissions) and Deployment.
The Deployment triggers the creation of a ReplicaSet which then creates the
Pod(s) with their containers.

```
$ kubectl -n openshift-storage describe deployment/csi-nvmeofplugin-provisioner
...
Conditions:
  Type           Status  Reason
  ----           ------  ------
  Available      True    MinimumReplicasAvailable
  Progressing    True    NewReplicaSetAvailable
OldReplicaSets:  <none>
NewReplicaSet:   csi-nvmeofplugin-provisioner-745f6d9947 (1/1 replicas created)
Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  22m   deployment-controller  Scaled up replica set csi-nvmeofplugin-provisioner-745f6d9947 from 0 to 1
```

The ReplicaSet has a random extension, the example above shows `csi-nvmeofplugin-provisioner-745f6d9947`:
```
$ kubectl -n openshift-storage describe rs/csi-nvmeofplugin-provisioner-745f6d9947
...
Events:
  Type    Reason            Age   From                   Message
  ----    ------            ----  ----                   -------
  Normal  SuccessfulCreate  23m   replicaset-controller  Created pod: csi-nvmeofplugin-provisioner-745f6d9947-hp5ds
```

The Deployment has `replicas: 1` configured, so that there is a single (no
high-availability) provisioner. The name of the provisioner Pod also contains a
random extension.

```
$ kubectl -n openshift-storage get pods -l app=csi-nvmeofplugin-provisioner
NAME                                            READY   STATUS    RESTARTS   AGE
csi-nvmeofplugin-provisioner-745f6d9947-hp5ds   2/2     Running   0          25m
```

### Create a StorageClass

The StorageClass contains the values for the configuration of the Ceph cluster
and the NVMe-oF gateway. All parameters that are valid for an RBD StorageClass,
are also valid/required for the NVMe-oF StorageClasses.

In addition of the RBD parameters, details of the NVMe-oF Gateway are required:

```
  listenerHostname: ceph-nvmeof-gateway.openshift-storage.svc.cluster.local
  listenerIpAddress: 172.30.156.139
  listenerPort: "4420"
  nvmeofGatewayAddress: ceph-nvmeof-gateway.openshift-storage.svc.cluster.local
  nvmeofGatewayPort: "5500"
  subsystemNQN: "nqn.2016-06.io.spdk:cnode1"
  hostNQN: "nqn.2014-08.org.nvmexpress:uuid:950ddadf-f995-47b7-9416-b9bb233f66e3"
```

> [ndevos] I haven't found clear examples of these parameters, current values
> may make no sense and could cause issues?

> [ndevos] It probably makes sense to have default parameters in case the
> parameters are not set in the StorageClass. Ideally everything works when
> only the `nvmeofGatewayAddress` is set in the StorageClass.

```
$ kubectl -n openshift-storage create -f deploy/nvmeof/kubernetes/storageclass.yaml
```

### Deploy a NVMe-oF Gateway

**Any other way of deploying a NVMe-oF Gateway should work.**

> [ndevos] this is probably the main issue I currently face. My gateway reports
> errors, so it is most likely that there is a mis-configuration there.

> [ndevos] there are also reports that the NVMe-oF gateway only works on Ceph
> Tentacle, which isn't what I have in my OpenShift with Data Foundation
> cluster.
> e.g. https://github.com/ceph/ceph-nvmeof/issues/1382

Steps in a [separate
document](https://github.com/nixpanic/ceph-nvmeof/tree/deploy/odf/deploy/kubernetes).

## Testing

Only `CreateVolume` and `DeleteVolume` needs to be tested for starters. A
PersistentVolumeClaim that references the StorageClass can be created:

```
$ kubectl create -f deploy/nvmeof/kubernetes/pvc.yaml
```

The result is most likely a PVC that is stuck in **PENDING** state:

```
$ kubectl describe pvc/nvmeof-test-pvc
Name:          nvmeof-test-pvc
Namespace:     default
StorageClass:  ocs-storagecluster-ceph-nvmeof
Status:        Pending
Volume:        
Labels:        <none>
Annotations:   volume.beta.kubernetes.io/storage-provisioner: nvmeof.csi.ceph.com
               volume.kubernetes.io/storage-provisioner: nvmeof.csi.ceph.com
Finalizers:    [kubernetes.io/pvc-protection]
Capacity:      
Access Modes:  
VolumeMode:    Filesystem
Used By:       <none>
Events:
  Type     Reason                Age                    From                                                                                                    Message
  ----     ------                ----                   ----                                                                                                    -------
  Warning  ProvisioningFailed    74m (x2 over 75m)      nvmeof.csi.ceph.com_csi-nvmeofplugin-provisioner-745f6d9947-hp5ds_074e8e19-768f-45c2-9cce-84e246a999b5  failed to provision volume with StorageClass "ocs-storagecluster-ceph-nvmeof": rpc error: code = Internal desc = NVMe-oF setup failed: subsystem setup failed: failed to list subsystems: rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing: dial tcp 172.30.24.72:5500: connect: connection refused"
```

## Troubleshooting

The `csi-nvmeofplugin-provisioner` Pod has two containers:

1. `csi-nvmeofplugin` that runs the main `cephcsi` executable
2. `csi-provisioner` that runs the Kubernetes external csi-provisioner

To watch the logs while the provisioner executes an action, the easiest is to
use a command like this:

```
$ kubectl -n openshift-storage logs -f -c csi-nvmeofplugin csi-nvmeofplugin-provisioner-745f6d9947-hp5ds
```

To stop logging, press _CTRL+c_.

When creation of a PersistentVolume(Claim) goes wrong, Kubernetes will usually
(depending on the failure) keep re-trying. The logs will likely grow quickly,
with lots of repeated failures. The most interesting failure will be the 1st,
somewhere in the top of the logs.

The deployment from this branch refers to `quay.io/nixpanic/cephcsi:nvmeof` as
a testing container image. Others will probably want to test their own
container-image, requiring to modify the deployment.

After making modifications to the container image, and pushing the updated
image to a container registry, it is needed to restart the provisioner. The
simplest way to do so, is by deleting the Pods (they get created again by the
Deployment/ReplicaSet):

```
$ kubectl -n openshift-storage delete pods -l app=csi-nvmeofplugin-provisioner
pod "csi-nvmeofplugin-provisioner-745f6d9947-hp5ds" deleted
```

## Current Status

- deploying the provisioner works
- the provisioner starts to handle the `CreateVolume` CSI procedure
- ... and then it fails

```
$ kubectl -n openshift-storage logs -f -c csi-nvmeofplugin csi-nvmeofplugin-provisioner-745f6d9947-fn4mq
I0801 14:03:07.587065       1 cephcsi.go:208] Driver version: canary and Git version: 7b86290f76c48d0c3f9456c1021eecda2daa3c0c
I0801 14:03:07.587552       1 cephcsi.go:249] Starting driver type: nvmeof with name: nvmeof.csi.ceph.com
I0801 14:03:07.588115       1 driver.go:126] Enabling controller service capability: CREATE_DELETE_VOLUME
I0801 14:03:07.588141       1 driver.go:139] Enabling volume access mode: SINGLE_NODE_WRITER
I0801 14:03:07.588149       1 driver.go:139] Enabling volume access mode: SINGLE_NODE_READER_ONLY
I0801 14:03:07.588741       1 server.go:131] Listening for connections on address: &net.UnixAddr{Name:"//csi/csi-provisioner.sock", Net:"unix"}
I0801 14:03:07.620209       1 utils.go:341] ID: 1 GRPC call: /csi.v1.Identity/Probe
I0801 14:03:07.620241       1 utils.go:342] ID: 1 GRPC request: {}
I0801 14:03:07.620325       1 utils.go:348] ID: 1 GRPC response: {}
I0801 14:03:07.623366       1 utils.go:341] ID: 2 GRPC call: /csi.v1.Identity/GetPluginInfo
I0801 14:03:07.623410       1 utils.go:342] ID: 2 GRPC request: {}
I0801 14:03:07.623435       1 identityserver-default.go:40] ID: 2 Using default GetPluginInfo
I0801 14:03:07.623676       1 utils.go:348] ID: 2 GRPC response: {"name":"nvmeof.csi.ceph.com","vendor_version":"canary"}
I0801 14:03:07.624173       1 utils.go:341] ID: 3 GRPC call: /csi.v1.Identity/GetPluginCapabilities
I0801 14:03:07.624199       1 utils.go:342] ID: 3 GRPC request: {}
I0801 14:03:07.624416       1 utils.go:348] ID: 3 GRPC response: {"capabilities":[{"service":{"type":"CONTROLLER_SERVICE"}}]}
I0801 14:03:07.625003       1 utils.go:341] ID: 4 GRPC call: /csi.v1.Controller/ControllerGetCapabilities
I0801 14:03:07.625029       1 utils.go:342] ID: 4 GRPC request: {}
I0801 14:03:07.625055       1 controllerserver-default.go:43] ID: 4 Using default ControllerGetCapabilities
I0801 14:03:07.625151       1 utils.go:348] ID: 4 GRPC response: {"capabilities":[{"rpc":{"type":"CREATE_DELETE_VOLUME"}}]}
I0801 14:03:25.502157       1 utils.go:341] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 GRPC call: /csi.v1.Controller/CreateVolume
I0801 14:03:25.502315       1 utils.go:342] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 GRPC request: {"capacity_range":{"required_bytes":1073741824},"name":"pvc-857210c8-29a2-490c-8af1-fdc35ea492b2","parameters":{"clusterID":"openshift-storage","csi.storage.k8s.io/pv/name":"pvc-857210c8-29a2-490c-8af1-fdc35ea492b2","csi.storage.k8s.io/pvc/name":"nvmeof-test-pvc","csi.storage.k8s.io/pvc/namespace":"default","hostNQN":"nqn.2014-08.org.nvmexpress:uuid:950ddadf-f995-47b7-9416-b9bb233f66e3","imageFeatures":"layering,deep-flatten,exclusive-lock,object-map,fast-diff","imageFormat":"2","listenerHostname":"ceph-nvmeof-gateway.openshift-storage.svc.cluster.local","listenerIpAddress":"172.30.156.139","listenerPort":"4420","nvmeofGatewayAddress":"ceph-nvmeof-gateway.openshift-storage.svc.cluster.local","nvmeofGatewayPort":"5500","pool":"ocs-storagecluster-cephblockpool","subsystemNQN":"nqn.2016-06.io.spdk:cnode1"},"secrets":"***stripped***","volume_capabilities":[{"access_mode":{"mode":"SINGLE_NODE_WRITER"},"mount":{"fs_type":"ext4"}}]}
I0801 14:03:25.502788       1 rbd_util.go:1421] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 setting disableInUseChecks: false image features: [exclusive-lock deep-flatten layering object-map fast-diff] mounter: rbd
I0801 14:03:25.531614       1 omap.go:89] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 got omap values: (pool="ocs-storagecluster-cephblockpool", namespace="", name="csi.volumes.default"): map[csi.volume.pvc-857210c8-29a2-490c-8af1-fdc35ea492b2:705d2484-faef-4b3d-8162-c7f6655c7b1d]
I0801 14:03:25.542560       1 omap.go:89] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 got omap values: (pool="ocs-storagecluster-cephblockpool", namespace="", name="csi.volume.705d2484-faef-4b3d-8162-c7f6655c7b1d"): map[csi.imageid:79911503631e csi.imagename:csi-vol-705d2484-faef-4b3d-8162-c7f6655c7b1d csi.volname:pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 csi.volume.owner:default]
I0801 14:03:25.562946       1 rbd_journal.go:356] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 found existing volume (0001-0011-openshift-storage-0000000000000002-705d2484-faef-4b3d-8162-c7f6655c7b1d) with image name (csi-vol-705d2484-faef-4b3d-8162-c7f6655c7b1d) for request (pvc-857210c8-29a2-490c-8af1-fdc35ea492b2)
I0801 14:03:25.563023       1 controllerserver.go:96] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 RBD volume created successfully: pool=ocs-storagecluster-cephblockpool, image=csi-vol-705d2484-faef-4b3d-8162-c7f6655c7b1d, id=0001-0011-openshift-storage-0000000000000002-705d2484-faef-4b3d-8162-c7f6655c7b1d
I0801 14:03:25.563229       1 nvmeof.go:226] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 Checking if subsystem nqn.2016-06.io.spdk:cnode1 exists on gateway ceph-nvmeof-gateway.openshift-storage.svc.cluster.local:5500
I0801 14:03:25.592370       1 nvmeof.go:247] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 Subsystem nqn.2016-06.io.spdk:cnode1 exists
I0801 14:03:25.592401       1 nvmeof.go:118] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 Creating namespace for RBD ocs-storagecluster-cephblockpool/csi-vol-705d2484-faef-4b3d-8162-c7f6655c7b1d in subsystem nqn.2016-06.io.spdk:cnode1
E0801 14:03:25.996762       1 controllerserver.go:102] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 NVMe-oF resource setup failed for volume 0001-0011-openshift-storage-0000000000000002-705d2484-faef-4b3d-8162-c7f6655c7b1d: namespace creation failed: gateway NamespaceAdd returned error: Shutting down server
E0801 14:03:25.996785       1 utils.go:346] ID: 5 Req-ID: pvc-857210c8-29a2-490c-8af1-fdc35ea492b2 GRPC error: rpc error: code = Internal desc = NVMe-oF setup failed: namespace creation failed: gateway NamespaceAdd returned error: Shutting down server
...
```

To check the logs of a single CSI procedure, filter on a gRPC ID, like `ID: 5
Req-ID`. The log for a CSI procedure should start with a line containing `GRPC
call: /csi`, the last line should contain a `GRPC response` or `GRPC error`.
