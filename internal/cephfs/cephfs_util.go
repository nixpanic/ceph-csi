/*
Copyright 2019 The Ceph-CSI Authors.

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

package cephfs

import (
	"context"
	"fmt"

	"github.com/ceph/ceph-csi/internal/util"
)

// MDSMap is a representation of the mds map sub-structure returned by 'ceph fs get'.
type MDSMap struct {
	FilesystemName string `json:"fs_name"`
}

// CephFilesystemDetails is a representation of the main json structure returned by 'ceph fs get'.
type CephFilesystemDetails struct {
	ID     int64  `json:"id"`
	MDSMap MDSMap `json:"mdsmap"`
}

func (vo *volumeOptions) getFscID(ctx context.Context) (int64, error) {
	fsa, err := vo.conn.GetFSAdmin()
	if err != nil {
		util.ErrorLog(ctx, "could not get FSAdmin, can not fetch filesystem ID for %s:", vo.FsName, err)
		return 0, err
	}

	volumes, err := fsa.EnumerateVolumes()
	if err != nil {
		util.ErrorLog(ctx, "could not list volumes, can not fetch filesystem ID for %s:", vo.FsName, err)
		return 0, err
	}

	for _, vol := range volumes {
		if vol.Name == vo.FsName {
			return vol.ID, nil
		}
	}

	err = fmt.Errorf("volume %s not found", vo.FsName)
	util.ErrorLog(ctx, "failed o list volume %s: %s", vo.FsName, err)
	return 0, err
}

// CephFilesystem is a representation of the json structure returned by 'ceph fs ls'.
type CephFilesystem struct {
	Name           string   `json:"name"`
	MetadataPool   string   `json:"metadata_pool"`
	MetadataPoolID int      `json:"metadata_pool_id"`
	DataPools      []string `json:"data_pools"`
	DataPoolIDs    []int    `json:"data_pool_ids"`
}

func getMetadataPool(ctx context.Context, monitors string, cr *util.Credentials, fsName string) (string, error) {
	// ./tbox ceph fs ls --format=json
	// [{"name":"myfs","metadata_pool":"myfs-metadata","metadata_pool_id":4,...},...]
	var filesystems []CephFilesystem
	err := execCommandJSON(ctx, &filesystems,
		"ceph",
		"-m", monitors,
		"--id", cr.ID,
		"--keyfile="+cr.KeyFile,
		"-c", util.CephConfigPath,
		"fs", "ls", "--format=json",
	)
	if err != nil {
		return "", err
	}

	for _, fs := range filesystems {
		if fs.Name == fsName {
			return fs.MetadataPool, nil
		}
	}

	return "", fmt.Errorf("%w: fsName (%s) not found in Ceph cluster", util.ErrPoolNotFound, fsName)
}

// CephFilesystemDump is a representation of the main json structure returned by 'ceph fs dump'.
type CephFilesystemDump struct {
	Filesystems []CephFilesystemDetails `json:"filesystems"`
}

func getFsName(ctx context.Context, monitors string, cr *util.Credentials, fscID int64) (string, error) {
	// ./tbox ceph fs dump --format=json
	// JSON: {...,"filesystems":[{"mdsmap":{},"id":<n>},...],...}
	var fsDump CephFilesystemDump
	err := execCommandJSON(ctx, &fsDump,
		"ceph",
		"-m", monitors,
		"--id", cr.ID,
		"--keyfile="+cr.KeyFile,
		"-c", util.CephConfigPath,
		"fs", "dump", "--format=json",
	)
	if err != nil {
		return "", err
	}

	for _, fs := range fsDump.Filesystems {
		if fs.ID == fscID {
			return fs.MDSMap.FilesystemName, nil
		}
	}

	return "", fmt.Errorf("%w: fscID (%d) not found in Ceph cluster", util.ErrPoolNotFound, fscID)
}
