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
func (provisioner *ElProvisioner) ReleaseInstallationGroup(installationGroup *model.InstallationGroup, release *model.RingRelease) error {
	logger := provisioner.logger.WithField("installationgroup", installationGroup.ID)
	logger.Infof("Releasing installation group %s", installationGroup.ID)

	client := provisioner.NewProvisionerClient()

	logger.Info("Getting provisioner installation groups")

	group, err := client.GetGroup(installationGroup.ProvisionerGroupID)
	if group == nil || err != nil {
		return errors.Wrapf(err, "failed to get group %s, make sure it exists", installationGroup.ProvisionerGroupID)
	}

	newEnvVars, err := release.EnvVariables.ToJSON()
	if err != nil {
		return errors.Wrap(err, "failed to create newEnvVars JSON")
	}

	if string(newEnvVars) == "{}" {
		release.EnvVariables = group.MattermostEnv
	}

	if group.Image != release.Image || group.Version != release.Version || checkChangeGroupEnvVariables(group.MattermostEnv, release.EnvVariables) {
		logger.Infof("Image or group env variable changes were detected. Current provisioner group image is %s:%s and new image is %s:%s", group.Image, group.Version, release.Image, release.Version)
		request := &cmodel.PatchGroupRequest{
			ID:             installationGroup.ProvisionerGroupID,
			Version:        &release.Version,
			Image:          &release.Image,
			MattermostEnv:  release.EnvVariables,
			ReadinessProbe: release.ReadinessProbe,
			LivenessProbe:  release.LivenessProbe,
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

// SoakInstallationGroup soaks an installation group
func (provisioner *ElProvisioner) SoakInstallationGroup(installationGroup *model.InstallationGroup) error {
	logger := provisioner.logger.WithField("installationgroup", installationGroup.ID)
	logger.Infof("Soaking installation group %s", installationGroup.ID)
	// err := soakInstallationGroup(provisioner, installationGroup, logger)
	// if err != nil {
	// 	return err
	// }
	return nil
}

func checkChangeGroupEnvVariables(oldGroup, newGroup cmodel.EnvVarMap) bool {
	if len(oldGroup) != len(newGroup) {
		logger.Info("There is a difference in the number of env variables in the two groups. Probably a new env variable is added...")
		return true
	}
	for i, envVar1 := range oldGroup {
		for j, envVar2 := range newGroup {
			if i == j {
				if envVar1.Value != envVar2.Value {
					logger.Infof("Env var %s has changed from %s to %s", i, envVar1.Value, envVar2.Value)
					return true
				}
			}
		}
	}
	return false
}
