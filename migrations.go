package servermanager

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	CurrentMigrationVersion = 2
	versionMetaKey          = "version"
)

func Migrate(store Store) error {
	var storeVersion int

	err := store.GetMeta(versionMetaKey, &storeVersion)

	if err != nil && err != ErrMetaValueNotSet {
		return err
	}

	for i := storeVersion; i < CurrentMigrationVersion; i++ {
		err := migrations[i](store)

		if err != nil {
			return err
		}
	}

	return store.SetMeta(versionMetaKey, CurrentMigrationVersion)
}

type migrationFunc func(Store) error

var migrations = []migrationFunc{
	addEntrantIDToChampionships,
	addAdminAccount,
}

func addEntrantIDToChampionships(rs Store) error {
	logrus.Infof("Running migration: Add Internal UUID to Championship Entrants")

	championships, err := rs.ListChampionships()

	if err != nil {
		return err
	}

	for _, c := range championships {
		for _, class := range c.Classes {
			for _, entrant := range class.Entrants {
				if entrant.InternalUUID == uuid.Nil {
					entrant.InternalUUID = uuid.New()
				}
			}
		}

		err = rs.UpsertChampionship(c)

		if err != nil {
			return err
		}
	}

	return nil
}

func addAdminAccount(rs Store) error {
	logrus.Infof("Running migration: Add Admin Account")

	account := NewAccount()
	account.Name = adminUserName
	account.DefaultPassword = "servermanager"
	account.Group = GroupAdmin

	return rs.UpsertAccount(account)
}
