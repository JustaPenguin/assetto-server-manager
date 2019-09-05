package servermanager

type FilterError string

func (f FilterError) Error() string {
	return string(f)
}

type EntrantPositionFilter struct {
	ResultStart     int
	ResultEnd       int
	ReverseEntrants bool

	EntryListStart int
	EntryListEnd   int
}

func (epf EntrantPositionFilter) Name() string {
	return "Entrant Position Filter"
}

func (epf EntrantPositionFilter) Description() string {
	return "Choose a start and finish position. Entrants between start and end will remain, others will be removed"
}

func (epf EntrantPositionFilter) Key() string {
	return "ENTRANT_POSITION_FILTER"
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

func (epf EntrantPositionFilter) Filter(entrants []*RaceWeekendSessionEntrant, entryList EntryList) error {
	entrants = filterDisqualifiedResults(entrants)

	if epf.ResultEnd < epf.ResultStart {
		// normalise result end to be greater than result start
		epf.ResultEnd, epf.ResultStart = epf.ResultStart, epf.ResultEnd
	}

	if epf.EntryListEnd < epf.EntryListStart {
		// normalise entrylist end to be greater than entrylist start
		epf.EntryListEnd, epf.EntryListStart = epf.EntryListStart, epf.EntryListEnd
	}

	if epf.ResultEnd > len(entrants) {
		epf.ResultEnd = len(entrants)
	}

	if epf.EntryListEnd > len(entrants) {
		epf.EntryListEnd = len(entrants)
	}

	// shift down one, remove the user friendliness
	epf.ResultStart--
	epf.EntryListStart--

	if epf.ResultStart < 0 || epf.ResultStart > len(entrants) || epf.ResultEnd < 0 || epf.ResultEnd > len(entrants) {
		return FilterError("Invalid bounds for Start or End")
	}

	if epf.ResultEnd-epf.ResultStart != epf.EntryListEnd-epf.EntryListStart {
		return FilterError("Interval between result and entrylist splits must be equal.")
	}

	split := entrants[epf.ResultStart:epf.ResultEnd]

	splitIndex := 0

	if epf.ReverseEntrants {
		splitIndex = len(split) - 1
	}

	for pitBox := epf.EntryListStart; pitBox < epf.EntryListEnd; pitBox++ {
		entryList.AddInPitBox(split[splitIndex].GetEntrant(), pitBox)

		if epf.ReverseEntrants {
			splitIndex--
		} else {
			splitIndex++
		}
	}

	return nil
}
