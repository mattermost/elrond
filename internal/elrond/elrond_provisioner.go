// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package elrond

import (
	cmodel "github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// ProvisioningParams represent configuration used during various provisioning operations.
type ProvisioningParams struct {
	ProvisionerGroupReleaseTimeout int
	GrafanaURL                     string
	GrafanaTokens                  []string
	ThanosURL                      string
}

// ElProvisioner provisions release rings.
type ElProvisioner struct {
	params                   ProvisioningParams
	logger                   log.FieldLogger
	ProvisionerServer        string
	ProvisionerClientID      string
	ProvisionerClientSecret  string
	ProvisionerTokenEndpoint string
}

func (elp *ElProvisioner) NewProvisionerClient() *cmodel.Client {
	if elp.ProvisionerClientID == "" || elp.ProvisionerClientSecret == "" || elp.ProvisionerTokenEndpoint == "" {
		return cmodel.NewClient(elp.ProvisionerServer)
	}

	return cmodel.NewClientWithOAuth(elp.ProvisionerServer, nil, elp.ProvisionerClientID, elp.ProvisionerClientSecret, elp.ProvisionerTokenEndpoint)
}

// NewElrondProvisioner creates a new ElrondProvisioner.
func NewElrondProvisioner(provisioningParams ProvisioningParams, logger log.FieldLogger, provisionerServer, provisionerClientID, provisionerClientSecret, provisionerTokenEndpoint string) *ElProvisioner {
	logger = logger.WithField("provisioner", "elrond")

	return &ElProvisioner{
		params:            provisioningParams,
		logger:            logger,
		ProvisionerServer: provisionerServer,
	}
}
