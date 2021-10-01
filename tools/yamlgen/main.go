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

package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/ceph/ceph-csi/api/deploy/kubernetes/rbd"
	"github.com/ceph/ceph-csi/api/deploy/ocp"
)

const header = `---
#
# /!\ DO NOT MODIFY THIS FILE
#
# This file has been automatically generated by Ceph-CSI yamlgen.
# The source for the contents can be found in the api/deploy directory, make
# your modifications there.
#
`

type deploymentArtifact struct {
	filename string
	yamlFunc reflect.Value
	defaults reflect.Value
}

var yamlArtifacts = []deploymentArtifact{
	{
		"../deploy/scc.yaml",
		reflect.ValueOf(ocp.NewSecurityContextConstraintsYAML),
		reflect.ValueOf(ocp.SecurityContextConstraintsDefaults),
	},
	{
		"../deploy/rbd/kubernetes/csidriver.yaml",
		reflect.ValueOf(rbd.NewCSIDriverYAML),
		reflect.ValueOf(rbd.CSIDriverDefaults),
	},
}

func main() {
	for _, artifact := range yamlArtifacts {
		writeArtifact(artifact)
	}
}

func writeArtifact(artifact deploymentArtifact) {
	fmt.Printf("creating %q...", artifact.filename)

	f, err := os.Create(artifact.filename)
	if err != nil {
		panic(fmt.Sprintf("failed to create file %q: %v", artifact.filename, err))
	}

	_, err = f.WriteString(header)
	if err != nil {
		panic(fmt.Sprintf("failed to write header to %q: %v", artifact.filename, err))
	}

	result := artifact.yamlFunc.Call([]reflect.Value{artifact.defaults})
	data := result[0].String()
	if data == "" {
		panic(fmt.Sprintf("failed to generate YAML for %q: %v", artifact.filename, result[1].String()))
	}

	_, err = f.WriteString(data)
	if err != nil {
		panic(fmt.Sprintf("failed to write contents to %q: %v", artifact.filename, err))
	}

	fmt.Println("done!")
}
