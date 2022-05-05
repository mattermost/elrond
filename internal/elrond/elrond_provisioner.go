// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package elrond

import (
	log "github.com/sirupsen/logrus"
)

// ElProvisioner provisions release rings.
type ElProvisioner struct {
	logger log.FieldLogger
}

// NewElrondProvisioner creates a new ElrondProvisioner.
func NewElrondProvisioner(logger log.FieldLogger) *ElProvisioner {
	logger = logger.WithField("provisioner", "elrond")

	return &ElProvisioner{
		logger: logger,
	}
}
