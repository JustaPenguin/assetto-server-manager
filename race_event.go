package servermanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type RaceEvent interface {
	GetRaceConfig() CurrentRaceConfig
	GetEntryList() EntryList

	IsLooping() bool
	IsChampionship() bool
	IsRaceWeekend() bool
	IsPractice() bool
	IsTimeAttack() bool

	OverrideServerPassword() bool
	ReplacementServerPassword() string

	GetForceStopTime() time.Duration
	GetForceStopWithDrivers() bool

	EventName() string
	EventDescription() string
	GetURL() string
}

type marshalledRaceEvent struct {
	Name      string
	RaceEvent RaceEvent
}

func nameOf(s interface{}) string {
	return fmt.Sprintf("%T", s)
}

func getRaceEvent(name string) (RaceEvent, error) {
	var r RaceEvent

	switch name {
	case nameOf(&QuickRace{}):
		r = &QuickRace{}
	case nameOf(&CustomRace{}):
		r = &CustomRace{}
	case nameOf(&ActiveChampionship{}):
		r = &ActiveChampionship{}
	case nameOf(&ActiveRaceWeekend{}):
		r = &ActiveRaceWeekend{}
	default:
		return nil, fmt.Errorf("servermanager: unknown race event tyre specified: %s", name)
	}

	return r, nil
}

func marshalRaceEvent(r RaceEvent) ([]byte, error) {
	return json.Marshal(marshalledRaceEvent{
		Name:      nameOf(r),
		RaceEvent: r,
	})
}

func unmarshalRaceEvent(data []byte) (RaceEvent, error) {
	var rawData map[string]json.RawMessage

	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, err
	}

	nameData, ok := rawData["Name"]

	if !ok {
		return nil, errors.New("servermanager: race event name not found")
	}

	var name string

	if err := json.Unmarshal(nameData, &name); err != nil {
		return nil, err
	}

	raceEventData, ok := rawData["RaceEvent"]

	if !ok {
		return nil, errors.New("servermanager: race event name not found")
	}

	raceEvent, err := getRaceEvent(name)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(raceEventData, &raceEvent); err != nil {
		return nil, err
	}

	return raceEvent, err
}

func describeRaceEvent(raceEvent RaceEvent) string {
	cfg := raceEvent.GetRaceConfig()

	return fmt.Sprintf("%d sessions with %d entrants at %s (%s)", len(cfg.Sessions), len(raceEvent.GetEntryList()), cfg.Track, cfg.TrackLayout)
}
