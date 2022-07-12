// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"github.com/mattermost/elrond/model"
	"github.com/sirupsen/logrus"
)

// Supervisor describes the interface to notify the background jobs of an actionable change.
type Supervisor interface {
	Do() error
}

// Store describes the interface required to persist changes made via API requests.
type Store interface {
	CreateRing(ring *model.Ring, installationGroup *model.InstallationGroup) error
	GetRing(ringID string) (*model.Ring, error)
	GetRings(filter *model.RingFilter) ([]*model.Ring, error)
	UpdateRing(ring *model.Ring) error
	LockRing(ringID, lockerID string) (bool, error)
	UnlockRing(ringID, lockerID string, force bool) (bool, error)
	LockRingAPI(ringID string) error
	UnlockRingAPI(ringID string) error
	DeleteRing(ringID string) error

	GetInstallationGroupsForRings(filter *model.RingFilter) (map[string][]*model.InstallationGroup, error)
	GetInstallationGroupsForRing(ringID string) ([]*model.InstallationGroup, error)
	CreateRingInstallationGroup(ringID string, installationGroup *model.InstallationGroup) (*model.InstallationGroup, error)
	DeleteRingInstallationGroup(ringID string, installationGroup string) error
	UpdateInstallationGroup(installationGroup *model.InstallationGroup) error
	GetInstallationGroupByID(installationGroupID string) (*model.InstallationGroup, error)
	LockRingInstallationGroup(installationGroupID, lockerID string) (bool, error)
	UnlockRingInstallationGroup(installationGroupID, lockerID string, force bool) (bool, error)

	CreateWebhook(webhook *model.Webhook) error
	GetWebhook(webhookID string) (*model.Webhook, error)
	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
	DeleteWebhook(webhookID string) error
}

// TODO: will be used

// Elrond describes the interface.
type Elrond interface {
}

// Context provides the API with all necessary data and interfaces for responding to requests.
//
// It is cloned before each request, allowing per-request changes such as logger annotations.
type Context struct {
	Store             Store
	Supervisor        Supervisor
	Elrond            Elrond
	RequestID         string
	Environment       string
	Logger            logrus.FieldLogger
	ProvisionerServer string
}

// Clone creates a shallow copy of context, allowing clones to apply per-request changes.
func (c *Context) Clone() *Context {
	return &Context{
		Store:      c.Store,
		Supervisor: c.Supervisor,
		Elrond:     c.Elrond,
		Logger:     c.Logger,
	}
}
