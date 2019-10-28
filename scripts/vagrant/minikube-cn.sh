#!/bin/bash
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

# TODO: build ceph-csi container and push to local docker registry
# needs sudo as it pushes the images to the (docker) registry
make image-cephcsi CONTAINER_CMD='sudo docker'

mkdir -p $GOPATH/src/github.com/ceph
cd $GOPATH/src/github.com/ceph
git clone https://github.com/ceph/cn
cd cn
make
install -D cn $HOME/bin/cn

mkdir -p $GOPATH/src/github.com/kubernetes-csi
cd $GOPATH/src/github.com/kubernetes-csi
git clone https://github.com/kubernetes-csi/csi-test
cd csi-test
make build-sanity
install -D cmd/csi-sanity/csi-sanity $HOME/bin/csi-sanity

###
### Installation of tools finished, start deployment
###

# when CRI-O is used, pass --container-runtime=cri-o
sudo /usr/local/bin/minikube start --vm-driver=none

# download kubectl and setup access for local user
KUBE_VERSION=$(sudo /usr/local/bin/minikube kubectl version -- --client -o yaml | awk '/gitVersion:/{print $2}')
sudo cp /root/.minikube/cache/${KUBE_VERSION}/kubectl /usr/local/bin/
sudo cp -r /root/.kube /root/.minikube $HOME
sudo chown $USER -R $HOME/.kube $HOME/.minikube
sed "s|/root/|$HOME/|g" -i $HOME/.kube/config
kubectl version

# show the version, might dump some non-yaml to stdout
cn version
cn kube > cn.yaml
sed -i 's/memory: 512M/memory: 1024M/g' cn.yaml
kubectl apply -f cn.yaml

# need to wait until everything is ready
while ! kubectl exec -t ceph-nano-0 -- /usr/bin/ceph status
do
	sleep 10
done

cd ${GOPATH}/src/github.com/ceph/ceph-csi/examples/rbd
sed 's/<plaintext ID>/admin/' -i secret.yaml
ADMIN_KEY=$(kubectl exec -t ceph-nano-0 -- /usr/bin/ceph auth get client.admin | awk '/key =/{print $3}')
sed "s|<Ceph auth key corresponding to ID above>|${ADMIN_KEY}|" -i secret.yaml

kubectl apply -f secret.yaml

cd ${GOPATH}/src/github.com/ceph/ceph-csi/examples/rbd
. plugin-deploy.sh
# plugin-deploy.sh changes working dir :-/
cd ${GOPATH}/src/github.com/ceph/ceph-csi/examples/rbd

# need to get the configuration of the Ceph cluster
CLUSTER_ID=$(kubectl exec -t ceph-nano-0 -- /usr/bin/ceph status | awk '/id:/{print $2}' | tr -d '\r')
# single mon is on the ceph-nano pod
MON_IP=$(kubectl get pod/ceph-nano-0 --template='{{.status.podIP}}')
MON_PORT='3300'

# based on ceph-csi/examples/csi-config-map-sample.yaml
cat << EOF > csi-config-map.yaml
---
apiVersion: v1
kind: ConfigMap
data:
  config.json: |-
    [
      {
        "clusterID": "${CLUSTER_ID}",
        "monitors": [
          "${MON_IP}:${MON_PORT}"
        ]
      }
    ]
metadata:
  name: ceph-csi-config
EOF

kubectl replace -f csi-config-map.yaml
kubectl create -f storageclass.yaml

# csi-sanity needs its own secrets file
cat << EOF > csi-sanity-secrets.yaml
CreateVolumeSecret:
  userID: admin
  userKey: ${ADMIN_KEY}
DeleteVolumeSecret:
  userID: admin
  userKey: ${ADMIN_KEY}
ControllerPublishVolumeSecret:
  userID: admin
  userKey: ${ADMIN_KEY}
ControllerUnpublishVolumeSecret:
  userID: admin
  userKey: ${ADMIN_KEY}
NodeStageVolumeSecret:
  userID: admin
  userKey: ${ADMIN_KEY}
NodePublishVolumeSecret:
  userID: admin
  userKey: ${ADMIN_KEY}
CreateSnapshotSecret:
  userID: admin
  userKey: ${ADMIN_KEY}
DeleteSnapshotSecret:
  userID: admin
  userKey: ${ADMIN_KEY}
ControllerValidateVolumeCapabilitiesSecret:
  userID: admin
  userKey: ${ADMIN_KEY}
EOF

cat << EOF > csi-sanity-parameters.yaml
clusterID: ${CLUSTER_ID}
monitors: ${MON_IP}:${MON_PORT}
pool: rbd
#dataPool: 
imageFeatures: layering
#mounter: rbd
EOF

# copy /usr/local/bin/csi-sanity and secrets to csi-rbdplugin pod(s)
CSI_PROVISIONER_POD=$(kubectl get pods -l app=csi-rbdplugin-provisioner -ojsonpath='{.items[0].metadata.name}')

# the provisioner may not be ready in time?
STATUS_PHASE='Unknown'
while [[ "${STATUS_PHASE}" != 'Running' ]]
do
	sleep 10
	STATUS_PHASE=$(kubectl get pods -l app=csi-rbdplugin-provisioner -ojsonpath='{.items[0].status.phase}')
done

tar c csi-sanity-secrets.yaml csi-sanity-parameters.yaml ${HOME}/bin/csi-sanity | kubectl exec -i -c csi-rbdplugin ${CSI_PROVISIONER_POD} -- tar x -C /tmp

# finally run the csi-sanity tests
if ! kubectl exec -t -c csi-rbdplugin ${CSI_PROVISIONER_POD} -- /tmp/$HOME/bin/csi-sanity --csi.endpoint=/csi/csi-provisioner.sock --csi.secrets=/tmp/csi-sanity-secrets.yaml --csi.testvolumeparameters=/tmp/csi-sanity-parameters.yaml -ginkgo.failFast
then
	echo "sometimes logs have not been flushed yet, waiting 30 seconds..."
	sleep 30
	kubectl logs -c csi-rbdplugin ${CSI_PROVISIONER_POD}
	exit 1
fi

