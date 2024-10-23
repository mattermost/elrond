// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
)

type migration struct {
	fromVersion   semver.Version
	toVersion     semver.Version
	migrationFunc func(execer) error
}

// migrations defines the set of migrations necessary to advance the database to the latest
// expected version.
//
// Note that the canonical schema is currently obtained by applying all migrations to an empty
// database.
var migrations = []migration{
	{semver.MustParse("0.0.0"), semver.MustParse("0.1.0"), func(e execer) error {
		_, systemErr := e.Exec(`
			CREATE TABLE System (
				Key VARCHAR(64) PRIMARY KEY,
				Value VARCHAR(1024) NULL
			);
		`)
		if systemErr != nil {
			return errors.Wrap(systemErr, "failed to create System table")
		}

		if _, ringErr := e.Exec(`
			CREATE TABLE Ring (
				ID CHAR(26) PRIMARY KEY,
				State TEXT NOT NULL,
				Name TEXT NOT NULL,
				Priority INT NOT NULL,
				Provisioner TEXT NOT NULL,
				SoakTime INT NOT NULL,
				ActiveReleaseID TEXT NOT NULL, 
				DesiredReleaseID TEXT NOT NULL,
				ReleaseAt BIGINT NOT NULL,
				CreateAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL,
				APISecurityLock BOOLEAN NOT NULL,
				LockAcquiredBy CHAR(26) NULL,
				LockAcquiredAt BIGINT NOT NULL
			);
		`); ringErr != nil {
			return errors.Wrap(ringErr, "failed to create Ring table")
		}

		if _, installationGroupErr := e.Exec(`
			CREATE TABLE InstallationGroup (
				ID TEXT PRIMARY KEY,
				Name TEXT NOT NULL,
				State TEXT NOT NULL,
				ReleaseAt BIGINT NOT NULL,
				SoakTime INT NOT NULL,
				ProvisionerGroupID TEXT NOT NULL,
				LockAcquiredBy CHAR(26) NULL,
				LockAcquiredAt BIGINT NOT NULL
			);
		`); installationGroupErr != nil {
			return errors.Wrap(installationGroupErr, "failed to create InstallationGroup table")
		}

		if _, ringInstallationGroupErr := e.Exec(`
			CREATE TABLE RingInstallationGroup (
				ID TEXT PRIMARY KEY,
				RingID TEXT NOT NULL,
				InstallationGroupID TEXT NOT NULL
			);
		`); ringInstallationGroupErr != nil {
			return ringInstallationGroupErr
		}

		_, uniqueIndexErr := e.Exec(`
		CREATE UNIQUE INDEX RingInstallationGroup_RingID_InstallationGroupID ON RingInstallationGroup (RingID, InstallationGroupID);
	`)
		if uniqueIndexErr != nil {
			return errors.Wrap(uniqueIndexErr, "failed to create unique installation group index")
		}

		if _, ringReleaseErr := e.Exec(`
			CREATE TABLE RingRelease (
				ID TEXT PRIMARY KEY,
				Image TEXT NOT NULL,
				Version TEXT NOT NULL,
				CreateAt BIGINT NOT NULL,
				Force BOOLEAN NOT NULL
			);
		`); ringReleaseErr != nil {
			return ringReleaseErr
		}

		_, ringReleaseIndexErr := e.Exec(`
			CREATE UNIQUE INDEX RingRelease_Image_Version_Force ON RingRelease (Image, Version, Force);
		`)
		if ringReleaseIndexErr != nil {
			return ringReleaseIndexErr
		}

		// Add webhook table.
		if _, webhookErr := e.Exec(`
			CREATE TABLE Webhooks (
				ID TEXT PRIMARY KEY,
				OwnerID TEXT NOT NULL,
				URL TEXT NOT NULL,
				CreateAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL
			);
		`); webhookErr != nil {
			return errors.Wrap(webhookErr, "failed to create Webhooks table")
		}

		if _, webhookIndexErr := e.Exec(`
			CREATE UNIQUE INDEX Webhook_URL_DeleteAt ON Webhooks (URL, DeleteAt);
		`); webhookIndexErr != nil {
			return errors.Wrap(webhookIndexErr, "failed to create unique webhook index")
		}

		return nil
	}},
	{semver.MustParse("0.1.0"), semver.MustParse("0.2.0"), func(e execer) error {
		_, renameTableErr := e.Exec(`ALTER TABLE RingRelease RENAME TO RingReleaseTemp;`)
		if renameTableErr != nil {
			return renameTableErr
		}

		_, createTableErr := e.Exec(`
				CREATE TABLE RingRelease (
					ID TEXT PRIMARY KEY,
					Image TEXT NOT NULL,
					Version TEXT NOT NULL,
					EnvVariables TEXT NOT NULL, 
					CreateAt BIGINT NOT NULL,
					Force BOOLEAN NOT NULL
				);
			`)
		if createTableErr != nil {
			return createTableErr
		}

		_, insertDataErr := e.Exec(`
				INSERT INTO RingRelease
				SELECT
					ID,
					Image,
					Version,
					'',
					CreateAt,
					Force
				FROM
				RingReleaseTemp;
			`)
		if insertDataErr != nil {
			return insertDataErr
		}

		_, dropTableErr := e.Exec(`DROP TABLE RingReleaseTemp;`)
		if dropTableErr != nil {
			return dropTableErr
		}
		return nil
	}},
	{semver.MustParse("0.2.0"), semver.MustParse("0.3.0"), func(e execer) error {
		_, renameTableErr := e.Exec(`ALTER TABLE RingRelease RENAME TO RingReleaseTemp;`)
		if renameTableErr != nil {
			return renameTableErr
		}

		_, createTableErr := e.Exec(`
				CREATE TABLE RingRelease (
					ID TEXT PRIMARY KEY,
					Image TEXT NOT NULL,
					Version TEXT NOT NULL,
					EnvVariables BYTEA NULL, 
					CreateAt BIGINT NOT NULL,
					Force BOOLEAN NOT NULL
				);
			`)
		if createTableErr != nil {
			return createTableErr
		}

		_, insertDataErr := e.Exec(`
				INSERT INTO RingRelease
				SELECT
					ID,
					Image,
					Version,
					NULL,
					CreateAt,
					Force
				FROM
				RingReleaseTemp;
			`)
		if insertDataErr != nil {
			return insertDataErr
		}

		_, dropTableErr := e.Exec(`DROP TABLE RingReleaseTemp;`)
		if dropTableErr != nil {
			return dropTableErr
		}
		return nil
	}},
}
