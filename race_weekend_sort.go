package servermanager

import (
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
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
		Key:      "", // key intentionally left blank
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
		Name:     "Fewest Collisions",
		Key:      "fewest_collisions",
		SortFunc: FewestCollisionsRaceWeekendEntryListSort,
	},
	{
		Name:     "Fewest Cuts",
		Key:      "fewest_cuts",
		SortFunc: FewestCutsRaceWeekendEntryListSort,
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
	{
		Name:     "Alphabetical (Using Driver Name)",
		Key:      "alphabetical",
		SortFunc: AlphabeticalRaceWeekendEntryListSort,
	},
}

func GetRaceWeekendEntryListSort(key string) RaceWeekendEntryListSorter {
	for _, sorter := range RaceWeekendEntryListSorters {
		if sorter.Key == key {
			return PerClassSort(sorter.SortFunc)
		}
	}

	return PerClassSort(UnchangedRaceWeekendEntryListSort)
}

func PerClassSort(sorter RaceWeekendEntryListSorter) RaceWeekendEntryListSorter {
	return func(session *RaceWeekendSession, allEntrants []*RaceWeekendSessionEntrant) error {
		fastestLapForClass := make(map[uuid.UUID]int)
		entrantsForClass := make(map[uuid.UUID][]*RaceWeekendSessionEntrant)

		for _, entrant := range allEntrants {
			fastestLap, ok := fastestLapForClass[entrant.EntrantResult.ClassID]

			if entrant.EntrantResult.BestLap > 0 {
				if !ok || (ok && entrant.EntrantResult.BestLap < fastestLap) {
					fastestLapForClass[entrant.EntrantResult.ClassID] = entrant.EntrantResult.BestLap
				}
			}

			entrantsForClass[entrant.EntrantResult.ClassID] = append(entrantsForClass[entrant.EntrantResult.ClassID], entrant)
		}

		var classes []uuid.UUID

		for class := range fastestLapForClass {
			classes = append(classes, class)
		}

		// sort each class by the fastest lap in that class
		sort.Slice(classes, func(i, j int) bool {
			return fastestLapForClass[classes[i]] < fastestLapForClass[classes[j]]
		})

		lastStartPos := 0

		for _, class := range classes {
			entrants := entrantsForClass[class]

			err := sorter(session, entrants)

			if err != nil {
				return err
			}

			reverseEntrants(session.NumEntrantsToReverse, entrants)

			sortDriversWithNoTimeToBackOfGrid(entrants)

			for index, entrant := range entrants {
				allEntrants[lastStartPos+index] = entrant
			}

			lastStartPos += len(entrants)
		}

		return nil
	}
}

func sortDriversWithNoTimeToBackOfGrid(entrants []*RaceWeekendSessionEntrant) {
	sort.SliceStable(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]

		if entrantI.EntrantResult.TotalTime == 0 {
			return false
		}

		if entrantJ.EntrantResult.TotalTime == 0 {
			return true
		}

		return i < j
	})
}

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
	if entrantI.SessionResults.GetNumLaps(entrantI.Car.Driver.GUID) == entrantJ.SessionResults.GetNumLaps(entrantJ.Car.Driver.GUID) {
		// drivers have completed the same number of laps, so compare their total time
		entrantITime := entrantI.SessionResults.GetTime(entrantI.EntrantResult.TotalTime, entrantI.Car.Driver.GUID, true)
		entrantJTime := entrantJ.SessionResults.GetTime(entrantJ.EntrantResult.TotalTime, entrantJ.Car.Driver.GUID, true)

		return entrantITime < entrantJTime
	} else {
		return entrantI.SessionResults.GetNumLaps(entrantI.Car.Driver.GUID) > entrantJ.SessionResults.GetNumLaps(entrantJ.Car.Driver.GUID)
	}
}

func lessBestLapTime(session *RaceWeekendSession, entrantI, entrantJ *RaceWeekendSessionEntrant) bool {
	if entrantI.EntrantResult.BestLap == 0 {
		// entrantI has a zero best lap. they must be not-less than J
		return false
	}

	if entrantJ.EntrantResult.BestLap == 0 {
		// entrantJ has a zero best lap. entrantI must be less than J
		return true
	}

	if entrantI.EntrantResult.BestLap == entrantJ.EntrantResult.BestLap {
		// if equal, compare safety
		entrantICrashes := entrantI.SessionResults.GetCrashes(entrantI.Car.Driver.GUID)
		entrantJCrashes := entrantJ.SessionResults.GetCrashes(entrantJ.Car.Driver.GUID)

		if entrantICrashes == entrantJCrashes {
			return entrantI.SessionResults.GetCuts(entrantI.Car.Driver.GUID) < entrantJ.SessionResults.GetCuts(entrantJ.Car.Driver.GUID)
		}

		return entrantICrashes < entrantJCrashes
	}

	return entrantI.EntrantResult.BestLap < entrantJ.EntrantResult.BestLap
}

func FewestCollisionsRaceWeekendEntryListSort(session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]
		entrantICrashes := entrantI.SessionResults.GetCrashes(entrantI.Car.Driver.GUID)
		entrantJCrashes := entrantJ.SessionResults.GetCrashes(entrantJ.Car.Driver.GUID)

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

func FewestCutsRaceWeekendEntryListSort(session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]
		entrantICuts := entrantI.SessionResults.GetCuts(entrantI.Car.Driver.GUID)
		entrantJCuts := entrantJ.SessionResults.GetCuts(entrantJ.Car.Driver.GUID)

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
		entrantICrashes := entrantI.SessionResults.GetCrashes(entrantI.Car.Driver.GUID)
		entrantJCrashes := entrantJ.SessionResults.GetCrashes(entrantJ.Car.Driver.GUID)
		entrantICuts := entrantI.SessionResults.GetCuts(entrantI.Car.Driver.GUID)
		entrantJCuts := entrantJ.SessionResults.GetCuts(entrantJ.Car.Driver.GUID)

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

func AlphabeticalRaceWeekendEntryListSort(_ *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]

		return entrantI.Car.Driver.Name < entrantJ.Car.Driver.Name
	})

	return nil
}
