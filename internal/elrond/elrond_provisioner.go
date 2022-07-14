// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package elrond

import (
	log "github.com/sirupsen/logrus"
)

// ElProvisioner provisions release rings.
type ElProvisioner struct {
	logger            log.FieldLogger
	ProvisionerServer string
}

// NewElrondProvisioner creates a new ElrondProvisioner.
func NewElrondProvisioner(logger log.FieldLogger, provisionerServer string) *ElProvisioner {
	logger = logger.WithField("provisioner", "elrond")

	return &ElProvisioner{
		logger:            logger,
		ProvisionerServer: provisionerServer,
	}
}
