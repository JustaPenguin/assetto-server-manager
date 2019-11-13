package servermanager

import (
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
)

// RaceWeekendEntryListSorter is a function which takes a race weekend, session and entrylist and sorts the entrylist based on some criteria.
type RaceWeekendEntryListSorter interface {
	Sort(*RaceWeekend, *RaceWeekendSession, []*RaceWeekendSessionEntrant) error
}

type RaceWeekendEntryListSorterDescription struct {
	Name               string
	Key                string
	Sorter             RaceWeekendEntryListSorter
	NeedsParentSession bool
}

type RaceWeekendEntryListSortFunc func(*RaceWeekend, *RaceWeekendSession, []*RaceWeekendSessionEntrant) error

func (rwelsf RaceWeekendEntryListSortFunc) Sort(rw *RaceWeekend, rws *RaceWeekendSession, rwes []*RaceWeekendSessionEntrant) error {
	return rwelsf(rw, rws, rwes)
}

var RaceWeekendEntryListSorters = []RaceWeekendEntryListSorterDescription{
	{
		Name:               "No Sort (Use Finishing Grid)",
		Key:                "", // key intentionally left blank
		Sorter:             RaceWeekendEntryListSortFunc(UnchangedRaceWeekendEntryListSort),
		NeedsParentSession: false,
	},
	{
		Name:               "Fastest Lap",
		Key:                "fastest_lap",
		Sorter:             RaceWeekendEntryListSortFunc(FastestLapRaceWeekendEntryListSort),
		NeedsParentSession: true,
	},
	{
		Name:               "Total Race Time",
		Key:                "total_race_time",
		Sorter:             RaceWeekendEntryListSortFunc(TotalRaceTimeRaceWeekendEntryListSort),
		NeedsParentSession: true,
	},
	{
		Name:               "Fewest Collisions",
		Key:                "fewest_collisions",
		Sorter:             RaceWeekendEntryListSortFunc(FewestCollisionsRaceWeekendEntryListSort),
		NeedsParentSession: true,
	},
	{
		Name:               "Fewest Cuts",
		Key:                "fewest_cuts",
		Sorter:             RaceWeekendEntryListSortFunc(FewestCutsRaceWeekendEntryListSort),
		NeedsParentSession: true,
	},
	{
		Name:               "Safety (Collisions then Cuts)",
		Key:                "safety",
		Sorter:             RaceWeekendEntryListSortFunc(SafetyRaceWeekendEntryListSort),
		NeedsParentSession: true,
	},
	{
		Name:               "Championship Standings Order",
		Key:                "championship_standings_order",
		Sorter:             &ChampionshipStandingsOrderEntryListSort{},
		NeedsParentSession: false,
	},
	{
		Name:               "Random",
		Key:                "random",
		Sorter:             RaceWeekendEntryListSortFunc(RandomRaceWeekendEntryListSort),
		NeedsParentSession: false,
	},
	{
		Name:               "Alphabetical (Using Driver Name)",
		Key:                "alphabetical",
		Sorter:             RaceWeekendEntryListSortFunc(AlphabeticalRaceWeekendEntryListSort),
		NeedsParentSession: false,
	},
}

func GetRaceWeekendEntryListSort(key string) RaceWeekendEntryListSorter {
	for _, sorter := range RaceWeekendEntryListSorters {
		if sorter.Key == key {
			return PerClassSort(sorter.Sorter)
		}
	}

	return PerClassSort(RaceWeekendEntryListSortFunc(UnchangedRaceWeekendEntryListSort))
}

