// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/elrond/model"
	cmodel "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
// ringReleaseTable = "RingRelease"
)

var ringReleaseSelect sq.SelectBuilder
var ringReleaseColumns = []string{
	"RingRelease.ID",
	"RingRelease.Image",
	"RingRelease.Version",
	"RingRelease.CreateAt",
	"RingRelease.Force",
	"RingRelease.EnvVariables",
	"RingRelease.ReadinessProbe",
	"RingRelease.LivenessProbe",
}

type rawRingRelease struct {
	*model.RingRelease
	EnvVariables   []byte
	ReadinessProbe []byte
	LivenessProbe  []byte
}

func init() {
	ringReleaseSelect = sq.Select(ringReleaseColumns...).
		From("RingRelease")
}

func (r *rawRingRelease) toRingRelease() (*model.RingRelease, error) {
	// We only need to set values that are converted from a raw database format.
	var err error
	mattermostEnv := &cmodel.EnvVarMap{}
	if r.EnvVariables != nil {
		mattermostEnv, err = cmodel.EnvVarFromJSON(r.EnvVariables)
		if err != nil {
			return nil, err
		}
	}

	r.RingRelease.EnvVariables = *mattermostEnv

	// Handle ReadinessProbe
	if r.ReadinessProbe != nil {
		var readinessProbe corev1.Probe
		if err := json.Unmarshal(r.ReadinessProbe, &readinessProbe); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal readiness probe")
		}
		r.RingRelease.ReadinessProbe = &readinessProbe
	}

	// Handle LivenessProbe
	if r.LivenessProbe != nil {
		var livenessProbe corev1.Probe
		if err := json.Unmarshal(r.LivenessProbe, &livenessProbe); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal liveness probe")
		}
		r.RingRelease.LivenessProbe = &livenessProbe
	}

	return r.RingRelease, nil
}

// GetRingRelease fetches the given ring release by ID.
func (sqlStore *SQLStore) GetRingRelease(releaseID string) (*model.RingRelease, error) {
	return sqlStore.getRingRelease(sqlStore.db, releaseID)
}

// GetOrCreateRingRelease checks if the given ring release exists otherwise it creates it.
func (sqlStore *SQLStore) GetOrCreateRingRelease(ringRelease *model.RingRelease) (*model.RingRelease, error) {
	return sqlStore.getOrCreateRingRelease(sqlStore.db, ringRelease)
}

func (sqlStore *SQLStore) getRingRelease(db queryer, releaseID string) (*model.RingRelease, error) {
	var rawRingReleaseOutput rawRingRelease

	builder := ringReleaseSelect.
		Where("ID = ?", releaseID).
		Limit(1)
	err := sqlStore.getBuilder(db, &rawRingReleaseOutput, builder)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get ring release by id")
	}

	return rawRingReleaseOutput.toRingRelease()
}

func (sqlStore *SQLStore) getOrCreateRingRelease(db execer, ringRelease *model.RingRelease) (*model.RingRelease, error) {
	var rawRingReleaseOutput rawRingRelease
	ringRelease.ID = model.NewID()

	envVarMap, err := ringRelease.EnvVariables.ToJSON()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new EnvVarMap JSON")
	}

	// Marshal probe data
	var readinessProbeData []byte
	var livenessProbeData []byte

	if ringRelease.ReadinessProbe != nil {
		readinessProbeData, err = json.Marshal(ringRelease.ReadinessProbe)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal readiness probe")
		}
	}

	if ringRelease.LivenessProbe != nil {
		livenessProbeData, err = json.Marshal(ringRelease.LivenessProbe)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal liveness probe")
		}
	}

	builder := ringReleaseSelect.
		Where("Image = ?", ringRelease.Image).
		Where("Version = ?", ringRelease.Version).
		Where("Force = ?", ringRelease.Force).
		Where("EnvVariables = ?", envVarMap).
		Where("ReadinessProbe = ?", readinessProbeData).
		Where("LivenessProbe = ?", livenessProbeData).
		Limit(1)

	err = sqlStore.getBuilder(sqlStore.db, &rawRingReleaseOutput, builder)
	if err != nil {
		if err == sql.ErrNoRows {
			sqlStore.logger.Debug("Entry does not exist in the db. Inserting...")
			_, err = sqlStore.execBuilder(db, sq.Insert("RingRelease").
				SetMap(map[string]interface{}{
					"ID":             ringRelease.ID,
					"Image":          ringRelease.Image,
					"Version":        ringRelease.Version,
					"EnvVariables":   envVarMap,
					"ReadinessProbe": readinessProbeData,
					"LivenessProbe":  livenessProbeData,
					"CreateAt":       ringRelease.CreateAt,
					"Force":          ringRelease.Force,
				}))
			if err != nil {
				return nil, errors.Wrap(err, "failed to create ring release")
			}
			return ringRelease, nil
		}
		return nil, errors.Wrap(err, "failed to get ring release by image and version")
	}

	return rawRingReleaseOutput.toRingRelease()
}
