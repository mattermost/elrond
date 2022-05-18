// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

// Ring represents a deployment ring.
type Ring struct {
	ID                 string
	Name               string
	Priority           int
	SoakTime           int
	Image              string
	Version            string
	State              string
	Provisioner        string
	CreateAt           int64
	DeleteAt           int64
	ReleaseAt          int64
	InstallationGroups []*InstallationGroup `json:"InstallationGroups,omitempty"`
	APISecurityLock    bool
	LockAcquiredBy     *string
	LockAcquiredAt     int64
}

// RingRelease stores information neeeded for a ring release.
type RingRelease struct {
	Image   string
	Version string
}

// Clone returns a deep copy the ring.
func (a *Ring) Clone() (*Ring, error) {
	var clone Ring
	data, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(data, &clone); err != nil {
		return nil, err
	}

	return &clone, nil
}

// RingFromReader decodes a json-encoded ring from the given io.Reader.
func RingFromReader(reader io.Reader) (*Ring, error) {
	ring := Ring{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&ring)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &ring, nil
}

// RingsFromReader decodes a json-encoded list of rings from the given io.Reader.
func RingsFromReader(reader io.Reader) ([]*Ring, error) {
	rings := []*Ring{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&rings)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return rings, nil
}

// RingFilter describes the parameters used to constrain a set of rings.
type RingFilter struct {
	Paging
	InstallationGroups *InstallationGroupsFilter
	Page               int
	PerPage            int
	IncludeDeleted     bool
}

// InstallationGroupsFilter describes filter based on Installation Groups.
type InstallationGroupsFilter struct {
	// MatchAllIDs contains all Installation Group IDs which need to be set on a Ring for it to be included in the result.
	MatchAllIDs []string
}
