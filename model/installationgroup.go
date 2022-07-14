// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/pkg/errors"
)

// InstallationGroup represents a provisioner installation group.
type InstallationGroup struct {
	ID                 string `json:"id,omitempty"`
	Name               string `json:"name,omitempty"`
	State              string `json:"state,omitempty"`
	ReleaseAt          int64  `json:"releaseAt,omitempty"`
	SoakTime           int    `json:"soakTime,omitempty"`
	ProvisionerGroupID string `json:"provisionerGroupID,omitempty"`
	LockAcquiredBy     *string
	LockAcquiredAt     int64
}

// RegisterInstallationGroupRequest represent parameters passed to register an installation group to the Ring.
type RegisterInstallationGroupRequest struct {
	Name               string `json:"name,omitempty"`
	SoakTime           int    `json:"soakTime,omitempty"`
	ProvisionerGroupID string `json:"provisionerGroupID,omitempty"`
}

// UpdateInstallationGroupRequest specifies the parameters to update an installation group.
type UpdateInstallationGroupRequest struct {
	Name               string `json:"name,omitempty"`
	SoakTime           int    `json:"soakTime,omitempty"`
	ProvisionerGroupID string `json:"provisionerGroupID,omitempty"`
}

// SortInstallationGroups sorts installation groups by name alphabetically.
func SortInstallationGroups(installationGroups []*InstallationGroup) []*InstallationGroup {
	sort.Slice(installationGroups, func(i, j int) bool {
		return installationGroups[i].Name < installationGroups[j].Name
	})
	return installationGroups
}

// NewRegisterInstallationGroupRequestFromReader will create a RegisterInstallationGroupRequest from an
// io.Reader with JSON data.
func NewRegisterInstallationGroupRequestFromReader(reader io.Reader) (*RegisterInstallationGroupRequest, error) {
	var registerInstallationGroupRequest RegisterInstallationGroupRequest
	err := json.NewDecoder(reader).Decode(&registerInstallationGroupRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode register installation group request")
	}

	return &registerInstallationGroupRequest, nil
}

// NewUpdateInstallationGroupRequestFromReader will create an UpdateRingRequest from an io.Reader with JSON data.
func NewUpdateInstallationGroupRequestFromReader(reader io.Reader) (*UpdateInstallationGroupRequest, error) {
	var updateInstallationGroupRequest UpdateInstallationGroupRequest
	err := json.NewDecoder(reader).Decode(&updateInstallationGroupRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode provision ring request")
	}
	return &updateInstallationGroupRequest, nil
}

// ContainsInstallationGroup determines whether slice of InstallationGroups contains a specific installation group.
func ContainsInstallationGroup(installationGroups []*InstallationGroup, installationGroup *InstallationGroup) bool {
	for _, ann := range installationGroups {
		if ann.ID == installationGroup.ID {
			return true
		}
	}
	return false
}

// InstallationGroupFromReader decodes a json-encoded ring from the given io.Reader.
func InstallationGroupFromReader(reader io.Reader) (*InstallationGroup, error) {
	installationGroup := InstallationGroup{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&installationGroup)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &installationGroup, nil
}
