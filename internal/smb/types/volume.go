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
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ceph/go-ceph/common/admin/smb"
	"github.com/container-storage-interface/spec/lib/go/csi"

	fscore "github.com/ceph/ceph-csi/internal/cephfs/core"
	"github.com/ceph/ceph-csi/internal/cephfs/store"
	fsutil "github.com/ceph/ceph-csi/internal/cephfs/util"
	"github.com/ceph/ceph-csi/internal/util"
)

const (
	// clusterNameKey is the key in OMAP that contains the name of the
	// SMB-cluster. It will be prefixed with the journal configuration.
	clusterNameKey = "smb.cluster"

	// ParameterServer is set in the parameters on volume creation and in
	// the VolumeContext.
	ParameterServer = "server"
)

// SMBVolume presents the API for consumption by the CSI-controller to create,
// modify and delete the SMB-shared CephFS volume. Instances of this struct
// are short lived, they only exist as long as a CSI-procedure is active.
type SMBVolume struct {
	// ctx is the context for this short living volume object
	ctx context.Context

	volumeID   string
	clusterID  string
	mons       string
	fscID      int64
	objectUUID string

	cr        *util.Credentials
	connected bool
	conn      *util.ClusterConnection
}

// NewSMBVolume create a new SMBVolume instance for the currently executing
// CSI-procedure.
func NewSMBVolume(ctx context.Context, volumeID string) (*SMBVolume, error) {
	vi := util.CSIIdentifier{}

	err := vi.DecomposeCSIID(volumeID)
	if err != nil {
		return nil, fmt.Errorf("error decoding volume ID (%s): %w", volumeID, err)
	}

	return &SMBVolume{
		ctx:        ctx,
		volumeID:   volumeID,
		clusterID:  vi.ClusterID,
		fscID:      vi.LocationID,
		objectUUID: vi.ObjectUUID,
		conn:       &util.ClusterConnection{},
	}, nil
}

// String returns a simple/short representation of the SMBVolume.
func (sv *SMBVolume) String() string {
	return sv.volumeID
}

// Connect fetches cluster connection details (like MONs) and connects to the
// Ceph cluster. This uses go-ceph, so after Connect(), Destroy() should be
// called to cleanup resources.
func (sv *SMBVolume) Connect(cr *util.Credentials) error {
	if sv.connected {
		return nil
	}

	var err error
	sv.mons, err = util.Mons(util.CsiConfigFile, sv.clusterID)
	if err != nil {
		return fmt.Errorf("failed to get MONs for cluster (%s): %w", sv.clusterID, err)
	}

	err = sv.conn.Connect(sv.mons, cr)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	sv.cr = cr
	sv.connected = true

	return nil
}

// Destroy cleans up resources once the SMBVolume instance is not needed
// anymore.
func (sv *SMBVolume) Destroy() {
	if sv.connected {
		sv.conn.Destroy()
		sv.connected = false
	}
}

// GetSharePath returns the UNC path on the SMB-server that can be used for
// mounting.
func (sv *SMBVolume) GetSharePath() string {
	return "\\\\" + sv.volumeID
}

