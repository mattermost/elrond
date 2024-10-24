// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"time"

	"github.com/mattermost/elrond/internal/webhook"

	"github.com/mattermost/elrond/model"
	log "github.com/sirupsen/logrus"
)

// ringStore abstracts the database operations required to manage rings.
type ringStore interface {
	GetRing(ringID string) (*model.Ring, error)
	GetUnlockedRingsPendingWork() ([]*model.Ring, error)
	GetRings(ringFilter *model.RingFilter) ([]*model.Ring, error)
	CreateRing(ring *model.Ring, installationGroup *model.InstallationGroup) error
	UpdateRing(ring *model.Ring) error
	LockRing(ringID, lockerID string) (bool, error)
	UnlockRing(ringID string, lockerID string, force bool) (bool, error)
	DeleteRing(ringID string) error
	GetRingInstallationGroupsPendingWork(ringID string) ([]*model.InstallationGroup, error)
	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
	GetRingsLocked() ([]*model.Ring, error)
	GetRingsReleaseInProgress() ([]*model.Ring, error)
	GetInstallationGroupsForRing(ringID string) ([]*model.InstallationGroup, error)
	UpdateInstallationGroup(installationGroup *model.InstallationGroup) error
	GetRingRelease(releaseID string) (*model.RingRelease, error)
	GetRingsPendingWork() ([]*model.Ring, error)
	UpdateRings(rings []*model.Ring) error
}

// ringProvisioner abstracts the provisioning operations required by the ring supervisor.
type ringProvisioner interface {
	PrepareRing(ring *model.Ring) bool
	CreateRing(ring *model.Ring) error
	ReleaseRing(ring *model.Ring) error
	SoakRing(ring *model.Ring) error
	RollBackRing(ring *model.Ring) error
	DeleteRing(ring *model.Ring) error
}

// RingSupervisor finds rings pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type RingSupervisor struct {
	store       ringStore
	provisioner ringProvisioner
	instanceID  string
	logger      log.FieldLogger
}

// NewRingSupervisor creates a new RingSupervisor.
func NewRingSupervisor(store ringStore, ringProvisioner ringProvisioner, instanceID string, logger log.FieldLogger) *RingSupervisor {
	return &RingSupervisor{
		store:       store,
		provisioner: ringProvisioner,
		instanceID:  instanceID,
		logger:      logger,
	}
}

// Shutdown performs graceful shutdown tasks for the ring supervisor.
func (s *RingSupervisor) Shutdown() {
	s.logger.Debug("Shutting down ring supervisor")
}

// Do looks for work to be done on any pending rings and attempts to schedule the required work.
func (s *RingSupervisor) Do() error {
	rings, err := s.store.GetUnlockedRingsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for rings pending work")
		return nil
	}

	for _, ring := range rings {
		s.Supervise(ring)
	}

	return nil
}

// Supervise schedules the required work on the given ring.
func (s *RingSupervisor) Supervise(ring *model.Ring) {
	logger := s.logger.WithFields(log.Fields{
		"ring": ring.ID,
	})

	lock := newRingLock(ring.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the ring, it is crucial that we ensure that it was
	// not updated to a new state by another elrond server.
	originalState := ring.State
	ring, err := s.store.GetRing(ring.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed ring")
		return
	}
	if ring.State != originalState {
		logger.WithField("oldRingState", originalState).
			WithField("newRingState", ring.State).
			Warn("Another provisioner has worked on this ring; skipping...")
		return
	}

	logger.Debugf("Supervising ring in state %s", ring.State)

	newState := s.transitionRing(ring, logger)

	ring, err = s.store.GetRing(ring.ID)
	if err != nil {
		logger.WithError(err).Warnf("failed to get ring and thus persist state %s", newState)
		return
	}

	if ring.State == newState {
		return
	}

	oldState := ring.State
	ring.State = newState

	if oldState == model.RingStateReleaseInProgress && (newState == model.RingStateSoakingRequested || newState == model.RingStateStable) {
		ring.ReleaseAt = time.Now().UnixNano()
	}

	if err = s.store.UpdateRing(ring); err != nil {
		logger.WithError(err).Warnf("failed to set ring state to %s", newState)
		return
	}

	//Move pending rings to release-failed as soon as an ring release fails
	if newState == model.RingStateReleaseFailed || newState == model.RingStateSoakingFailed {
		logger.Info("Ring release has failed, moving pending rings to failed state")
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
		ID:        ring.ID,
		NewState:  newState,
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
	}
	if err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState)); err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned ring from %s to %s", oldState, newState)
}

