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
		Select("Ring.ID", "Name", "Priority", "SoakTime", "ActiveReleaseID", "DesiredReleaseID", "Provisioner", "State", "CreateAt", "DeleteAt", "ReleaseAt", "APISecurityLock", "LockAcquiredBy", "LockAcquiredAt").
		From("Ring")
}

// GetRing fetches the given ring by id.
func (sqlStore *SQLStore) GetRing(id string) (*model.Ring, error) {
	var ring model.Ring
	err := sqlStore.getBuilder(sqlStore.db, &ring, ringSelect.Where("ID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get ring by id")
	}

	return &ring, nil
}

// GetRings fetches the given page of created rings. The first page is 0.
func (sqlStore *SQLStore) GetRings(filter *model.RingFilter) ([]*model.Ring, error) {
	builder := ringSelect.
		OrderBy("CreateAt ASC")
	builder = sqlStore.applyRingsFilter(builder, filter)

	var rings []*model.Ring
	err := sqlStore.selectBuilder(sqlStore.db, &rings, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for rings")
	}

	return rings, nil
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

// GetUnlockedRingsPendingWork returns rings pending work.
func (sqlStore *SQLStore) GetUnlockedRingsPendingWork() ([]*model.Ring, error) {
	builder := ringSelect.
		Where(sq.Eq{
			"State": model.AllRingStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("CreateAt ASC")

	var rings []*model.Ring
	err := sqlStore.selectBuilder(sqlStore.db, &rings, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for rings pending work")
	}

	return rings, nil
}

// GetRingsInPendingState returns rings in pending state.
func (sqlStore *SQLStore) GetRingsInPendingState() ([]*model.Ring, error) {
	builder := ringSelect.
		Where(sq.Eq{
			"State": model.AllRingStatesReleasePending,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("CreateAt ASC")

	var rings []*model.Ring
	err := sqlStore.selectBuilder(sqlStore.db, &rings, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for rings in pending state")
	}

	return rings, nil
}

// GetRingsLocked returns all rings that are under lock.
func (sqlStore *SQLStore) GetRingsLocked() ([]*model.Ring, error) {
	var rings []*model.Ring

	builder := ringSelect.
		Where("LockAcquiredAt > 0")

	err := sqlStore.selectBuilder(sqlStore.db, &rings, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for locked rings")
	}

	return rings, nil
}

// GetRingsReleaseInProgress returns all rings in a releasing state.
func (sqlStore *SQLStore) GetRingsReleaseInProgress() ([]*model.Ring, error) {
	var rings []*model.Ring

	builder := ringSelect.
		Where(sq.Eq{
			"State": model.AllRingStatesReleaseInProgress,
		})

	err := sqlStore.selectBuilder(sqlStore.db, &rings, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for rings with release in progress")
	}

	return rings, nil
}

// GetRingsPendingWork returns all rings in a pending work state.
func (sqlStore *SQLStore) GetRingsPendingWork() ([]*model.Ring, error) {
	var rings []*model.Ring

	builder := ringSelect.
		Where(sq.Eq{
			"State": model.AllRingStatesPendingWork,
		})
	err := sqlStore.selectBuilder(sqlStore.db, &rings, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for rings")
	}

	return rings, nil
}

// CreateRing records the given ring to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateRing(ring *model.Ring, installationGroup *model.InstallationGroup) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	if err = sqlStore.createRing(tx, ring); err != nil {
		return errors.Wrap(err, "failed to create ring")
	}

	if installationGroup != nil {
		if installationGroup.Name != "" {
			installationGroup, err := sqlStore.getOrCreateInstallationGroup(tx, installationGroup)
			if err != nil {
				return errors.Wrap(err, "failed to get or create installation group")
			}

			_, err = sqlStore.createRingInstallationGroup(tx, ring.ID, installationGroup)
			if err != nil {
				return errors.Wrap(err, "failed to register installation group for ring")
			}
		}
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
			"ID":               ring.ID,
			"Name":             ring.Name,
			"Priority":         ring.Priority,
			"State":            ring.State,
			"SoakTime":         ring.SoakTime,
			"ActiveReleaseID":  ring.ActiveReleaseID,
			"DesiredReleaseID": ring.DesiredReleaseID,
			"Provisioner":      ring.Provisioner,
			"CreateAt":         ring.CreateAt,
			"ReleaseAt":        ring.ReleaseAt,
			"DeleteAt":         ring.DeleteAt,
			"APISecurityLock":  ring.APISecurityLock,
			"LockAcquiredBy":   nil,
			"LockAcquiredAt":   0,
		}),
	); err != nil {
		return errors.Wrap(err, "failed to create ring")
	}

	return nil
}

// updateRings updates the given rings to the database when a single transaction is needed.
func (sqlStore *SQLStore) updateRings(execer execer, rings []*model.Ring) error {
	for _, ring := range rings {
		if _, err := sqlStore.execBuilder(execer, sq.
			Update("Ring").
			SetMap(map[string]interface{}{
				"Name":             ring.Name,
				"Priority":         ring.Priority,
				"State":            ring.State,
				"SoakTime":         ring.SoakTime,
				"Provisioner":      ring.Provisioner,
				"ActiveReleaseID":  ring.ActiveReleaseID,
				"DesiredReleaseID": ring.DesiredReleaseID,
				"ReleaseAt":        ring.ReleaseAt,
			}).
			Where("ID = ?", ring.ID),
		); err != nil {
			return errors.Wrap(err, "failed to update ring")
		}
	}

	return nil
}

// UpdateRing updates the given ring in the database.
func (sqlStore *SQLStore) UpdateRing(ring *model.Ring) error {
	if _, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Ring").
		SetMap(map[string]interface{}{
			"Name":             ring.Name,
			"Priority":         ring.Priority,
			"State":            ring.State,
			"SoakTime":         ring.SoakTime,
			"Provisioner":      ring.Provisioner,
			"ActiveReleaseID":  ring.ActiveReleaseID,
			"DesiredReleaseID": ring.DesiredReleaseID,
			"ReleaseAt":        ring.ReleaseAt,
		}).
		Where("ID = ?", ring.ID),
	); err != nil {
		return errors.Wrap(err, "failed to update ring")
	}

	return nil
}

// UpdateRings updates the given rings in the database.
func (sqlStore *SQLStore) UpdateRings(rings []*model.Ring) error {

	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	if err = sqlStore.updateRings(tx, rings); err != nil {
		return errors.Wrap(err, "failed to update rings")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit the transaction")
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

	_, err = sqlStore.DeleteInstallationGroupsFromRing(id)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation groups from deleted ring")
	}

	return nil
}

// LockRing marks the ring as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockRing(ringID, lockerID string) (bool, error) {
	return sqlStore.lockRows("Ring", []string{ringID}, lockerID)
}

// LockRings marks the rings as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockRings(rings []string, lockerID string) (bool, error) {
	return sqlStore.lockRows("Ring", rings, lockerID)
}

// UnlockRing releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockRing(ringID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("Ring", []string{ringID}, lockerID, force)
}

// UnlockRings releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockRings(rings []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("Ring", rings, lockerID, force)
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
