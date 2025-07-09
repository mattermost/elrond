// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"testing"
	"time"

	"github.com/mattermost/elrond/internal/store"
	"github.com/mattermost/elrond/internal/supervisor"
	"github.com/mattermost/elrond/internal/testlib"
	"github.com/mattermost/elrond/model"
	"github.com/stretchr/testify/require"
)

type mockRingStore struct {
	Ring                     *model.Ring
	UnlockedRingsPendingWork []*model.Ring
	Rings                    []*model.Ring

	UnlockChan      chan interface{}
	UpdateRingCalls int
}

func (s *mockRingStore) GetRing(_ string) (*model.Ring, error) {
	return s.Ring, nil
}

func (s *mockRingStore) GetUnlockedRingsPendingWork() ([]*model.Ring, error) {
	return s.UnlockedRingsPendingWork, nil
}

func (s *mockRingStore) GetRings(_ *model.RingFilter) ([]*model.Ring, error) {
	return s.Rings, nil
}

func (s *mockRingStore) GetRingsLocked() ([]*model.Ring, error) {
	return s.Rings, nil
}

func (s *mockRingStore) GetRingsReleaseInProgress() ([]*model.Ring, error) {
	return s.Rings, nil
}

func (s *mockRingStore) GetRingsPendingWork() ([]*model.Ring, error) {
	return s.Rings, nil
}

func (s *mockRingStore) UpdateRing(_ *model.Ring) error {
	s.UpdateRingCalls++
	return nil
}

func (s *mockRingStore) UpdateRings(_ []*model.Ring) error {
	s.UpdateRingCalls++
	return nil
}

func (s *mockRingStore) CreateRing(_ *model.Ring, _ *model.InstallationGroup) error {
	return nil
}

func (s *mockRingStore) LockRing(_, _ string) (bool, error) {
	return true, nil
}

func (s *mockRingStore) UnlockRing(_ string, _ string, _ bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s *mockRingStore) DeleteRing(_ string) error {
	return nil
}

func (s *mockRingStore) GetWebhooks(_ *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

func (s *mockRingStore) GetRingInstallationGroupsPendingWork(_ string) ([]*model.InstallationGroup, error) {
	return nil, nil
}

func (s *mockRingStore) GetInstallationGroupsForRing(_ string) ([]*model.InstallationGroup, error) {
	return nil, nil
}

func (s *mockRingStore) UpdateInstallationGroup(_ *model.InstallationGroup) error {
	return nil
}

func (s *mockRingStore) GetRingRelease(_ string) (*model.RingRelease, error) {
	return nil, nil
}

type mockRingProvisioner struct{}

func (p *mockRingProvisioner) PrepareRing(_ *model.Ring) bool {
	return true
}

func (p *mockRingProvisioner) CreateRing(_ *model.Ring) error {
	return nil
}

func (p *mockRingProvisioner) ReleaseRing(_ *model.Ring) error {
	return nil
}

func (p *mockRingProvisioner) SoakRing(_ *model.Ring) error {
	return nil
}

func (p *mockRingProvisioner) RollBackRing(_ *model.Ring) error {
	return nil
}

func (p *mockRingProvisioner) DeleteRing(_ *model.Ring) error {
	return nil
}

func TestRingSupervisorDo(t *testing.T) {
	t.Run("no Rings pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockRingStore{}

		supervisor := supervisor.NewRingSupervisor(mockStore, &mockRingProvisioner{}, "instanceID", logger)
		err := supervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateRingCalls)
	})

	t.Run("mock Ring creation", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockRingStore{}

		mockStore.UnlockedRingsPendingWork = []*model.Ring{{
			ID:    model.NewID(),
			State: model.RingStateCreationRequested,
		}}
		mockStore.Ring = mockStore.UnlockedRingsPendingWork[0]
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewRingSupervisor(mockStore, &mockRingProvisioner{}, "instanceID", logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 2, mockStore.UpdateRingCalls)
	})
}

func TestRingSupervisorSupervise(t *testing.T) {
	testCases := []struct {
		Description   string
		InitialState  string
		ExpectedState string
	}{
		{"unexpected state", model.RingStateStable, model.RingStateStable},
		{"creation requested", model.RingStateCreationRequested, model.RingStateStable},
		{"release pending", model.RingStateReleasePending, model.RingStateReleaseRequested},
		{"release requested", model.RingStateReleaseRequested, model.RingStateSoakingRequested},
		{"soaking requested", model.RingStateSoakingRequested, model.RingStateStable},
		{"rollback requested", model.RingStateReleaseRollbackRequested, model.RingStateReleaseRollbackComplete},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewRingSupervisor(sqlStore, &mockRingProvisioner{}, "instanceID", logger)

			release, err := sqlStore.GetOrCreateRingRelease(&model.RingRelease{
				Version:      "test-version",
				Image:        "test-image",
				Force:        false,
				EnvVariables: nil,
				CreateAt:     time.Now().UnixNano(),
			})
			require.NoError(t, err)

			Ring := &model.Ring{
				State:            tc.InitialState,
				ActiveReleaseID:  release.ID,
				DesiredReleaseID: release.ID,
			}

			installationGroup := model.InstallationGroup{
				Name:  "group1",
				State: model.InstallationGroupStable,
			}

			err = sqlStore.CreateRing(Ring, &installationGroup)
			require.NoError(t, err)

			supervisor.Supervise(Ring)

			Ring, err = sqlStore.GetRing(Ring.ID)
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedState, Ring.State)
		})

		t.Run(tc.Description, func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewRingSupervisor(sqlStore, &mockRingProvisioner{}, "instanceID", logger)

			release, err := sqlStore.GetOrCreateRingRelease(&model.RingRelease{
				Version:      "test-version",
				Image:        "test-image",
				Force:        true,
				EnvVariables: nil,
				CreateAt:     time.Now().UnixNano(),
			})
			require.NoError(t, err)

			Ring := &model.Ring{
				State:            tc.InitialState,
				ActiveReleaseID:  release.ID,
				DesiredReleaseID: release.ID,
			}

			installationGroup := model.InstallationGroup{
				Name:  "group1",
				State: model.InstallationGroupStable,
			}

			err = sqlStore.CreateRing(Ring, &installationGroup)
			require.NoError(t, err)

			supervisor.Supervise(Ring)

			Ring, err = sqlStore.GetRing(Ring.ID)
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedState, Ring.State)
		})
	}

	t.Run("state has changed since Ring was selected to be worked on", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewRingSupervisor(sqlStore, &mockRingProvisioner{}, "instanceID", logger)

		Ring := &model.Ring{
			State: model.RingStateDeletionRequested,
		}
		installationGroup := model.InstallationGroup{Name: "group2"}

		err := sqlStore.CreateRing(Ring, &installationGroup)
		require.NoError(t, err)

		// The stored Ring is RingStateDeletionRequested, so we will pass
		// in a Ring with state of RingStateCreationRequested to simulate
		// stale state.
		Ring.State = model.RingStateCreationRequested

		supervisor.Supervise(Ring)

		Ring, err = sqlStore.GetRing(Ring.ID)
		require.NoError(t, err)
		require.Equal(t, model.RingStateDeletionRequested, Ring.State)
	})
}
