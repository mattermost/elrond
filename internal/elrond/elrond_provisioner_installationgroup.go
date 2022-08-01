// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package elrond

import (
	"time"

	"github.com/mattermost/elrond/model"
	cmodel "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// ReleaseInstallationGroup releases an installation group ring.
func (provisioner *ElProvisioner) ReleaseInstallationGroup(installationGroup *model.InstallationGroup, ring *model.Ring) error {
	logger := provisioner.logger.WithField("installationgroup", installationGroup.ID)
	logger.Infof("Releasing installation group %s", installationGroup.ID)

	client := cmodel.NewClient(provisioner.ProvisionerServer)

	logger.Info("Getting provisioner installation groups")

	group, err := client.GetGroup(installationGroup.ProvisionerGroupID)
	if group == nil || err != nil {
		return errors.Wrapf(err, "failed to get group %s, make sure it exists", installationGroup.ProvisionerGroupID)
	}

	if group.Image != ring.ChangeRequest.Image || group.Version != ring.ChangeRequest.Version {
		logger.Infof("Current provisioner group image is %s:%s and should be updated to %s:%s", group.Image, group.Version, &ring.ChangeRequest.Image, &ring.ChangeRequest.Version)
		request := &cmodel.PatchGroupRequest{
			ID:      installationGroup.ProvisionerGroupID,
			Version: &ring.ChangeRequest.Version,
			Image:   &ring.ChangeRequest.Image,
		}

		logger.Infof("Updating provisioner group %s", installationGroup.ProvisionerGroupID)
		_, err := client.UpdateGroup(request)

		if err != nil {
			return errors.Wrap(err, "failed to patch provisioner group")
		}

		logger.Infof("Update provisioner group %s successful. Waiting up to %d seconds for the group release to complete...", installationGroup.ProvisionerGroupID, provisioner.params.ProvisionerGroupReleaseTimeout)
		err = waitForGroupRelease(client, provisioner.params.ProvisionerGroupReleaseTimeout, installationGroup.ProvisionerGroupID)
		if err != nil {
			return err
		}
	} else {
		logger.Infof("Provisioner group image and version are already up to date with image %s:%s", group.Image, group.Version)
	}

	return nil
}

func waitForGroupRelease(client *cmodel.Client, timeout int, groupID string) error {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return errors.New("timed out waiting for group release to complete")
		default:
			status, err := client.GetGroupStatus(groupID)
			if err != nil {
				return errors.Wrap(err, "failed to get provisioner group status")
			}
			if status.InstallationsAwaitingUpdate == 0 && status.InstallationsUpdating == 0 {
				return nil
			}
			logger.Infof("Provisioner group %s release in progress...", groupID)
			time.Sleep(60 * time.Second)
		}
	}
}