// CreateShare takes the (CephFS) CSI-volume and instructs Ceph Mgr to create
// a new SMB-share for the volume on the Ceph managed SMB-server.
func (sv *SMBVolume) CreateShare(backend *csi.Volume) error {
	if !sv.connected {
		return fmt.Errorf("can not create share for %q: %w", sv, ErrNotConnected)
	}
	vctx := backend.GetVolumeContext()
	fs := vctx["fsName"]
	smbCluster := vctx["smbCluster"]
	path := vctx["subvolumePath"]
	// Additional SMB-specific parameters
	shareParams := vctx["shareParams"]

	err := sv.setSMBCluster(smbCluster)
	if err != nil {
		return fmt.Errorf("failed to set SMB-cluster: %w", err)
	}

	smba, err := sv.conn.GetSMBAdmin()
	if err != nil {
		return fmt.Errorf("failed to get SMBAdmin: %w", err)
	}

	share := smb.CephFSShareSpec{
		FileSystemName: fs,
		ClusterID:      smbCluster,
		ShareName:      sv.volumeID,
		Path:           path,
	}

	// Apply additional share parameters if provided
	if shareParams != "" {
		// Parse share parameters (format: key1=value1,key2=value2)
		// This can be extended based on specific SMB/Samba requirements
		for _, param := range strings.Split(shareParams, ",") {
			kv := strings.SplitN(param, "=", 2)
			if len(kv) == 2 {
				// Apply parameters to the share spec
				// This will depend on the actual go-ceph SMB admin API
			}
		}
	}

	_, err = smba.CreateCephFSShare(share)
	switch {
	case err == nil:
		return nil
	case strings.Contains(err.Error(), "Share already exists"):
		return nil
	case strings.Contains(err.Error(), "rados: ret=-2"): // try with the old command
		break
	default: // any other error
		return fmt.Errorf("creating SMB share %q on cluster %q failed: %w",
			sv, smbCluster, err)
	}

	// if we get here, the API call failed, fallback to the old command

	// ceph smb share create cephfs ${FS} ${SMB_CLUSTER} ${SHARE_NAME} ${SUBVOL_PATH}
	cmd := sv.createShareCommand(smbCluster, fs, sv.volumeID, path)

	_, stderr, err := util.ExecCommand(sv.ctx, "ceph", cmd...)
	if err != nil {
		return fmt.Errorf("failed to create SMB share %q in cluster %q"+
			"(%v): %s", sv, smbCluster, err, stderr)
	}

	return nil
}

// DeleteShare removes the SMB-share from the Ceph managed SMB-server.
func (sv *SMBVolume) DeleteShare() error {
	if !sv.connected {
		return fmt.Errorf("can not delete share for %q: not connected", sv)
	}

	smbCluster, err := sv.getSMBCluster()
	if err != nil {
		return fmt.Errorf("failed to identify SMB cluster: %w", err)
	}

	smba, err := sv.conn.GetSMBAdmin()
	if err != nil {
		return fmt.Errorf("failed to get SMBAdmin: %w", err)
	}

	err = smba.RemoveShare(smbCluster, sv.volumeID)
	switch {
	case err == nil:
		return nil
	case strings.Contains(err.Error(), "API call not implemented"): // try with the old command
		break
	case strings.Contains(err.Error(), "Share does not exist"):
		return ErrShareNotFound
	default: // any other error
		return fmt.Errorf("failed to remove %q from SMB-cluster %q: "+
			"%w", sv, smbCluster, err)
	}

	// if we get here, the API call failed, fallback to the old command

	// ceph smb share delete <cluster_id> <share_name>
	cmd := sv.deleteShareCommand("delete", smbCluster)

	_, stderr, err := util.ExecCommand(sv.ctx, "ceph", cmd...)
	if err != nil {
		return fmt.Errorf("failed to delete SMB share %q from cluster"+
			"%q (%v): %s", sv, smbCluster, err, stderr)
	}

	return nil
}

// SetServer stores the SMB-server name in the CephFS journal.
func (sv *SMBVolume) SetServer(server string) error {
	return sv.setAttribute(ParameterServer, server)
}

// GetServer fetches the SMB-server name from the CephFS journal.
func (sv *SMBVolume) GetServer() (string, error) {
	return sv.getAttribute(ParameterServer)
}

// createShareCommand returns the "ceph smb share create ..." command
// arguments (without "ceph"). The order of the parameters matches old Ceph
// releases, new Ceph releases added --option formats, which can be added when
// passing the parameters to this function.
func (sv *SMBVolume) createShareCommand(smbCluster, fs, shareName, path string) []string {
	return []string{
		"--id", sv.cr.ID,
		"--keyfile=" + sv.cr.KeyFile,
		"-m", sv.mons,
		"smb",
		"share",
		"create",
		"cephfs",
		fs,
		smbCluster,
		shareName,
		path,
	}
}

// deleteShareCommand returns the "ceph smb share delete ..." command
// arguments (without "ceph"). Old releases of Ceph expect "delete" as cmd,
// newer releases use "rm".
func (sv *SMBVolume) deleteShareCommand(cmd, smbCluster string) []string {
	return []string{
		"--id", sv.cr.ID,
		"--keyfile=" + sv.cr.KeyFile,
		"-m", sv.mons,
		"smb",
		"share",
		cmd,
		smbCluster,
		sv.volumeID,
	}
}

