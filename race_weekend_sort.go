package servermanager

import (
	"math/rand"
	"sort"
	"time"
)

// RaceWeekendEntryListSorter is a function which takes a race weekend, session and entrylist and sorts the entrylist based on some criteria.
type RaceWeekendEntryListSorter func(*RaceWeekendSession, []*RaceWeekendSessionEntrant) error

type RaceWeekendEntryListSorterDescription struct {
	Name     string
	Key      string
	SortFunc RaceWeekendEntryListSorter
}

var RaceWeekendEntryListSorters = []RaceWeekendEntryListSorterDescription{
	{
		Name:     "No Sort (Use Finishing Grid)",
		Key:      "no_sort",
		SortFunc: UnchangedRaceWeekendEntryListSort,
	},
	{
		Name:     "Fastest Lap",
		Key:      "fastest_lap",
		SortFunc: FastestLapRaceWeekendEntryListSort,
	},
	{
		Name:     "Total Race Time",
		Key:      "total_race_time",
		SortFunc: TotalRaceTimeRaceWeekendEntryListSort,
	},
	{
		Name:     "Least Collisions",
		Key:      "least_collisions",
		SortFunc: LeastCollisionsRaceWeekendEntryListSort,
	},
	{
		Name:     "Least Cuts",
		Key:      "least_cuts",
		SortFunc: LeastCutsRaceWeekendEntryListSort,
	},
	{
		Name:     "Safety (Collisions then Cuts)",
		Key:      "safety",
		SortFunc: SafetyRaceWeekendEntryListSort,
	},
	{
		Name:     "Random",
		Key:      "random",
		SortFunc: RandomRaceWeekendEntryListSort,
	},
}

func GetRaceWeekendEntryListSort(key string) RaceWeekendEntryListSorter {
	for _, sorter := range RaceWeekendEntryListSorters {
		if sorter.Key == key {
			return sorter.SortFunc
		}
	}

	return UnchangedRaceWeekendEntryListSort
}

// @TODO when classes work, sorting should be _in class_

func UnchangedRaceWeekendEntryListSort(_ *RaceWeekendSession, _ []*RaceWeekendSessionEntrant) error {
	return nil // do nothing
}

func FastestLapRaceWeekendEntryListSort(session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]

		return lessBestLapTime(session, entrantI, entrantJ)
	})

	return nil
}

func TotalRaceTimeRaceWeekendEntryListSort(session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]

		return lessTotalEntrantTime(session, entrantI, entrantJ)
	})

	return nil
}

func lessTotalEntrantTime(session *RaceWeekendSession, entrantI, entrantJ *RaceWeekendSessionEntrant) bool {
	if session.Results.GetNumLaps(entrantI.Car.Driver.GUID) == session.Results.GetNumLaps(entrantJ.Car.Driver.GUID) {
		// drivers have completed the same number of laps, so compare their total time
		entrantITime := session.Results.GetTime(entrantI.Results.TotalTime, entrantI.Car.Driver.GUID, true)
		entrantJTime := session.Results.GetTime(entrantJ.Results.TotalTime, entrantJ.Car.Driver.GUID, true)

		return entrantITime < entrantJTime
	} else {
		return session.Results.GetNumLaps(entrantI.Car.Driver.GUID) > session.Results.GetNumLaps(entrantJ.Car.Driver.GUID)
	}
}

func lessBestLapTime(session *RaceWeekendSession, entrantI, entrantJ *RaceWeekendSessionEntrant) bool {
	if entrantI.Results.BestLap == 0 {
		// entrantI has a zero best lap. they must be not-less than J
		return false
	}

	if entrantJ.Results.BestLap == 0 {
		// entrantJ has a zero best lap. entrantI must be less than J
		return true
	}

	if entrantI.Results.BestLap == entrantJ.Results.BestLap {
		// if equal, compare safety
		entrantICrashes := session.Results.GetCrashes(entrantI.Car.Driver.GUID)
		entrantJCrashes := session.Results.GetCrashes(entrantJ.Car.Driver.GUID)

		if entrantICrashes == entrantJCrashes {
			return session.Results.GetCuts(entrantI.Car.Driver.GUID) < session.Results.GetCuts(entrantJ.Car.Driver.GUID)
		}

		return entrantICrashes < entrantJCrashes
	}

	return entrantI.Results.BestLap < entrantJ.Results.BestLap
}

func LeastCollisionsRaceWeekendEntryListSort(session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]
		entrantICrashes := session.Results.GetCrashes(entrantI.Car.Driver.GUID)
		entrantJCrashes := session.Results.GetCrashes(entrantJ.Car.Driver.GUID)

		if entrantICrashes == entrantJCrashes {
			if session.SessionType() == SessionTypeRace {
				return lessTotalEntrantTime(session, entrantI, entrantJ)
			} else {
				return lessBestLapTime(session, entrantI, entrantJ)
			}
		}

		return entrantICrashes < entrantJCrashes
	})

	return nil
}

func LeastCutsRaceWeekendEntryListSort(session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]
		entrantICuts := session.Results.GetCuts(entrantI.Car.Driver.GUID)
		entrantJCuts := session.Results.GetCuts(entrantJ.Car.Driver.GUID)

		if entrantICuts == entrantJCuts {
			if session.SessionType() == SessionTypeRace {
				return lessTotalEntrantTime(session, entrantI, entrantJ)
			} else {
				return lessBestLapTime(session, entrantI, entrantJ)
			}
		}

		return entrantICuts < entrantJCuts
	})

	return nil
}

func SafetyRaceWeekendEntryListSort(session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]
		entrantICrashes := session.Results.GetCrashes(entrantI.Car.Driver.GUID)
		entrantJCrashes := session.Results.GetCrashes(entrantJ.Car.Driver.GUID)
		entrantICuts := session.Results.GetCuts(entrantI.Car.Driver.GUID)
		entrantJCuts := session.Results.GetCuts(entrantJ.Car.Driver.GUID)

		if entrantICrashes == entrantJCrashes {
			if entrantICuts == entrantJCuts {
				if session.SessionType() == SessionTypeRace {
					return lessTotalEntrantTime(session, entrantI, entrantJ)
				} else {
					return lessBestLapTime(session, entrantI, entrantJ)
				}
			} else {
				return entrantICuts < entrantJCuts
			}
		}

		return entrantICrashes < entrantJCrashes
	})

	return nil
}

func RandomRaceWeekendEntryListSort(_ *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	r.Shuffle(len(entrants), func(i, j int) {
		entrants[i], entrants[j] = entrants[j], entrants[i]
	})

	return nil
}
