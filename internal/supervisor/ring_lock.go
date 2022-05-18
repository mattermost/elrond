// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type ringLockStore interface {
	LockRing(ringID, lockerID string) (bool, error)
	UnlockRing(ringID, lockerID string, force bool) (bool, error)
}

type ringLock struct {
	ringID   string
	lockerID string
	store    ringLockStore
	logger   log.FieldLogger
}

func newRingLock(ringID, lockerID string, store ringLockStore, logger log.FieldLogger) *ringLock {
	return &ringLock{
		ringID:   ringID,
		lockerID: lockerID,
		store:    store,
		logger:   logger,
	}
}

func (l *ringLock) TryLock() bool {
	locked, err := l.store.LockRing(l.ringID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock ring")
		return false
	}

	return locked
}

func (l *ringLock) Unlock() {
	unlocked, err := l.store.UnlockRing(l.ringID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock ring")
	} else if !unlocked {
		l.logger.Error("failed to release lock for ring")
	}
}
