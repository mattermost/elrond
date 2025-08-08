// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"fmt"
	"time"

	"github.com/mattermost/elrond/internal/webhook"
	"github.com/mattermost/elrond/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// installationGroupStore abstracts the database operations required to manage installation groups.
type installationGroupStore interface {
	GetInstallationGroupsPendingWork() ([]*model.InstallationGroup, error)
	GetInstallationGroupByID(id string) (*model.InstallationGroup, error)
	UpdateInstallationGroup(installationGroup *model.InstallationGroup) error
	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
	GetRingFromInstallationGroupID(installationGroupID string) (*model.Ring, error)
	LockRingInstallationGroup(installationGroupID, lockerID string) (bool, error)
	UnlockRingInstallationGroup(installationGroupID string, lockerID string, force bool) (bool, error)
	GetInstallationGroupsLocked() ([]*model.InstallationGroup, error)
	GetInstallationGroupsReleaseInProgress() ([]*model.InstallationGroup, error)
	GetRingRelease(releaseID string) (*model.RingRelease, error)
	GetRingsPendingWork() ([]*model.Ring, error)
	UpdateRings(rings []*model.Ring) error
}

// installationGroupProvisioner abstracts the provisioning operations required by the installation group supervisor.
type installationGroupProvisioner interface {
	ReleaseInstallationGroup(installationGroup *model.InstallationGroup, release *model.RingRelease) error
	SoakInstallationGroup(installationGroup *model.InstallationGroup) error
	AddGrafanaAnnotations(text string, ring *model.Ring, installationGroup *model.InstallationGroup, release *model.RingRelease) error
}

// InstallationGroupSupervisor finds installation groups pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type InstallationGroupSupervisor struct {
	store       installationGroupStore
	provisioner installationGroupProvisioner
	instanceID  string
	logger      log.FieldLogger
}

// NewInstallationGroupSupervisor creates a new InstallationGroupSupervisor.
func NewInstallationGroupSupervisor(store installationGroupStore, installationGroupProvisioner installationGroupProvisioner, instanceID string, logger log.FieldLogger) *InstallationGroupSupervisor {
	return &InstallationGroupSupervisor{
		store:       store,
		provisioner: installationGroupProvisioner,
		instanceID:  instanceID,
		logger:      logger,
	}
}

// Shutdown performs graceful shutdown tasks for the installation group supervisor.
func (s *InstallationGroupSupervisor) Shutdown() {
	s.logger.Debug("Shutting down installation group supervisor")
}

// Do looks for work to be done on any pending rings and attempts to schedule the required work.
func (s *InstallationGroupSupervisor) Do() error {
	installationGroups, err := s.store.GetInstallationGroupsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for installation groups pending work")
		return nil
	}

	s.logger.WithFields(log.Fields{
		"pendingCount": len(installationGroups),
		"action":       "starting_supervision_cycle",
	}).Debug("Starting installation group supervision cycle")

	// Clean up locks for groups that are ready in provisioner but still locked
	s.forceUnlockStaleLocks(s.logger)

	for _, installationGroup := range installationGroups {
		s.Supervise(installationGroup)
	}

	return nil
}

