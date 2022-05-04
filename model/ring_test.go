// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRingClone(t *testing.T) {
	ring := &Ring{
		Provisioner: "elrond",
		Priority:    2,
	}

	clone, err := ring.Clone()
	require.NoError(t, err)
	require.Equal(t, ring, clone)

	// Verify changing pointers in the clone doesn't affect the original.
	clone.Priority = 3
	require.NotEqual(t, ring, clone)
}

func TestRingFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		ring, err := RingFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &Ring{}, ring)
	})

	t.Run("invalid request", func(t *testing.T) {
		ring, err := RingFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, ring)
	})

	t.Run("request", func(t *testing.T) {
		ring, err := RingFromReader(bytes.NewReader([]byte(
			`{"ID":"id","InstallationGroup":"12345"}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &Ring{ID: "id", InstallationGroup: "12345"}, ring)
	})
}

func TestRingsFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		rings, err := RingsFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*Ring{}, rings)
	})

	t.Run("invalid request", func(t *testing.T) {
		rings, err := RingsFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, rings)
	})

	t.Run("request", func(t *testing.T) {
		ring, err := RingsFromReader(bytes.NewReader([]byte(
			`[{"ID":"id1", "InstallationGroup":"12345"}, {"ID":"id2", "InstallationGroup":"123456"}]`,
		)))
		require.NoError(t, err)
		require.Equal(t, []*Ring{
			{ID: "id1", InstallationGroup: "12345"},
			{ID: "id2", InstallationGroup: "123456"},
		}, ring)
	})
}
