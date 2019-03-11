package servermanager

import (
	"time"
	
	"github.com/sirupsen/logrus"
)

var CustomRaceStartTimers map[string]*time.Timer
var ChampionshipEventStartTimers map[string]*time.Timer

func InitialiseScheduledCustomRaces() error {
	CustomRaceStartTimers = make(map[string]*time.Timer)

	races, err := raceManager.raceStore.ListCustomRaces()

	if err != nil {
		return err
	}

	for _, race := range races {
		if race.Scheduled.After(time.Now()) {
			// add a scheduled event on date
			duration := race.Scheduled.Sub(time.Now())

			CustomRaceStartTimers[race.UUID.String()] = time.AfterFunc(duration, func() {
				err := raceManager.StartCustomRace(race.UUID.String(), false)

				if err != nil {
					logrus.Errorf("couldn't start scheduled race, err: %s", err)
				}
			})

			err := raceManager.raceStore.UpsertCustomRace(race)

			if err != nil {
				return err
			}
		} else {
			emptyTime := time.Time{}
			if race.Scheduled != emptyTime {
				logrus.Infof("Looks like the server was offline whilst a scheduled event was meant to start!"+
					" Start time: %s. The schedule has been cleared. Start the event manually if you wish to run it.", race.Scheduled.String())

				race.Scheduled = emptyTime

				err := raceManager.raceStore.UpsertCustomRace(race)

				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func InitialiseScheduledChampionshipEvents() error {
	ChampionshipEventStartTimers = make(map[string]*time.Timer)

	championships, err := championshipManager.ListChampionships()

	if err != nil {
		return err
	}

	for _, championship := range championships {
		for _, event := range championship.Events {
			if event.Scheduled.After(time.Now()) {
				// add a scheduled event on date
				duration := event.Scheduled.Sub(time.Now())

				ChampionshipEventStartTimers[event.ID.String()] = time.AfterFunc(duration, func() {
					err := championshipManager.StartEvent(championship.ID.String(), event.ID.String())

					if err != nil {
						logrus.Errorf("couldn't start scheduled race, err: %s", err)
					}
				})

				return championshipManager.UpsertChampionship(championship)
			} else {
				emptyTime := time.Time{}
				if event.Scheduled != emptyTime {
					logrus.Infof("Looks like the server was offline whilst a scheduled event was meant to start!"+
						" Start time: %s. The schedule has been cleared. Start the event manually if you wish to run it.", event.Scheduled.String())

					event.Scheduled = emptyTime

					return championshipManager.UpsertChampionship(championship)
				}

			}
		}
	}

	return nil
}
