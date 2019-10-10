package servermanager

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type PenaltiesHandler struct {
	*BaseHandler

	championshipManager *ChampionshipManager
	raceWeekendManager  *RaceWeekendManager
}

func NewPenaltiesHandler(baseHandler *BaseHandler, championshipManager *ChampionshipManager, raceWeekendManager *RaceWeekendManager) *PenaltiesHandler {
	return &PenaltiesHandler{
		BaseHandler:         baseHandler,
		championshipManager: championshipManager,
		raceWeekendManager:  raceWeekendManager,
	}
}

func (ph *PenaltiesHandler) managePenalty(w http.ResponseWriter, r *http.Request) {
	remove, err := ph.applyPenalty(r)

	if err != nil {
		AddErrorFlash(w, r, "Could not add/remove penalty")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	if remove {
		AddFlash(w, r, "Penalty Removed!")
	} else {
		AddFlash(w, r, "Penalty Added!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ph *PenaltiesHandler) applyPenalty(r *http.Request) (bool, error) {
	var results *SessionResults
	var remove bool
	var penaltyTime float64

	jsonFileName := chi.URLParam(r, "sessionFile")
	GUID := chi.URLParam(r, "driverGUID")

	results, err := LoadResult(jsonFileName + ".json")

	if err != nil {
		logrus.WithError(err).Errorf("could not load session result file")
		return false, err
	}

	err = r.ParseForm()

	if err != nil {
		logrus.WithError(err).Errorf("could not load parse form")
		return false, err
	}

	if r.FormValue("action") == "add" {
		penaltyString := r.FormValue("time-penalty")

		if penaltyString == "" {
			penaltyTime = 0
		} else {
			pen, err := strconv.ParseFloat(penaltyString, 64)

			if err != nil {
				logrus.WithError(err).Errorf("could not parse penalty time")
				return false, err
			}

			penaltyTime = pen
		}
	} else {
		// remove penalty
		remove = true
	}

	for _, result := range results.Result {
		if result.DriverGUID == GUID {
			if remove {
				result.HasPenalty = false
				result.Disqualified = false
				result.PenaltyTime = 0
				result.LapPenalty = 0
			} else {
				if penaltyTime == 0 {
					result.Disqualified = true
					result.HasPenalty = false
					result.LapPenalty = 0
				} else {
					result.HasPenalty = true
					result.Disqualified = false

					timeParsed, err := time.ParseDuration(fmt.Sprintf("%.1fs", penaltyTime))

					if err != nil {
						logrus.WithError(err).Errorf("could not parse penalty time")
						return false, err
					}

					result.PenaltyTime = timeParsed

					// If penalty time is greater than a lap then add a lap penalty and change penalty time by one lap
					lastLapTime := results.GetLastLapTime(result.DriverGUID)

					if result.PenaltyTime > lastLapTime {
						result.LapPenalty = int(result.PenaltyTime / lastLapTime)
					}
				}
			}
		}
	}

	// sort results.Result, if disqualified go to back, if time penalty sort by laps completed then lap time
	sort.Slice(results.Result, func(i, j int) bool {
		if !results.Result[i].Disqualified && !results.Result[j].Disqualified {

			// if both drivers aren't disqualified
			if results.GetNumLaps(results.Result[i].DriverGUID) == results.GetNumLaps(results.Result[j].DriverGUID) {
				// if their number of laps are equal, compare lap times

				return results.GetTime(results.Result[i].TotalTime, results.Result[i].DriverGUID, true) <
					results.GetTime(results.Result[j].TotalTime, results.Result[j].DriverGUID, true)
			}

			return results.GetNumLaps(results.Result[i].DriverGUID) >= results.GetNumLaps(results.Result[j].DriverGUID)

		} else if results.Result[i].Disqualified && results.Result[j].Disqualified {

			// if both drivers ARE disqualified, compare their lap times / num laps
			if results.GetNumLaps(results.Result[i].DriverGUID) == results.GetNumLaps(results.Result[j].DriverGUID) {
				// if their number of laps are equal, compare lap times
				return results.GetTime(results.Result[i].TotalTime, results.Result[i].DriverGUID, true) <
					results.GetTime(results.Result[j].TotalTime, results.Result[j].DriverGUID, true)
			}

			return results.GetNumLaps(results.Result[i].DriverGUID) >= results.GetNumLaps(results.Result[j].DriverGUID)

		} else {
			// driver i is closer to the front than j if they are not disqualified and j is
			return results.Result[j].Disqualified
		}
	})

	err = saveResults(jsonFileName+".json", results)

	if err != nil {
		logrus.WithError(err).Errorf("could not encode to session result file")
		return false, err
	}

	if results.ChampionshipID != "" {
		championship, err := ph.championshipManager.LoadChampionship(results.ChampionshipID)

		if err != nil {
			logrus.WithError(err).Errorf("Couldn't load championship with ID: %s")
			return false, err
		}

	champEvents:
		for i, event := range championship.Events {
			for key, session := range event.Sessions {
				if session.Results.SessionFile == jsonFileName {
					championship.Events[i].Sessions[key].Results = results

					break champEvents
				}
			}
		}

		err = ph.championshipManager.UpsertChampionship(championship)

		if err != nil {
			logrus.WithError(err).Errorf("Couldn't save championship with ID: %s", results.ChampionshipID)
			return false, err
		}
	}

	if results.RaceWeekendID != "" {
		raceWeekend, err := ph.raceWeekendManager.LoadRaceWeekend(results.RaceWeekendID)

		if err != nil {
			logrus.WithError(err).Errorf("Couldn't load race weekend with id: %s", results.RaceWeekendID)
			return false, err
		}

		for _, session := range raceWeekend.Sessions {
			if session.Results.SessionFile == jsonFileName {
				session.Results = results
				break
			}
		}

		// @TODO change this to use rwm.UpsertRaceWeekend
		err = ph.raceWeekendManager.store.UpsertRaceWeekend(raceWeekend)

		if err != nil {
			logrus.WithError(err).Errorf("Could not update race weekend: %s", raceWeekend.ID.String())
			return false, err
		}
	}

	return remove, nil
}
