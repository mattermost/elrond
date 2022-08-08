// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package elrond

import (
	"github.com/mattermost/elrond/model"
)

// PrepareRing ensures a ring object is ready for provisioning.
func (provisioner *ElProvisioner) PrepareRing(ring *model.Ring) bool {
	return true
}

// CreateRing creates a ring.
func (provisioner *ElProvisioner) CreateRing(ring *model.Ring) error {
	logger := provisioner.logger.WithField("ring", ring.ID)
	logger.Infof("Creating ring %s", ring.ID)
	// err := createRing(provisioner, ring, logger)
	// if err != nil {
	// 	return err
	// }

	return nil
}

// DeleteRing deletes a ring.
func (provisioner *ElProvisioner) DeleteRing(ring *model.Ring) error {
	logger := provisioner.logger.WithField("ring", ring.ID)
	logger.Infof("Deleting ring %s", ring.ID)
	// err := deleteRing(provisioner, ring, logger)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// ReleaseRing releases a ring.
func (provisioner *ElProvisioner) ReleaseRing(ring *model.Ring) error {
	logger := provisioner.logger.WithField("ring", ring.ID)
	logger.Infof("Releasing ring %s", ring.ID)
	// err := releaseRing(provisioner, ring, logger)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// RollBackRing rolls back a ring.
func (provisioner *ElProvisioner) RollBackRing(ring *model.Ring) error {
	logger := provisioner.logger.WithField("ring", ring.ID)
	logger.Infof("Rolling back ring %s", ring.ID)
	// err := releaseRing(provisioner, ring, logger)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// SoakRing soaks a ring.
func (provisioner *ElProvisioner) SoakRing(ring *model.Ring) error {
	logger := provisioner.logger.WithField("ring", ring.ID)
	logger.Infof("Soaking ring %s", ring.ID)
	// err := soakRing(provisioner, ring, logger)
	// if err != nil {
	// 	return err
	// }
	return nil
}
