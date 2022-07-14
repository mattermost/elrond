// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/elrond/model"
	"github.com/pkg/errors"
)

const (
	ringInstallationGroupTable = "RingInstallationGroup"
)

var installationGroupSelect sq.SelectBuilder
var installationGroupColumns = []string{
	"InstallationGroup.ID",
	"InstallationGroup.Name",
	"InstallationGroup.State",
	"InstallationGroup.SoakTime",
	"InstallationGroup.ProvisionerGroupID",
	"InstallationGroup.LockAcquiredBy",
	"InstallationGroup.LockAcquiredAt",
}

type ringInstallationGroup struct {
	RingID                              string
	InstallationGroupID                 string
	InstallationGroupName               string
	InstallationGroupState              string
	InstallationGroupReleaseAt          int64
	InstallationGroupSoakTime           int
	InstallationGroupProvisionerGroupID string
}

func init() {
	installationGroupSelect = sq.Select(installationGroupColumns...).
		From("InstallationGroup")
}

// GetInstallationGroupByName fetches the given installation group by name.
func (sqlStore *SQLStore) GetInstallationGroupByName(name string) (*model.InstallationGroup, error) {
	return sqlStore.getInstallationGroupByName(sqlStore.db, name)
}

// GetInstallationGroupByID fetches the given installation group by ID.
func (sqlStore *SQLStore) GetInstallationGroupByID(id string) (*model.InstallationGroup, error) {
	return sqlStore.getInstallationGroupByID(sqlStore.db, id)
}

func (sqlStore *SQLStore) getInstallationGroupByName(db queryer, name string) (*model.InstallationGroup, error) {
	var installationGroup model.InstallationGroup

	builder := installationGroupSelect.
		Where("Name = ?", name).
		Limit(1)
	err := sqlStore.getBuilder(db, &installationGroup, builder)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get installation group by name")
	}

	return &installationGroup, nil
}

func (sqlStore *SQLStore) getInstallationGroupByID(db queryer, id string) (*model.InstallationGroup, error) {
	var installationGroup model.InstallationGroup

	builder := installationGroupSelect.
		Where("ID = ?", id).
		Limit(1)
	err := sqlStore.getBuilder(db, &installationGroup, builder)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get installation group by id")
	}

	return &installationGroup, nil
}

// CreateInstallationGroup creates the given installation group to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallationGroup(installationGroup *model.InstallationGroup) error {
	return sqlStore.createInstallationGroup(sqlStore.db, installationGroup)
}

func (sqlStore *SQLStore) createInstallationGroup(db execer, installationGroup *model.InstallationGroup) error {
	installationGroup.ID = model.NewID()

	_, err := sqlStore.execBuilder(db, sq.Insert("InstallationGroup").
		SetMap(map[string]interface{}{
			"ID":                 installationGroup.ID,
			"Name":               installationGroup.Name,
			"State":              installationGroup.State,
			"ReleaseAt":          installationGroup.ReleaseAt,
			"SoakTime":           installationGroup.SoakTime,
			"ProvisionerGroupID": installationGroup.ProvisionerGroupID,
			"LockAcquiredBy":     nil,
			"LockAcquiredAt":     0,
		}))
	if err != nil {
		return errors.Wrap(err, "failed to create installation group")
	}

	return nil
}

// GetOrCreateInstallationGroups fetches installation groups by name or creates them if they do not exist.
func (sqlStore *SQLStore) GetOrCreateInstallationGroups(installationGroups []*model.InstallationGroup) ([]*model.InstallationGroup, error) {
	return sqlStore.getOrCreateInstallationGroups(sqlStore.db, installationGroups)
}

func (sqlStore *SQLStore) getOrCreateInstallationGroups(db dbInterface, installationGroups []*model.InstallationGroup) ([]*model.InstallationGroup, error) {
	for i, ig := range installationGroups {
		installationGroup, err := sqlStore.getOrCreateInstallationGroup(db, ig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get or create installation group '%s'", ig.Name)
		}
		installationGroups[i] = installationGroup
	}
	return installationGroups, nil
}

func (sqlStore *SQLStore) getOrCreateInstallationGroup(db dbInterface, installationGroup *model.InstallationGroup) (*model.InstallationGroup, error) {
	fetched, err := sqlStore.getInstallationGroupByName(db, installationGroup.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installation group by name")
	}
	if fetched != nil {
		return fetched, nil
	}

	err = sqlStore.createInstallationGroup(db, installationGroup)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create installation group")
	}

	return installationGroup, nil
}

// CreateRingInstallationGroup maps the selected installation group to ring and stores it in the database.
func (sqlStore *SQLStore) CreateRingInstallationGroup(ringID string, installationGroup *model.InstallationGroup) (*model.InstallationGroup, error) {
	installationGroup, err := sqlStore.getOrCreateInstallationGroup(sqlStore.db, installationGroup)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get or create installation groups")
	}

	return sqlStore.createRingInstallationGroup(sqlStore.db, ringID, installationGroup)
}

