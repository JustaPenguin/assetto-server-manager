package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	servermanager "github.com/JustaPenguin/assetto-server-manager"

	"github.com/sirupsen/logrus"
)

var raceOutFile string
var nameToGUIDFile string

func init() {
	flag.StringVar(&raceOutFile, "r", "race_out.json", "race out file")
	flag.StringVar(&nameToGUIDFile, "n", "name_to_guid.json", "name to GUID file")
	flag.Parse()
}

func main() {
	var nameToGUIDMap map[string]string

	f1, err := os.Open(nameToGUIDFile)
	checkError("open name to guid map", err)

	defer f1.Close()

	err = json.NewDecoder(f1).Decode(&nameToGUIDMap)
	checkError("decode name to guid map", err)

	var raceOut *RaceOut

	f2, err := os.Open(raceOutFile)
	checkError("open race out", err)

	defer f2.Close()

	err = json.NewDecoder(f2).Decode(&raceOut)
	checkError("decode race out", err)

	for _, session := range raceOut.Sessions {
		playerToCarIDMap := make(map[string]int)

		var sessionType servermanager.SessionType

		switch session.Type {
		case 1:
			sessionType = servermanager.SessionTypePractice
		case 2:
			sessionType = servermanager.SessionTypeQualifying
		case 3:
			sessionType = servermanager.SessionTypeRace
		}

		results := &servermanager.SessionResults{
			TrackName:   raceOut.Track,
			TrackConfig: "", // no way to know afaik?
			Type:        sessionType,
			Date:        time.Now(),
		}

		for carID, player := range raceOut.Players {
			playerToCarIDMap[player.Name] = carID

			results.Cars = append(results.Cars, &servermanager.SessionCar{
				BallastKG: 0,
				CarID:     carID,
				Driver: servermanager.SessionDriver{
					GUID:      nameToGUIDMap[player.Name],
					GuidsList: []string{nameToGUIDMap[player.Name]},
					Name:      player.Name,
					Nation:    "",
					Team:      "",
				},
				Model:      player.Car,
				Restrictor: 0,
				Skin:       player.Skin,
			})
		}

		lapTimeStamp := make(map[int]int)

		for _, lap := range session.Laps {
			if _, ok := lapTimeStamp[lap.Car]; !ok {
				lapTimeStamp[lap.Car] = 0
			}

			lapTimeStamp[lap.Car] += lap.Time

			results.Laps = append(results.Laps, &servermanager.SessionLap{
				BallastKG:  0,
				CarID:      lap.Car,
				CarModel:   results.Cars[lap.Car].Model,
				Cuts:       lap.Cuts,
				DriverGUID: results.Cars[lap.Car].Driver.GUID,
				DriverName: results.Cars[lap.Car].Driver.Name,
				LapTime:    lap.Time,
				Restrictor: 0,
				Sectors:    lap.Sectors,
				Timestamp:  lapTimeStamp[lap.Car],
				Tyre:       lap.Tyre,
			})
		}

		sort.Slice(results.Laps, func(i, j int) bool {
			lapI := results.Laps[i]
			lapJ := results.Laps[j]

			return lapI.Timestamp < lapJ.Timestamp
		})

		for _, carID := range session.RaceResult {
			totalTime := 0
			bestLap := -1

			for _, lap := range session.Laps {
				if lap.Car == carID {
					totalTime += lap.Time

					if bestLap < 0 || lap.Time < bestLap {
						bestLap = lap.Time
					}
				}
			}

			results.Result = append(results.Result, &servermanager.SessionResult{
				BallastKG:  0,
				BestLap:    bestLap,
				CarID:      carID,
				CarModel:   results.Cars[carID].Model,
				DriverGUID: results.Cars[carID].Driver.GUID,
				DriverName: results.Cars[carID].Driver.Name,
				Restrictor: 0,
				TotalTime:  totalTime,
			})
		}

		err = saveResults(results)
		checkError("save results", err)
	}
}

func checkError(what string, err error) {
	if err == nil {
		return
	}

	logrus.WithError(err).Fatalf("could not: %s", what)
}

func saveResults(r *servermanager.SessionResults) error {
	f, err := os.Create(fmt.Sprintf("%d_%d_%d_%d_%d_%s.json", r.Date.Year(), r.Date.Month(), r.Date.Day(), r.Date.Hour(), r.Date.Minute(), r.Type.OriginalString()))

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")

	return enc.Encode(r)
}
