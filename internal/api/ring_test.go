// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/elrond/internal/api"
	"github.com/mattermost/elrond/internal/store"
	"github.com/mattermost/elrond/internal/testlib"
	"github.com/mattermost/elrond/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRings(t *testing.T) {
	logger := testlib.MakeLogger(t)

	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)
	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("unknown ring", func(t *testing.T) {
		ring, err := client.GetRing(model.NewID())
		require.NoError(t, err)
		require.Nil(t, ring)
	})

	t.Run("no rings", func(t *testing.T) {
		rings, err := client.GetRings(&model.GetRingsRequest{
			Page:           0,
			PerPage:        10,
			IncludeDeleted: true,
		})
		require.NoError(t, err)
		require.Empty(t, rings)
	})

	t.Run("get rings", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/rings?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/rings?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/rings", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/rings?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/rings?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})
	t.Run("rings", func(t *testing.T) {
		ring1, err := client.CreateRing(&model.CreateRingRequest{
			Priority: 1,
			SoakTime: 7200,
		})
		require.NoError(t, err)
		require.NotNil(t, ring1)
		require.Equal(t, 7200, ring1.SoakTime)

		actualRing1, err := client.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, ring1.ID, actualRing1.ID)
		require.Equal(t, model.RingStateCreationRequested, actualRing1.State)
		require.Equal(t, 7200, actualRing1.SoakTime)

		time.Sleep(1 * time.Millisecond)

		ring2, err := client.CreateRing(&model.CreateRingRequest{
			Priority: 2,
			SoakTime: 3600,
		})
		require.NoError(t, err)
		require.NotNil(t, ring2)

		actualRing2, err := client.GetRing(ring2.ID)
		require.NoError(t, err)
		require.Equal(t, ring2.ID, actualRing2.ID)
		require.Equal(t, model.RingStateCreationRequested, actualRing2.State)
		require.Equal(t, 2, actualRing2.Priority)
		require.Equal(t, 3600, actualRing2.SoakTime)

		time.Sleep(1 * time.Millisecond)

		ring3, err := client.CreateRing(&model.CreateRingRequest{
			Priority: 2,
			SoakTime: 3600,
		})
		require.NoError(t, err)
		require.NotNil(t, ring3)

		actualRing3, err := client.GetRing(ring3.ID)
		require.NoError(t, err)
		require.Equal(t, ring3.ID, actualRing3.ID)

		time.Sleep(1 * time.Millisecond)

		t.Run("get rings, page 0, perPage 2, exclude deleted", func(t *testing.T) {
			rings, err := client.GetRings(&model.GetRingsRequest{
				Page:           0,
				PerPage:        2,
				IncludeDeleted: false,
			})
			require.NoError(t, err)
			require.Equal(t, []*model.Ring{actualRing1, actualRing2}, rings)
		})

		t.Run("get rings, page 1, perPage 2, exclude deleted", func(t *testing.T) {
			rings, err := client.GetRings(&model.GetRingsRequest{
				Page:           1,
				PerPage:        2,
				IncludeDeleted: false,
			})
			require.NoError(t, err)
			require.Equal(t, []*model.Ring{actualRing3}, rings)
		})

		t.Run("delete ring", func(t *testing.T) {
			ring2.State = model.RingStateStable
			err := sqlStore.UpdateRing(ring2)
			require.NoError(t, err)

			err = client.DeleteRing(ring2.ID)
			require.NoError(t, err)

			ring2, err = client.GetRing(ring2.ID)
			require.NoError(t, err)
			require.Equal(t, model.RingStateDeletionRequested, ring2.State)
		})

		t.Run("get rings after deletion request", func(t *testing.T) {
			t.Run("page 0, perPage 2, exclude deleted", func(t *testing.T) {
				rings, err := client.GetRings(&model.GetRingsRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Ring{actualRing1, ring2}, rings)
			})

			t.Run("page 1, perPage 2, exclude deleted", func(t *testing.T) {
				rings, err := client.GetRings(&model.GetRingsRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Ring{actualRing3}, rings)
			})

			t.Run("page 0, perPage 2, include deleted", func(t *testing.T) {
				rings, err := client.GetRings(&model.GetRingsRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Ring{actualRing1, ring2}, rings)
			})

			t.Run("page 1, perPage 2, include deleted", func(t *testing.T) {
				rings, err := client.GetRings(&model.GetRingsRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Ring{actualRing3}, rings)
			})
		})

		err = sqlStore.DeleteRing(ring2.ID)
		require.NoError(t, err)

		ring2, err = client.GetRing(ring2.ID)
		require.NoError(t, err)
		require.NotEqual(t, 0, ring2.DeleteAt)

		t.Run("get rings after actual deletion", func(t *testing.T) {
			t.Run("page 0, perPage 2, exclude deleted", func(t *testing.T) {
				rings, err := client.GetRings(&model.GetRingsRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Ring{actualRing1, actualRing3}, rings)
			})

			t.Run("page 1, perPage 2, exclude deleted", func(t *testing.T) {
				rings, err := client.GetRings(&model.GetRingsRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Ring{}, rings)
			})

			t.Run("page 0, perPage 2, include deleted", func(t *testing.T) {
				rings, err := client.GetRings(&model.GetRingsRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Ring{actualRing1, ring2}, rings)
			})

			t.Run("page 1, perPage 2, include deleted", func(t *testing.T) {
				rings, err := client.GetRings(&model.GetRingsRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Ring{actualRing3}, rings)
			})
		})
	})
}

