// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/mattermost/elrond/model"
	"github.com/stretchr/testify/require"
)

func TestNewCreateRingRequestFromReader(t *testing.T) {
	defaultCreateRingRequest := func() *model.CreateRingRequest {
		return &model.CreateRingRequest{
			Priority:           1,
			InstallationGroups: []string{"prod-12345", "prod-1234567"},
			SoakTime:           7200,
		}
	}

	t.Run("invalid request", func(t *testing.T) {
		ringRequest, err := model.NewCreateRingRequestFromReader(bytes.NewReader([]byte(
			`{`,
		)))
		require.Error(t, err)
		require.Nil(t, ringRequest)
	})

	t.Run("partial request", func(t *testing.T) {
		ringRequest, err := model.NewCreateRingRequestFromReader(bytes.NewReader([]byte(
			`{"InstallationGroups": "prod-12345", "Priority": 2}`,
		)))
		require.NoError(t, err)
		modifiedDefaultCreateRingRequest := defaultCreateRingRequest()
		modifiedDefaultCreateRingRequest.InstallationGroups = []string{"prod-12345", "prod-1234567"}
		modifiedDefaultCreateRingRequest.Priority = 2
		require.Equal(t, modifiedDefaultCreateRingRequest, ringRequest)
	})

	t.Run("full request", func(t *testing.T) {
		ringRequest, err := model.NewCreateRingRequestFromReader(bytes.NewReader([]byte(
			`{"InstallationGroup": "prod-12345", "Priority": 2, "Name": "test"}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &model.CreateRingRequest{
			Priority:           2,
			InstallationGroups: []string{"prod-12345", "prod-1234567"},
			Name:               "test",
			SoakTime:           7200,
		}, ringRequest)
	})
}

func TestGetRingsRequestApplyToURL(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		u, err := url.Parse("http://localhost:3018")
		require.NoError(t, err)

		getRingsRequest := &model.GetRingsRequest{}
		getRingsRequest.ApplyToURL(u)

		require.Equal(t, "page=0&per_page=0", u.RawQuery)
	})

	t.Run("changes", func(t *testing.T) {
		u, err := url.Parse("http://localhost:3018")
		require.NoError(t, err)

		getRingsRequest := &model.GetRingsRequest{
			Page:           10,
			PerPage:        123,
			IncludeDeleted: true,
		}
		getRingsRequest.ApplyToURL(u)

		require.Equal(t, "include_deleted=true&page=10&per_page=123", u.RawQuery)
	})
}
