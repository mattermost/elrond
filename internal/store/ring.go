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

var ringSelect sq.SelectBuilder

func init() {
	ringSelect = sq.
		Select("Ring.ID", "Name", "Priority", "InstallationGroup", "SoakTime", "Image", "Version", "Provisioner", "State", "CreateAt", "DeleteAt", "ReleaseAt", "APISecurityLock", "LockAcquiredBy", "LockAcquiredAt").
		From("Ring")
}

type rawRing struct {
	*model.Ring
}

type rawRings []*rawRing

func (r *rawRing) toRing() (*model.Ring, error) {
	return r.Ring, nil
}

func (rc *rawRings) toRings() ([]*model.Ring, error) {
	var rings []*model.Ring
	for _, rawRing := range *rc {
		ring, err := rawRing.toRing()
		if err != nil {
			return nil, err
		}
		rings = append(rings, ring)
	}

	return rings, nil
}

// GetRing fetches the given ring by id.
func (sqlStore *SQLStore) GetRing(id string) (*model.Ring, error) {
	var rawRing rawRing
	err := sqlStore.getBuilder(sqlStore.db, &rawRing, ringSelect.Where("ID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get ring by id")
	}

	return rawRing.toRing()
}

// GetRings fetches the given page of created rings. The first page is 0.
func (sqlStore *SQLStore) GetRings(filter *model.RingFilter) ([]*model.Ring, error) {
	builder := ringSelect.
		OrderBy("CreateAt ASC")
	builder = sqlStore.applyRingsFilter(builder, filter)

	var rawRings rawRings
	err := sqlStore.selectBuilder(sqlStore.db, &rawRings, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for rings")
	}

	return rawRings.toRings()
}

func (sqlStore *SQLStore) applyRingsFilter(builder sq.SelectBuilder, filter *model.RingFilter) sq.SelectBuilder {
	if filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	if !filter.IncludeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	return builder
}

// GetUnlockedRingsPendingWork returns an unlocked ring in a pending state.
func (sqlStore *SQLStore) GetUnlockedRingsPendingWork() ([]*model.Ring, error) {
	builder := ringSelect.
		Where(sq.Eq{
			"State": model.AllRingStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("CreateAt ASC")

	var rawRings rawRings
	err := sqlStore.selectBuilder(sqlStore.db, &rawRings, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for rings")
	}

	return rawRings.toRings()
}

// CreateRing records the given ring to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateRing(ring *model.Ring) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	if err = sqlStore.createRing(tx, ring); err != nil {
		return errors.Wrap(err, "failed to create ring")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit the transaction")
	}

	return nil
}

// createRing records the given ring to the database, assigning it a unique ID.
func (sqlStore *SQLStore) createRing(execer execer, ring *model.Ring) error {
	ring.ID = model.NewID()
	ring.CreateAt = GetMillis()

	if _, err := sqlStore.execBuilder(execer, sq.
		Insert("Ring").
		SetMap(map[string]interface{}{
			"ID":                ring.ID,
			"Name":              ring.Name,
			"Priority":          ring.Priority,
			"State":             ring.State,
			"InstallationGroup": ring.InstallationGroup,
			"SoakTime":          ring.SoakTime,
			"Image":             ring.Image,
			"Version":           ring.Version,
			"Provisioner":       ring.Provisioner,
			"CreateAt":          ring.CreateAt,
			"ReleaseAt":         ring.ReleaseAt,
			"DeleteAt":          ring.DeleteAt,
			"APISecurityLock":   ring.APISecurityLock,
			"LockAcquiredBy":    nil,
			"LockAcquiredAt":    0,
		}),
	); err != nil {
		return errors.Wrap(err, "failed to create ring")
	}

	return nil
}

// UpdateRing updates the given ring in the database.
func (sqlStore *SQLStore) UpdateRing(ring *model.Ring) error {

	if _, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Ring").
		SetMap(map[string]interface{}{
			"Name":              ring.Name,
			"Priority":          ring.Priority,
			"State":             ring.State,
			"InstallationGroup": ring.InstallationGroup,
			"SoakTime":          ring.SoakTime,
			"Provisioner":       ring.Provisioner,
			"Image":             ring.Image,
			"Version":           ring.Version,
		}).
		Where("ID = ?", ring.ID),
	); err != nil {
		return errors.Wrap(err, "failed to update ring")
	}

	return nil
}

// DeleteRing marks the given ring as deleted, but does not remove the record from the
// database.
func (sqlStore *SQLStore) DeleteRing(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Ring").
		Set("DeleteAt", GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark ring as deleted")
	}

	return nil
}

// LockRing marks the ring as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockRing(ringID, lockerID string) (bool, error) {
	return sqlStore.lockRows("Ring", []string{ringID}, lockerID)
}

// UnlockRing releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockRing(ringID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("Ring", []string{ringID}, lockerID, force)
}

// LockRingAPI locks updates to the ring from the API.
func (sqlStore *SQLStore) LockRingAPI(ringID string) error {
	return sqlStore.setRingAPILock(ringID, true)
}

// UnlockRingAPI unlocks updates to the ring from the API.
func (sqlStore *SQLStore) UnlockRingAPI(ringID string) error {
	return sqlStore.setRingAPILock(ringID, false)
}

func (sqlStore *SQLStore) setRingAPILock(ringID string, lock bool) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Ring").
		Set("APISecurityLock", lock).
		Where("ID = ?", ringID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to store ring API lock")
	}

	return nil
}
