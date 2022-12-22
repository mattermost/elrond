// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// Client is the programmatic interface to the elrond server API.
type Client struct {
	address    string
	headers    map[string]string
	httpClient *http.Client
}

// NewClient creates a client to the elrond server at the given address.
func NewClient(address string) *Client {
	return &Client{
		address:    address,
		headers:    make(map[string]string),
		httpClient: &http.Client{},
	}
}

// NewClientWithHeaders creates a client to the elrond server at the given
// address and uses the provided headers.
func NewClientWithHeaders(address string, headers map[string]string) *Client {
	return &Client{
		address:    address,
		headers:    headers,
		httpClient: &http.Client{},
	}
}

// closeBody ensures the Body of an http.Response is properly closed.
func closeBody(r *http.Response) {
	if r.Body != nil {
		_, _ = ioutil.ReadAll(r.Body)
		_ = r.Body.Close()
	}
}

func (c *Client) buildURL(urlPath string, args ...interface{}) string {
	return fmt.Sprintf("%s%s", c.address, fmt.Sprintf(urlPath, args...))
}

func (c *Client) doGet(u string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	return c.httpClient.Do(req)
}

func (c *Client) doPost(u string, request interface{}) (*http.Response, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(requestBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) doDelete(u string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	return c.httpClient.Do(req)
}

// CreateRing requests the creation of a ring from the configured elrond server.
func (c *Client) CreateRing(request *CreateRingRequest) (*Ring, error) {
	resp, err := c.doPost(c.buildURL("/api/rings"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return RingFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// RetryCreateRing retries the creation of a ring from the configured elrond server.
func (c *Client) RetryCreateRing(ringID string) error {
	resp, err := c.doPost(c.buildURL("/api/ring/%s", ringID), nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UpdateRing requests the update of a ring from the configured elrond server.
func (c *Client) UpdateRing(ringID string, request *UpdateRingRequest) (*Ring, error) {
	resp, err := c.doPost(c.buildURL("/api/ring/%s/update", ringID), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return RingFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// ReleaseRing releases a ring deployment form the configured elrond server.
func (c *Client) ReleaseRing(ringID string, request *RingReleaseRequest) (*Ring, error) {
	resp, err := c.doPost(c.buildURL("/api/ring/%s/release", ringID), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return RingFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetRingRelease fetches the specified ring release from the configured elrond server.
func (c *Client) GetRingRelease(releaseID string) (*RingRelease, error) {
	resp, err := c.doGet(c.buildURL("/api/release/%s", releaseID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return RingReleaseFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// ReleaseAllRings releases all ring deployments from the configured elrond server.
func (c *Client) ReleaseAllRings(request *RingReleaseRequest) ([]*Ring, error) {
	resp, err := c.doPost(c.buildURL("/api/rings/release"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return RingsFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// PauseRelease pauses all ring deployments from the configured elrond server.
func (c *Client) PauseRelease() error {
	resp, err := c.doPost(c.buildURL("/api/rings/release/pause"), nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// ResumeRelease resumes all paused ring deployments from the configured elrond server.
func (c *Client) ResumeRelease() error {
	resp, err := c.doPost(c.buildURL("/api/rings/release/resume"), nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// CancelRelease cancels all ring deployments in pending stat for the configured elrond server.
func (c *Client) CancelRelease() error {
	resp, err := c.doPost(c.buildURL("/api/rings/release/cancel"), nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetRing fetches the specified ring from the configured elrond server.
func (c *Client) GetRing(ringID string) (*Ring, error) {
	resp, err := c.doGet(c.buildURL("/api/ring/%s", ringID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return RingFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetRings fetches the list of rings from the configured elrond server.
func (c *Client) GetRings(request *GetRingsRequest) ([]*Ring, error) {
	u, err := url.Parse(c.buildURL("/api/rings"))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return RingsFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteRing deletes the given ring from the configured elrond server.
func (c *Client) DeleteRing(ringID string) error {
	resp, err := c.doDelete(c.buildURL("/api/ring/%s", ringID))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// CreateWebhook requests the creation of a webhook from the configured elrond server.
func (c *Client) CreateWebhook(request *CreateWebhookRequest) (*Webhook, error) {
	resp, err := c.doPost(c.buildURL("/api/webhooks"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return WebhookFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetWebhook fetches the webhook from the configured elrond server.
func (c *Client) GetWebhook(webhookID string) (*Webhook, error) {
	resp, err := c.doGet(c.buildURL("/api/webhook/%s", webhookID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return WebhookFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetWebhooks fetches the list of webhooks from the configured elrond server.
func (c *Client) GetWebhooks(request *GetWebhooksRequest) ([]*Webhook, error) {
	u, err := url.Parse(c.buildURL("/api/webhooks"))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return WebhooksFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteWebhook deletes the given webhook from the configured elrond server.
func (c *Client) DeleteWebhook(webhookID string) error {
	resp, err := c.doDelete(c.buildURL("/api/webhook/%s", webhookID))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// LockAPIForRing locks API changes for a given ring.
func (c *Client) LockAPIForRing(ringID string) error {
	return c.makeSecurityCall("ring", ringID, "api", "lock")
}

// UnlockAPIForRing unlocks API changes for a given ring.
func (c *Client) UnlockAPIForRing(ringID string) error {
	return c.makeSecurityCall("ring", ringID, "api", "unlock")
}

func (c *Client) makeSecurityCall(resourceType, id, securityType, action string) error {
	resp, err := c.doPost(c.buildURL("/api/security/%s/%s/%s/%s", resourceType, id, securityType, action), nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}

}

// RegisterRingInstallationGroup registers an installation group to the given ring.
func (c *Client) RegisterRingInstallationGroup(ringID string, installationGroupRequest *RegisterInstallationGroupRequest) (*Ring, error) {
	resp, err := c.doPost(c.buildURL("/api/ring/%s/installationgroup", ringID), installationGroupRequest)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return RingFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteRingInstallationGroup deletes installation group from the given ring.
func (c *Client) DeleteRingInstallationGroup(ringID string, installationGroupID string) error {
	resp, err := c.doDelete(c.buildURL("/api/ring/%s/installationgroup/%s", ringID, installationGroupID))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UpdateInstallationGroup requests the update of an installation group from the configured elrond server.
func (c *Client) UpdateInstallationGroup(installationGroup string, request *UpdateInstallationGroupRequest) (*InstallationGroup, error) {
	resp, err := c.doPost(c.buildURL("/api/installationgroup/%s/update", installationGroup), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return InstallationGroupFromReader(resp.Body)
	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}
