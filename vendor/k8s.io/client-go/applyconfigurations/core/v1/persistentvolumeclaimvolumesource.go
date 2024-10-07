/*
Copyright The Kubernetes Authors.

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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// PersistentVolumeClaimVolumeSourceApplyConfiguration represents a declarative configuration of the PersistentVolumeClaimVolumeSource type for use
// with apply.
type PersistentVolumeClaimVolumeSourceApplyConfiguration struct {
	ClaimName *string `json:"claimName,omitempty"`
	ReadOnly  *bool   `json:"readOnly,omitempty"`
}

// PersistentVolumeClaimVolumeSourceApplyConfiguration constructs a declarative configuration of the PersistentVolumeClaimVolumeSource type for use with
// apply.
func PersistentVolumeClaimVolumeSource() *PersistentVolumeClaimVolumeSourceApplyConfiguration {
	return &PersistentVolumeClaimVolumeSourceApplyConfiguration{}
}

// WithClaimName sets the ClaimName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClaimName field is set to the value of the last call.
func (b *PersistentVolumeClaimVolumeSourceApplyConfiguration) WithClaimName(value string) *PersistentVolumeClaimVolumeSourceApplyConfiguration {
	b.ClaimName = &value
	return b
}

// WithReadOnly sets the ReadOnly field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ReadOnly field is set to the value of the last call.
func (b *PersistentVolumeClaimVolumeSourceApplyConfiguration) WithReadOnly(value bool) *PersistentVolumeClaimVolumeSourceApplyConfiguration {
	b.ReadOnly = &value
	return b
}