// Supervise schedules the required work on the given installation group.
func (s *InstallationGroupSupervisor) Supervise(installationGroup *model.InstallationGroup) {
	logger := s.logger.WithFields(log.Fields{
		"installationgroup": installationGroup.ID,
	})

	lock := newInstallationGroupLock(installationGroup.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the installation group, it is crucial that we ensure that it was
	// not updated to a new state by another elrond server.
	originalState := installationGroup.State
	installationGroup, err := s.store.GetInstallationGroupByID(installationGroup.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed installation group")
		return
	}
	if installationGroup.State != originalState {
		logger.WithField("oldInstallationGroupState", originalState).
			WithField("newInstallationGroupState", installationGroup.State).
			Warn("Another provisioner has worked on this installationGroup; skipping...")
		return
	}

	logger.Debugf("Supervising installation group in state %s", installationGroup.State)

	newState := s.transitionInstallationGroup(installationGroup, logger)

	installationGroup, err = s.store.GetInstallationGroupByID(installationGroup.ID)
	if err != nil {
		logger.WithError(err).Warnf("failed to get installation group and thus persist state %s", newState)
		return
	}

	if installationGroup.State == newState {
		return
	}

	oldState := installationGroup.State
	installationGroup.State = newState
	if oldState == model.InstallationGroupReleaseRequested && (newState == model.InstallationGroupReleaseSoakingRequested || newState == model.InstallationGroupStable) {
		installationGroup.ReleaseAt = time.Now().UnixNano()
	}

	if err = s.store.UpdateInstallationGroup(installationGroup); err != nil {
		logger.WithError(err).Warnf("failed to set installation group state to %s", newState)
		return
	}

	//Move rings to release-failed as soon as an IG release fails
	if newState == model.InstallationGroupReleaseFailed || newState == model.InstallationGroupReleaseSoakingFailed {
		logger.Info("Installation group release has failed, moving ring to failed state")
		rings, getErr := s.store.GetRingsPendingWork()
		if getErr != nil {
			logger.WithError(getErr).Error("failed to get all rings pending work")
			return
		}
		for _, ring := range rings {
			ring.State = model.RingStateReleaseFailed
		}

		if err = s.store.UpdateRings(rings); err != nil {
			logger.WithError(err).Error("failed to move rings to failed state")
			return
		}
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeRing,
		ID:        installationGroup.ID,
		NewState:  newState,
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
	}
	if err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState)); err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned installation group from %s to %s", oldState, newState)
}

// Do works with the given ring to transition it to a final state.
func (s *InstallationGroupSupervisor) transitionInstallationGroup(installationGroup *model.InstallationGroup, logger log.FieldLogger) string {
	switch installationGroup.State {
	case model.InstallationGroupReleasePending:
		return s.checkInstallationGroupPending(installationGroup, logger)
	case model.InstallationGroupReleaseRequested:
		return s.releaseInstallationGroup(installationGroup, logger)
	case model.InstallationGroupReleaseSoakingRequested:
		return s.soakInstallationGroup(installationGroup, logger)
	default:
		logger.Warnf("Found installation group pending work in unexpected state %s", installationGroup.State)
		return installationGroup.State
	}
}

func (s *InstallationGroupSupervisor) checkInstallationGroupPending(installationGroup *model.InstallationGroup, logger log.FieldLogger) string {
	logger.Debugf("Checking if installation group %s ring is in state to move forward with installation group releases...", installationGroup.ID)
	ring, err := s.store.GetRingFromInstallationGroupID(installationGroup.ID)
	if err != nil {
		logger.WithError(err).Error("Failed to query for the ring of the installation group")
		return model.InstallationGroupReleaseFailed
	}

	if ring.State == model.RingStateReleaseFailed {
		return model.InstallationGroupReleaseFailed
	}

	if ring.State != model.RingStateReleaseRequested && ring.State != model.RingStateReleaseInProgress {
		return model.InstallationGroupReleasePending
	}

	logger.Debug("Checking if other Installation Groups are locked...")

	installationGroupsLocked, err := s.store.GetInstallationGroupsLocked()
	if err != nil {
		logger.WithError(err).Error("Failed to query for installation groups that are under lock")
		return model.InstallationGroupReleaseFailed
	}

	installationGroupsReleaseInProgress, err := s.store.GetInstallationGroupsReleaseInProgress()
	if err != nil {
		logger.WithError(err).Error("Failed to query for installation groups that are under release")
		return model.InstallationGroupReleaseFailed
	}

	//The total installation groups locked at this time will be at least 1
	if len(installationGroupsLocked) > 1 || len(installationGroupsReleaseInProgress) > 0 {
		logger.Debug("Another installation group is under lock and being updated...")
		return model.InstallationGroupReleasePending
	}

	return model.InstallationGroupReleaseRequested
}