// Do works with the given ring to transition it to a final state.
func (s *RingSupervisor) transitionRing(ring *model.Ring, logger log.FieldLogger) string {
	switch ring.State {
	case model.RingStateCreationRequested:
		return s.createRing(ring, logger)
	case model.RingStateReleasePending:
		return s.checkRingReleasePending(ring, logger)
	case model.RingStateReleaseRequested:
		return s.releaseRing(ring, logger)
	case model.RingStateReleaseInProgress:
		return s.checkReleaseProgress(ring, logger)
	case model.RingStateSoakingRequested:
		return s.soakRing(ring, logger)
	case model.RingStateDeletionRequested:
		return s.deleteRing(ring, logger)
	case model.RingStateReleaseRollbackRequested:
		return s.rollbackRing(ring, logger)
	default:
		logger.Warnf("Found ring pending work in unexpected state %s", ring.State)
		return ring.State
	}
}

func (s *RingSupervisor) createRing(ring *model.Ring, logger log.FieldLogger) string {
	var err error

	if s.provisioner.PrepareRing(ring) {
		if err = s.store.UpdateRing(ring); err != nil {
			logger.WithError(err).Error("Failed to record updated ring after creation")
			return model.RingStateCreationFailed
		}
	}

	if err = s.provisioner.CreateRing(ring); err != nil {
		logger.WithError(err).Error("Failed to create ring")
		return model.RingStateCreationFailed
	}

	logger.Infof("Finished creating ring %s", ring.ID)
	return model.RingStateStable
}

func (s *RingSupervisor) releaseRing(ring *model.Ring, logger log.FieldLogger) string {
	err := s.provisioner.ReleaseRing(ring)
	if err != nil {
		logger.WithError(err).Error("Failed to release ring")
		return model.RingStateReleaseFailed
	}

	installationGroups, err := s.store.GetRingInstallationGroupsPendingWork(ring.ID)
	if err != nil {
		logger.WithError(err).Error("Failed to get ring installation groups pending work")
		return model.RingStateReleaseFailed
	}
	if len(installationGroups) > 0 {
		logger.Info("There are installation groups pending work...")
		return model.RingStateReleaseInProgress
	}

	logger.Infof("Finished releasing ring %s", ring.ID)
	return model.RingStateSoakingRequested
}

func (s *RingSupervisor) checkRingReleasePending(ring *model.Ring, logger log.FieldLogger) string {
	logger.Debugf("Checking if pending ring release should be forced...")

	release, err := s.store.GetRingRelease(ring.DesiredReleaseID)
	if err != nil {
		logger.WithError(err).Error("Failed to get the ring release for the ring pending work")
		return model.RingStateReleaseFailed
	}

	if !release.Force {
		logger.Debug("Checking if other Rings are locked...")

		ringsLocked, getLockedErr := s.store.GetRingsLocked()
		if getLockedErr != nil {
			logger.WithError(getLockedErr).Error("Failed to query for rings that are under lock")
			return model.RingStateReleaseFailed
		}

		ringsReleaseInProgress, ringsReleaseInProgressErr := s.store.GetRingsReleaseInProgress()
		if ringsReleaseInProgressErr != nil {
			logger.WithError(ringsReleaseInProgressErr).Error("Failed to query for rings that are under release")
			return model.RingStateReleaseFailed
		}

		//The total rings locked at this time will be at least 1
		if len(ringsLocked) > 1 || len(ringsReleaseInProgress) > 0 {
			logger.Debug("Another ring is under lock and being updated...")
			return model.InstallationGroupReleasePending
		}

		logger.Debugf("Checking ring %s prioritization", ring.ID)
		rings, ringsErr := s.store.GetUnlockedRingsPendingWork()
		if ringsErr != nil {
			logger.WithError(ringsErr).Error("Failed to get rings pending work for prioritization check")
			return model.RingStateReleaseFailed
		}

		for _, rg := range rings {
			if rg.Priority < ring.Priority {
				logger.Debugf("Ring %s is in priority", rg.ID)
				return model.RingStateReleasePending
			}
		}
	}

	installationGroups, err := s.store.GetInstallationGroupsForRing(ring.ID)
	if err != nil {
		logger.WithError(err).Error("failed to get installation groups for ring")
		return model.RingStateReleaseFailed
	}

	for _, ig := range installationGroups {
		newInstallationGroupState := model.InstallationGroupReleasePending

		logger.Infof("Setting Installation group %s to %s state", ig.Name, newInstallationGroupState)

		if !ig.ValidInstallationGroupTransitionState(newInstallationGroupState) {
			logger.Warnf("Unable to change installation group state change while in state %s", ig.State)
			return model.RingStateReleaseFailed
		}

		logger.Infof("Setting Installation group %s to %s state", ig.Name, newInstallationGroupState)

		ig.State = model.InstallationGroupReleasePending
		if err = s.store.UpdateInstallationGroup(ig); err != nil {
			logger.WithError(err).Error("failed to update installation group")
			return model.RingStateReleaseFailed
		}
	}

	return model.RingStateReleaseRequested
}

