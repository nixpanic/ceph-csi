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

package rbd

import (
	"testing"

	"github.com/ceph/ceph-csi/internal/util"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsThickProvisionRequest(t *testing.T) {
	cs := &ControllerServer{}
	req := &csi.CreateVolumeRequest{
		Name: "fake",
		Parameters: map[string]string{
			"unkownOption": "not-set",
		},
	}

	// pass disabled/invalid values for "thickProvision" option
	if cs.isThickProvisionRequest(req) {
		t.Error("request is not for thick-provisioning")
	}

	req.Parameters["thickProvision"] = ""
	if cs.isThickProvisionRequest(req) {
		t.Errorf("request is not for thick-provisioning: %s", req.Parameters["thickProvision"])
	}

	req.Parameters["thickProvision"] = "false"
	if cs.isThickProvisionRequest(req) {
		t.Errorf("request is not for thick-provisioning: %s", req.Parameters["thickProvision"])
	}

	req.Parameters["thickProvision"] = "off"
	if cs.isThickProvisionRequest(req) {
		t.Errorf("request is not for thick-provisioning: %s", req.Parameters["thickProvision"])
	}

	req.Parameters["thickProvision"] = "no"
	if cs.isThickProvisionRequest(req) {
		t.Errorf("request is not for thick-provisioning: %s", req.Parameters["thickProvision"])
	}

	req.Parameters["thickProvision"] = "**true**"
	if cs.isThickProvisionRequest(req) {
		t.Errorf("request is not for thick-provisioning: %s", req.Parameters["thickProvision"])
	}

	// only "true" should enable thick provisioning
	req.Parameters["thickProvision"] = "true"
	if !cs.isThickProvisionRequest(req) {
		t.Errorf("request should be for thick-provisioning: %s", req.Parameters["thickProvision"])
	}
}

func TestCheckValidEncryptionRequest(t *testing.T) {
	secrets := map[string]string{
		"encryptionPassphrase": "workflow test",
	}

	kms, err := util.GetKMS("tenant", "", secrets)
	assert.NoError(t, err)
	require.NotNil(t, kms)

	ve, err := util.NewVolumeEncryption("", kms)

	type args struct {
		dest    *rbdVolume
		src     *rbdVolume
		rbdSnap *rbdSnapshot
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Parent snapshot is encrypted and request volume is not encrypted",
			args: args{
				dest: &rbdVolume{
					rbdImage: rbdImage{encryption: nil},
				},
				src: nil,
				rbdSnap: &rbdSnapshot{
					rbdImage: rbdImage{encryption: ve},
				},
			},
			wantErr: true,
		},
		{
			name: "Parent snapshot is not encrypted and request volume is encrypted",
			args: args{
				dest: &rbdVolume{
					rbdImage: rbdImage{encryption: ve},
				},
				src: nil,
				rbdSnap: &rbdSnapshot{
					rbdImage: rbdImage{encryption: nil},
				},
			},
			wantErr: true,
		},
		{
			name: "Parent snapshot is encrypted and request volume is also encrypted",
			args: args{
				dest: &rbdVolume{
					rbdImage: rbdImage{encryption: ve},
				},
				src: nil,
				rbdSnap: &rbdSnapshot{
					rbdImage: rbdImage{encryption: ve},
				},
			},
			wantErr: false,
		},
		{
			name: "Parent snapshot is not encrypted and request volume is also not encrypted",
			args: args{
				dest: &rbdVolume{
					rbdImage: rbdImage{encryption: nil},
				},
				src: nil,
				rbdSnap: &rbdSnapshot{
					rbdImage: rbdImage{encryption: nil},
				},
			},
			wantErr: false,
		},

		{
			name: "Parent volume is encrypted and request volume is not encrypted",
			args: args{
				dest: &rbdVolume{
					rbdImage: rbdImage{encryption: nil},
				},
				src: &rbdVolume{
					rbdImage: rbdImage{encryption: ve},
				},
				rbdSnap: nil,
			},
			wantErr: true,
		},
		{
			name: "Parent snapshot is not encrypted and request volume is encrypted",
			args: args{
				dest: &rbdVolume{
					rbdImage: rbdImage{encryption: ve},
				},
				src: &rbdVolume{
					rbdImage: rbdImage{encryption: nil},
				},
				rbdSnap: nil,
			},
			wantErr: true,
		},
		{
			name: "Parent snapshot is encrypted and request volume is also encrypted",
			args: args{
				dest: &rbdVolume{
					rbdImage: rbdImage{encryption: ve},
				},
				src: &rbdVolume{
					rbdImage: rbdImage{encryption: ve},
				},
				rbdSnap: nil,
			},
			wantErr: false,
		},
		{
			name: "Parent snapshot is not encrypted and request volume is also not encrypted",
			args: args{
				dest: &rbdVolume{
					rbdImage: rbdImage{encryption: nil},
				},
				src: &rbdVolume{
					rbdImage: rbdImage{encryption: nil},
				},
				rbdSnap: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		et := tt
		t.Run(et.name, func(t *testing.T) {
			if err = checkValidEncryptionRequest(et.args.src, et.args.dest, et.args.rbdSnap); (err != nil) != et.wantErr {
				t.Errorf("checkValidEncryptionRequest() error = %v, wantErr %v", err, et.wantErr)
			}
		})
	}
}
