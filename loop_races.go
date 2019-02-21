package servermanager

import (
	"github.com/sirupsen/logrus"
	"time"
)

func LoopRaces () {
	var i int
	ticker := time.NewTicker(10 * time.Second)

	for {
		select{
		case <- ticker.C:
			currentRace, _ := raceManager.CurrentRace()

			if currentRace != nil {
				println("Race")
				println(currentRace.CurrentRaceConfig.LoopMode)
				println(currentRace.CurrentRaceConfig.RaceOverTime)
				println(currentRace.CurrentRaceConfig.SleepTime)
				break
			}

			println("No race")

			_, _, looped, err := raceManager.ListCustomRaces()

			if err != nil {
				logrus.Errorf("couldn't list custom races, err: %s", err)
				return
			}

			if looped != nil {
				if i >= len(looped) {
					i = 0
				}

				err := raceManager.StartCustomRace(looped[i].UUID.String(), true)

				if err != nil {
					logrus.Errorf("couldn't start auto loop custom race, err: %s", err)
					return
				}
			}

			i++
		}
	}

}