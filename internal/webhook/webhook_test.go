// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package webhook

import (
	"testing"
	"time"

	"github.com/mattermost/elrond/internal/testlib"
	"github.com/mattermost/elrond/model"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type mockWebhookStore struct {
	Webhooks []*model.Webhook
}

func (s *mockWebhookStore) GetWebhooks(_ *model.WebhookFilter) ([]*model.Webhook, error) {
	return s.Webhooks, nil
}

func TestGetAndSendWebhooks(t *testing.T) {
	mockStore := &mockWebhookStore{}
	logger := testlib.MakeLogger(t).WithFields(log.Fields{
		"webhooks-tests": true,
	})

	t.Run("no webhooks", func(t *testing.T) {
		err := SendToAllWebhooks(mockStore, nil, logger)
		require.NoError(t, err)
	})

	mockStore.Webhooks = append(mockStore.Webhooks, &model.Webhook{
		ID:       model.NewID(),
		OwnerID:  model.NewID(),
		URL:      "https://test.com",
		CreateAt: 10,
		DeleteAt: 0,
	})

	t.Run("1 webhook", func(t *testing.T) {
		err := SendToAllWebhooks(mockStore, nil, logger)
		require.NoError(t, err)
	})

	mockStore.Webhooks = append(mockStore.Webhooks, &model.Webhook{
		ID:       model.NewID(),
		OwnerID:  model.NewID(),
		URL:      "https://test2.com",
		CreateAt: 10,
		DeleteAt: 0,
	})

	t.Run("2 webhooks", func(t *testing.T) {
		err := SendToAllWebhooks(mockStore, nil, logger)
		require.NoError(t, err)
	})
}

func TestSendWebhooks(t *testing.T) {
	logger := testlib.MakeLogger(t).WithFields(log.Fields{
		"webhooks-tests": true,
	})
	hook := &model.Webhook{
		ID:       model.NewID(),
		OwnerID:  model.NewID(),
		URL:      "https://not-a-real-host",
		CreateAt: 10,
		DeleteAt: 0,
	}
	payload := &model.WebhookPayload{
		Type:      "type",
		ID:        model.NewID(),
		NewState:  "new_state",
		OldState:  "old_state",
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"RingID": model.NewID()},
	}

	err := sendWebhook(hook, payload, logger)
	require.Contains(t, err.Error(), "unable to send webhook")
}
