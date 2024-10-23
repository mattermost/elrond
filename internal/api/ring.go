// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/mattermost/elrond/internal/webhook"
	"github.com/mattermost/elrond/model"
)

// initRing registers ring endpoints on the given router.
func initRing(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	ringsRouter := apiRouter.PathPrefix("/rings").Subrouter()
	ringsRouter.Handle("", addContext(handleGetRings)).Methods("GET")
	ringsRouter.Handle("", addContext(handleCreateRing)).Methods("POST")
	ringsRouter.Handle("/release", addContext(handleReleaseAllRings)).Methods("POST")
	ringsRouter.Handle("/release/pause", addContext(handlePauseReleaseRing)).Methods("POST")
	ringsRouter.Handle("/release/resume", addContext(handleResumeReleaseRing)).Methods("POST")
	ringsRouter.Handle("/release/cancel", addContext(handleCancelReleaseRing)).Methods("POST")

	ringRouter := apiRouter.PathPrefix("/ring/{ring:[A-Za-z0-9]{26}}").Subrouter()
	ringRouter.Handle("", addContext(handleGetRing)).Methods("GET")
	ringRouter.Handle("", addContext(handleRetryCreateRing)).Methods("POST")
	ringRouter.Handle("/update", addContext(handleUpdateRing)).Methods("POST")
	ringRouter.Handle("/release", addContext(handleReleaseRing)).Methods("POST")
	ringRouter.Handle("/release", addContext(handleRetryReleaseRing)).Methods("POST")
	ringRouter.Handle("/installationgroup", addContext(handleRegisterRingInstallationGroup)).Methods("POST")
	ringRouter.Handle("/installationgroup/{installation-group-id}", addContext(handleDeleteRingInstallationGroup)).Methods("DELETE")
	ringRouter.Handle("", addContext(handleDeleteRing)).Methods("DELETE")

	ringReleaseRouter := apiRouter.PathPrefix("/release/{release:[A-Za-z0-9]{26}}").Subrouter()
	ringReleaseRouter.Handle("", addContext(handleGetRingRelease)).Methods("GET")

}

// handleGetRing responds to GET /api/ring/{ring}, returning the ring in question.
func handleGetRing(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringID := vars["ring"]
	c.Logger = c.Logger.WithField("ring", ringID)

	ring, err := c.Store.GetRing(ringID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query ring")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ring == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	installationGroups, err := c.Store.GetInstallationGroupsForRing(ringID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get installation groups for ring")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ring.InstallationGroups = installationGroups

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, ring)
}

// handleGetRings responds to GET /api/rings, returning the specified page of rings.
func handleGetRings(c *Context, w http.ResponseWriter, r *http.Request) {
	page, perPage, includeDeleted, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.RingFilter{
		Page:           page,
		PerPage:        perPage,
		IncludeDeleted: includeDeleted,
	}

	rings, err := c.Store.GetRings(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query rings")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if rings == nil {
		rings = []*model.Ring{}
	}

	installationGroups, err := c.Store.GetInstallationGroupsForRings(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get installation groups for ring")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, r := range rings {
		r.InstallationGroups = installationGroups[r.ID]
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, rings)
}

// handleCreateRing responds to POST /api/rings, beginning the process of creating a new
// ring.
// sample body:
//
//	{
//			"priority": 1,
//	}
func handleCreateRing(c *Context, w http.ResponseWriter, r *http.Request) {
	createRingRequest, err := model.NewCreateRingRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	release, err := c.Store.GetOrCreateRingRelease(&model.RingRelease{
		Version:      createRingRequest.Version,
		Image:        createRingRequest.Image,
		Force:        false,
		EnvVariables: nil,
		CreateAt:     time.Now().UnixNano(),
	})

	if err != nil {
		c.Logger.WithError(err).Error("failed to get or create new ring release")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ring := model.Ring{
		Name:             createRingRequest.Name,
		Priority:         createRingRequest.Priority,
		SoakTime:         createRingRequest.SoakTime,
		ActiveReleaseID:  release.ID,
		DesiredReleaseID: release.ID,
		Provisioner:      "elrond",
		APISecurityLock:  createRingRequest.APISecurityLock,
		State:            model.RingStateCreationRequested,
	}
	iGroup := model.InstallationGroup{}
	if createRingRequest.InstallationGroup != nil {
		if createRingRequest.InstallationGroup.Name != "" {
			iGroup = model.InstallationGroup{
				Name:               createRingRequest.InstallationGroup.Name,
				State:              model.InstallationGroupStable,
				ProvisionerGroupID: createRingRequest.InstallationGroup.ProvisionerGroupID,
				SoakTime:           createRingRequest.InstallationGroup.SoakTime,
			}
		}
	}

	if err = c.Store.CreateRing(&ring, &iGroup); err != nil {
		c.Logger.WithError(err).Error("failed to create ring")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ring.InstallationGroups = append(ring.InstallationGroups, &iGroup)

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeRing,
		ID:        ring.ID,
		Name:      ring.Name,
		NewState:  model.RingStateCreationRequested,
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
	}
	if err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState)); err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
	}

	c.Supervisor.Do() //nolint

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, ring)
}

