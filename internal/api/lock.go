// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"sync"

	"github.com/mattermost/elrond/model"
)

// lockRing synchronizes access to the given elrond across potentially
// multiple elrond servers.
func lockRing(c *Context, ringID string) (*model.Ring, int, func()) {
	ring, err := c.Store.GetRing(ringID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query ring")
		return nil, http.StatusInternalServerError, nil
	}
	if ring == nil {
		return nil, http.StatusNotFound, nil
	}

	locked, err := c.Store.LockRing(ringID, c.RequestID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to lock ring")
		return nil, http.StatusInternalServerError, nil
	} else if !locked {
		c.Logger.Error("failed to acquire lock for ring")
		return nil, http.StatusConflict, nil
	}

	unlockOnce := sync.Once{}

	return ring, 0, func() {
		unlockOnce.Do(func() {
			unlocked, err := c.Store.UnlockRing(ring.ID, c.RequestID, false)
			if err != nil {
				c.Logger.WithError(err).Errorf("failed to unlock ring")
			} else if !unlocked {
				c.Logger.Error("failed to release lock for ring")
			}
		})
	}
}
