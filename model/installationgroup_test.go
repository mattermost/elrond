// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallationGroupsFromStringSlice(t *testing.T) {

	t.Run("valid installation groups", func(t *testing.T) {
		for _, testCase := range []struct {
			description        string
			names              []string
			installationGroups []*InstallationGroup
		}{
			{"nil array", nil, nil},
			{"empty array", []string{}, []*InstallationGroup{}},
			{
				"valid names",
				[]string{"abcd", "multi-tenant", "awesome_installation_group"},
				[]*InstallationGroup{{Name: "abcd"}, {Name: "multi-tenant"}, {Name: "awesome_installation_group"}},
			},
			{
				"long names",
				[]string{"multi-tenant-1234-abcd-very-long-name", "super-awesome-long_name"},
				[]*InstallationGroup{{Name: "multi-tenant-1234-abcd-very-long-name"}, {Name: "super-awesome-long_name"}},
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				installationGroups, err := InstallationGroupsFromStringSlice(testCase.names)
				require.NoError(t, err)
				assert.Equal(t, testCase.installationGroups, installationGroups)
			})
		}
	})
}

func TestSortInstallationGroups(t *testing.T) {

	for _, testCase := range []struct {
		description        string
		installationGroups []*InstallationGroup
		expected           []*InstallationGroup
	}{
		{
			description: "sort installation groups",
			installationGroups: []*InstallationGroup{
				{Name: "xyz"}, {Name: "other-installation-group"}, {Name: "other_installation_group"}, {Name: "abcdefgh"}, {Name: "abcd"},
			},
			expected: []*InstallationGroup{
				{Name: "abcd"}, {Name: "abcdefgh"}, {Name: "other-installation-group"}, {Name: "other_installation_group"}, {Name: "xyz"},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			SortInstallationGroups(testCase.installationGroups)
			assert.Equal(t, testCase.expected, testCase.installationGroups)
		})
	}
}

func TestNewRegisterInstallationGroupsRequestFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		installationGroupsRequest, err := NewRegisterInstallationGroupsRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &RegisterInstallationGroupsRequest{}, installationGroupsRequest)
	})

	t.Run("invalid request", func(t *testing.T) {
		installationGroupsRequest, err := NewRegisterInstallationGroupsRequestFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installationGroupsRequest)
	})

	t.Run("request", func(t *testing.T) {
		installationGroupsRequest, err := NewRegisterInstallationGroupsRequestFromReader(bytes.NewReader([]byte(
			`{"installationGroups":["abcd", "super-awesome"]}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &RegisterInstallationGroupsRequest{InstallationGroups: []string{"abcd", "super-awesome"}}, installationGroupsRequest)
	})
}

func TestContainsInstallationGroup(t *testing.T) {

	installationGroups := []*InstallationGroup{
		{ID: "1", Name: "installation-group1"},
		{ID: "2", Name: "my-installation-group"},
		{ID: "3", Name: "super-awesome"},
		{ID: "4", Name: "fourth"},
		{ID: "5", Name: "multi_tenant"},
	}

	for _, testCase := range []struct {
		description       string
		slice             []*InstallationGroup
		installationGroup *InstallationGroup
		isPresent         bool
	}{
		{
			description:       "should find installation group",
			slice:             installationGroups,
			installationGroup: &InstallationGroup{ID: "3", Name: "super-awesome"},
			isPresent:         true,
		},
		{
			description:       "should find with ID only",
			slice:             installationGroups,
			installationGroup: &InstallationGroup{ID: "5"},
			isPresent:         true,
		},
		{
			description:       "should not find installation group",
			slice:             installationGroups,
			installationGroup: &InstallationGroup{ID: "10", Name: "fourth"},
			isPresent:         false,
		},
		{
			description:       "should not find in empty slice",
			slice:             []*InstallationGroup{},
			installationGroup: &InstallationGroup{ID: "1"},
			isPresent:         false,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			found := ContainsInstallationGroup(testCase.slice, testCase.installationGroup)
			assert.Equal(t, testCase.isPresent, found)
		})
	}
}
