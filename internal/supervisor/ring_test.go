// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"testing"

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

func (s *mockRingStore) GetRing(RingID string) (*model.Ring, error) {
	return s.Ring, nil
}

func (s *mockRingStore) GetUnlockedRingsPendingWork() ([]*model.Ring, error) {
	return s.UnlockedRingsPendingWork, nil
}

func (s *mockRingStore) GetRings(RingFilter *model.RingFilter) ([]*model.Ring, error) {
	return s.Rings, nil
}

func (s *mockRingStore) UpdateRing(Ring *model.Ring) error {
	s.UpdateRingCalls++
	return nil
}

func (s *mockRingStore) CreateRing(Ring *model.Ring, installationGroup *model.InstallationGroup) error {
	return nil
}

func (s *mockRingStore) LockRing(RingID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockRingStore) UnlockRing(RingID string, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s *mockRingStore) DeleteRing(RingID string) error {
	return nil
}

func (s *mockRingStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

func (s *mockRingStore) GetRingInstallationGroupsPendingWork(ringID string) ([]*model.InstallationGroup, error) {
	return nil, nil
}

type mockRingProvisioner struct{}

func (p *mockRingProvisioner) PrepareRing(Ring *model.Ring) bool {
	return true
}

func (p *mockRingProvisioner) CreateRing(Ring *model.Ring) error {
	return nil
}

func (p *mockRingProvisioner) ReleaseRing(Ring *model.Ring) error {
	return nil
}

func (p *mockRingProvisioner) SoakRing(Ring *model.Ring) error {
	return nil
}

func (p *mockRingProvisioner) RollBackRing(Ring *model.Ring) error {
	return nil
}

func (p *mockRingProvisioner) DeleteRing(Ring *model.Ring) error {
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
		{"release requested", model.RingStateReleaseRequested, model.RingStateSoakingRequested},
		{"soaking requested", model.RingStateSoakingRequested, model.RingStateStable},
		{"rollback requested", model.RingStateReleaseRollbackRequested, model.RingStateReleaseRollbackComplete},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewRingSupervisor(sqlStore, &mockRingProvisioner{}, "instanceID", logger)

			Ring := &model.Ring{
				State: tc.InitialState,
			}

			installationGroup := model.InstallationGroup{Name: "group1"}

			err := sqlStore.CreateRing(Ring, &installationGroup)
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
