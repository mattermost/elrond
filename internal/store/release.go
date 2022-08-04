// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/elrond/model"
	"github.com/pkg/errors"
)

const (
	ringReleaseTable = "RingRelease"
)

var ringReleaseSelect sq.SelectBuilder
var ringReleaseColumns = []string{
	"RingRelease.ID",
	"RingRelease.Image",
	"RingRelease.Version",
	"RingRelease.CreateAt",
	"RingRelease.Force",
}

type ringRelease struct {
	ID       string
	Image    string
	Version  string
	CreateAt int64
	Force    bool
}

func init() {
	ringReleaseSelect = sq.Select(ringReleaseColumns...).
		From("RingRelease")
}

// GetRingRelease fetches the given ring release by ID.
func (sqlStore *SQLStore) GetRingRelease(releaseID string) (*model.RingRelease, error) {
	return sqlStore.getRingRelease(sqlStore.db, releaseID)
}

// CreateRingRelease creates the given ring release.
func (sqlStore *SQLStore) CreateRingRelease(ringRelease *model.RingRelease) (*model.RingRelease, error) {
	return sqlStore.createRingRelease(sqlStore.db, ringRelease)
}

func (sqlStore *SQLStore) getRingRelease(db queryer, releaseID string) (*model.RingRelease, error) {
	var ringRelease model.RingRelease

	builder := ringReleaseSelect.
		Where("ID = ?", releaseID).
		Limit(1)
	err := sqlStore.getBuilder(db, &ringRelease, builder)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get ring release by id")
	}

	return &ringRelease, nil
}

func (sqlStore *SQLStore) createRingRelease(db execer, ringRelease *model.RingRelease) (*model.RingRelease, error) {
	ringRelease.ID = model.NewID()

	_, err := sqlStore.execBuilder(db, sq.Insert("RingRelease").
		SetMap(map[string]interface{}{
			"ID":       ringRelease.ID,
			"Image":    ringRelease.Image,
			"Version":  ringRelease.Version,
			"CreateAt": ringRelease.CreateAt,
			"Force":    ringRelease.Force,
		}))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create installation group")
	}

	return ringRelease, nil
}
