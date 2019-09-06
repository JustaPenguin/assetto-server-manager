package servermanager

import (
	"fmt"

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
	EntryListEnd   int
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

	if f.ResultEnd < f.ResultStart {
		// normalise result end to be greater than result start
		f.ResultEnd, f.ResultStart = f.ResultStart, f.ResultEnd
	}

	if f.EntryListEnd < f.EntryListStart {
		// normalise entrylist end to be greater than entrylist start
		f.EntryListEnd, f.EntryListStart = f.EntryListStart, f.EntryListEnd
	}

	// shift down one, remove the user friendliness
	f.ResultStart--
	f.EntryListStart--

	if f.ResultStart < 0 || f.ResultStart > len(parentSessionResults) || f.ResultEnd < 0 || f.ResultEnd > len(parentSessionResults) {
		return FilterError("Invalid bounds for Start or End")
	}

	if f.ResultEnd-f.ResultStart != f.EntryListEnd-f.EntryListStart {
		return FilterError(fmt.Sprintf("Interval between result and entrylist splits must be equal. (%d vs %d)", f.ResultEnd-f.ResultStart, f.EntryListEnd-f.EntryListStart))
	}

	split := parentSessionResults[f.ResultStart:f.ResultEnd]

	splitIndex := 0

	if f.ReverseEntrants {
		splitIndex = len(split) - 1
	}

	for pitBox := f.EntryListStart; pitBox < f.EntryListEnd; pitBox++ {
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
