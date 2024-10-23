// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
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
		ring, getErr := client.GetRing(model.NewID())
		require.NoError(t, getErr)
		require.Nil(t, ring)
	})

	t.Run("no rings", func(t *testing.T) {
		rings, getErr := client.GetRings(&model.GetRingsRequest{
			Page:           0,
			PerPage:        10,
			IncludeDeleted: true,
		})
		require.NoError(t, getErr)
		require.Empty(t, rings)
	})

	t.Run("get rings", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, httpErr := http.Get(fmt.Sprintf("%s/api/rings?page=invalid&per_page=100", ts.URL))
			require.NoError(t, httpErr)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, httpErr := http.Get(fmt.Sprintf("%s/api/rings?page=0&per_page=invalid", ts.URL))
			require.NoError(t, httpErr)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, httpErr := http.Get(fmt.Sprintf("%s/api/rings", ts.URL))
			require.NoError(t, httpErr)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, httpErr := http.Get(fmt.Sprintf("%s/api/rings?per_page=100", ts.URL))
			require.NoError(t, httpErr)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, httpErr := http.Get(fmt.Sprintf("%s/api/rings?page=1", ts.URL))
			require.NoError(t, httpErr)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})
	t.Run("rings", func(t *testing.T) {
		ring1, createErr := client.CreateRing(&model.CreateRingRequest{
			Priority: 1,
			SoakTime: 7200,
		})
		require.NoError(t, createErr)
		require.NotNil(t, ring1)
		require.Equal(t, 7200, ring1.SoakTime)

		actualRing1, getErr := client.GetRing(ring1.ID)
		require.NoError(t, getErr)
		require.Equal(t, ring1.ID, actualRing1.ID)
		require.Equal(t, model.RingStateCreationRequested, actualRing1.State)
		require.Equal(t, 7200, actualRing1.SoakTime)

		time.Sleep(1 * time.Millisecond)

		ring2, createErr := client.CreateRing(&model.CreateRingRequest{
			Priority: 2,
			SoakTime: 3600,
		})
		require.NoError(t, createErr)
		require.NotNil(t, ring2)

		actualRing2, getErr := client.GetRing(ring2.ID)
		require.NoError(t, getErr)
		require.Equal(t, ring2.ID, actualRing2.ID)
		require.Equal(t, model.RingStateCreationRequested, actualRing2.State)
		require.Equal(t, 2, actualRing2.Priority)
		require.Equal(t, 3600, actualRing2.SoakTime)

		time.Sleep(1 * time.Millisecond)

		ring3, createErr := client.CreateRing(&model.CreateRingRequest{
			Priority: 2,
			SoakTime: 3600,
		})
		require.NoError(t, createErr)
		require.NotNil(t, ring3)

		actualRing3, getErr := client.GetRing(ring3.ID)
		require.NoError(t, getErr)
		require.Equal(t, ring3.ID, actualRing3.ID)

		time.Sleep(1 * time.Millisecond)

		t.Run("get rings, page 0, perPage 2, exclude deleted", func(t *testing.T) {
			rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
				Page:           0,
				PerPage:        2,
				IncludeDeleted: false,
			})
			require.NoError(t, getRingsErr)
			require.Equal(t, []*model.Ring{actualRing1, actualRing2}, rings)
		})

		t.Run("get rings, page 1, perPage 2, exclude deleted", func(t *testing.T) {
			rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
				Page:           1,
				PerPage:        2,
				IncludeDeleted: false,
			})
			require.NoError(t, getRingsErr)
			require.Equal(t, []*model.Ring{actualRing3}, rings)
		})

		t.Run("delete ring", func(t *testing.T) {
			ring2.State = model.RingStateStable
			updateErr := sqlStore.UpdateRing(ring2)
			require.NoError(t, updateErr)

			deleteErr := client.DeleteRing(ring2.ID)
			require.NoError(t, deleteErr)

			ring2, getErr = client.GetRing(ring2.ID)
			require.NoError(t, getErr)
			require.Equal(t, model.RingStateDeletionRequested, ring2.State)
		})

		t.Run("get rings after deletion request", func(t *testing.T) {
			t.Run("page 0, perPage 2, exclude deleted", func(t *testing.T) {
				rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, getRingsErr)
				require.Equal(t, []*model.Ring{actualRing1, ring2}, rings)
			})

			t.Run("page 1, perPage 2, exclude deleted", func(t *testing.T) {
				rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, getRingsErr)
				require.Equal(t, []*model.Ring{actualRing3}, rings)
			})

			t.Run("page 0, perPage 2, include deleted", func(t *testing.T) {
				rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, getRingsErr)
				require.Equal(t, []*model.Ring{actualRing1, ring2}, rings)
			})

			t.Run("page 1, perPage 2, include deleted", func(t *testing.T) {
				rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, getRingsErr)
				require.Equal(t, []*model.Ring{actualRing3}, rings)
			})
		})

		deleteErr := sqlStore.DeleteRing(ring2.ID)
		require.NoError(t, deleteErr)

		ring2, getErr = client.GetRing(ring2.ID)
		require.NoError(t, getErr)
		require.NotEqual(t, 0, ring2.DeleteAt)

		t.Run("get rings after actual deletion", func(t *testing.T) {
			t.Run("page 0, perPage 2, exclude deleted", func(t *testing.T) {
				rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, getRingsErr)
				require.Equal(t, []*model.Ring{actualRing1, actualRing3}, rings)
			})

			t.Run("page 1, perPage 2, exclude deleted", func(t *testing.T) {
				rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, getRingsErr)
				require.Equal(t, []*model.Ring{}, rings)
			})

			t.Run("page 0, perPage 2, include deleted", func(t *testing.T) {
				rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, getRingsErr)
				require.Equal(t, []*model.Ring{actualRing1, ring2}, rings)
			})

			t.Run("page 1, perPage 2, include deleted", func(t *testing.T) {
				rings, getRingsErr := client.GetRings(&model.GetRingsRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, getRingsErr)
				require.Equal(t, []*model.Ring{actualRing3}, rings)
			})
		})
	})
}
