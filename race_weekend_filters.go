package servermanager

// An EntryListFilter takes a given EntryList, and (based on some criteria) filters out invalid Entrants
type EntryListFilter interface {
	Name() string
	Description() string
	Key() string
	Filter(results []*SessionResult) ([]*SessionResult, error)
}

type FilterError string

func (f FilterError) Error() string {
	return string(f)
}

type EntrantPositionFilter struct {
	Start           int
	End             int
	ReverseEntrants bool
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

func (epf EntrantPositionFilter) Filter(results []*SessionResult) ([]*SessionResult, error) {
	results = filterDisqualifiedResults(results)

	if epf.Start < 0 || epf.Start >= len(results) || epf.End < 0 || epf.End >= len(results) {
		return nil, FilterError("Invalid bounds for Start or End")
	}

	split := results[epf.Start:epf.End]

	if epf.ReverseEntrants {
		var out []*SessionResult

		for i := len(split) - 1; i >= 0; i-- {
			out = append(out, split[i])
		}

		return out, nil
	} else {
		return split, nil
	}
}
