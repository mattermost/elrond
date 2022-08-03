// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
)

// ChangeRequest is the Ring change request stored in a model.Ring.
type ChangeRequest struct {
	Image        string
	Version      string
	Force        bool
	ReleaseStart int64
}

// NewChangeRequestMetadata creates an instance of ChangeRequestMetadata given the raw elrond change request.
func NewChangeRequestMetadata(metadataBytes []byte) (*ChangeRequest, error) {
	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(metadataBytes) == 0 || string(metadataBytes) == "null" {
		// TODO: remove "null" check after sqlite is gone.
		return nil, nil
	}

	changeRequest := ChangeRequest{}
	err := json.Unmarshal(metadataBytes, &changeRequest)
	if err != nil {
		return nil, err
	}

	return &changeRequest, nil
}
