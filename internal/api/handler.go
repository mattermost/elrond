// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/mattermost/elrond/model"
	log "github.com/sirupsen/logrus"
)

type contextHandlerFunc func(c *Context, w http.ResponseWriter, r *http.Request)

type contextHandler struct {
	context *Context
	handler contextHandlerFunc
}

func (h contextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := h.context.Clone()
	context.RequestID = model.NewID()
	context.Logger = context.Logger.WithFields(log.Fields{
		"path":    r.URL.Path,
		"request": context.RequestID,
	})

	h.handler(context, w, r)
}

func newContextHandler(context *Context, handler contextHandlerFunc) *contextHandler {
	return &contextHandler{
		context: context,
		handler: handler,
	}
}
