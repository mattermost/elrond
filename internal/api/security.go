// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

// initSecurity registers security endpoints on the given router.
func initSecurity(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	securityRouter := apiRouter.PathPrefix("/security").Subrouter()

	securityClusterRouter := securityRouter.PathPrefix("/ring/{ring:[A-Za-z0-9]{26}}").Subrouter()
	securityClusterRouter.Handle("/api/lock", addContext(handleRingLockAPI)).Methods("POST")
	securityClusterRouter.Handle("/api/unlock", addContext(handleRingUnlockAPI)).Methods("POST")
}

// handleRingLockAPI responds to POST /api/security/ring/{ring}/api/lock,
// locking API changes for this ring.
func handleRingLockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
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

	if !ring.APISecurityLock {
		if err := c.Store.LockRingAPI(ring.ID); err != nil {
			c.Logger.WithError(err).Error("failed to lock ring API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleRingUnlockAPI responds to POST /api/security/ring/{ring}/api/unlock,
// unlocking API changes for this ring.
func handleRingUnlockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
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

	if ring.APISecurityLock {
		if err = c.Store.UnlockRingAPI(ring.ID); err != nil {
			c.Logger.WithError(err).Error("failed to unlock ring API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