func (sqlStore *SQLStore) createRingInstallationGroup(db execer, ringID string, installationGroup *model.InstallationGroup) (*model.InstallationGroup, error) {
	builder := sq.Insert(ringInstallationGroupTable).
		Columns("ID", "RingID", "InstallationGroupID")

	builder = builder.Values(model.NewID(), ringID, installationGroup.ID)

	_, err := sqlStore.execBuilder(db, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ring installation group")
	}

	return installationGroup, nil
}

// GetInstallationGroupsForRing fetches all installation groups registered to the ring.
func (sqlStore *SQLStore) GetInstallationGroupsForRing(ringID string) ([]*model.InstallationGroup, error) {
	return sqlStore.getInstallationGroupsForRing(sqlStore.db, ringID)
}

func (sqlStore *SQLStore) getInstallationGroupsForRing(db dbInterface, ringID string) ([]*model.InstallationGroup, error) {
	var installationGroups []*model.InstallationGroup

	builder := sq.Select(installationGroupColumns...).
		From(ringInstallationGroupTable).
		Where("RingID = ?", ringID).
		LeftJoin("InstallationGroup ON InstallationGroup.ID=InstallationGroupID")
	err := sqlStore.selectBuilder(db, &installationGroups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installation groups for Ring")
	}

	return installationGroups, nil
}

// GetRingFromInstallationGroupID gets the ring that has the associated installation group.
func (sqlStore *SQLStore) GetRingFromInstallationGroupID(installationGroupID string) (*model.Ring, error) {
	return sqlStore.getRingFromInstallationGroupID(sqlStore.db, installationGroupID)
}

func (sqlStore *SQLStore) getRingFromInstallationGroupID(db dbInterface, installationGroupID string) (*model.Ring, error) {
	var ringID []string

	builder := sq.Select("RingID").
		From(ringInstallationGroupTable).
		Where("InstallationGroupID = ?", installationGroupID)
	err := sqlStore.selectBuilder(db, &ringID, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installation groups for Ring")
	}

	return sqlStore.GetRing(ringID[0])
}

// DeleteInstallationGroupsFromRing deletes all installation groups registered to the ring.
func (sqlStore *SQLStore) DeleteInstallationGroupsFromRing(ringID string) ([]*model.InstallationGroup, error) {
	return sqlStore.deleteInstallationGroupsFromRing(sqlStore.db, ringID)
}

func (sqlStore *SQLStore) deleteInstallationGroupsFromRing(db dbInterface, ringID string) ([]*model.InstallationGroup, error) {
	var installationGroups []*model.InstallationGroup

	builder := sq.Select(installationGroupColumns...).
		From(ringInstallationGroupTable).
		Where("RingID = ?", ringID).
		LeftJoin("InstallationGroup ON InstallationGroup.ID=InstallationGroupID")
	err := sqlStore.selectBuilder(db, &installationGroups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installation groups for Ring")
	}

	for _, ig := range installationGroups {
		err = sqlStore.DeleteRingInstallationGroup(ringID, ig.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to delete installation group from Ring")
		}

		err = sqlStore.deleteInstallationGroup(ig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to delete installation group")
		}
	}

	return installationGroups, nil
}

// GetInstallationGroupsForRings fetches all installation groups registered to rings.
func (sqlStore *SQLStore) GetInstallationGroupsForRings(filter *model.RingFilter) (map[string][]*model.InstallationGroup, error) {
	var ringInstallationGroups []*ringInstallationGroup

	builder := sq.Select(
		"Ring.ID as RingID",
		"InstallationGroup.ID as InstallationGroupID",
		"InstallationGroup.Name as InstallationGroupName",
		"InstallationGroup.State as InstallationGroupState",
		"InstallationGroup.ReleaseAt as InstallationGroupReleaseAt",
		"InstallationGroup.SoakTime as InstallationGroupSoakTime",
		"InstallationGroup.ProvisionerGroupID as InstallationGroupProvisionerGroupID").
		From("Ring").
		LeftJoin(fmt.Sprintf("%s ON %s.RingID = Ring.ID", ringInstallationGroupTable, ringInstallationGroupTable)).
		Join("InstallationGroup ON InstallationGroup.ID=InstallationGroupID")
	builder = sqlStore.applyRingsFilter(builder, filter)

	err := sqlStore.selectBuilder(sqlStore.db, &ringInstallationGroups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installation groups for ring")
	}

	installationGroups := map[string][]*model.InstallationGroup{}
	for _, rig := range ringInstallationGroups {
		installationGroups[rig.RingID] = append(
			installationGroups[rig.RingID],
			&model.InstallationGroup{
				ID:                 rig.InstallationGroupID,
				Name:               rig.InstallationGroupName,
				State:              rig.InstallationGroupState,
				ReleaseAt:          rig.InstallationGroupReleaseAt,
				SoakTime:           rig.InstallationGroupSoakTime,
				ProvisionerGroupID: rig.InstallationGroupProvisionerGroupID,
			},
		)
	}

	return installationGroups, nil
}