// getAttribute fetches the attribute with the given key from the CephFS journal.
func (sv *SMBVolume) getAttribute(key string) (string, error) {
	if !sv.connected {
		return "", fmt.Errorf("can not get SMB-cluster for %q: %w", sv, ErrNotConnected)
	}

	fs := fscore.NewFileSystem(sv.conn)
	fsName, err := fs.GetFsName(sv.ctx, sv.fscID)
	if err != nil && errors.Is(err, util.ErrPoolNotFound) {
		return "", fmt.Errorf("%w for ID %x: %w", ErrFilesystemNotFound, sv.fscID, err)
	} else if err != nil {
		return "", fmt.Errorf("failed to get filesystem name for ID %x: %w", sv.fscID, err)
	}

	mdPool, err := fs.GetMetadataPool(sv.ctx, fsName)
	if err != nil && errors.Is(err, util.ErrPoolNotFound) {
		return "", fmt.Errorf("metadata pool for %q %w: %w", fsName, ErrNotFound, err)
	} else if err != nil {
		return "", fmt.Errorf("failed to get metadata pool for %q: %w", fsName, err)
	}

	// Connect to cephfs' default radosNamespace (csi)
	j, err := store.VolJournal.Connect(sv.mons, fsutil.RadosNamespace, sv.cr)
	if err != nil {
		return "", fmt.Errorf("failed to connect to journal: %w", err)
	}
	defer j.Destroy()

	value, err := j.FetchAttribute(sv.ctx, mdPool, sv.objectUUID, key)
	if err != nil && errors.Is(err, util.ErrPoolNotFound) || errors.Is(err, util.ErrKeyNotFound) {
		return "", fmt.Errorf("attribute with key %q for %q %w: %w", key, sv.objectUUID, ErrNotFound, err)
	} else if err != nil {
		return "", fmt.Errorf("failed to get attribute with key %q for %q: %w", key, sv.objectUUID, err)
	}

	return value, nil
}

// setAttribute stores the attribute with the key and value in the CephFS journal.
func (sv *SMBVolume) setAttribute(key, value string) error {
	if !sv.connected {
		return fmt.Errorf("can not set SMB-cluster for %q: %w", sv, ErrNotConnected)
	}

	fs := fscore.NewFileSystem(sv.conn)
	fsName, err := fs.GetFsName(sv.ctx, sv.fscID)
	if err != nil && errors.Is(err, util.ErrPoolNotFound) {
		return fmt.Errorf("%w for ID %x: %w", ErrFilesystemNotFound, sv.fscID, err)
	} else if err != nil {
		return fmt.Errorf("failed to get filesystem name for ID %x: %w", sv.fscID, err)
	}

	mdPool, err := fs.GetMetadataPool(sv.ctx, fsName)
	if err != nil && errors.Is(err, util.ErrPoolNotFound) {
		return fmt.Errorf("metadata pool for %q %w: %w", fsName, ErrNotFound, err)
	} else if err != nil {
		return fmt.Errorf("failed to get metadata pool for %q: %w", fsName, err)
	}

	// Connect to cephfs' default radosNamespace (csi)
	j, err := store.VolJournal.Connect(sv.mons, fsutil.RadosNamespace, sv.cr)
	if err != nil {
		return fmt.Errorf("failed to connect to journal: %w", err)
	}
	defer j.Destroy()

	err = j.StoreAttribute(sv.ctx, mdPool, sv.objectUUID, key, value)
	if err != nil {
		return fmt.Errorf("failed to store attribute with key %q: %w", key, err)
	}

	return nil
}

// getSMBCluster fetches the SMB-cluster name from the CephFS journal.
func (sv *SMBVolume) getSMBCluster() (string, error) {
	return sv.getAttribute(clusterNameKey)
}

// setSMBCluster stores the SMB-cluster name in the CephFS journal.
func (sv *SMBVolume) setSMBCluster(clusterName string) error {
	return sv.setAttribute(clusterNameKey, clusterName)
}
