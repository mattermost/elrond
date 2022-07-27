// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package elrond

import (
	log "github.com/sirupsen/logrus"
)

// ProvisioningParams represent configuration used during various provisioning operations.
type ProvisioningParams struct {
	ProvisionerGroupReleaseTimeout int
}

// ElProvisioner provisions release rings.
type ElProvisioner struct {
	params            ProvisioningParams
	logger            log.FieldLogger
	ProvisionerServer string
}

// NewElrondProvisioner creates a new ElrondProvisioner.
func NewElrondProvisioner(provisioningParams ProvisioningParams, logger log.FieldLogger, provisionerServer string) *ElProvisioner {
	logger = logger.WithField("provisioner", "elrond")

	return &ElProvisioner{
		params:            provisioningParams,
		logger:            logger,
		ProvisionerServer: provisionerServer,
	}
}
