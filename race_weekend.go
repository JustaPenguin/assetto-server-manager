package servermanager

import (
	"html/template"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// RaceWeekends are a collection of events, where one event influences the EntryList of the next.
// 'Base' events are configured much like Custom Races (but they only have one session!),
// Inherited Events are also like Custom Races, but their EntryList is just an ordered set
// of finishing positions from the Event that is inherited.
type RaceWeekend struct {
	ID      uuid.UUID
	Created time.Time
	Updated time.Time
	Deleted time.Time

	Name string

	Entrants EntryList
	Events   []*RaceWeekendEvent
}

func NewRaceWeekend() *RaceWeekend {
	return &RaceWeekend{
		ID:      uuid.New(),
		Created: time.Now(),
	}
}

func (rw *RaceWeekend) AddEvent() {

}

var (
	ErrRaceWeekendNotFound      = errors.New("servermanager: race weekend not found")
	ErrRaceWeekendEventNotFound = errors.New("servermanager: race weekend event not found")
)

func (rw *RaceWeekend) FindEventByID(id string) (*RaceWeekendEvent, error) {
	for _, event := range rw.Events {
		if event.ID.String() == id {
			return event, nil
		}
	}

	return nil, ErrRaceWeekendEventNotFound
}

type RaceWeekendEvent struct {
	ID      uuid.UUID
	Created time.Time
	Updated time.Time
	Deleted time.Time

	InheritsIDs []uuid.UUID

	Filters []EntryListFilter

	RaceConfig CurrentRaceConfig
	Results    *SessionResults
}

func NewRaceWeekendEvent() *RaceWeekendEvent {
	return &RaceWeekendEvent{
		ID:      uuid.New(),
		Created: time.Now(),
	}
}

func (rwe *RaceWeekendEvent) IsBase() bool {
	return rwe.InheritsIDs == nil
}

type RaceWeekendEntrant struct {
	*Entrant

	PreviousSessionResults *SessionResult
	PreviousSessionCar     *SessionCar
}

type RaceWeekendEntryList map[string]*RaceWeekendEntrant

var ErrRaceWeekendEventDependencyIncomplete = errors.New("servermanager: race weekend event dependency incomplete")

func (rwe *RaceWeekendEvent) GetEntryList(rw *RaceWeekend) (EntryList, error) {
	var entryList EntryList

	if rwe.IsBase() {
		entryList = rw.Entrants
	} else {
		entryList = make(EntryList)

		for _, inheritedID := range rwe.InheritsIDs {
			// find previous event
			previousEvent, err := rw.FindEventByID(inheritedID.String())

			if err != nil {
				return nil, err
			}

			if previousEvent.Results == nil {
				return nil, ErrRaceWeekendEventDependencyIncomplete
			}

			for pos, result := range previousEvent.Results.Result {
				e := NewEntrant()

				car, err := previousEvent.Results.FindCarByGUID(result.DriverGUID)

				if err != nil {
					return nil, err
				}

				e.AssignFromResult(result, car)
				e.PitBox = pos

				entryList.Add(e)
			}
		}
	}

	for _, filter := range rwe.Filters {
		err := filter.Filter(entryList)

		if err != nil {
			return nil, errors.Wrapf(err, "could not apply filter: %s", filter.Name())
		}
	}

	return entryList, nil
}

// An EntryListFilter takes a given EntryList, and (based on some criteria) filters out invalid Entrants
type EntryListFilter interface {
	Name() string
	Filter(e EntryList) error
	Render() *template.HTML
}
