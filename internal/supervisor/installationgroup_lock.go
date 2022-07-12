// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type installationGroupLockStore interface {
	LockRingInstallationGroup(installationGroupID, lockerID string) (bool, error)
	UnlockRingInstallationGroup(installationGroupID, lockerID string, force bool) (bool, error)
}

type installationGroupLock struct {
	installationGroupID string
	lockerID            string
	store               installationGroupLockStore
	logger              log.FieldLogger
}

func newInstallationGroupLock(installationGroupID, lockerID string, store installationGroupLockStore, logger log.FieldLogger) *installationGroupLock {
	return &installationGroupLock{
		installationGroupID: installationGroupID,
		lockerID:            lockerID,
		store:               store,
		logger:              logger,
	}
}

func (l *installationGroupLock) TryLock() bool {
	locked, err := l.store.LockRingInstallationGroup(l.installationGroupID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock installation group")
		return false
	}

	return locked
}

func (l *installationGroupLock) Unlock() {
	unlocked, err := l.store.UnlockRingInstallationGroup(l.installationGroupID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock installation group")
	} else if !unlocked {
		l.logger.Error("failed to release lock for installation group")
	}
}