func PerClassSort(sorter RaceWeekendEntryListSorter) RaceWeekendEntryListSorter {
	return RaceWeekendEntryListSortFunc(func(rw *RaceWeekend, session *RaceWeekendSession, allEntrants []*RaceWeekendSessionEntrant) error {
		classMap := make(map[uuid.UUID]bool)
		fastestLapForClass := make(map[uuid.UUID]int)
		entrantsForClass := make(map[uuid.UUID][]*RaceWeekendSessionEntrant)

		for _, entrant := range allEntrants {
			classMap[entrant.EntrantResult.ClassID] = true

			fastestLap, ok := fastestLapForClass[entrant.EntrantResult.ClassID]

			if entrant.EntrantResult.BestLap > 0 {
				if !ok || (ok && entrant.EntrantResult.BestLap < fastestLap) {
					fastestLapForClass[entrant.EntrantResult.ClassID] = entrant.EntrantResult.BestLap
				}
			}

			entrantsForClass[entrant.EntrantResult.ClassID] = append(entrantsForClass[entrant.EntrantResult.ClassID], entrant)
		}

		var classes []uuid.UUID

		for class := range classMap {
			classes = append(classes, class)
		}

		if len(fastestLapForClass) == len(classes) {
			// sort each class by the fastest lap in that class
			sort.Slice(classes, func(i, j int) bool {
				return fastestLapForClass[classes[i]] < fastestLapForClass[classes[j]]
			})
		}

		lastStartPos := 0

		for _, class := range classes {
			entrants := entrantsForClass[class]

			err := sorter.Sort(rw, session, entrants)

			if err != nil {
				return err
			}

			reverseEntrants(session.NumEntrantsToReverse, entrants)

			if _, isChampionshipOrderSort := sorter.(*ChampionshipStandingsOrderEntryListSort); isChampionshipOrderSort && rw.HasLinkedChampionship() && rw.Championship != nil {
				sortDriversWithNoChampionshipRacesToBackOfGrid(rw.Championship, entrants)
			} else {
				sortDriversWithNoTimeToBackOfGrid(entrants)
			}

			for index, entrant := range entrants {
				allEntrants[lastStartPos+index] = entrant
			}

			lastStartPos += len(entrants)
		}

		return nil
	})
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

func UnchangedRaceWeekendEntryListSort(_ *RaceWeekend, _ *RaceWeekendSession, _ []*RaceWeekendSessionEntrant) error {
	return nil // do nothing
}

func FastestLapRaceWeekendEntryListSort(rw *RaceWeekend, session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]

		return lessBestLapTime(rw, session, entrantI, entrantJ)
	})

	return nil
}

func TotalRaceTimeRaceWeekendEntryListSort(rw *RaceWeekend, session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]

		return lessTotalEntrantTime(rw, session, entrantI, entrantJ)
	})

	return nil
}

func lessTotalEntrantTime(_ *RaceWeekend, _ *RaceWeekendSession, entrantI, entrantJ *RaceWeekendSessionEntrant) bool {
	if entrantI.SessionResults.GetNumLaps(entrantI.Car.Driver.GUID, entrantI.Car.Model) == entrantJ.SessionResults.GetNumLaps(entrantJ.Car.Driver.GUID, entrantJ.Car.Model) {
		// drivers have completed the same number of laps, so compare their total time
		entrantITime := entrantI.SessionResults.GetTime(entrantI.EntrantResult.TotalTime, entrantI.Car.Driver.GUID, entrantI.Car.Model, true)
		entrantJTime := entrantJ.SessionResults.GetTime(entrantJ.EntrantResult.TotalTime, entrantJ.Car.Driver.GUID, entrantJ.Car.Model, true)

		return entrantITime < entrantJTime
	} else {
		return entrantI.SessionResults.GetNumLaps(entrantI.Car.Driver.GUID, entrantI.Car.Model) > entrantJ.SessionResults.GetNumLaps(entrantJ.Car.Driver.GUID, entrantJ.Car.Model)
	}
}

func lessBestLapTime(_ *RaceWeekend, _ *RaceWeekendSession, entrantI, entrantJ *RaceWeekendSessionEntrant) bool {
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
			return entrantI.SessionResults.GetCuts(entrantI.Car.Driver.GUID, entrantI.Car.Model) < entrantJ.SessionResults.GetCuts(entrantJ.Car.Driver.GUID, entrantJ.Car.Model)
		}

		return entrantICrashes < entrantJCrashes
	}

	return entrantI.EntrantResult.BestLap < entrantJ.EntrantResult.BestLap
}

