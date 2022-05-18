// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

type mockSupervisor struct {
}

func (s *mockSupervisor) Do() error {
	return nil
}
