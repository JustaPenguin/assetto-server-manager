package servermanager

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type PenaltiesHandler struct {
	*BaseHandler

	penaltiesManager *PenaltiesManager
}

func NewPenaltiesHandler(baseHandler *BaseHandler, penaltiesManager *PenaltiesManager) *PenaltiesHandler {
	return &PenaltiesHandler{
		BaseHandler:      baseHandler,
		penaltiesManager: penaltiesManager,
	}
}

func (ph *PenaltiesHandler) managePenalty(w http.ResponseWriter, r *http.Request) {
	jsonFileName := chi.URLParam(r, "sessionFile")
	guid := chi.URLParam(r, "driverGUID")
	carModel := r.URL.Query().Get("model")

	err := r.ParseForm()

	if err != nil {
		AddErrorFlash(w, r, "Could not parse penalty form")
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	add := r.FormValue("action") == "add"

	penalty := 0.0

	if add {
		penaltyString := r.FormValue("time-penalty")

		if penaltyString == "" {
			penalty = 0
		} else {
			pen, err := strconv.ParseFloat(penaltyString, 64)

			if err != nil {
				logrus.WithError(err).Errorf("could not parse penalty time")
				AddErrorFlash(w, r, "Could not parse penalty time")
				http.Redirect(w, r, r.Referer(), http.StatusFound)
				return
			}

			penalty = pen
		}
	}

	err = ph.penaltiesManager.applyPenalty(jsonFileName, guid, carModel, penalty, add)

	if err != nil {
		AddErrorFlash(w, r, "Could not add/remove penalty")
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	if !add {
		AddFlash(w, r, "Penalty Removed!")
	} else {
		AddFlash(w, r, "Penalty Added!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

type PenaltiesManager struct {
	store Store
}

func NewPenaltiesManager(store Store) *PenaltiesManager {
	return &PenaltiesManager{
		store: store,
	}
}

func (pm *PenaltiesManager) applyPenalty(jsonFileName, guid, carModel string, penalty float64, add bool) error {
	var results *SessionResults

	var fullFileName string

	if !strings.HasSuffix(jsonFileName, ".json") {
		fullFileName = jsonFileName + ".json"
	} else {
		fullFileName = jsonFileName

		jsonFileName = strings.TrimSuffix(jsonFileName, ".json")
	}

	results, err := LoadResult(fullFileName)

	if err != nil {
		logrus.WithError(err).Errorf("could not load session result file")
		return err
	}

	for _, result := range results.Result {
		if result.DriverGUID == guid && result.CarModel == carModel {
			if !add {
				result.HasPenalty = false
				result.Disqualified = false
				result.PenaltyTime = 0
				result.LapPenalty = 0

				logrus.Infof("All penalties cleared from Driver: %s", guid)
			} else {
				if penalty == 0 {
					result.Disqualified = true
					result.HasPenalty = false
					result.LapPenalty = 0

					logrus.Infof("Driver: %s disqualified", guid)
				} else {
					result.HasPenalty = true
					result.Disqualified = false

					timeParsed, err := time.ParseDuration(fmt.Sprintf("%.1fs", penalty))

					if err != nil {
						logrus.WithError(err).Errorf("could not parse penalty time")
						return err
					}

					result.PenaltyTime = timeParsed

					// If penalty time is greater than a lap then add a lap penalty and change penalty time by one lap
					lastLapTime := results.GetLastLapTime(result.DriverGUID, result.CarModel)

					if result.PenaltyTime > lastLapTime {
						result.LapPenalty = int(result.PenaltyTime / lastLapTime)
					}

					logrus.Infof("%s penalty applied to driver: %s", timeParsed.String(), guid)
				}
			}

			break
		}
	}

	switch results.Type {
	case SessionTypePractice, SessionTypeQualifying:
		sort.Slice(results.Result, func(i, j int) bool {
			if (!results.Result[i].Disqualified && !results.Result[j].Disqualified) || (results.Result[i].Disqualified && results.Result[j].Disqualified) {

				if results.Result[i].BestLap == 0 {
					return false
				}

				if results.Result[j].BestLap == 0 {
					return true
				}

				// if both drivers are/aren't disqualified
				return results.GetTime(results.Result[i].BestLap, results.Result[i].DriverGUID, results.Result[i].CarModel, true) <
					results.GetTime(results.Result[j].BestLap, results.Result[j].DriverGUID, results.Result[j].CarModel, true)

			}

			// driver i is closer to the front than j if they are not disqualified and j is
			return results.Result[j].Disqualified
		})
	case SessionTypeRace:
		// sort results.Result, if disqualified go to back, if time penalty sort by laps completed then lap time
		sort.Slice(results.Result, func(i, j int) bool {
			if !results.Result[i].Disqualified && !results.Result[j].Disqualified {

				// if both drivers aren't disqualified
				if results.GetNumLaps(results.Result[i].DriverGUID, results.Result[i].CarModel) == results.GetNumLaps(results.Result[j].DriverGUID, results.Result[j].CarModel) {
					// if their number of laps are equal, compare lap times

					return results.GetTime(results.Result[i].TotalTime, results.Result[i].DriverGUID, results.Result[i].CarModel, true) <
						results.GetTime(results.Result[j].TotalTime, results.Result[j].DriverGUID, results.Result[j].CarModel, true)
				}

				return results.GetNumLaps(results.Result[i].DriverGUID, results.Result[i].CarModel) >= results.GetNumLaps(results.Result[j].DriverGUID, results.Result[j].CarModel)

			} else if results.Result[i].Disqualified && results.Result[j].Disqualified {

				// if both drivers ARE disqualified, compare their lap times / num laps
				if results.GetNumLaps(results.Result[i].DriverGUID, results.Result[i].CarModel) == results.GetNumLaps(results.Result[j].DriverGUID, results.Result[j].CarModel) {
					// if their number of laps are equal, compare lap times
					return results.GetTime(results.Result[i].TotalTime, results.Result[i].DriverGUID, results.Result[i].CarModel, true) <
						results.GetTime(results.Result[j].TotalTime, results.Result[j].DriverGUID, results.Result[j].CarModel, true)
				}

				return results.GetNumLaps(results.Result[i].DriverGUID, results.Result[i].CarModel) >= results.GetNumLaps(results.Result[j].DriverGUID, results.Result[j].CarModel)

			} else {
				// driver i is closer to the front than j if they are not disqualified and j is
				return results.Result[j].Disqualified
			}
		})
	}

	err = saveResults(fullFileName, results)

	if err != nil {
		logrus.WithError(err).Errorf("could not encode to session result file")
		return err
	}

	if results.ChampionshipID != "" {
		championship, err := pm.store.LoadChampionship(results.ChampionshipID)

		if err != nil {
			logrus.WithError(err).Errorf("Couldn't load championship with ID: %s", results.ChampionshipID)
			return err
		}

	champEvents:
		for i, event := range championship.Events {
			if event.IsRaceWeekend() {
				raceWeekend, err := pm.store.LoadRaceWeekend(event.RaceWeekendID.String())

				if err != nil {
					return err
				}

				for key, session := range raceWeekend.Sessions {
					if !session.Completed() {
						continue
					}

					if session.Results.SessionFile == jsonFileName {
						raceWeekend.Sessions[key].Results = results

						break champEvents
					}
				}
			} else {
				for key, session := range event.Sessions {
					if !session.Completed() {
						continue
					}

					if session.Results.SessionFile == jsonFileName {
						championship.Events[i].Sessions[key].Results = results

						break champEvents
					}
				}
			}
		}

		err = pm.store.UpsertChampionship(championship)

		if err != nil {
			logrus.WithError(err).Errorf("Couldn't save championship with ID: %s", results.ChampionshipID)
			return err
		}
	}

	if results.RaceWeekendID != "" {
		raceWeekend, err := pm.store.LoadRaceWeekend(results.RaceWeekendID)

		if err != nil {
			logrus.WithError(err).Errorf("Couldn't load race weekend with id: %s", results.RaceWeekendID)
			return err
		}

		for _, session := range raceWeekend.Sessions {
			if !session.Completed() {
				continue
			}

			if session.Results.SessionFile == jsonFileName {
				session.Results = results
				break
			}
		}

		err = pm.store.UpsertRaceWeekend(raceWeekend)

		if err != nil {
			logrus.WithError(err).Errorf("Could not update race weekend: %s", raceWeekend.ID.String())
			return err
		}
	}

	return nil
}
