// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"strings"
	"testing"

	"github.com/mattermost/elrond/internal/testlib"
	"github.com/mattermost/elrond/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallationGroups_Ring(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installationGroup1 := model.InstallationGroup{Name: "installationGroup1"}
	installationGroup2 := model.InstallationGroup{Name: "installationGroup2"}

	err := sqlStore.CreateInstallationGroup(&installationGroup1)
	require.NoError(t, err)

	err = sqlStore.CreateInstallationGroup(&installationGroup2)
	require.NoError(t, err)

	t.Run("fail to create installation group with same name", func(t *testing.T) {
		err := sqlStore.CreateInstallationGroup(&model.InstallationGroup{Name: installationGroup1.Name})
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "unique constraint") // Make sure error comes from DB
	})

	t.Run("get installation group by name", func(t *testing.T) {
		installationGroup, err := sqlStore.GetInstallationGroupByName(installationGroup1.Name)
		require.NoError(t, err)
		assert.Equal(t, &installationGroup1, installationGroup)
	})

	t.Run("get unknown installation group", func(t *testing.T) {
		installationGroup, err := sqlStore.GetInstallationGroupByName("unknown")
		require.NoError(t, err)
		assert.Nil(t, installationGroup)
	})

	installationGroups := []*model.InstallationGroup{&installationGroup1, &installationGroup2}

	ring1 := model.Ring{}
	err = sqlStore.createRing(sqlStore.db, &ring1)
	require.NoError(t, err)

	_, err = sqlStore.CreateRingInstallationGroups(ring1.ID, installationGroups)
	require.NoError(t, err)

	t.Run("get installation groups for ring", func(t *testing.T) {
		installationGroupsForRing, err := sqlStore.GetInstallationGroupsForRing(ring1.ID)
		require.NoError(t, err)
		assert.Equal(t, len(installationGroups), len(installationGroupsForRing))
		assert.True(t, model.ContainsInstallationGroup(installationGroupsForRing, &installationGroup1))
		assert.True(t, model.ContainsInstallationGroup(installationGroupsForRing, &installationGroup2))
	})

	t.Run("fail to assign the same installation group to the ring twice", func(t *testing.T) {
		_, err = sqlStore.CreateRingInstallationGroups(ring1.ID, installationGroups)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "unique constraint") // Make sure error comes from DB
	})

	ring2 := model.Ring{}
	err = sqlStore.CreateRing(&ring2, installationGroups)
	require.NoError(t, err)

	t.Run("get installation groups for ring2", func(t *testing.T) {
		installationGroupsForRing, err := sqlStore.GetInstallationGroupsForRing(ring2.ID)
		require.NoError(t, err)
		assert.Equal(t, len(installationGroups), len(installationGroupsForRing))
		assert.True(t, model.ContainsInstallationGroup(installationGroupsForRing, &installationGroup1))
		assert.True(t, model.ContainsInstallationGroup(installationGroupsForRing, &installationGroup2))
	})

	t.Run("get installation groups for rings", func(t *testing.T) {
		installationGroupsForRings, err := sqlStore.GetInstallationGroupsForRings(&model.RingFilter{Paging: model.AllPagesNotDeleted()})
		require.NoError(t, err)
		assert.Equal(t, 2, len(installationGroupsForRings))
		assert.True(t, model.ContainsInstallationGroup(installationGroupsForRings[ring1.ID], &installationGroup1))
		assert.True(t, model.ContainsInstallationGroup(installationGroupsForRings[ring1.ID], &installationGroup2))
		assert.True(t, model.ContainsInstallationGroup(installationGroupsForRings[ring2.ID], &installationGroup1))
		assert.True(t, model.ContainsInstallationGroup(installationGroupsForRings[ring2.ID], &installationGroup2))
	})

	t.Run("delete ring installation group", func(t *testing.T) {
		err = sqlStore.DeleteRingInstallationGroup(ring1.ID, installationGroup1.Name)
		require.NoError(t, err)
		installationGroupsForRing, err := sqlStore.GetInstallationGroupsForRing(ring1.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, len(installationGroupsForRing))

		t.Run("do not fail when deleting ring installation group twice", func(t *testing.T) {
			err = sqlStore.DeleteRingInstallationGroup(ring1.ID, installationGroup1.Name)
			require.NoError(t, err)
		})
	})

	t.Run("delete unknown installation group", func(t *testing.T) {
		err = sqlStore.DeleteRingInstallationGroup(ring1.ID, "unknown-installation-group")
		require.NoError(t, err)
	})
}
