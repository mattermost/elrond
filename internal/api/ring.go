// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

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

	ringRouter := apiRouter.PathPrefix("/ring/{ring:[A-Za-z0-9]{26}}").Subrouter()
	ringRouter.Handle("", addContext(handleGetRing)).Methods("GET")
	ringRouter.Handle("", addContext(handleRetryCreateRing)).Methods("POST")
	ringRouter.Handle("/update", addContext(handleUpdateRing)).Methods("POST")
	ringRouter.Handle("/release", addContext(handleReleaseRing)).Methods("POST")
	ringRouter.Handle("/release", addContext(handleRetryReleaseRing)).Methods("POST")
	ringRouter.Handle("/installationgroups", addContext(handleRegisterRingInstallationGroups)).Methods("POST")
	ringRouter.Handle("/installationgroup/{installation-group-name}", addContext(handleDeleteRingInstallationGroup)).Methods("DELETE")
	ringRouter.Handle("", addContext(handleDeleteRing)).Methods("DELETE")
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
// {
//		"priority": 1,
// }
func handleCreateRing(c *Context, w http.ResponseWriter, r *http.Request) {
	createRingRequest, err := model.NewCreateRingRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ring := model.Ring{
		Name:            createRingRequest.Name,
		Priority:        createRingRequest.Priority,
		SoakTime:        createRingRequest.SoakTime,
		Image:           createRingRequest.Image,
		Version:         createRingRequest.Version,
		Provisioner:     "elrond",
		APISecurityLock: createRingRequest.APISecurityLock,
		State:           model.RingStateCreationRequested,
	}

	installationGroups, err := model.InstallationGroupsFromStringSlice(createRingRequest.InstallationGroups)
	if err != nil {
		c.Logger.WithError(err).Error("failed to validate installation groups")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err = c.Store.CreateRing(&ring, installationGroups); err != nil {
		c.Logger.WithError(err).Error("failed to create ring")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ring.InstallationGroups = installationGroups

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeRing,
		ID:        ring.ID,
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
// releasing a deployment in a ring.
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

	c.Logger.Info(updateRingRequest)

	if updateRingRequest.Name != "" {
		ring.Name = updateRingRequest.Name
	}

	if updateRingRequest.Image != "" {
		ring.Image = updateRingRequest.Image
	}

	if updateRingRequest.Version != "" {
		ring.Version = updateRingRequest.Version
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

	releaseRingRequest, err := model.NewReleaseRingRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to deserialize ring release request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newState := model.RingStateReleaseRequested

	if !ring.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to do a ring release while in state %s", ring.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ring.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeRing,
			ID:        ring.ID,
			NewState:  newState,
			OldState:  ring.State,
			Timestamp: time.Now().UnixNano(),
			ExtraData: map[string]string{"Environment": c.Environment},
		}
		ring.State = newState

		//ADD LOGIC FOR RING RELEASES

		c.Logger.Info(releaseRingRequest)

		if err = c.Store.UpdateRing(ring); err != nil {
			c.Logger.WithError(err).Error("failed to update ring")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//TODO Set here the release logic

		if err := webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState)); err != nil {
			c.Logger.WithError(err).Error("unable to process and send webhooks")
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

	newState := model.RingStateReleaseRequested

	if !ring.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to retry ring release while in state %s", ring.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ring.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeRing,
			ID:        ring.ID,
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

// handleRegisterRingInstallationGroups responds to POST /api/ring/{ring}/installationgroups,
// registers the set of installation groups to the Ring.
func handleRegisterRingInstallationGroups(c *Context, w http.ResponseWriter, r *http.Request) {
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

	installationGroups, err := installationGroupsFromRequest(r)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get installation groups from request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installationGroups, err = c.Store.CreateRingInstallationGroups(ringID, installationGroups)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create ring installation groups")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ring.InstallationGroups = append(ring.InstallationGroups, installationGroups...)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, ring)
}

// handleDeleteRingInstallationGroup responds to DELETE /api/ring/{ring}/installationgroup/{installation-group-name},
// removes installation group from the Ring.
func handleDeleteRingInstallationGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ringID := vars["ring"]
	installationGroupName := vars["installation-group-name"]
	c.Logger = c.Logger.
		WithField("ring", ringID).
		WithField("action", "delete-ring-installatio-group").
		WithField("installation-group-name", installationGroupName)

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

	err := c.Store.DeleteRingInstallationGroup(ringID, installationGroupName)
	if err != nil {
		c.Logger.WithError(err).Error("failed delete ring installation group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func installationGroupsFromRequest(req *http.Request) ([]*model.InstallationGroup, error) {
	installationGroupsRequest, err := model.NewRegisterInstallationGroupsRequestFromReader(req.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}
	defer req.Body.Close()

	installationGroups, err := model.InstallationGroupsFromStringSlice(installationGroupsRequest.InstallationGroups)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate installation groups")
	}

	return installationGroups, nil
}
