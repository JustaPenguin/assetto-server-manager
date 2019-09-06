package servermanager

import (
	"github.com/google/uuid"
)

type FilterError string

func (f FilterError) Error() string {
	return string(f)
}

type RaceWeekendSessionToSessionFilter struct {
	ResultStart     int
	ResultEnd       int
	ReverseEntrants bool

	EntryListStart int
}

func filterDisqualifiedResults(entrants []*RaceWeekendSessionEntrant) []*RaceWeekendSessionEntrant {
	var out []*RaceWeekendSessionEntrant

	for _, entrant := range entrants {
		if !entrant.Results.Disqualified {
			out = append(out, entrant)
		}
	}

	return out
}

// Filter takes a set of RaceWeekendSessionEntrants formed by the results of the parent session and filters them into a child session entry list.
func (f RaceWeekendSessionToSessionFilter) Filter(parentSessionID uuid.UUID, parentSessionResults []*RaceWeekendSessionEntrant, childSessionEntryList RaceWeekendEntryList) error {
	parentSessionResults = filterDisqualifiedResults(parentSessionResults)

	resultStart, resultEnd, entryListStart := f.ResultStart, f.ResultEnd, f.EntryListStart

	resultStart--
	entryListStart--

	if resultStart > len(parentSessionResults) {
		return nil
	}

	if resultEnd > len(parentSessionResults) {
		resultEnd = len(parentSessionResults)
	}

	split := parentSessionResults[resultStart:resultEnd]

	splitIndex := 0

	if f.ReverseEntrants {
		splitIndex = len(split) - 1
	}

	for pitBox := entryListStart; pitBox < entryListStart+(resultEnd-resultStart); pitBox++ {
		entrant := split[splitIndex]
		entrant.SessionID = parentSessionID

		childSessionEntryList.AddInPitBox(entrant, pitBox)

		if f.ReverseEntrants {
			splitIndex--
		} else {
			splitIndex++
		}
	}

	return nil
}
