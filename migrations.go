package servermanager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	defaultcontent "github.com/JustaPenguin/assetto-server-manager/fixtures/default-content"

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
		addAvailableCarsToChampionshipClass,
		addTyresForP13c,
		changeNotificationTimer,
		addContentExamples,
		addServerIDToScheduledEvents,
		addLoopServerToCustomRace,
		amendChampionshipClassIDIncorrectValues,
		enableLoggingWith5LogsKept,
		convertAccountGroupToServerIDGroupMap,
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
	if err := initServerID(rs); err != nil {
		return err
	}

	accounts, err := rs.ListAccounts()

	if err != nil {
		return err
	}

	for _, account := range accounts {
		// in a multi-server scenario, don't reset the admin account password to 'servermanager'
		if account.Name == adminUserName {
			return nil
		}
	}

	logrus.Infof("Running migration: Add Admin Account")

	account := NewAccount()
	account.Name = adminUserName
	account.DefaultPassword = "servermanager"
	account.Groups[serverID] = GroupAdmin

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
	logrus.Infof("Running migration: Add Result Screen Time")

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
	logrus.Infof("Running migration: Add Pit Box Definition To Entrants")

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
	logrus.Infof("Running migration: Add Last Seen Version to Accounts")
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

	err := json.Unmarshal(defaultcontent.RaceWeekendF12004spa, &raceWeekend)

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

func changeNotificationTimer(s Store) error {
	logrus.Infof("Running migration: Change Notification Timer")

	opts, err := s.LoadServerOptions()

	if err != nil {
		return err
	}

	opts.NotificationReminderTimers = strconv.Itoa(opts.NotificationReminderTimer)

	if opts.NotificationReminderTimers == "0" {
		opts.NotificationReminderTimers = ""
	}

	return s.UpsertServerOptions(opts)
}

func addAvailableCarsToChampionshipClass(s Store) error {
	logrus.Infof("Running migration: Add Available Cars to Championship Class")

	championships, err := s.ListChampionships()

	if err != nil {
		return err
	}

	sort.Slice(championships, func(i, j int) bool {
		return championships[i].Updated.Before(championships[j].Updated)
	})

	for _, champ := range championships {
		for _, class := range champ.Classes {
			cars := make(map[string]bool)

			for _, e := range class.Entrants {
				cars[e.Model] = true
			}

			for car := range cars {
				class.AvailableCars = append(class.AvailableCars, car)
			}
		}

		err := s.UpsertChampionship(champ)

		if err != nil {
			return err
		}
	}

	return nil
}

const IERP13c = "ier_p13c"

var IERP13cTyres = []string{"S1", "S2", "S3", "S4", "S5", "S6", "S7", "S8"}

func addTyresForP13c(s Store) error {
	logrus.Debugf("Running migration: add tyres for IER P13c")

	tyres := make(map[string]string)

	for _, tyre := range IERP13cTyres {
		tyres[tyre] = tyre
	}

	return addTyresToModTyres(IERP13c, tyres)
}

func addContentExamples(s Store) error {
	logrus.Infof("Running migration: Add Content examples")

	championships, err := s.ListChampionships()

	if err != nil {
		return err
	}

	if len(championships) == 0 {
		var mx5Championship *Championship

		if err := json.Unmarshal(defaultcontent.ChampionshipMX5CrashCourse, &mx5Championship); err != nil {
			return err
		}

		if err := s.UpsertChampionship(mx5Championship); err != nil {
			return err
		}

		var multiclassChampionship *Championship

		if err := json.Unmarshal(defaultcontent.ChampionshipMulticlassEndurance, &multiclassChampionship); err != nil {
			return err
		}

		if err := s.UpsertChampionship(multiclassChampionship); err != nil {
			return err
		}
	}

	customRaces, err := s.ListCustomRaces()

	if err != nil {
		return err
	}

	if len(customRaces) == 0 {
		var f2004RaceSpa *CustomRace

		if err := json.Unmarshal(defaultcontent.CustomRaceF2004Spa, &f2004RaceSpa); err != nil {
			return err
		}

		if err := s.UpsertCustomRace(f2004RaceSpa); err != nil {
			return err
		}

		var bmw235iRace *CustomRace

		if err := json.Unmarshal(defaultcontent.CustomRaceBMWZandvoort, &bmw235iRace); err != nil {
			return err
		}

		if err := s.UpsertCustomRace(bmw235iRace); err != nil {
			return err
		}
	}

	return nil
}

