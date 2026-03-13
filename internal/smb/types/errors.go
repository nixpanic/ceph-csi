/*
Copyright 2026 The Ceph-CSI Authors.

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

package types

import (
	"errors"
	"fmt"
)

var (
	// ErrNotConnected is returned when components from the SMB
	// ControllerServer can not connect to the Ceph cluster or SMB/Samba
	// service.
	ErrNotConnected = errors.New("not connected")

	// ErrNotFound is a generic error that is the parent of other "not
	// found" failures. Callers can check if something was "not found",
	// even if the actual error is more specific.
	ErrNotFound = errors.New("not found")

	// ErrShareNotFound is returned by components that communicate with the
	// SMB/Samba service, and have identified that the SMB-Share does
	// not exist (anymore). This error is also a ErrNotFound.
	ErrShareNotFound = fmt.Errorf("SMB-share %w", ErrNotFound)

	// ErrFilesystemNotFound is returned in case the filesystem
	// does not exist.
	ErrFilesystemNotFound = fmt.Errorf("filesystem %w", ErrNotFound)
)
