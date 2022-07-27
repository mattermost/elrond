// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/elrond/model"
)

// initRing registers ring endpoints on the given router.
func initInstallationGroup(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	installationGroupRouter := apiRouter.PathPrefix("/installationgroup/{installationgroup:[A-Za-z0-9]{26}}").Subrouter()
	installationGroupRouter.Handle("/update", addContext(handleUpdateInstallationGroup)).Methods("POST")
}

// handleUpdateInstallationGroup responds to POST /api/installationgroup/{installationgroup}/update,
// updating an installation group.
func handleUpdateInstallationGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationGroupID := vars["installationgroup"]
	c.Logger = c.Logger.
		WithField("installationgroup", installationGroupID).
		WithField("action", "update-installation-group")

	installationGroup, err := c.Store.GetInstallationGroupByID(installationGroupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to update installation group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	updateInstallationGroupRequest, err := model.NewUpdateInstallationGroupRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to deserialize ring update request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if updateInstallationGroupRequest.Name != "" {
		installationGroup.Name = updateInstallationGroupRequest.Name
	}

	if updateInstallationGroupRequest.SoakTime != installationGroup.SoakTime && updateInstallationGroupRequest.SoakTime != 0 {
		installationGroup.SoakTime = updateInstallationGroupRequest.SoakTime
	}

	if updateInstallationGroupRequest.ProvisionerGroupID != "" {
		installationGroup.ProvisionerGroupID = updateInstallationGroupRequest.ProvisionerGroupID
	}

	if err = c.Store.UpdateInstallationGroup(installationGroup); err != nil {
		c.Logger.WithError(err).Error("failed to update installation group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationGroup)
}
