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
	"fmt"

	secv1 "github.com/openshift/api/security/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewSecurityContextConstraints takes a namespace and the name of the deployer
// (like Rook) and returns a SecurityContextConstraints object that can be
// deployed on OpenShift.
//
// The deployer parameter (when not an empty string) is used as a prefix for
// the name of the SCC and the linked ServiceAccounts.
func NewSecurityContextConstraints(namespace, deployer string) *secv1.SecurityContextConstraints {
	scc := &secv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.openshift.io/v1",
			Kind:       "SecurityContextConstraints",
		},
	}
	prefix := ""

	if deployer != "" {
		prefix = deployer + "-"
	}

	scc.Name = fmt.Sprintf("%sceph-csi", prefix)
	scc.AllowPrivilegedContainer = true
	scc.AllowHostNetwork = true
	scc.AllowHostDirVolumePlugin = true
	scc.AllowedCapabilities = []corev1.Capability{
		secv1.AllowAllCapabilities,
	}
	scc.AllowHostPorts = true
	scc.AllowHostPID = true
	scc.AllowHostIPC = true
	scc.ReadOnlyRootFilesystem = false
	scc.RequiredDropCapabilities = []corev1.Capability{}
	scc.DefaultAddCapabilities = []corev1.Capability{}
	scc.RunAsUser = secv1.RunAsUserStrategyOptions{
		Type: secv1.RunAsUserStrategyRunAsAny,
	}
	scc.SELinuxContext = secv1.SELinuxContextStrategyOptions{
		Type: secv1.SELinuxStrategyRunAsAny,
	}
	scc.FSGroup = secv1.FSGroupStrategyOptions{
		Type: secv1.FSGroupStrategyRunAsAny,
	}
	scc.SupplementalGroups = secv1.SupplementalGroupsStrategyOptions{
		Type: secv1.SupplementalGroupsStrategyRunAsAny,
	}
	scc.Volumes = []secv1.FSType{
		secv1.FSTypeAll,
	}
	scc.Users = []string{
		fmt.Sprintf("system:serviceaccount:%s:%scsi-rbd-plugin-sa", namespace, prefix),
		fmt.Sprintf("system:serviceaccount:%s:%scsi-rbd-provisioner-sa", namespace, prefix),
		fmt.Sprintf("system:serviceaccount:%s:%scsi-rbd-attacher-sa", namespace, prefix),
		fmt.Sprintf("system:serviceaccount:%s:%scsi-cephfs-plugin-sa", namespace, prefix),
		fmt.Sprintf("system:serviceaccount:%s:%scsi-cephfs-provisioner-sa", namespace, prefix),
	}

	return scc
}
