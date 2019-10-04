package servermanager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/cj123/assetto-server-manager/fixtures/race-weekend-examples"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	versionMetaKey = "version"
)

func Migrate(store Store) error {
	var storeVersion int

	err := store.GetMeta(versionMetaKey, &storeVersion)

	if err != nil && err != ErrValueNotSet {
		return err
	}

	if jsonStore, ok := store.(*JSONStore); ok {
		err := separateJSONStores(jsonStore)

		if err != nil {
			return err
		}
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

var (
	CurrentMigrationVersion = len(migrations)

	migrations = []migrationFunc{
		addEntrantIDToChampionships,
		addAdminAccount,
		// migration 2 (below) is left intentionally blank. it replaces a migration which worked with deprecated data.
		func(Store) error { return nil },
		addEntrantsToChampionshipEvents,
		addIDToChampionshipClasses,
		enhanceOldChampionshipResultFiles,
		addResultScreenTimeDefault,
		// migration 8 (below) has been left intentionally blank, as it is now migration 9
		// due to it needing re-running in some environments.
		func(Store) error { return nil },
		addPitBoxDefinitionToEntrants,
		addLastSeenVersionToAccounts,
		addSleepTime1ToServerOptions,
		addPersistOpenEntrantsToChampionship,
		addThemeChoiceToAccounts,
		addRaceWeekendExamples,
		addServerNameTemplate,
	}
)

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

func addEntrantsToChampionshipEvents(rs Store) error {
	logrus.Infof("Running migration: Add entrants to championship events")

	championships, err := rs.ListChampionships()

	if err != nil {
		return err
	}

	for _, c := range championships {
		for _, event := range c.Events {
			if event.EntryList == nil || len(event.EntryList) == 0 {
				event.EntryList = c.AllEntrants()
			}
		}

		err = rs.UpsertChampionship(c)

		if err != nil {
			return err
		}
	}

	return nil
}

func addIDToChampionshipClasses(rs Store) error {
	logrus.Infof("Running migration: Add class ID to championship classes")

	championships, err := rs.ListChampionships()

	if err != nil {
		return err
	}

	for _, c := range championships {
		for _, class := range c.Classes {
			if class.ID == uuid.Nil {
				class.ID = uuid.New()
			}
		}

		err = rs.UpsertChampionship(c)

		if err != nil {
			return err
		}
	}

	return nil
}

func enhanceOldChampionshipResultFiles(rs Store) error {
	logrus.Infof("Running migration: Enhance Old Championship Results Files")

	championships, err := rs.ListChampionships()

	if err != nil {
		return err
	}

	for _, c := range championships {
		for _, event := range c.Events {
			for _, session := range event.Sessions {
				c.EnhanceResults(session.Results)
			}
		}

		err = rs.UpsertChampionship(c)

		if err != nil {
			return err
		}
	}

	return nil
}

func addResultScreenTimeDefault(rs Store) error {
	logrus.Errorf("Running migration: Add Result Screen Time")

	const defaultResultScreenTime = 90

	customRaces, err := rs.ListCustomRaces()

	if err != nil {
		return err
	}

	sort.Slice(customRaces, func(i, j int) bool {
		return customRaces[i].Updated.After(customRaces[j].Updated)
	})

	for _, race := range customRaces {
		race.RaceConfig.ResultScreenTime = defaultResultScreenTime

		err := rs.UpsertCustomRace(race)

		if err != nil {
			return err
		}
	}

	championships, err := rs.ListChampionships()

	if err != nil {
		return err
	}

	sort.Slice(championships, func(i, j int) bool {
		return championships[i].Updated.After(championships[j].Updated)
	})

	for _, champ := range championships {
		for _, event := range champ.Events {
			event.RaceSetup.ResultScreenTime = defaultResultScreenTime
		}

		err := rs.UpsertChampionship(champ)

		if err != nil {
			return err
		}
	}

	return nil
}

func addPitBoxDefinitionToEntrants(rs Store) error {
	logrus.Errorf("Running migration: Add Pit Box Definition To Entrants")

	customRaces, err := rs.ListCustomRaces()

	if err != nil {
		return err
	}

	sort.Slice(customRaces, func(i, j int) bool {
		return customRaces[i].Updated.After(customRaces[j].Updated)
	})

	for _, customRace := range customRaces {
		for i, entrant := range customRace.EntryList.AsSlice() {
			entrant.PitBox = i
		}

		err := rs.UpsertCustomRace(customRace)

		if err != nil {
			return err
		}
	}

	championships, err := rs.ListChampionships()

	if err != nil {
		return err
	}

	sort.Slice(championships, func(i, j int) bool {
		return championships[i].Updated.After(championships[j].Updated)
	})

	for _, championship := range championships {
		for _, event := range championship.Events {
			event.championship = championship

			for i, entrant := range event.EntryList.AsSlice() {
				entrant.PitBox = i
			}
		}

		err := rs.UpsertChampionship(championship)

		if err != nil {
			return err
		}
	}

	return nil
}

func moveStoreFiles(oldPath string, newPath string) error {
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	logrus.WithField("from", oldPath).WithField("to", newPath).Infof("Migrating JSON private store to shared store")

	return os.Rename(oldPath, newPath)
}

func separateJSONStores(rs *JSONStore) error {
	if rs.base != rs.shared {
		err := os.MkdirAll(rs.shared, 0755)

		if err != nil {
			return err
		}

		err = moveStoreFiles(filepath.Join(rs.base, championshipsDir), filepath.Join(rs.shared, championshipsDir))

		if err != nil {
			return err
		}

		err = moveStoreFiles(filepath.Join(rs.base, customRacesDir), filepath.Join(rs.shared, customRacesDir))

		if err != nil {
			return err
		}

		err = moveStoreFiles(filepath.Join(rs.base, entrantsFile), filepath.Join(rs.shared, entrantsFile))

		if err != nil {
			return err
		}
	}

	return nil
}

const lastReleaseVersionPreChangelogShowUpdate = "v1.3.4"

func addLastSeenVersionToAccounts(s Store) error {
	logrus.Errorf("Running migration: Add Last Seen Version to Accounts")
	accounts, err := s.ListAccounts()

	if err != nil {
		return err
	}

	for _, account := range accounts {
		account.LastSeenVersion = lastReleaseVersionPreChangelogShowUpdate

		err := s.UpsertAccount(account)

		if err != nil {
			return err
		}
	}

	return nil
}

func addSleepTime1ToServerOptions(s Store) error {
	logrus.Infof("Running migration: Set Server Options Sleep Time to 1")
	opts, err := s.LoadServerOptions()

	if err != nil {
		return err
	}

	opts.SleepTime = 1

	return s.UpsertServerOptions(opts)
}

func addPersistOpenEntrantsToChampionship(s Store) error {
	logrus.Infof("Running migration: enable 'Persist Open Entrants' in Championships")

	championships, err := s.ListChampionships()

	if err != nil {
		return err
	}

	sort.Slice(championships, func(i, j int) bool {
		return championships[i].Updated.Before(championships[j].Updated)
	})

	for _, champ := range championships {
		if champ.OpenEntrants {
			champ.PersistOpenEntrants = true
		}

		err := s.UpsertChampionship(champ)

		if err != nil {
			return err
		}
	}

	return nil
}

func addThemeChoiceToAccounts(s Store) error {
	logrus.Infof("Running migration: Add Theme Choice to Accounts")

	accounts, err := s.ListAccounts()

	if err != nil {
		return err
	}

	for _, account := range accounts {
		account.Theme = ThemeDefault

		if err := s.UpsertAccount(account); err != nil {
			return err
		}
	}

	return nil
}

func addRaceWeekendExamples(s Store) error {
	logrus.Infof("Running migration: Add Race Weekend examples")

	var raceWeekend *RaceWeekend

	err := json.Unmarshal(raceweekendexamples.F12004spa, &raceWeekend)

	if err != nil {
		return err
	}

	return s.UpsertRaceWeekend(raceWeekend)
}

func addServerNameTemplate(s Store) error {
	logrus.Infof("Running migration: Add Server Name Template")

	opts, err := s.LoadServerOptions()

	if err != nil {
		return err
	}

	opts.ServerNameTemplate = defaultServerNameTemplate

	return s.UpsertServerOptions(opts)
}