// DeleteRingInstallationGroup removes an installation group from a given ring.
func (sqlStore *SQLStore) DeleteRingInstallationGroup(ringID string, installationGroupID string) error {
	installationGroup, err := sqlStore.GetInstallationGroupByID(installationGroupID)
	if err != nil {
		return errors.Wrapf(err, "failed to get installation group '%s' by id", installationGroupID)
	}
	if installationGroup == nil {
		return nil
	}

	builder := sq.Delete(ringInstallationGroupTable).
		Where("RingID = ?", ringID).
		Where("InstallationGroupID = ?", installationGroup.ID)

	_, err = sqlStore.execBuilder(sqlStore.db, builder)
	if err != nil {
		return errors.Wrap(err, "failed to delete ring installation group")
	}

	return nil
}

// GetInstallationGroupsPendingWork returns all installation groups in a pending state.
func (sqlStore *SQLStore) GetInstallationGroupsPendingWork() ([]*model.InstallationGroup, error) {
	var installationGroups []*model.InstallationGroup

	builder := installationGroupSelect.
		Where(sq.Eq{
			"State": model.AllInstallationGroupStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0")

	err := sqlStore.selectBuilder(sqlStore.db, &installationGroups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation groups")
	}

	return installationGroups, nil
}

// GetInstallationGroupsReleaseInProgress returns all installation groups in a releasing state.
func (sqlStore *SQLStore) GetInstallationGroupsReleaseInProgress() ([]*model.InstallationGroup, error) {
	var installationGroups []*model.InstallationGroup

	builder := installationGroupSelect.
		Where(sq.Eq{
			"State": model.AllInstallationGroupStatesReleaseInProgress,
		}).
		Where("LockAcquiredAt = 0")

	err := sqlStore.selectBuilder(sqlStore.db, &installationGroups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation groups")
	}

	return installationGroups, nil
}

// GetInstallationGroupsLocked returns all installation groups that are under lock.
func (sqlStore *SQLStore) GetInstallationGroupsLocked() ([]*model.InstallationGroup, error) {
	var installationGroups []*model.InstallationGroup

	builder := installationGroupSelect.
		Where("LockAcquiredAt > 0")

	err := sqlStore.selectBuilder(sqlStore.db, &installationGroups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for locked installation groups")
	}

	return installationGroups, nil
}

// GetRingInstallationGroupsPendingWork returns all installation groups of a specific ring in a pending state.
func (sqlStore *SQLStore) GetRingInstallationGroupsPendingWork(ringID string) ([]*model.InstallationGroup, error) {
	var installationGroups []*model.InstallationGroup

	builder := sq.Select(installationGroupColumns...).
		From(ringInstallationGroupTable).
		Where("RingID = ?", ringID).
		Where(sq.Eq{
			"State": model.AllInstallationGroupStatesPendingWork,
		}).
		LeftJoin("InstallationGroup ON InstallationGroup.ID=InstallationGroupID")
	err := sqlStore.selectBuilder(sqlStore.db, &installationGroups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pending installation groups for Ring")
	}

	return installationGroups, nil
}

// UpdateInstallationGroup updates the given installation group in the database.
func (sqlStore *SQLStore) UpdateInstallationGroup(installationGroup *model.InstallationGroup) error {

	if _, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("InstallationGroup").
		SetMap(map[string]interface{}{
			"Name":               installationGroup.Name,
			"State":              installationGroup.State,
			"ReleaseAt":          installationGroup.ReleaseAt,
			"SoakTime":           installationGroup.SoakTime,
			"ProvisionerGroupID": installationGroup.ProvisionerGroupID,
		}).
		Where("ID = ?", installationGroup.ID),
	); err != nil {
		return errors.Wrap(err, "failed to update installation group")
	}

	return nil
}

func (sqlStore *SQLStore) deleteInstallationGroup(installationGroup *model.InstallationGroup) error {

	if _, err := sqlStore.execBuilder(sqlStore.db, sq.
		Delete("InstallationGroup").
		Where("ID = ?", installationGroup.ID),
	); err != nil {
		return errors.Wrap(err, "failed to delete installation group")
	}

	return nil
}

// LockRingInstallationGroup marks the ring as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockRingInstallationGroup(installationGroupID, lockerID string) (bool, error) {
	return sqlStore.lockRows("InstallationGroup", []string{installationGroupID}, lockerID)
}

// UnlockRingInstallationGroup releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockRingInstallationGroup(installationGroupID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("InstallationGroup", []string{installationGroupID}, lockerID, force)
}
