package main

import (
	"encoding/json"
	"fmt"
	servermanager "github.com/cj123/assetto-server-manager"
	"github.com/google/uuid"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const url = "http://ergast.com/api/f1/2019/13/qualifying.json"
const sessionType = "qualifying"

func main() {
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	var results *Qualifying

	err = json.NewDecoder(resp.Body).Decode(&results)

	if err != nil {
		panic(err)
	}

	for _, race := range results.MRData.RaceTable.Races {
		for _, session := range []string{"Q1", "Q2", "Q3"} {
			date, err := time.Parse("2006-01-02_15:04:05Z", race.Date+"_"+race.Time)

			if err != nil {
				panic(err)
			}

			filename := fmt.Sprintf("%d_%d_%d_%d_%d_%s.json", date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute(), session)

			output := servermanager.SessionResults{
				TrackConfig:    "",
				TrackName:      "spa",
				Type:           "QUALIFY",
				Date:           date,
				SessionFile:    filename,
				ChampionshipID: "",
			}

			for index, result := range race.QualifyingResults {

				var lapTime time.Duration

				if session == "Q1" {
					if result.Q1 == "" {

					} else {
						lapTime = parseDuration(result.Q1)
					}

				} else if session == "Q2" {
					if result.Q2 == "" {
						continue
					}

					lapTime = parseDuration(result.Q2)
				} else if session == "Q3" {
					if result.Q3 == "" {
						continue
					}

					lapTime = parseDuration(result.Q3)
				}

				output.Cars = append(output.Cars, &servermanager.SessionCar{
					BallastKG: 0,
					CarID:     index,
					Driver: servermanager.SessionDriver{
						GUID:      result.Driver.DriverID,
						GuidsList: []string{result.Driver.DriverID},
						Name:      result.Driver.GivenName + " " + result.Driver.FamilyName,
						Nation:    result.Driver.Nationality,
						Team:      result.Constructor.Name,
						ClassID:   uuid.UUID{},
					},
					Model:      "rss_formula_hybrid_2019",
					Restrictor: 0,
					Skin:       "",
				})

				output.Laps = append(output.Laps, &servermanager.SessionLap{
					CarID:      index,
					CarModel:   "rss_formula_hybrid_2019",
					Cuts:       0,
					DriverGUID: result.Driver.DriverID,
					DriverName: result.Driver.GivenName + " " + result.Driver.FamilyName,
					LapTime:    int(lapTime.Milliseconds()),
					Restrictor: 0,
					Sectors:    []int{},
					Timestamp:  int(date.Unix()),
					Tyre:       "C1",
					ClassID:    uuid.UUID{},
				})

				output.Result = append(output.Result, &servermanager.SessionResult{
					BallastKG:    0,
					BestLap:      int(lapTime.Milliseconds()),
					CarID:        index,
					CarModel:     "rss_formula_hybrid_2019",
					DriverGUID:   result.Driver.DriverID,
					DriverName:   result.Driver.GivenName + " " + result.Driver.FamilyName,
					Restrictor:   0,
					HasPenalty:   false,
					PenaltyTime:  0,
					LapPenalty:   0,
					Disqualified: false,
					TotalTime:    10,
					ClassID:      uuid.UUID{},
				})
			}

			sort.Slice(output.Result, func(i, j int) bool {
				if output.Result[i].BestLap == 0 {
					return false
				}

				return output.Result[i].BestLap < output.Result[j].BestLap
			})

			f, err := os.Create(filename)

			if err != nil {
				panic(err)
			}

			defer f.Close() // meh

			json.NewEncoder(f).Encode(output)
		}
	}
}

func parseDuration(duration string) time.Duration {
	d, err := time.ParseDuration(strings.Replace(duration, ":", "m", -1) + "s")

	if err != nil {
		panic(err)
	}

	return d
}