func (s *InstallationGroupSupervisor) releaseInstallationGroup(installationGroup *model.InstallationGroup, logger log.FieldLogger) string {
	ring, err := s.store.GetRingFromInstallationGroupID(installationGroup.ID)
	if err != nil {
		logger.WithError(err).Error("Failed to get the ring from the installation group pending work")
		return model.InstallationGroupReleaseFailed
	}

	release, err := s.store.GetRingRelease(ring.DesiredReleaseID)
	if err != nil {
		logger.WithError(err).Error("Failed to get the ring release for the installation group pending work")
		return model.InstallationGroupReleaseFailed
	}

	err = s.provisioner.AddGrafanaAnnotations(fmt.Sprintf("Initiating release for ring %s and installation group %s", ring.Name, installationGroup.ProvisionerGroupID), ring, installationGroup, release)
	if err != nil {
		logger.WithError(err).Error("Failed to add release Grafana Annotations")
		return model.InstallationGroupReleaseFailed
	}

	err = s.provisioner.ReleaseInstallationGroup(installationGroup, release)
	if err != nil {
		logger.WithError(err).Error("Failed to release installation group")
		return model.InstallationGroupReleaseFailed
	}
	logger.Infof("Finished releasing installation group %s", installationGroup.ID)
	if release.Force {
		logger.Info("This is a forced release. Skipping installation group soaking time...")

		err = s.provisioner.AddGrafanaAnnotations(fmt.Sprintf("Release for ring %s and installation group %s is complete", ring.Name, installationGroup.ProvisionerGroupID), ring, installationGroup, release)
		if err != nil {
			logger.WithError(err).Error("Failed to add release Grafana Annotations")
			return model.InstallationGroupReleaseFailed
		}
		return model.InstallationGroupStable
	}
	return model.InstallationGroupReleaseSoakingRequested
}

func (s *InstallationGroupSupervisor) soakInstallationGroup(installationGroup *model.InstallationGroup, logger log.FieldLogger) string {
	timePassed := ((time.Now().UnixNano() - installationGroup.ReleaseAt) / int64(time.Second))
	if timePassed < int64(installationGroup.SoakTime) {
		logger.Infof("Installation Group %s will be soaking for another %d seconds...", installationGroup.ID, int64(installationGroup.SoakTime)-timePassed)
		return model.InstallationGroupReleaseSoakingRequested
	}

	err := s.provisioner.SoakInstallationGroup(installationGroup)
	if err != nil {
		logger.WithError(err).Error("Failed to soak ring")
		return model.InstallationGroupReleaseSoakingFailed
	}

	logger.Info("Finished soaking installation group")

	ring, err := s.store.GetRingFromInstallationGroupID(installationGroup.ID)
	if err != nil {
		logger.WithError(err).Error("Failed to get the ring from the installation group pending work")
		return model.InstallationGroupReleaseFailed
	}

	release, err := s.store.GetRingRelease(ring.DesiredReleaseID)
	if err != nil {
		logger.WithError(err).Error("Failed to get the ring release for the installation group pending work")
		return model.InstallationGroupReleaseFailed
	}

	err = s.provisioner.AddGrafanaAnnotations(fmt.Sprintf("Release for ring %s and installation group %s is complete", ring.Name, installationGroup.ProvisionerGroupID), ring, installationGroup, release)
	if err != nil {
		logger.WithError(err).Error("Failed to add release Grafana Annotations")
		return model.InstallationGroupReleaseFailed
	}

	return model.InstallationGroupStable
}

