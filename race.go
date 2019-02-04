package servermanager

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var raceManager = &RaceManager{}

type RaceManager struct {
	currentRace *ServerConfig
}

func (rm *RaceManager) CurrentRace() *ServerConfig {
	if !AssettoProcess.IsRunning() {
		return nil
	}

	return rm.currentRace
}

func (rm *RaceManager) applyConfigAndStart(config ServerConfig, entryList EntryList) error {
	err := config.Write()

	if err != nil {
		return err
	}

	err = entryList.Write()

	if err != nil {
		return err
	}

	rm.currentRace = &config

	if AssettoProcess.IsRunning() {
		return AssettoProcess.Restart()
	}

	err = AssettoProcess.Start()

	if err != nil {
		return err
	}

	return nil
}

func (rm *RaceManager) SetupQuickRace(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	// load default config values
	quickRace := ConfigIniDefault

	cars := r.Form["Cars"]

	quickRace.CurrentRaceConfig.Cars = strings.Join(cars, ";")
	quickRace.CurrentRaceConfig.Track = r.Form.Get("Track")
	quickRace.CurrentRaceConfig.TrackLayout = r.Form.Get("TrackLayout")

	quickRace.CurrentRaceConfig.Sessions = make(map[SessionType]SessionConfig)

	qualifyingTime, err := strconv.ParseInt(r.Form.Get("Qualifying.Time"), 10, 0)

	if err != nil {
		return err
	}

	quickRace.AddSession(SessionTypeQualifying, SessionConfig{
		Name:   "Qualify",
		Time:   int(qualifyingTime),
		IsOpen: 1,
	})

	raceTime, err := strconv.ParseInt(r.Form.Get("Race.Time"), 10, 0)

	if err != nil {
		return err
	}

	raceLaps, err := strconv.ParseInt(r.Form.Get("Race.Laps"), 10, 0)

	if err != nil {
		return err
	}

	quickRace.AddSession(SessionTypeRace, SessionConfig{
		Name:     "Race",
		Time:     int(raceTime),
		Laps:     int(raceLaps),
		IsOpen:   1,
		WaitTime: 60,
	})

	if len(cars) == 0 {
		return errors.New("you must submit a car")
	}

	entryList := EntryList{}

	// @TODO this should work to the number of grid slots on the track rather than MaxClients.
	for i := 0; i < quickRace.GlobalServerConfig.MaxClients; i++ {
		entryList.Add(Entrant{
			Model: cars[i%len(cars)],
		})
	}

	return rm.applyConfigAndStart(quickRace, entryList)
}

func (rm *RaceManager) SetupCustomRace(r *http.Request) error {
	spew.Dump(r.Form)

	return nil // @TODO
}

// BuildRaceOpts builds a quick race form
func (rm *RaceManager) BuildRaceOpts() (map[string]interface{}, error) {
	cars, err := ListCars()

	if err != nil {
		return nil, err
	}

	tracks, err := ListTracks()

	if err != nil {
		return nil, err
	}

	var carNames, trackNames, trackLayouts []string

	for _, car := range cars {
		carNames = append(carNames, car.Name)
	}

	tyres, err := ListTyres()

	if err != nil {
		return nil, err
	}

	weather, err := ListWeather()

	if err != nil {
		return nil, err
	}

	// @TODO eventually this will be loaded from somewhere
	race := &ConfigIniDefault

	for _, track := range tracks {
		trackNames = append(trackNames, track.Name)

		for _, layout := range track.Layouts {
			trackLayouts = append(trackLayouts, fmt.Sprintf("%s:%s", track.Name, layout))
		}
	}

	for i, layout := range trackLayouts {
		if layout == fmt.Sprintf("%s:%s", race.CurrentRaceConfig.Track, race.CurrentRaceConfig.TrackLayout) {
			// mark the current track layout so the javascript can correctly set it up.
			trackLayouts[i] += ":current"
			break
		}
	}

	return map[string]interface{}{
		"CarOpts":           carNames,
		"TrackOpts":         trackNames,
		"TrackLayoutOpts":   trackLayouts,
		"MaxClients":        race.GlobalServerConfig.MaxClients,
		"AvailableSessions": AvailableSessions,
		"Tyres":             tyres,
		"Weather":           weather,
		"Current":           race.CurrentRaceConfig,
	}, nil
}

type CustomRace struct {
	Name    string
	Created time.Time
	Deleted time.Time

	ServerSetup CurrentRaceConfig
}

func (rm *RaceManager) ListCustomRaces() ([]CustomRace, error) {
	return nil, nil
}
