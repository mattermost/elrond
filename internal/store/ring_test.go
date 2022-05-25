// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"
	"time"

	"github.com/mattermost/elrond/internal/testlib"
	"github.com/mattermost/elrond/model"
	"github.com/stretchr/testify/require"
)

func TestRings(t *testing.T) {
	t.Run("get unknown ring", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)
		defer CloseConnection(t, sqlStore)

		ring, err := sqlStore.GetRing("unknown")
		require.NoError(t, err)
		require.Nil(t, ring)
	})

	t.Run("get rings", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)
		defer CloseConnection(t, sqlStore)

		ring1 := &model.Ring{
			Provisioner: "elrond",
			Name:        "test",
			Priority:    1,
			SoakTime:    60,
			State:       model.RingStateCreationRequested,
		}

		installationGroups := []*model.InstallationGroup{{Name: "12345"}, {Name: "123456"}}

		err := sqlStore.CreateRing(ring1, installationGroups)
		require.NoError(t, err)

		actualRing1, err := sqlStore.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, ring1, actualRing1)

		actualRings, err := sqlStore.GetRings(&model.RingFilter{Page: 0, PerPage: 0, IncludeDeleted: false})
		require.NoError(t, err)
		require.Empty(t, actualRings)

		actualRings, err = sqlStore.GetRings(&model.RingFilter{Page: 0, PerPage: 1, IncludeDeleted: false})
		require.NoError(t, err)
		require.Equal(t, []*model.Ring{ring1}, actualRings)

		actualRings, err = sqlStore.GetRings(&model.RingFilter{Page: 0, PerPage: 10, IncludeDeleted: false})
		require.NoError(t, err)
		require.Equal(t, []*model.Ring{ring1}, actualRings)

		actualRings, err = sqlStore.GetRings(&model.RingFilter{Page: 0, PerPage: 1, IncludeDeleted: true})
		require.NoError(t, err)
		require.Equal(t, []*model.Ring{ring1}, actualRings)

		actualRings, err = sqlStore.GetRings(&model.RingFilter{Page: 0, PerPage: 10, IncludeDeleted: true})
		require.NoError(t, err)
		require.Equal(t, []*model.Ring{ring1}, actualRings)

		actualRings, err = sqlStore.GetRings(&model.RingFilter{PerPage: model.AllPerPage, IncludeDeleted: true})
		require.NoError(t, err)
		require.Equal(t, []*model.Ring{ring1}, actualRings)
	})

	t.Run("update rings", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)
		defer CloseConnection(t, sqlStore)

		ring1 := &model.Ring{
			Provisioner: "elrond",
			Name:        "test",
			Priority:    1,
			SoakTime:    60,
			State:       model.RingStateCreationRequested,
		}

		installationGroups := []*model.InstallationGroup{{Name: "group1"}, {Name: "group2"}}

		err := sqlStore.CreateRing(ring1, installationGroups)
		require.NoError(t, err)

		ring1.Priority = 2
		ring1.SoakTime = 120
		ring1.State = model.RingStateDeletionRequested

		err = sqlStore.UpdateRing(ring1)
		require.NoError(t, err)

		actualRing1, err := sqlStore.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, ring1, actualRing1)
	})

	t.Run("delete ring", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)
		defer CloseConnection(t, sqlStore)

		ring1 := &model.Ring{
			Provisioner: "elrond",
			Name:        "test",
			Priority:    1,
			SoakTime:    60,
			State:       model.RingStateCreationRequested,
		}

		installationGroups := []*model.InstallationGroup{{Name: "group1"}, {Name: "group2"}}

		err := sqlStore.CreateRing(ring1, installationGroups)
		require.NoError(t, err)

		err = sqlStore.DeleteRing(ring1.ID)
		require.NoError(t, err)

		actualRing1, err := sqlStore.GetRing(ring1.ID)
		require.NoError(t, err)
		require.NotEqual(t, 0, actualRing1.DeleteAt)
		ring1.DeleteAt = actualRing1.DeleteAt
		require.Equal(t, ring1, actualRing1)

		actualRings, err := sqlStore.GetRings(&model.RingFilter{Page: 0, PerPage: 0, IncludeDeleted: false})
		require.NoError(t, err)
		require.Empty(t, actualRings)

		actualRings, err = sqlStore.GetRings(&model.RingFilter{Page: 0, PerPage: 1, IncludeDeleted: true})
		require.NoError(t, err)
		require.Equal(t, []*model.Ring{ring1}, actualRings)

		actualRings, err = sqlStore.GetRings(&model.RingFilter{Page: 0, PerPage: 10, IncludeDeleted: true})
		require.NoError(t, err)
		require.Equal(t, []*model.Ring{ring1}, actualRings)

		time.Sleep(1 * time.Millisecond)

		// Deleting again shouldn't change timestamp
		err = sqlStore.DeleteRing(ring1.ID)
		require.NoError(t, err)

		actualRing1, err = sqlStore.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, ring1, actualRing1)

	})
}

func TestGetUnlockedRingsPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	creationRequestedRing := &model.Ring{
		State: model.RingStateCreationRequested,
	}

	err := sqlStore.CreateRing(creationRequestedRing, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	releaseRequestedRing := &model.Ring{
		State: model.RingStateReleaseRequested,
	}
	err = sqlStore.CreateRing(releaseRequestedRing, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	deletionRequestedRing := &model.Ring{
		State: model.RingStateDeletionRequested,
	}
	err = sqlStore.CreateRing(deletionRequestedRing, nil)
	require.NoError(t, err)

	// Store rings with states that should be ignored by GetUnlockedRingsPendingWork()
	otherStates := []string{
		model.RingStateCreationFailed,
		model.RingStateReleaseFailed,
		model.RingStateDeletionFailed,
		model.RingStateDeleted,
		model.RingStateStable,
	}
	for _, otherState := range otherStates {
		err = sqlStore.CreateRing(&model.Ring{State: otherState}, nil)
		require.NoError(t, err)
	}

	rings, err := sqlStore.GetUnlockedRingsPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Ring{creationRequestedRing, releaseRequestedRing, deletionRequestedRing}, rings)

	lockerID := model.NewID()

	locked, err := sqlStore.LockRing(creationRequestedRing.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	rings, err = sqlStore.GetUnlockedRingsPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Ring{releaseRequestedRing, deletionRequestedRing}, rings)

	locked, err = sqlStore.LockRing(releaseRequestedRing.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	rings, err = sqlStore.GetUnlockedRingsPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Ring{deletionRequestedRing}, rings)

	locked, err = sqlStore.LockRing(deletionRequestedRing.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	rings, err = sqlStore.GetUnlockedRingsPendingWork()
	require.NoError(t, err)
	require.Empty(t, rings)
}

func TestLockRing(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	lockerID1 := model.NewID()
	lockerID2 := model.NewID()

	ring1 := &model.Ring{}
	err := sqlStore.CreateRing(ring1, nil)
	require.NoError(t, err)

	ring2 := &model.Ring{}
	err = sqlStore.CreateRing(ring2, nil)
	require.NoError(t, err)

	t.Run("rings should start unlocked", func(t *testing.T) {
		ring1, err = sqlStore.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), ring1.LockAcquiredAt)
		require.Nil(t, ring1.LockAcquiredBy)

		ring2, err = sqlStore.GetRing(ring2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), ring2.LockAcquiredAt)
		require.Nil(t, ring2.LockAcquiredBy)
	})

	t.Run("lock an unlocked ring", func(t *testing.T) {
		locked, err := sqlStore.LockRing(ring1.ID, lockerID1)
		require.NoError(t, err)
		require.True(t, locked)

		ring1, err = sqlStore.GetRing(ring1.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), ring1.LockAcquiredAt)
		require.Equal(t, lockerID1, *ring1.LockAcquiredBy)
	})

	t.Run("lock a previously locked ring", func(t *testing.T) {
		t.Run("by the same locker", func(t *testing.T) {
			locked, err := sqlStore.LockRing(ring1.ID, lockerID1)
			require.NoError(t, err)
			require.False(t, locked)
		})

		t.Run("by a different locker", func(t *testing.T) {
			locked, err := sqlStore.LockRing(ring1.ID, lockerID2)
			require.NoError(t, err)
			require.False(t, locked)
		})
	})

	t.Run("lock a second ring from a different locker", func(t *testing.T) {
		locked, err := sqlStore.LockRing(ring2.ID, lockerID2)
		require.NoError(t, err)
		require.True(t, locked)

		ring2, err = sqlStore.GetRing(ring2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), ring2.LockAcquiredAt)
		require.Equal(t, lockerID2, *ring2.LockAcquiredBy)
	})

	t.Run("unlock the first ring", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockRing(ring1.ID, lockerID1, false)
		require.NoError(t, err)
		require.True(t, unlocked)

		ring1, err = sqlStore.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), ring1.LockAcquiredAt)
		require.Nil(t, ring1.LockAcquiredBy)
	})

	t.Run("unlock the first ring again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockRing(ring1.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		ring1, err = sqlStore.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), ring1.LockAcquiredAt)
		require.Nil(t, ring1.LockAcquiredBy)
	})

	t.Run("force unlock the first ring again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockRing(ring1.ID, lockerID1, true)
		require.NoError(t, err)
		require.False(t, unlocked)

		ring1, err = sqlStore.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), ring1.LockAcquiredAt)
		require.Nil(t, ring1.LockAcquiredBy)
	})

	t.Run("unlock the second ring from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockRing(ring2.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		ring2, err = sqlStore.GetRing(ring2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), ring2.LockAcquiredAt)
		require.Equal(t, lockerID2, *ring2.LockAcquiredBy)
	})

	t.Run("force unlock the second ring from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockRing(ring2.ID, lockerID1, true)
		require.NoError(t, err)
		require.True(t, unlocked)

		ring2, err = sqlStore.GetRing(ring2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), ring2.LockAcquiredAt)
		require.Nil(t, ring2.LockAcquiredBy)
	})
}