// forceUnlockStaleLocks unlocks installation groups that are stuck in release-pending state
// with no legitimate reason to be pending
func (s *InstallationGroupSupervisor) forceUnlockStaleLocks(logger log.FieldLogger) {
	lockedGroups, err := s.store.GetInstallationGroupsLocked()
	if err != nil {
		logger.WithError(err).Error("Failed to get locked installation groups for lock cleanup")
		return
	}

	if len(lockedGroups) == 0 {
		return
	}

	for _, group := range lockedGroups {
		// Only check groups that are in release-pending state
		if group.State != model.InstallationGroupReleasePending {
			continue
		}

		// First check if there are legitimate reasons to be pending
		isLegitimatelyPending, err := s.checkLegitimateWait(group, logger)
		if err != nil {
			logger.WithError(err).WithFields(log.Fields{
				"installationGroupID": group.ID,
				"action":              "check_legitimate_wait_failed",
			}).Error("Failed to check if group is legitimately pending")
			continue
		}

		if isLegitimatelyPending {
			logger.WithFields(log.Fields{
				"installationGroupID": group.ID,
				"action":              "legitimate_pending_state",
			}).Debug("Group is legitimately pending, not stuck")
			continue
		}

		// If no legitimate reason to be pending, check if lock is too old
		isStuck, err := s.checkPendingGroupStuck(group, logger)
		if err != nil {
			logger.WithError(err).WithFields(log.Fields{
				"installationGroupID": group.ID,
				"action":              "check_stuck_failed",
			}).Error("Failed to check if group is stuck")
			continue
		}

		// If stuck with no legitimate reason, force unlock
		if isStuck {
			lockAcquiredBy := ""
			if group.LockAcquiredBy != nil {
				lockAcquiredBy = *group.LockAcquiredBy
			}

			logger.WithFields(log.Fields{
				"installationGroupID": group.ID,
				"name":                group.Name,
				"lockAcquiredBy":      lockAcquiredBy,
				"action":              "force_unlock_no_legitimate_wait",
			}).Warn("Release-pending installation group locked with no legitimate reason, force unlocking")

			unlocked, err := s.store.UnlockRingInstallationGroup(group.ID, lockAcquiredBy, true)
			if err != nil {
				logger.WithError(err).WithFields(log.Fields{
					"installationGroupID": group.ID,
					"lockAcquiredBy":      lockAcquiredBy,
					"action":              "force_unlock_failed",
				}).Error("Failed to force unlock installation group")
			} else if unlocked {
				logger.WithFields(log.Fields{
					"installationGroupID": group.ID,
					"lockAcquiredBy":      lockAcquiredBy,
					"action":              "force_unlock_successful",
				}).Info("Successfully force unlocked stuck release-pending installation group")
			}
		}
	}
}

// checkLegitimateWait checks if there are valid reasons for a group to be in release-pending state
func (s *InstallationGroupSupervisor) checkLegitimateWait(group *model.InstallationGroup, logger log.FieldLogger) (bool, error) {
	// Check if waiting for other groups in progress
	inProgressGroups, err := s.store.GetInstallationGroupsReleaseInProgress()
	if err != nil {
		return false, errors.Wrap(err, "failed to get installation groups in progress")
	}

	if len(inProgressGroups) > 0 {
		logger.WithFields(log.Fields{
			"installationGroupID": group.ID,
			"waitingForGroup":     inProgressGroups[0].ID,
			"waitingForState":     inProgressGroups[0].State,
			"action":              "waiting_for_group_in_progress",
		}).Debug("Group is legitimately waiting for another group in progress")
		return true, nil
	}

	// Check if other groups are legitimately locked
	lockedGroups, err := s.store.GetInstallationGroupsLocked()
	if err != nil {
		return false, errors.Wrap(err, "failed to get locked installation groups")
	}

	// If there are other locked groups that are not this one
	for _, lockedGroup := range lockedGroups {
		if lockedGroup.ID != group.ID {
			logger.WithFields(log.Fields{
				"installationGroupID": group.ID,
				"waitingForGroup":     lockedGroup.ID,
				"waitingForState":     lockedGroup.State,
				"action":              "waiting_for_locked_group",
			}).Debug("Group is legitimately waiting for another locked group")
			return true, nil
		}
	}

	// No legitimate reason found to be pending
	logger.WithFields(log.Fields{
		"installationGroupID": group.ID,
		"action":              "no_legitimate_wait_reason",
	}).Debug("No legitimate reason found for group to be pending")
	return false, nil
}

// checkPendingGroupStuck checks if a release-pending group has been locked too long
func (s *InstallationGroupSupervisor) checkPendingGroupStuck(group *model.InstallationGroup, logger log.FieldLogger) (bool, error) {
	// Calculate how long the lock has been held
	lockDuration := time.Duration(time.Now().UnixNano() - group.LockAcquiredAt)

	// If no legitimate reason to be pending, use a shorter threshold
	stuckThreshold := 5 * time.Minute

	isStuck := lockDuration > stuckThreshold

	logger.WithFields(log.Fields{
		"installationGroupID": group.ID,
		"lockDuration":        lockDuration.String(),
		"stuckThreshold":      stuckThreshold.String(),
		"isStuck":             isStuck,
		"action":              "pending_group_stuck_check",
	}).Debug("Checking if release-pending group with no legitimate wait reason is stuck")

	return isStuck, nil
}
