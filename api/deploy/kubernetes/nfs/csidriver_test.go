/*
Copyright 2022 The Ceph-CSI Authors.

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

package nfs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCSIDriver(t *testing.T) {
	driver, err := NewCSIDriver(CSIDriverDefaults)

	require.NoError(t, err)
	require.NotNil(t, driver)
	require.Equal(t, driver.Name, CSIDriverDefaults.Name)
}

func TestNewCSIDriverYAML(t *testing.T) {
	yaml, err := NewCSIDriverYAML(CSIDriverDefaults)

	require.NoError(t, err)
	require.NotEqual(t, "", yaml)
}
