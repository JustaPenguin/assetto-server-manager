package servermanager

type FilterError string

func (f FilterError) Error() string {
	return string(f)
}

type RaceWeekendSessionToSessionFilter struct {
	ResultStart     int
	ResultEnd       int
	ReverseEntrants bool

	EntryListStart int

	SortType string
}

// Filter takes a set of RaceWeekendSessionEntrants formed by the results of the parent session and filters them into a child session entry list.
func (f RaceWeekendSessionToSessionFilter) Filter(parentSession, childSession *RaceWeekendSession, parentSessionResults []*RaceWeekendSessionEntrant, childSessionEntryList *RaceWeekendEntryList) error {
	if parentSession.Completed() {
		sorter := GetRaceWeekendEntryListSort(f.SortType)

		// race weekend session is completed and has a valid sorter, use it to sort results before filtering.
		if err := sorter(parentSession, parentSessionResults); err != nil {
			return err
		}

		if f.ReverseEntrants {
			for i := len(parentSessionResults)/2 - 1; i >= 0; i-- {
				opp := len(parentSessionResults) - 1 - i
				parentSessionResults[i], parentSessionResults[opp] = parentSessionResults[opp], parentSessionResults[i]
			}
		}
	}

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

	for pitBox := entryListStart; pitBox < entryListStart+(resultEnd-resultStart); pitBox++ {
		entrant := split[splitIndex]
		entrant.SessionID = parentSession.ID

		childSessionEntryList.AddInPitBox(entrant, pitBox)

		splitIndex++
	}

	return nil
}