func addServerIDToScheduledEvents(s Store) error {
	logrus.Infof("Running migration: Add Server ID to Scheduled Events")

	if err := initServerID(s); err != nil {
		return err
	}

	customRaces, err := s.ListCustomRaces()

	if err != nil {
		return err
	}

	sort.Slice(customRaces, func(i, j int) bool {
		return customRaces[i].Updated.Before(customRaces[j].Updated)
	})

	for _, customRace := range customRaces {
		if customRace.Scheduled.IsZero() {
			continue
		}

		customRace.ScheduledServerID = serverID

		err := s.UpsertCustomRace(customRace)

		if err != nil {
			return err
		}
	}

	championships, err := s.ListChampionships()

	if err != nil {
		return err
	}

	sort.Slice(championships, func(i, j int) bool {
		return championships[i].Updated.Before(championships[j].Updated)
	})

	for _, championship := range championships {
		for _, event := range championship.Events {
			if event.Scheduled.IsZero() {
				continue
			}

			event.ScheduledServerID = serverID
		}

		err := s.UpsertChampionship(championship)

		if err != nil {
			return err
		}
	}

	raceWeekends, err := s.ListRaceWeekends()

	if err != nil {
		return err
	}

	sort.Slice(raceWeekends, func(i, j int) bool {
		return raceWeekends[i].Updated.Before(raceWeekends[j].Updated)
	})

	for _, raceWeekend := range raceWeekends {
		for _, session := range raceWeekend.Sessions {
			if session.ScheduledTime.IsZero() {
				continue
			}

			session.ScheduledServerID = serverID
		}

		err := s.UpsertRaceWeekend(raceWeekend)

		if err != nil {
			return err
		}
	}

	return nil
}

func addLoopServerToCustomRace(s Store) error {
	logrus.Infof("Running migration: Add Loop Per Server to Custom Race")

	if err := initServerID(s); err != nil {
		return err
	}

	customRaces, err := s.ListCustomRaces()

	if err != nil {
		return err
	}

	for _, customRace := range customRaces {
		if customRace.Loop {
			customRace.LoopServer = make(map[ServerID]bool)

			customRace.LoopServer[serverID] = true

			err := s.UpsertCustomRace(customRace)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func amendChampionshipClassIDIncorrectValues(s Store) error {
	logrus.Infof("Running migration: Correcting Multiclass Championship incorrect ClassIDs (if any)")

	championships, err := s.ListChampionships()

	if err != nil {
		return err
	}

	for _, championship := range championships {
		if !championship.IsMultiClass() {
			continue
		}

		foundClasses := make(map[uuid.UUID]bool)

		for _, class := range championship.Classes {
			if _, ok := foundClasses[class.ID]; ok {
				logrus.Infof("Duplicated class id: %s in Championship: %s", class.ID, championship.ID)

				class.ID = uuid.New()

				for _, event := range championship.Events {
					if event.IsRaceWeekend() {
						rw, err := s.LoadRaceWeekend(event.RaceWeekendID.String())

						if err != nil {
							continue
						}

						for _, session := range rw.Sessions {

							if session.SessionType() == SessionTypeRace {
								session.Points[class.ID] = &class.Points
							} else {
								var points []int

								for range class.Points.Places {
									points = append(points, 0)
								}

								session.Points[class.ID] = &ChampionshipPoints{
									Places:               points,
									BestLap:              0,
									PolePosition:         0,
									CollisionWithDriver:  0,
									CollisionWithEnv:     0,
									CutTrack:             0,
									SecondRaceMultiplier: 0,
								}
							}

							if !session.Completed() {
								continue
							}

							for _, car := range session.Results.Cars {
								class, err := championship.FindClassForCarModel(car.Model)

								if err != nil {
									continue
								}

								car.Driver.ClassID = class.ID
							}

							for _, result := range session.Results.Result {
								class, err := championship.FindClassForCarModel(result.CarModel)

								if err != nil {
									continue
								}

								result.ClassID = class.ID
							}

							for _, lap := range session.Results.Laps {
								class, err := championship.FindClassForCarModel(lap.CarModel)

								if err != nil {
									continue
								}

								lap.ClassID = class.ID
							}
						}

						if err := s.UpsertRaceWeekend(rw); err != nil {
							return err
						}
					} else {
						for _, session := range event.Sessions {
							if !session.Completed() {
								continue
							}

							for _, car := range session.Results.Cars {
								class, err := championship.FindClassForCarModel(car.Model)

								if err != nil {
									continue
								}

								car.Driver.ClassID = class.ID
							}

							for _, result := range session.Results.Result {
								class, err := championship.FindClassForCarModel(result.CarModel)

								if err != nil {
									continue
								}

								result.ClassID = class.ID
							}

							for _, lap := range session.Results.Laps {
								class, err := championship.FindClassForCarModel(lap.CarModel)

								if err != nil {
									continue
								}

								lap.ClassID = class.ID
							}
						}
					}
				}
			}

			foundClasses[class.ID] = true
		}

		if err := s.UpsertChampionship(championship); err != nil {
			return err
		}
	}

	return nil
}

func enableLoggingWith5LogsKept(s Store) error {
	logrus.Infof("Running migration: Enable AC Server Logging")

	opts, err := s.LoadServerOptions()

	if err != nil {
		return err
	}

	opts.LogACServerOutputToFile = true
	opts.NumberOfACServerLogsToKeep = 5

	return s.UpsertServerOptions(opts)
}

func convertAccountGroupToServerIDGroupMap(s Store) error {
	logrus.Infof("Running migration: Convert Account Group to Server ID Group Map")

	if err := initServerID(s); err != nil {
		return err
	}

	accounts, err := s.ListAccounts()

	if err != nil {
		return err
	}

	for _, account := range accounts {
		if account.Groups != nil {
			if _, ok := account.Groups[serverID]; ok {
				continue
			}
		}

		account.Groups = make(map[ServerID]Group)
		account.Groups[serverID] = account.DeprecatedGroup

		if err := s.UpsertAccount(account); err != nil {
			return err
		}
	}

	return nil
}