func FewestCollisionsRaceWeekendEntryListSort(rw *RaceWeekend, session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]
		entrantICrashes := entrantI.SessionResults.GetCrashes(entrantI.Car.Driver.GUID)
		entrantJCrashes := entrantJ.SessionResults.GetCrashes(entrantJ.Car.Driver.GUID)

		if entrantICrashes == entrantJCrashes {
			if session.SessionType() == SessionTypeRace {
				return lessTotalEntrantTime(rw, session, entrantI, entrantJ)
			} else {
				return lessBestLapTime(rw, session, entrantI, entrantJ)
			}
		}

		return entrantICrashes < entrantJCrashes
	})

	return nil
}

func FewestCutsRaceWeekendEntryListSort(rw *RaceWeekend, session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]
		entrantICuts := entrantI.SessionResults.GetCuts(entrantI.Car.Driver.GUID, entrantI.Car.Model)
		entrantJCuts := entrantJ.SessionResults.GetCuts(entrantJ.Car.Driver.GUID, entrantJ.Car.Model)

		if entrantICuts == entrantJCuts {
			if session.SessionType() == SessionTypeRace {
				return lessTotalEntrantTime(rw, session, entrantI, entrantJ)
			} else {
				return lessBestLapTime(rw, session, entrantI, entrantJ)
			}
		}

		return entrantICuts < entrantJCuts
	})

	return nil
}

func SafetyRaceWeekendEntryListSort(rw *RaceWeekend, session *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]
		entrantICrashes := entrantI.SessionResults.GetCrashes(entrantI.Car.Driver.GUID)
		entrantJCrashes := entrantJ.SessionResults.GetCrashes(entrantJ.Car.Driver.GUID)
		entrantICuts := entrantI.SessionResults.GetCuts(entrantI.Car.Driver.GUID, entrantI.Car.Model)
		entrantJCuts := entrantJ.SessionResults.GetCuts(entrantJ.Car.Driver.GUID, entrantJ.Car.Model)

		if entrantICrashes == entrantJCrashes {
			if entrantICuts == entrantJCuts {
				if session.SessionType() == SessionTypeRace {
					return lessTotalEntrantTime(rw, session, entrantI, entrantJ)
				} else {
					return lessBestLapTime(rw, session, entrantI, entrantJ)
				}
			} else {
				return entrantICuts < entrantJCuts
			}
		}

		return entrantICrashes < entrantJCrashes
	})

	return nil
}

func RandomRaceWeekendEntryListSort(_ *RaceWeekend, _ *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	r.Shuffle(len(entrants), func(i, j int) {
		entrants[i], entrants[j] = entrants[j], entrants[i]
	})

	return nil
}

func AlphabeticalRaceWeekendEntryListSort(_ *RaceWeekend, _ *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]

		return entrantI.Car.Driver.Name < entrantJ.Car.Driver.Name
	})

	return nil
}

type ChampionshipStandingsOrderEntryListSort struct{}

func (ChampionshipStandingsOrderEntryListSort) Sort(rw *RaceWeekend, _ *RaceWeekendSession, entrants []*RaceWeekendSessionEntrant) error {
	if !rw.HasLinkedChampionship() || rw.Championship == nil {
		return nil
	}

	if len(entrants) == 0 {
		return nil
	}

	class := entrants[0].ChampionshipClass(rw)
	standings := class.Standings(rw.Championship.Events)

	sort.Slice(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]

		iPos := len(standings)
		jPos := len(standings)

		for i, standing := range standings {
			if standing.Car.GetGUID() == entrantI.GetEntrant().GUID {
				iPos = i
			}

			if standing.Car.GetGUID() == entrantJ.GetEntrant().GUID {
				jPos = i
			}
		}

		return iPos < jPos
	})

	return nil
}

func sortDriversWithNoChampionshipRacesToBackOfGrid(championship *Championship, entrants []*RaceWeekendSessionEntrant) {
	sort.SliceStable(entrants, func(i, j int) bool {
		entrantI, entrantJ := entrants[i], entrants[j]

		entrantIAttendance := championship.EntrantAttendance(entrantI.GetEntrant().GUID)
		entrantJAttendance := championship.EntrantAttendance(entrantJ.GetEntrant().GUID)

		if entrantIAttendance == 0 {
			return false
		}

		if entrantJAttendance == 0 {
			return true
		}

		return i < j
	})
}
