/*
Copyright 2024 The Ceph-CSI Authors.

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
	"context"
	"errors"
	"fmt"

	"github.com/ceph/ceph-csi/internal/journal"
	"github.com/ceph/ceph-csi/internal/rbd/types"
	group "github.com/ceph/ceph-csi/internal/rbd_group"
	"github.com/ceph/ceph-csi/internal/util"
)

var _ types.Manager = &rbdManager{}

type rbdManager struct {
	csiID      string
	parameters map[string]string
	secrets    map[string]string

	creds     *util.Credentials
	vgJournal journal.VolumeGroupJournal
}

// NewManager returns a new manager for handling Volume and Volume Group
// operations, combining the requests for RBD and the journalling in RADOS.
func NewManager(csiID string, parameters, secrets map[string]string) types.Manager {
	return &rbdManager{
		csiID:      csiID,
		parameters: parameters,
		secrets:    secrets,
	}
}

func (mgr *rbdManager) Destroy(ctx context.Context) {
	if mgr.creds != nil {
		mgr.creds.DeleteCredentials()
		mgr.creds = nil
	}

	if mgr.vgJournal != nil {
		mgr.vgJournal.Destroy()
		mgr.vgJournal = nil
	}
}

// getCredentials sets up credentials and connects to the journal.
func (mgr *rbdManager) getCredentials() (*util.Credentials, error) {
	if mgr.creds != nil {
		return mgr.creds, nil
	}

	creds, err := util.NewUserCredentials(mgr.secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	mgr.creds = creds

	return creds, nil
}

func (mgr *rbdManager) getVolumeGroupJournal(clusterID string) (journal.VolumeGroupJournal, error) {
	if mgr.vgJournal != nil {
		return mgr.vgJournal, nil
	}

	creds, err := mgr.getCredentials()
	if err != nil {
		return nil, err
	}

	monitors, err := util.Mons(util.CsiConfigFile, clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to find MONs for cluster %q: %w", clusterID, err)
	}

	ns, err := util.GetRadosNamespace(util.CsiConfigFile, clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to find the RADOS namespace for cluster %q: %w", clusterID, err)
	}

	vgJournalConfig := journal.NewCSIVolumeGroupJournalWithNamespace(mgr.csiID, ns)

	vgJournal, err := vgJournalConfig.Connect(monitors, ns, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to journal: %w", err)
	}

	mgr.vgJournal = vgJournal

	return vgJournal, nil
}

func (mgr *rbdManager) GetVolumeByID(ctx context.Context, id string) (types.Volume, error) {
	creds, err := mgr.getCredentials()
	if err != nil {
		return nil, err
	}

	volume, err := GenVolFromVolID(ctx, id, creds, mgr.secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume from id %q: %w", id, err)
	}

	return volume, nil
}

func (mgr *rbdManager) GetVolumeGroupByID(ctx context.Context, id string) (types.VolumeGroup, error) {
	vi := &util.CSIIdentifier{}
	if err := vi.DecomposeCSIID(id); err != nil {
		return nil, fmt.Errorf("failed to parse volume group id %q: %w", id, err)
	}

	vgJournal, err := mgr.getVolumeGroupJournal(vi.ClusterID)
	if err != nil {
		return nil, err
	}

	creds, err := mgr.getCredentials()
	if err != nil {
		return nil, err
	}

	vg, err := group.GetVolumeGroup(ctx, id, vgJournal, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume group with id %q: %w", id, err)
	}

	return vg, nil
}

func (mgr *rbdManager) CreateVolumeGroup(ctx context.Context, name string) (types.VolumeGroup, error) {
	creds, err := mgr.getCredentials()
	if err != nil {
		return nil, err
	}

	clusterID, err := util.GetClusterID(mgr.parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster-id: %w", err)
	}

	vgJournal, err := mgr.getVolumeGroupJournal(clusterID)
	if err != nil {
		return nil, err
	}

	// pool is a required parameter
	pool, ok := mgr.parameters["pool"]
	if !ok {
		return nil, errors.New("required 'pool' parameter missing in volume attributes")
	}

	// journalPool is an optional parameter, use pool if it is not set
	journalPool, ok := mgr.parameters["journalPool"]
	if !ok {
		journalPool = pool
	}

	// volumeNamePrefix is an optional parameter, can be an empty string
	prefix := mgr.parameters["volumeNamePrefix"]

	uuid, vgName, err := vgJournal.ReserveName(ctx, journalPool, name, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve volume group for name %q: %w", name, err)
	}

	monitors, err := util.Mons(util.CsiConfigFile, clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to find MONs for cluster %q: %w", clusterID, err)
	}

	_ /*journalPoolID*/, poolID, err := util.GetPoolIDs(ctx, monitors, journalPool, pool, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a unique CSI volume group id for %q: %w", vgName, err)
	}

	csiID, err := util.GenerateVolID(ctx, monitors, creds, poolID, pool, clusterID, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a unique CSI volume group id for %q: %w", vgName, err)
	}

	vg, err := group.GetVolumeGroup(ctx, csiID, vgJournal, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume group %q at cluster %q: %w", name, clusterID, err)
	}

	err = vg.Create(ctx, vgName)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume group %q: %w", name, err)
	}

	// TODO: create the volume group in the journal

	return vg, nil
}

func (mgr *rbdManager) DeleteVolumeGroup(ctx context.Context, vg types.VolumeGroup) error {
	// TODO: remove from journal
	return vg.Delete(ctx)
}
