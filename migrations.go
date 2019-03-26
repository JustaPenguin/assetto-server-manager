package servermanager

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const CurrentMigrationVersion = 1

func Migrate(store RaceStore) error {
	storeVersion, err := store.GetVersion()

	if err != nil {
		return err
	}

	for i := storeVersion; i < CurrentMigrationVersion; i++ {
		err := migrations[i](store)

		if err != nil {
			return err
		}
	}

	return store.SetVersion(CurrentMigrationVersion)
}

type migrationFunc func(RaceStore) error

var migrations = []migrationFunc{
	addEventIDtoChampionships,
}

func addEventIDtoChampionships(rs RaceStore) error {
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
