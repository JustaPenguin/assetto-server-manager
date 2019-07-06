package servermanager

import (
	"errors"
	"time"

	"github.com/google/uuid"
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

	Events []*RaceWeekendEvent
}

func NewRaceWeekend() *RaceWeekend {
	return &RaceWeekend{
		ID:      uuid.New(),
		Created: time.Now(),
	}
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

	InheritsID uuid.UUID

	RaceConfig CurrentRaceConfig
	EntryList  EntryList
}

func NewRaceWeekendEvent() *RaceWeekendEvent {
	return &RaceWeekendEvent{
		ID:      uuid.New(),
		Created: time.Now(),
	}
}

func (rwe *RaceWeekendEvent) IsBase() bool {
	return rwe.InheritsID == uuid.Nil
}