func (s *RingSupervisor) checkReleaseProgress(ring *model.Ring, logger log.FieldLogger) string {

	installationGroups, err := s.store.GetRingInstallationGroupsPendingWork(ring.ID)
	if err != nil {
		logger.WithError(err).Error("Failed to get ring installation groups pending work")
		return model.RingStateReleaseFailed
	}
	if len(installationGroups) > 0 {
		logger.Info("There are installation groups pending work...")
		return model.RingStateReleaseInProgress
	}

	logger.Infof("Finished releasing ring %s", ring.ID)

	release, err := s.store.GetRingRelease(ring.DesiredReleaseID)
	if err != nil {
		logger.WithError(err).Error("Failed to get the ring release for the installation group pending work")
		return model.InstallationGroupReleaseFailed
	}

	if release.Force {
		logger.Info("This is a forced release. Skipping ring soaking time...")
		logger.Infof("Ring %s release is now complete. Setting active release ID and moving ring to stable.", ring.ID)

		ring.ActiveReleaseID = ring.DesiredReleaseID

		if err = s.store.UpdateRing(ring); err != nil {
			logger.WithError(err).Error("Failed to record updated ring version and image")
			return model.RingStateReleaseFailed
		}
		return model.RingStateStable
	}
	return model.RingStateSoakingRequested
}

func (s *RingSupervisor) soakRing(ring *model.Ring, logger log.FieldLogger) string {

	installationGroups, err := s.store.GetInstallationGroupsForRing(ring.ID)
	if err != nil {
		logger.WithError(err).Error("failed to get installation groups for ring")
		return model.RingStateSoakingFailed
	}
	ring.InstallationGroups = installationGroups

	timePassed := ((time.Now().UnixNano() - ring.ReleaseAt) / int64(time.Second))
	if timePassed < int64(ring.SoakTime) {
		logger.Infof("Ring %s will be soaking for another %d seconds...", ring.ID, int64(ring.SoakTime)-timePassed)
		err := s.provisioner.SoakRing(ring)
		if err != nil {
			logger.WithError(err).Error("Failed to soak ring")
			return model.RingStateSoakingFailed
		}
		logger.Info("Sleeping for 30 seconds before next check...")
		time.Sleep(30 * time.Second)
		return model.RingStateSoakingRequested
	}

	logger.Infof("Finished soaking ring %s", ring.ID)
	logger.Infof("Ring %s release is now complete. Setting active release ID and moving ring to stable.", ring.ID)

	ring.ActiveReleaseID = ring.DesiredReleaseID

	if err := s.store.UpdateRing(ring); err != nil {
		logger.WithError(err).Error("Failed to record updated ring version and image")
		return model.RingStateSoakingFailed
	}
	return model.RingStateStable
}

func (s *RingSupervisor) rollbackRing(ring *model.Ring, logger log.FieldLogger) string {
	err := s.provisioner.RollBackRing(ring)
	if err != nil {
		logger.WithError(err).Error("Failed to rollback ring")
		return model.RingStateReleaseRollbackFailed
	}

	logger.Infof("Finished rolling back ring %s", ring.ID)
	return model.RingStateReleaseRollbackComplete
}

func (s *RingSupervisor) deleteRing(ring *model.Ring, logger log.FieldLogger) string {
	err := s.provisioner.DeleteRing(ring)
	if err != nil {
		logger.WithError(err).Error("Failed to delete ring")
		return model.RingStateDeletionFailed
	}

	if err = s.store.DeleteRing(ring.ID); err != nil {
		logger.WithError(err).Error("Failed to record updated ring after deletion")
		return model.RingStateDeletionFailed
	}

	logger.Infof("Finished deleting ring %s", ring.ID)
	return model.RingStateDeleted
}
