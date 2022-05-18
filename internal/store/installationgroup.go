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
	"InstallationGroup.ID", "InstallationGroup.Name",
}

var (
	// ErrInstallationGroupUsedByRing is an error returned when user attempts to delete a installation group
	// that is registered with a deployment ring.
	ErrInstallationGroupUsedByRing = errors.New("cannot delete installation group, " +
		"it is used by one or more deployment rings")
	// ErrInstallationGroupDoNotMatchRings is an error returned when user attempts to register an installation group to a
	// a ring that does not exist.
	ErrInstallationGroupDoNotMatchRings = errors.New("cannot register installation groups to ring, ring does not exist")
)

func init() {
	installationGroupSelect = sq.Select(installationGroupColumns...).
		From("InstallationGroup")
}

// GetInstallationGroupByName fetches the given installation group by name.
func (sqlStore *SQLStore) GetInstallationGroupByName(name string) (*model.InstallationGroup, error) {
	return sqlStore.getInstallationGroupByName(sqlStore.db, name)
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
		return nil, errors.Wrap(err, "failed to get intallation group by name")
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
			"ID":   installationGroup.ID,
			"Name": installationGroup.Name,
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

	err = sqlStore.createInstallationGroup(sqlStore.db, installationGroup)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create installation group")
	}

	return installationGroup, nil
}

// CreateRingInstallationGroups maps selected installation groups to ring and stores it in the database.
func (sqlStore *SQLStore) CreateRingInstallationGroups(ringID string, installationGroups []*model.InstallationGroup) ([]*model.InstallationGroup, error) {
	installationGroups, err := sqlStore.getOrCreateInstallationGroups(sqlStore.db, installationGroups)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get or create installation groups")
	}

	return sqlStore.createRingInstallationGroups(sqlStore.db, ringID, installationGroups)
}

func (sqlStore *SQLStore) createRingInstallationGroups(db execer, ringID string, installationGroups []*model.InstallationGroup) ([]*model.InstallationGroup, error) {
	builder := sq.Insert(ringInstallationGroupTable).
		Columns("ID", "RingID", "InstallationGroupID")

	for _, a := range installationGroups {
		builder = builder.Values(model.NewID(), ringID, a.ID)
	}
	_, err := sqlStore.execBuilder(db, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ring installation groups")
	}

	return installationGroups, nil
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

type ringInstallationGroup struct {
	RingID                string
	InstallationGroupID   string
	InstallationGroupName string
}

// GetInstallationGroupsForRings fetches all installation groups registered to the ring.
func (sqlStore *SQLStore) GetInstallationGroupsForRings(filter *model.RingFilter) (map[string][]*model.InstallationGroup, error) {
	var ringInstallationGroups []*ringInstallationGroup

	builder := sq.Select(
		"Ring.ID as RingID",
		"InstallationGroup.ID as InstallationGroupID",
		"InstallationGroup.Name as InstallationGroupName").
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
			&model.InstallationGroup{ID: rig.InstallationGroupID, Name: rig.InstallationGroupName},
		)
	}

	return installationGroups, nil
}

// DeleteRingInstallationGroup removes an installation group from a given ring.
func (sqlStore *SQLStore) DeleteRingInstallationGroup(ringID string, installationGroupName string) error {
	installationGroup, err := sqlStore.GetInstallationGroupByName(installationGroupName)
	if err != nil {
		return errors.Wrapf(err, "failed to get installation group '%s' by name", installationGroupName)
	}
	if installationGroup == nil {
		return nil
	}

	tx, err := sqlStore.beginCustomTransaction(sqlStore.db, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return errors.Wrap(err, "failed to begin the transaction")
	}
	defer tx.RollbackUnlessCommitted()

	builder := sq.Delete(ringInstallationGroupTable).
		Where("RingID = ?", ringID).
		Where("InstallationGroupID = ?", installationGroup.ID)

	result, err := sqlStore.execBuilder(tx, builder)
	if err != nil {
		return errors.Wrap(err, "failed to delete ring installation group")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to check affected rows when deleting ring installation group")
	}
	if rows > 1 { // Do not fail if installation group is not set on ring
		return fmt.Errorf("error deleting ring installation group, expected 0 or 1 rows to be affected was %d", rows)
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit the transaction")
	}

	return nil
}
