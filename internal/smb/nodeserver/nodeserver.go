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

package nodeserver

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mount "k8s.io/mount-utils"
	netutil "k8s.io/utils/net"

	"github.com/ceph/ceph-csi/internal/cephfs/store"
	fsutil "github.com/ceph/ceph-csi/internal/cephfs/util"
	csicommon "github.com/ceph/ceph-csi/internal/csi-common"
	"github.com/ceph/ceph-csi/internal/journal"
	smb "github.com/ceph/ceph-csi/internal/smb/types"
	"github.com/ceph/ceph-csi/internal/util"
	"github.com/ceph/ceph-csi/internal/util/log"
)

const (
	defaultMountPermission = os.FileMode(0o777)
	// Address of the SMB server.
	paramShare     = "share"
	paramClusterID = "clusterID"
)

var errInvalidParameter = errors.New("invalid parameter")

// NodeServer struct of ceph CSI driver with supported methods of CSI
// node server spec.
type NodeServer struct {
	csicommon.DefaultNodeServer
}

// NewNodeServer initialize a node server for ceph CSI driver.
func NewNodeServer(
	d *csicommon.CSIDriver,
	t string,
) *NodeServer {
	store.VolJournal = journal.NewCSIVolumeJournalWithNamespace(d.GetInstanceID(), fsutil.RadosNamespace)

	return &NodeServer{
		DefaultNodeServer: *csicommon.NewDefaultNodeServer(d, t, "", map[string]string{}, map[string]string{}),
	}
}

// NodePublishVolume mount the volume.
func (ns *NodeServer) NodePublishVolume(
	ctx context.Context,
	req *csi.NodePublishVolumeRequest,
) (*csi.NodePublishVolumeResponse, error) {
	err := validateNodePublishVolumeRequest(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	volumeID := req.GetVolumeId()
	volCap := req.GetVolumeCapability()
	targetPath := req.GetTargetPath()
	mountOptions := volCap.GetMount().GetMountFlags()
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}

	source, err := getSource(ctx, req)
	if err != nil && errors.Is(errInvalidParameter, err) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = ns.mountSMB(ctx,
		volumeID,
		source,
		targetPath,
		mountOptions)
	if err != nil {
		if os.IsPermission(err) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		if strings.Contains(err.Error(), "invalid argument") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}
	log.DebugLog(ctx, "smb: successfully mounted volume %q mount %q to %q succeeded",
		volumeID, source, targetPath)

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmount the volume.
func (ns *NodeServer) NodeUnpublishVolume(
	ctx context.Context,
	req *csi.NodeUnpublishVolumeRequest,
) (*csi.NodeUnpublishVolumeResponse, error) {
	err := util.ValidateNodeUnpublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	log.DebugLog(ctx, "smb: unmounting volume %s on %s", volumeID, targetPath)
	err = mount.CleanupMountPoint(targetPath, ns.Mounter, true)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmount target %q: %v",
			targetPath, err)
	}
	log.DebugLog(ctx, "smb: successfully unbounded volume %q from %q",
		volumeID, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetCapabilities returns the supported capabilities of the node server.
func (ns *NodeServer) NodeGetCapabilities(
	ctx context.Context,
	req *csi.NodeGetCapabilitiesRequest,
) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
					},
				},
			},
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER,
					},
				},
			},
		},
	}, nil
}

// NodeGetVolumeStats get volume stats.
func (ns *NodeServer) NodeGetVolumeStats(
	ctx context.Context,
	req *csi.NodeGetVolumeStatsRequest,
) (*csi.NodeGetVolumeStatsResponse, error) {
	var err error
	targetPath := req.GetVolumePath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument,
			fmt.Sprintf("targetpath %v is empty", targetPath))
	}

	stat, err := os.Stat(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			"failed to get stat for targetpath %q: %v", targetPath, err)
	}

	if !stat.Mode().IsDir() {
		return nil, status.Errorf(codes.InvalidArgument,
			"targetpath %q is not a directory or device", targetPath)
	}

	return csicommon.FilesystemNodeGetVolumeStats(ctx, ns.Mounter, targetPath, false)
}

// mountSMB mounts SMB/CIFS volumes.
func (ns *NodeServer) mountSMB(
	ctx context.Context,
	volumeID, source, mountPoint string,
	mountOptions []string,
) error {
	var err error

	isMnt, err := ns.Mounter.IsMountPoint(mountPoint)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(mountPoint, defaultMountPermission)
			if err != nil {
				return err
			}
			isMnt = false
		} else {
			return err
		}
	}
	if isMnt {
		log.DebugLog(ctx, "smb: volume is already mounted to %s", mountPoint)

		return nil
	}

	log.DefaultLog("smb: mounting volumeID(%v) source(%s) targetPath(%s) mountflags(%v)",
		volumeID, source, mountPoint, mountOptions)

	err = ns.Mounter.Mount(source, mountPoint, "cifs", mountOptions)
	if err != nil {
		return fmt.Errorf("smb: failed to mount %q to %q : %w",
			source, mountPoint, err)
	}

	return err
}

// validateNodePublishVolumeRequest validates node publish volume request.
func validateNodePublishVolumeRequest(req *csi.NodePublishVolumeRequest) error {
	if err := util.ValidateVolumeID(req.GetVolumeId(), util.IsStaticVol(req.GetVolumeContext())); err != nil {
		return err
	}

	switch {
	case req.GetVolumeCapability() == nil:
		return errors.New("volume capability missing in request")
	case req.GetTargetPath() == "":
		return errors.New("target path missing in request")
	}

	return nil
}

// getSource validates volume context, extracts and returns source.
// This function expects `server` and `share` parameters to be set
// and validates for the same.
func getSource(ctx context.Context, req *csi.NodePublishVolumeRequest) (string, error) {
	volContext := req.GetVolumeContext()

	// default server from the VolumeContext, updated server from journal
	server, err := getServerFromVolume(ctx, req)
	if err != nil {
		log.ErrorLog(ctx, "failed to get server from volume: %v", err)

		return "", err
	} else if server == "" {
		server = volContext[smb.ParameterServer]
		if server == "" {
			return "", fmt.Errorf("%w: %q missing in request", errInvalidParameter, smb.ParameterServer)
		}
	}

	baseDir := volContext[paramShare]
	if baseDir == "" {
		return "", fmt.Errorf("%v missing in request", paramShare)
	}

	if netutil.IsIPv6String(server) {
		// if server is IPv6, format to [IPv6].
		server = fmt.Sprintf("[%s]", server)
	}

	// SMB UNC path format: //server/share
	return fmt.Sprintf("//%s%s", server, baseDir), nil
}

func getServerFromVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (string, error) {
	secrets := req.GetSecrets()
	if secrets == nil {
		// no secrets, continue as if not set in metadata
		return "", nil
	}
	cr, err := util.NewAdminCredentials(secrets)
	if err != nil {
		log.ErrorLog(ctx, "failed to retrieve user credentials: %v", err)

		// invalid secrets, continue as if not set in metadata
		return "", nil
	}
	defer cr.DeleteCredentials()

	smbVolume, err := smb.NewSMBVolume(ctx, req.GetVolumeId())
	if err != nil {
		return "", fmt.Errorf("failed to instantiate volume with ID %q: %w", req.GetVolumeId(), err)
	}

	err = smbVolume.Connect(cr)
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer smbVolume.Destroy()

	server, err := smbVolume.GetServer()
	if err != nil && !errors.Is(err, smb.ErrNotFound) {
		return "", fmt.Errorf("failed to get server for volume with ID %q: %w", req.GetVolumeId(), err)
	}

	return server, nil
}