func TestCreateRing(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/rings", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/rings", ts.URL), "application/json", bytes.NewReader([]byte("")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid priority", func(t *testing.T) {
		_, err := client.CreateRing(&model.CreateRingRequest{
			Priority:          0,
			InstallationGroup: &model.InstallationGroup{Name: "prod-12345"},
			SoakTime:          3600,
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("valid", func(t *testing.T) {
		ring, err := client.CreateRing(&model.CreateRingRequest{
			Priority:          1,
			InstallationGroup: &model.InstallationGroup{Name: "prod-12345"},
			SoakTime:          3600,
		})
		require.NoError(t, err)
		require.Equal(t, model.RingStateCreationRequested, ring.State)
		require.Equal(t, "prod-12345", ring.InstallationGroups[0].Name)
		require.Equal(t, 1, ring.Priority)
		require.Equal(t, 3600, ring.SoakTime)
	})
}

func TestRetryCreateRing(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	ring1, err := client.CreateRing(&model.CreateRingRequest{
		Priority:          1,
		InstallationGroup: &model.InstallationGroup{Name: "prod-12345"},
		SoakTime:          3600,
	})
	require.NoError(t, err)

	t.Run("unknown ring", func(t *testing.T) {
		err := client.RetryCreateRing(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		ring1.State = model.RingStateStable
		err = sqlStore.UpdateRing(ring1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockRing(ring1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockRing(ring1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		err = client.RetryCreateRing(ring1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while creating", func(t *testing.T) {
		ring1.State = model.RingStateCreationRequested
		err = sqlStore.UpdateRing(ring1)
		require.NoError(t, err)

		err = client.RetryCreateRing(ring1.ID)
		require.NoError(t, err)

		ring1, err = client.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, model.RingStateCreationRequested, ring1.State)
	})

	t.Run("while stable", func(t *testing.T) {
		ring1.State = model.RingStateStable
		err = sqlStore.UpdateRing(ring1)
		require.NoError(t, err)

		err = client.RetryCreateRing(ring1.ID)
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("while creation failed", func(t *testing.T) {
		ring1.State = model.RingStateCreationFailed
		err = sqlStore.UpdateRing(ring1)
		require.NoError(t, err)

		err = client.RetryCreateRing(ring1.ID)
		require.NoError(t, err)

		ring1, err = client.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, model.RingStateCreationRequested, ring1.State)
	})
}

func TestReleaseRing(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	ring1, err := client.CreateRing(&model.CreateRingRequest{
		Priority:          1,
		InstallationGroup: &model.InstallationGroup{Name: "prod-12345"},
		SoakTime:          3600,
	})
	require.NoError(t, err)

	t.Run("unknown ring", func(t *testing.T) {
		ringResp, err := client.ReleaseRing(model.NewID(), nil)
		require.EqualError(t, err, "failed with status code 404")
		assert.Nil(t, ringResp)
	})

	t.Run("while locked", func(t *testing.T) {
		ring1.State = model.RingStateStable
		err = sqlStore.UpdateRing(ring1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockRing(ring1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockRing(ring1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		ringResp, err := client.ReleaseRing(ring1.ID, nil)
		require.EqualError(t, err, "failed with status code 409")
		assert.Nil(t, ringResp)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockRingAPI(ring1.ID)
		require.NoError(t, err)

		ringResp, err := client.ReleaseRing(ring1.ID, nil)
		require.EqualError(t, err, "failed with status code 403")
		assert.Nil(t, ringResp)

		err = sqlStore.UnlockRingAPI(ring1.ID)
		require.NoError(t, err)
	})

	t.Run("while releasing", func(t *testing.T) {
		ring1.State = model.RingStateReleaseRequested
		err = sqlStore.UpdateRing(ring1)
		require.NoError(t, err)

		ringResp, err := client.ReleaseRing(ring1.ID, nil)
		require.EqualError(t, err, "failed with status code 500")
		assert.Nil(t, ringResp)

		ring1, err = client.GetRing(ring1.ID)
		require.NoError(t, err)
		require.Equal(t, model.RingStateReleaseRequested, ring1.State)
	})

	t.Run("while deleting", func(t *testing.T) {
		ring1.State = model.RingStateDeletionRequested
		err = sqlStore.UpdateRing(ring1)
		require.NoError(t, err)

		ringResp, err := client.ReleaseRing(ring1.ID, nil)
		require.EqualError(t, err, "failed with status code 500")
		assert.Nil(t, ringResp)
	})
}

func TestDeleteRing(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	ring1, err := client.CreateRing(&model.CreateRingRequest{
		Priority:          1,
		InstallationGroup: &model.InstallationGroup{Name: "prod-12345"},
	})
	require.NoError(t, err)

	t.Run("unknown ring", func(t *testing.T) {
		err := client.DeleteRing(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		ring1.State = model.RingStateStable
		err = sqlStore.UpdateRing(ring1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockRing(ring1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockRing(ring1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)

			ring1, err = client.GetRing(ring1.ID)
			require.NoError(t, err)
			require.Equal(t, int64(0), ring1.LockAcquiredAt)
		}()

		err = client.DeleteRing(ring1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockRingAPI(ring1.ID)
		require.NoError(t, err)

		err := client.DeleteRing(ring1.ID)
		require.EqualError(t, err, "failed with status code 403")

		err = sqlStore.UnlockRingAPI(ring1.ID)
		require.NoError(t, err)
	})

	// valid unlocked states
	states := []string{
		model.RingStateStable,
		model.RingStateCreationRequested,
		model.RingStateCreationFailed,
		model.RingStateReleaseFailed,
		model.RingStateDeletionRequested,
		model.RingStateDeletionFailed,
	}

	t.Run("from a valid, unlocked state", func(t *testing.T) {
		for _, state := range states {
			t.Run(state, func(t *testing.T) {
				ring1.State = state
				err = sqlStore.UpdateRing(ring1)
				require.NoError(t, err)

				err = client.DeleteRing(ring1.ID)
				require.NoError(t, err)

				ring1, err = client.GetRing(ring1.ID)
				require.NoError(t, err)
				require.Equal(t, model.RingStateDeletionRequested, ring1.State)
			})
		}
	})
}
