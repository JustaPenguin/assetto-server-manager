package main

import (
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/cj123/assetto-server-manager"
	"github.com/etcd-io/bbolt"
)

var existingSessions = map[string]map[servermanager.SessionType]string{
	"imola": {
		servermanager.SessionTypePractice:   "2019_1_16_20_2_PRACTICE.json",
		servermanager.SessionTypeQualifying: "2019_1_16_20_32_QUALIFY.json",
		servermanager.SessionTypeRace:       "2019_1_16_20_47_RACE.json",
	},
	"ks_red_bull_ring": {
		servermanager.SessionTypePractice:   "2019_1_22_20_23_PRACTICE.json",
		servermanager.SessionTypeQualifying: "2019_1_22_20_41_QUALIFY.json",
		servermanager.SessionTypeRace:       "2019_1_22_21_12_RACE.json",
	},
	"t78_hockenheimring": {
		servermanager.SessionTypePractice:   "2019_1_31_20_39_PRACTICE.json",
		servermanager.SessionTypeQualifying: "2019_1_31_20_58_QUALIFY.json",
		servermanager.SessionTypeRace:       "2019_1_31_21_27_RACE.json",
	},
}

func main() {
	var err error
	servermanager.ServerInstallPath, err = filepath.Abs(filepath.Join("..", "server-manager", "assetto"))
	checkError(err)

	championshipID := "90b0dcf6-0e80-4738-a7ad-53470795660f"

	bb, err := bbolt.Open(filepath.Join(servermanager.ServerInstallPath, "store.db"), 0644, nil)
	checkError(err)

	cm := servermanager.NewChampionshipManager(servermanager.NewRaceManager(servermanager.NewBoltRaceStore(bb)))

	champ, err := cm.LoadChampionship(championshipID)
	checkError(err)

	for _, event := range champ.Events {
		if event.ID.String() == "00000000-0000-0000-0000-000000000000" {
			event.ID = uuid.New()
		}
	}

	for track, sessions := range existingSessions {
		for _, event := range champ.Events {
			if track == event.RaceSetup.Track {
				event.Sessions = make(map[servermanager.SessionType]*servermanager.ChampionshipSession)

				for sess, file := range sessions {
					result, err := servermanager.LoadResult(file)
					checkError(err)

					event.Sessions[sess] = &servermanager.ChampionshipSession{
						StartedTime:   result.Date.Add(time.Minute * -10),
						CompletedTime: result.Date,
						Results:       result,
					}
				}

				event.CompletedTime = event.Sessions[servermanager.SessionTypeRace].Results.Date

				break
			}
		}
	}

	champ.Points.Places = servermanager.DefaultChampionshipPoints.Places[:8]

	err = cm.UpsertChampionship(champ)
	checkError(err)
}

func checkError(err error) {
	if err == nil {
		return
	}

	panic(err)
}
