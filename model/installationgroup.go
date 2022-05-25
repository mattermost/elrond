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
	ID   string
	Name string
}

// RegisterInstallationGroupsRequest represent parameters passed to register set of installation groups to the Ring.
type RegisterInstallationGroupsRequest struct {
	InstallationGroups []string `json:"installationGroups"`
}

// InstallationGroupsFromStringSlice converts list of strings to list of installation groups.
func InstallationGroupsFromStringSlice(names []string) ([]*InstallationGroup, error) {
	if names == nil {
		return nil, nil
	}

	installationGroups := make([]*InstallationGroup, 0, len(names))
	for _, n := range names {
		installationGroups = append(installationGroups, &InstallationGroup{Name: n})
	}

	return installationGroups, nil
}

// SortInstallationGroups sorts installation groups by name alphabetically.
func SortInstallationGroups(installationGroups []*InstallationGroup) []*InstallationGroup {
	sort.Slice(installationGroups, func(i, j int) bool {
		return installationGroups[i].Name < installationGroups[j].Name
	})
	return installationGroups
}

// NewRegisterInstallationGroupsRequestFromReader will create a RegisterInstallationGroupsRequest from an
// io.Reader with JSON data.
func NewRegisterInstallationGroupsRequestFromReader(reader io.Reader) (*RegisterInstallationGroupsRequest, error) {
	var registerInstallationGroupsRequest RegisterInstallationGroupsRequest
	err := json.NewDecoder(reader).Decode(&registerInstallationGroupsRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode register installation groups request")
	}

	return &registerInstallationGroupsRequest, nil
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
