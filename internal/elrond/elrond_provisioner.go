// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package elrond

import (
	log "github.com/sirupsen/logrus"
)

// ElrondProvisioner provisions release rings.
type ElrondProvisioner struct {
	logger log.FieldLogger
}

// NewElrondProvisioner creates a new ElrondProvisioner.
func NewElrondProvisioner(logger log.FieldLogger) *ElrondProvisioner {
	logger = logger.WithField("provisioner", "elrond")

	return &ElrondProvisioner{
		logger: logger,
	}
}
