// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package webhook handles webhook management and delivery for the elrond server.
package webhook

import (
	"bytes"
	"net/http"
	"time"

	"github.com/mattermost/elrond/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type webhookStore interface {
	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// SendToAllWebhooks sends a given payload to all webhooks.
func SendToAllWebhooks(store webhookStore, payload *model.WebhookPayload, logger *log.Entry) error {
	hooks, err := store.GetWebhooks(&model.WebhookFilter{
		PerPage:        model.AllPerPage,
		IncludeDeleted: false,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to find webhooks")
	}

	sendWebhooks(hooks, payload, logger)

	return nil
}

// sendWebhooks sends webhooks via fire-and-forget goroutines. The send-webhook
// failures are logged, but not handled.
func sendWebhooks(hooks []*model.Webhook, payload *model.WebhookPayload, logger *log.Entry) {
	if len(hooks) == 0 {
		return
	}

	logger.Debugf("Sending %d webhook(s)", len(hooks))

	for _, hook := range hooks {
		go sendWebhook(hook, payload, logger) //nolint
	}
}

func sendWebhook(hook *model.Webhook, payload *model.WebhookPayload, logger *log.Entry) error {
	payloadStr, err := payload.ToJSON()
	if err != nil {
		logger.WithField("webhookURL", hook.URL).WithError(err).Error("Unable to create payload string to send to webhook")
		return errors.Wrap(err, "unable to create payload string to send to webhook")
	}

	req, err := http.NewRequest("POST", hook.URL, bytes.NewBuffer([]byte(payloadStr)))
	if err != nil {
		logger.WithField("webhookURL", hook.URL).WithError(err).Error("Unable to create webhook request")
		return errors.Wrap(err, "unable to create webhook request")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	if _, err = client.Do(req); err != nil {
		logger.WithField("webhookURL", hook.URL).WithError(err).Error("Unable to send webhook")
		return errors.Wrap(err, "unable to send webhook")
	}

	return nil
}
