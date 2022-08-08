package store

import (
	"testing"

	"github.com/mattermost/elrond/internal/testlib"
	"github.com/mattermost/elrond/model"
	"github.com/stretchr/testify/require"
)

func TestRingRelease(t *testing.T) {
	t.Run("get unknown ring release", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)
		defer CloseConnection(t, sqlStore)

		ringRelease, err := sqlStore.getRingRelease(sqlStore.db, "unknown")
		require.NoError(t, err)
		require.Nil(t, ringRelease)
	})

	t.Run("get ring release", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)
		defer CloseConnection(t, sqlStore)

		ringRelease1 := &model.RingRelease{
			Image:   "test",
			Version: "test",
			Force:   false,
		}

		ringRelease, err := sqlStore.GetOrCreateRingRelease(ringRelease1)
		require.NoError(t, err)

		actualRingRelease1, err := sqlStore.GetRingRelease(ringRelease.ID)
		require.NoError(t, err)
		require.Equal(t, ringRelease1, actualRingRelease1)
	})
}
