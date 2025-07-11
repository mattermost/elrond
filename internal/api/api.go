// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package api provides HTTP API handlers for the elrond server.
package api

import (
	"github.com/gorilla/mux"
)

// Register registers the API endpoints on the given router.
func Register(rootRouter *mux.Router, context *Context) {
	// api handler at /api
	apiRouter := rootRouter.PathPrefix("/api").Subrouter()
	initRing(apiRouter, context)
	initInstallationGroup(apiRouter, context)
	initWebhook(apiRouter, context)
	initSecurity(apiRouter, context)
}
