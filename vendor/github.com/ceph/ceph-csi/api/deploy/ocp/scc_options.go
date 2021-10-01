/*
Copyright 2021 The Ceph-CSI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocp

import (
	"bytes"
	"text/template"

	"github.com/ghodss/yaml"
	secv1 "github.com/openshift/api/security/v1"
)

type SecurityContextConstraintsValues struct {
	Namespace string
	Prefix string
}

var SecurityContextConstraintsDefaults = SecurityContextConstraintsValues{
	Namespace: "ceph-csi",
	Prefix: "",
}

// Maybe use go:embed to include a .yaml file?
const securityContextConstraintsTemplate = `---
# scc for the CSI driver
kind: SecurityContextConstraints
apiVersion: security.openshift.io/v1
metadata:
  name: {{ .Prefix }}ceph-csi
# To allow running privilegedContainers
allowPrivilegedContainer: true
# CSI daemonset pod needs hostnetworking
allowHostNetwork: true
# This need to be set to true as we use HostPath
allowHostDirVolumePlugin: true
priority:
# SYS_ADMIN is needed for rbd to execture rbd map command
allowedCapabilities: ["SYS_ADMIN"]
# Needed as we run liveness container on daemonset pods
allowHostPorts: true
# Needed as we are setting this in RBD plugin pod
allowHostPID: true
# Required for encryption
allowHostIPC: true
# Set to false as we write to RootFilesystem inside csi containers
readOnlyRootFilesystem: false
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: RunAsAny
fsGroup:
  type: RunAsAny
supplementalGroups:
  type: RunAsAny
# The type of volumes which are mounted to csi pods
volumes:
  - configMap
  - projected
  - emptyDir
  - hostPath
users:
  # A user needs to be added for each rook service account.
  # This assumes running in the default sample "rook-ceph" namespace.
  # If other namespaces or service accounts are configured, they need to be updated here.
  - system:serviceaccount:{{ .Namespace }}:{{ .Prefix }}csi-rbd-plugin-sa # serviceaccount:namespace:operator
  - system:serviceaccount:{{ .Namespace }}:{{ .Prefix }}csi-rbd-provisioner-sa # serviceaccount:namespace:operator
  - system:serviceaccount:{{ .Namespace }}:{{ .Prefix }}csi-cephfs-plugin-sa # serviceaccount:namespace:operator
  - system:serviceaccount:{{ .Namespace }}:{{ .Prefix }}csi-cephfs-provisioner-sa # serviceaccount:namespace:operator`

func NewSecurityContextConstraints2(values SecurityContextConstraintsValues) *secv1.SecurityContextConstraints {
	var buf bytes.Buffer

	tmpl, err := template.New("SCC").Parse(securityContextConstraintsTemplate)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(&buf, values)
	if err != nil {
		panic(err)
	}

	scc := &secv1.SecurityContextConstraints{}
	err = yaml.Unmarshal(buf.Bytes(), scc)
	if err != nil {
		panic(err)
	}

	return scc
}
