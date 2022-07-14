// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"

	"github.com/mattermost/elrond/model"
	"github.com/stretchr/testify/assert"
)

func TestCreateRingRequestValid(t *testing.T) {
	var testCases = []struct {
		testName     string
		request      *model.CreateRingRequest
		requireError bool
	}{
		{"defaults", &model.CreateRingRequest{SoakTime: 3600, Priority: 1, InstallationGroup: &model.InstallationGroup{Name: "test2"}}, false},
		{"invalid priority", &model.CreateRingRequest{Priority: 0}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			tc.request.SetDefaults()

			if tc.requireError {
				assert.Error(t, tc.request.Validate())
			} else {
				assert.NoError(t, tc.request.Validate())
			}
		})
	}
}
