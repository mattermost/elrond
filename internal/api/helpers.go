// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/url"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func logSecurityLockConflict(resourceType string, logger logrus.FieldLogger) {
	logger.WithField("api-security-lock-conflict", resourceType).Warn("API security lock conflict detected")
}

func parseInt(u *url.URL, name string, defaultValue int) (int, error) {
	valueStr := u.Query().Get(name)
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse %s as integer", name)
	}

	return value, nil
}

func parseBool(u *url.URL, name string, defaultValue bool) (bool, error) {
	valueStr := u.Query().Get(name)
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse %s as boolean", name)
	}

	return value, nil
}

func parsePaging(u *url.URL) (int, int, bool, error) {
	page, err := parseInt(u, "page", 0)
	if err != nil {
		return 0, 0, false, err
	}

	perPage, err := parseInt(u, "per_page", 100)
	if err != nil {
		return 0, 0, false, err
	}

	includeDeleted, err := parseBool(u, "include_deleted", false)
	if err != nil {
		return 0, 0, false, err
	}

	return page, perPage, includeDeleted, nil
}
