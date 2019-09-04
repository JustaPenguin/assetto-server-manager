package servermanager

import (
	"github.com/sirupsen/logrus"
)

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

func filterDisqualifiedResults(results []*SessionResult) []*SessionResult {
	var out []*SessionResult

	for _, result := range results {
		if !result.Disqualified {
			out = append(out, result)
		}
	}

	return out
}

func (epf EntrantPositionFilter) Filter(results *SessionResults, entryList EntryList) error {
	results.Result = filterDisqualifiedResults(results.Result)

	if epf.ResultEnd < epf.ResultStart {
		// normalise result end to be greater than result start
		epf.ResultEnd, epf.ResultStart = epf.ResultStart, epf.ResultEnd
	}

	if epf.EntryListEnd < epf.EntryListStart {
		// normalise entrylist end to be greater than entrylist start
		epf.EntryListEnd, epf.EntryListStart = epf.EntryListStart, epf.EntryListEnd
	}

	// shift down one, remove the user friendliness
	epf.ResultStart--
	epf.EntryListStart--

	if epf.ResultStart < 0 || epf.ResultStart > len(results.Result) || epf.ResultEnd < 0 || epf.ResultEnd > len(results.Result) {
		return FilterError("Invalid bounds for Start or End")
	}

	if epf.ResultEnd-epf.ResultStart != epf.EntryListEnd-epf.EntryListStart {
		return FilterError("Interval between result and entrylist splits must be equal.")
	}

	split := results.Result[epf.ResultStart:epf.ResultEnd]

	splitIndex := 0

	if epf.ReverseEntrants {
		splitIndex = len(split) - 1
	}

	for pitBox := epf.EntryListStart; pitBox < epf.EntryListEnd; pitBox++ {
		entrantAsResult := split[splitIndex]

		e := NewEntrant()
		car, err := results.FindCarByGUIDAndModel(entrantAsResult.DriverGUID, entrantAsResult.CarModel)

		if err != nil {
			logrus.WithError(err).Warnf("could not find car for guid/model: %s/%s", entrantAsResult.DriverGUID, entrantAsResult.CarModel)
			continue
		}

		e.AssignFromResult(entrantAsResult, car)

		entryList.AddInPitBox(e, pitBox)

		if epf.ReverseEntrants {
			splitIndex--
		} else {
			splitIndex++
		}
	}

	return nil
}