// handleRetryCreateRing responds to POST /api/ring/{ring}, retrying a previously
// failed creation.
func handleRetryCreateRing(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringID := vars["ring"]
	c.Logger = c.Logger.WithField("ring", ringID)

	ring, status, unlockOnce := lockRing(c, ringID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	newState := model.RingStateCreationRequested

	if !ring.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to retry ring creation while in state %s", ring.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ring.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeRing,
			ID:        ring.ID,
			Name:      ring.Name,
			NewState:  newState,
			OldState:  ring.State,
			Timestamp: time.Now().UnixNano(),
		}
		ring.State = newState

		if err := c.Store.UpdateRing(ring); err != nil {
			c.Logger.WithError(err).Errorf("failed to retry ring creation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState)); err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
		}
	}

	// Notify even if we didn't make changes, to expedite even the no-op operations above.
	unlockOnce()
	c.Supervisor.Do() //nolint

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, ring)
}

// handleUpdateRing responds to POST /api/ring/{ring}/update,
// updating a ring.
func handleUpdateRing(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringID := vars["ring"]
	c.Logger = c.Logger.WithField("ring", ringID)

	ring, status, unlockOnce := lockRing(c, ringID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if ring.APISecurityLock {
		logSecurityLockConflict("ring", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	updateRingRequest, err := model.NewUpdateRingRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to deserialize ring update request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if updateRingRequest.Name != "" {
		ring.Name = updateRingRequest.Name
	}

	if updateRingRequest.SoakTime != ring.SoakTime && updateRingRequest.SoakTime != 0 {
		ring.SoakTime = updateRingRequest.SoakTime
	}

	if updateRingRequest.Priority != ring.Priority && updateRingRequest.Priority != 0 {
		ring.Priority = updateRingRequest.Priority
	}

	if err = c.Store.UpdateRing(ring); err != nil {
		c.Logger.WithError(err).Error("failed to update ring")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, ring)
}

// handleReleaseAllRings responds to POST /api/rings/release,
// releasing a deployment in all rings.
func handleReleaseAllRings(c *Context, w http.ResponseWriter, r *http.Request) {
	ringReleaseRequest, err := model.NewRingReleaseRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to deserialize ring release request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	rings, err := c.Store.GetRings(&model.RingFilter{
		IncludeDeleted: false,
		PerPage:        10000,
	})

	if err != nil {
		c.Logger.WithError(err).Error("failed to get all rings from store")
		w.WriteHeader(http.StatusInternalServerError)
	}

	var ringIDs []string
	for _, ring := range rings {
		ringIDs = append(ringIDs, ring.ID)
	}

	status, unlockOnce := lockRings(c, ringIDs)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	var webhookPayloads []*model.WebhookPayload

	c.Logger.Debug("Checking if all rings can be released")

	ringsPending, err := c.Store.GetRingsInPendingState()
	if err != nil {
		c.Logger.WithError(err).Error("failed to get all rings pending work")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(ringsPending) > 0 {
		c.Logger.Error("Cannot start an all rings release, while another release is pending work")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ringRelease := model.RingRelease{
		Version:      ringReleaseRequest.Version,
		Image:        ringReleaseRequest.Image,
		Force:        ringReleaseRequest.Force,
		EnvVariables: ringReleaseRequest.EnvVariables,
		CreateAt:     time.Now().UnixNano(),
	}

	//Proactively checking or creating a ring release entry so that all rings to be released get the same release version
	desiredRelease, err := c.Store.GetOrCreateRingRelease(&ringRelease)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get or create new ring release")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, ring := range rings {
		c.Logger = c.Logger.WithField("ring", ring.ID)

		if ring.APISecurityLock {
			logSecurityLockConflict("ring", c.Logger)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if !ring.ValidTransitionState(model.RingStateReleasePending) {
			c.Logger.Warnf("unable to do a ring release while in state %s", ring.State)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if ring.State != model.RingStateReleasePending {
			webhookPayload := &model.WebhookPayload{
				Type:      model.TypeRing,
				ID:        ring.ID,
				Name:      ring.Name,
				NewState:  model.RingStateReleasePending,
				OldState:  ring.State,
				Timestamp: time.Now().UnixNano(),
				ExtraData: map[string]string{"Environment": c.Environment},
			}
			activeRelease, getErr := c.Store.GetRingRelease(ring.ActiveReleaseID)
			if getErr != nil {
				c.Logger.WithError(getErr).Error("failed to get ring active release details")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if activeRelease.Image != ringReleaseRequest.Image || activeRelease.Version != ringReleaseRequest.Version {
				ring.State = model.RingStateReleasePending
				ring.DesiredReleaseID = desiredRelease.ID

				webhookPayloads = append(webhookPayloads, webhookPayload)
			}
		}
	}

	c.Logger.Debug("Updating all rings in a single transaction")
	if err = c.Store.UpdateRings(rings); err != nil {
		c.Logger.WithError(err).Error("failed to update rings in a single transaction")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, payload := range webhookPayloads {
		if err := webhook.SendToAllWebhooks(c.Store, payload, c.Logger.WithField("webhookEvent", payload.NewState)); err != nil {
			c.Logger.WithError(err).Error("unable to process and send webhooks")
		}

		c.Logger.Infof("Ring %s updated", payload.ID)
	}

	c.Supervisor.Do() //nolint
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, rings)
}

// handleReleaseRing responds to POST /api/ring/{ring}/release,
// releasing a deployment in a ring.
func handleReleaseRing(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringID := vars["ring"]
	c.Logger = c.Logger.WithField("ring", ringID)

	ring, status, unlockOnce := lockRing(c, ringID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if ring.APISecurityLock {
		logSecurityLockConflict("ring", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	ringReleaseRequest, err := model.NewRingReleaseRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to deserialize ring release request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !ring.ValidTransitionState(model.RingStateReleasePending) {
		c.Logger.Warnf("unable to do a ring release while in state %s", ring.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ring.State != model.RingStateReleasePending {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeRing,
			ID:        ring.ID,
			Name:      ring.Name,
			NewState:  model.RingStateReleasePending,
			OldState:  ring.State,
			Timestamp: time.Now().UnixNano(),
			ExtraData: map[string]string{"Environment": c.Environment},
		}

		activeRelease, err := c.Store.GetRingRelease(ring.ActiveReleaseID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to get ring active release details")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if activeRelease.Image != ringReleaseRequest.Image || activeRelease.Version != ringReleaseRequest.Version || ringReleaseRequest.EnvVariables != nil {

			ringRelease := model.RingRelease{
				Version:      ringReleaseRequest.Version,
				Image:        ringReleaseRequest.Image,
				Force:        ringReleaseRequest.Force,
				EnvVariables: ringReleaseRequest.EnvVariables,
				CreateAt:     time.Now().UnixNano(),
			}

			desiredRelease, err := c.Store.GetOrCreateRingRelease(&ringRelease)
			if err != nil {
				c.Logger.WithError(err).Error("failed to get or create new ring release")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			ring.State = model.RingStateReleasePending
			ring.DesiredReleaseID = desiredRelease.ID

			if err = c.Store.UpdateRing(ring); err != nil {
				c.Logger.WithError(err).Error("failed to update ring")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if err := webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState)); err != nil {
				c.Logger.WithError(err).Error("unable to process and send webhooks")
			}
		}
	}

	// Notify even if we didn't make changes, to expedite even the no-op operations above.
	unlockOnce()
	c.Supervisor.Do() //nolint

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, ring)
}

// handleRetryReleaseRing responds to POST /api/ring/{ring}/release, retrying a previously
// failed creation.
func handleRetryReleaseRing(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringID := vars["ring"]
	c.Logger = c.Logger.WithField("ring", ringID)

	ring, status, unlockOnce := lockRing(c, ringID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	newState := model.RingStateReleasePending

	if !ring.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to retry ring release while in state %s", ring.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ring.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeRing,
			ID:        ring.ID,
			Name:      ring.Name,
			NewState:  newState,
			OldState:  ring.State,
			Timestamp: time.Now().UnixNano(),
		}
		ring.State = newState

		if err := c.Store.UpdateRing(ring); err != nil {
			c.Logger.WithError(err).Errorf("failed to retry ring release")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState)); err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
		}
	}

	// Notify even if we didn't make changes, to expedite even the no-op operations above.
	unlockOnce()
	c.Supervisor.Do() //nolint

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, ring)
}

// handlePauseReleaseRing responds to POST /api/rings/release/pause, pausing all pending releases
func handlePauseReleaseRing(c *Context, w http.ResponseWriter, r *http.Request) {
	ringsPending, err := c.Store.GetRingsInPendingState()
	if err != nil {
		c.Logger.WithError(err).Error("failed to get all rings pending work")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Logger.Info("pausing all releases in pending state. Releases already in progress cannot be paused")

	for _, ring := range ringsPending {
		ring.State = model.RingStateReleasePaused
	}

	c.Logger.Debug("Updating all rings in a single transaction")
	if err = c.Store.UpdateRings(ringsPending); err != nil {
		c.Logger.WithError(err).Error("failed to update rings status to paused in a single transaction")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// handleResumeReleaseRing responds to POST /api/rings/release/pause, pausing all pending releases
func handleResumeReleaseRing(c *Context, w http.ResponseWriter, r *http.Request) {
	ringsPaused, err := c.Store.GetRingsInPendingState()
	if err != nil {
		c.Logger.WithError(err).Error("failed to get all rings in paused state")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Logger.Info("resuming all releases in paused state")

	for _, ring := range ringsPaused {
		ring.State = model.RingStateReleasePending
	}

	c.Logger.Debug("Updating all rings in a single transaction")
	if err = c.Store.UpdateRings(ringsPaused); err != nil {
		c.Logger.WithError(err).Error("failed to update rings status to pending in a single transaction")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// handleCancelReleaseRing responds to POST /api/rings/release/pause, pausing all pending releases
func handleCancelReleaseRing(c *Context, w http.ResponseWriter, r *http.Request) {
	ringsPending, err := c.Store.GetRingsInPendingState()
	if err != nil {
		c.Logger.WithError(err).Error("failed to get all rings in pending state")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Logger.Info("canceling all releases in pending state. Setting desired release same as active. Releases already in progress cannot be cancelled")

	for _, ring := range ringsPending {
		ring.State = model.RingStateStable
		ring.DesiredReleaseID = ring.ActiveReleaseID
	}

	c.Logger.Debug("Updating all rings in a single transaction")
	if err = c.Store.UpdateRings(ringsPending); err != nil {
		c.Logger.WithError(err).Error("failed to update rings status to stable and set desired release in a single transaction")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// handleGetRingRelease responds to GET /api/release/{release}, returning the ring release in question.
func handleGetRingRelease(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringReleaseID := vars["release"]
	c.Logger = c.Logger.WithField("release", ringReleaseID)

	ringRelease, err := c.Store.GetRingRelease(ringReleaseID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query ring release")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ringRelease == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, ringRelease)
}

// handleDeleteRing responds to DELETE /api/ring/{ring}, beginning the process of
// deleting the ring.
func handleDeleteRing(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringID := vars["ring"]
	c.Logger = c.Logger.WithField("ring", ringID)

	ring, status, unlockOnce := lockRing(c, ringID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if ring.APISecurityLock {
		logSecurityLockConflict("ring", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	newState := model.RingStateDeletionRequested

	if !ring.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to delete ring while in state %s", ring.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ring.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeRing,
			ID:        ring.ID,
			Name:      ring.Name,
			NewState:  newState,
			OldState:  ring.State,
			Timestamp: time.Now().UnixNano(),
		}
		ring.State = newState

		if err := c.Store.UpdateRing(ring); err != nil {
			c.Logger.WithError(err).Error("failed to mark ring for deletion")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState)); err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
		}
	}

	unlockOnce()
	c.Supervisor.Do() //nolint

	w.WriteHeader(http.StatusAccepted)
}

// handleRegisterRingInstallationGroup responds to POST /api/ring/{ring}/installationgroup,
// registers the set of installation groups to the Ring.
func handleRegisterRingInstallationGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringID := vars["ring"]
	c.Logger = c.Logger.WithField("ring", ringID).WithField("action", "register-ring-installation-groups")
	ring, status, unlockOnce := lockRing(c, ringID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if ring.APISecurityLock {
		logSecurityLockConflict("ring", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	installationGroupRequest, err := model.NewRegisterInstallationGroupRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	iGroup := model.InstallationGroup{
		Name:               installationGroupRequest.Name,
		SoakTime:           installationGroupRequest.SoakTime,
		State:              model.InstallationGroupStable,
		ProvisionerGroupID: installationGroupRequest.ProvisionerGroupID,
	}

	installationGroup, err := c.Store.CreateRingInstallationGroup(ringID, &iGroup)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create ring installation groups")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ring.InstallationGroups = append(ring.InstallationGroups, installationGroup)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, ring)
}

// handleDeleteRingInstallationGroup responds to DELETE /api/ring/{ring}/installationgroup/{installation-group-id},
// removes installation group from the Ring.
func handleDeleteRingInstallationGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringID := vars["ring"]
	installationGroupID := vars["installation-group-id"]
	c.Logger = c.Logger.
		WithField("ring", ringID).
		WithField("action", "delete-ring-installation-group").
		WithField("installation-group-id", installationGroupID)

	ring, status, unlockOnce := lockRing(c, ringID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if ring.APISecurityLock {
		logSecurityLockConflict("ring", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := c.Store.DeleteRingInstallationGroup(ringID, installationGroupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed delete ring installation group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
