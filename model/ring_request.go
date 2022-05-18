// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

// CreateRingRequest specifies the parameters for a new ring.
type CreateRingRequest struct {
	Name               string   `json:"name,omitempty"`
	Priority           int      `json:"priority,omitempty"`
	InstallationGroups []string `json:"installationGroups,omitempty"`
	SoakTime           int      `json:"soakTime,omitempty"`
	Image              string   `json:"image,omitempty"`
	Version            string   `json:"version,omitempty"`
	APISecurityLock    bool     `json:"api-security-lock,omitempty"`
}

// UpdateRingRequest specifies the parameters to update a ring.
type UpdateRingRequest struct {
	Name               string   `json:"name,omitempty"`
	Priority           int      `json:"priority,omitempty"`
	InstallationGroups []string `json:"installationGroups,omitempty"`
	SoakTime           int      `json:"soakTime,omitempty"`
	Image              string   `json:"image,omitempty"`
	Version            string   `json:"version,omitempty"`
	APISecurityLock    bool     `json:"api-security-lock,omitempty"`
}

// SetDefaults sets the default values for a ring create request.
func (request *CreateRingRequest) SetDefaults() {
	if request.SoakTime == 0 {
		request.SoakTime = 7200
	}
}

// Validate validates the values of a ring create request.
func (request *CreateRingRequest) Validate() error {
	if request.Priority == 0 {
		return errors.New("Priority cannot be zero")
	}

	return nil
}

// NewCreateRingRequestFromReader will create a CreateRingRequest from an
// io.Reader with JSON data.
func NewCreateRingRequestFromReader(reader io.Reader) (*CreateRingRequest, error) {
	var createRingRequest CreateRingRequest
	err := json.NewDecoder(reader).Decode(&createRingRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create ring request")
	}

	createRingRequest.SetDefaults()
	if err = createRingRequest.Validate(); err != nil {
		return nil, errors.Wrap(err, "create ring request failed validation")
	}

	return &createRingRequest, nil
}

// NewUpdateRingRequestFromReader will create an UpdateRingRequest from an io.Reader with JSON data.
func NewUpdateRingRequestFromReader(reader io.Reader) (*UpdateRingRequest, error) {
	var updateRingRequest UpdateRingRequest
	err := json.NewDecoder(reader).Decode(&updateRingRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode provision ring request")
	}
	return &updateRingRequest, nil
}

// GetRingsRequest describes the parameters to request a list of rings.
type GetRingsRequest struct {
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetRingsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	if request.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
	u.RawQuery = q.Encode()
}

// ReleaseRingRequest contains metadata related to changing the installed ring state.
type ReleaseRingRequest struct {
	Image   string
	Version string
}

// NewReleaseRingRequestFromReader will create an UpdateRingRequest from an io.Reader with JSON data.
func NewReleaseRingRequestFromReader(reader io.Reader) (*ReleaseRingRequest, error) {
	var releaseRingRequest ReleaseRingRequest
	err := json.NewDecoder(reader).Decode(&releaseRingRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode provision ring request")
	}
	return &releaseRingRequest, nil
}
