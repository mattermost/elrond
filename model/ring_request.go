// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"

	dclient "github.com/docker/docker/client"
	"github.com/pkg/errors"
)

// CreateRingRequest specifies the parameters for a new ring.
type CreateRingRequest struct {
	Name              string             `json:"name,omitempty"`
	Priority          int                `json:"priority,omitempty"`
	InstallationGroup *InstallationGroup `json:"installationGroup,omitempty"`
	SoakTime          int                `json:"soakTime,omitempty"`
	Image             string             `json:"image,omitempty"`
	Version           string             `json:"version,omitempty"`
	APISecurityLock   bool               `json:"apiSecurityLock,omitempty"`
}

// UpdateRingRequest specifies the parameters to update a ring.
type UpdateRingRequest struct {
	Name            string `json:"name,omitempty"`
	Priority        int    `json:"priority,omitempty"`
	SoakTime        int    `json:"soakTime,omitempty"`
	Image           string `json:"image,omitempty"`
	Version         string `json:"version,omitempty"`
	APISecurityLock bool   `json:"apiSecurityLock,omitempty"`
}

// ReleaseRingRequest contains metadata related to changing the installed ring state.
type ReleaseRingRequest struct {
	Image   string
	Version string
}

// GetRingsRequest describes the parameters to request a list of rings.
type GetRingsRequest struct {
	Page           int
	PerPage        int
	IncludeDeleted bool
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

// NewReleaseRingRequestFromReader will create an UpdateRingRequest from an io.Reader with JSON data.
func NewReleaseRingRequestFromReader(reader io.Reader) (*ReleaseRingRequest, error) {
	var releaseRingRequest ReleaseRingRequest
	err := json.NewDecoder(reader).Decode(&releaseRingRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode provision ring request")
	}

	err = releaseRingRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid ring release request")
	}

	return &releaseRingRequest, nil
}

// Validate validates the values of a ring release request.
func (request *ReleaseRingRequest) Validate() error {
	ctx := context.Background()
	cli, err := dclient.NewClientWithOpts()
	if err != nil {
		panic(err)
	}

	_, err = cli.DistributionInspect(ctx, fmt.Sprintf("%s:%s", request.Image, request.Version), "")
	if err != nil {
		return errors.Wrapf(err, "cannot find the docker image and version specified. Please check they exist.")
	}

	return nil
}
