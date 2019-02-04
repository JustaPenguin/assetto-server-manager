package servermanager

import (
	"errors"
	"fmt"
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

	quickRace.CurrentRaceConfig.AddSession(SessionTypeQualifying, SessionConfig{
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

	quickRace.CurrentRaceConfig.AddSession(SessionTypeRace, SessionConfig{
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

func formValueAsInt(val string) int {
	if val == "on" {
		return 1
	}

	i, err := strconv.ParseInt(val, 10, 0)

	if err != nil {
		return 0
	}

	return int(i)
}

func (rm *RaceManager) SetupCustomRace(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	cars := r.Form["Cars"]

	raceConfig := CurrentRaceConfig{
		// general race config
		Cars:        strings.Join(cars, ";"),
		Track:       r.FormValue("Track"),
		TrackLayout: r.FormValue("TrackLayout"),

		// assists
		ABSAllowed:              formValueAsInt(r.FormValue("ABSAllowed")),
		TractionControlAllowed:  formValueAsInt(r.FormValue("TractionControlAllowed")),
		StabilityControlAllowed: formValueAsInt(r.FormValue("StabilityControlAllowed")),
		AutoClutchAllowed:       formValueAsInt(r.FormValue("AutoClutchAllowed")),
		TyreBlanketsAllowed:     formValueAsInt(r.FormValue("TyreBlanketsAllowed")),

		// weather
		SunAngle:               formValueAsInt(r.FormValue("SunAngle")),
		WindBaseSpeedMin:       formValueAsInt(r.FormValue("WindBaseSpeedMin")),
		WindBaseSpeedMax:       formValueAsInt(r.FormValue("WindBaseSpeedMax")),
		WindBaseDirection:      formValueAsInt(r.FormValue("WindBaseDirection")),
		WindVariationDirection: formValueAsInt(r.FormValue("WindVariationDirection")),

		// @TODO specific weather setups

		// realism
		LegalTyres:          strings.Join(r.Form["LegalTyres"], ";"),
		FuelRate:            formValueAsInt(r.FormValue("FuelRate")),
		DamageMultiplier:    formValueAsInt(r.FormValue("DamageMultiplier")),
		TyreWearRate:        formValueAsInt(r.FormValue("TyreWearRate")),
		ForceVirtualMirror:  formValueAsInt(r.FormValue("ForceVirtualMirror")),
		TimeOfDayMultiplier: formValueAsInt(r.FormValue("TimeOfDayMultiplier")),

		DynamicTrack: DynamicTrackConfig{
			SessionStart:    formValueAsInt(r.FormValue("SessionStart")),
			Randomness:      formValueAsInt(r.FormValue("Randomness")),
			SessionTransfer: formValueAsInt(r.FormValue("SessionTransfer")),
			LapGain:         formValueAsInt(r.FormValue("LapGain")),
		},

		// rules
		LockedEntryList:           formValueAsInt(r.FormValue("LockedEntryList")),
		RacePitWindowStart:        formValueAsInt(r.FormValue("RacePitWindowStart")),
		RacePitWindowEnd:          formValueAsInt(r.FormValue("RacePitWindowEnd")),
		ReversedGridRacePositions: formValueAsInt(r.FormValue("ReversedGridRacePositions")),
		QualifyMaxWaitPercentage:  formValueAsInt(r.FormValue("QualifyMaxWaitPercentage")),
		RaceGasPenaltyDisabled:    formValueAsInt(r.FormValue("RaceGasPenaltyDisabled")),
		MaxBallastKilograms:       formValueAsInt(r.FormValue("MaxBallastKilograms")),
		AllowedTyresOut:           formValueAsInt(r.FormValue("AllowedTyresOut")),
		PickupModeEnabled:         formValueAsInt(r.FormValue("PickupModeEnabled")),
		LoopMode:                  formValueAsInt(r.FormValue("LoopMode")),
		SleepTime:                 formValueAsInt(r.FormValue("SleepTime")),
		RaceOverTime:              formValueAsInt(r.FormValue("RaceOverTime")),
		StartRule:                 formValueAsInt(r.FormValue("StartRule")),
	}

	for _, session := range AvailableSessions {
		sessName := session.String()

		if r.FormValue(sessName+".Enabled") != "on" {
			continue
		}

		raceConfig.AddSession(session, SessionConfig{
			Name:     r.FormValue(sessName + ".Name"),
			Time:     formValueAsInt(r.FormValue(sessName + ".Time")),
			Laps:     formValueAsInt(r.FormValue(sessName + ".Laps")),
			IsOpen:   formValueAsInt(r.FormValue(sessName + ".IsOpen")),
			WaitTime: formValueAsInt(r.FormValue(sessName + ".WaitTime")),
		})
	}

	// weather
	for i := 0; i < len(r.Form["Graphics"]); i++ {
		raceConfig.AddWeather(WeatherConfig{
			Graphics:               r.Form["Graphics"][i],
			BaseTemperatureAmbient: formValueAsInt(r.Form["BaseTemperatureAmbient"][i]),
			BaseTemperatureRoad:    formValueAsInt(r.Form["BaseTemperatureRoad"][i]),
			VariationAmbient:       formValueAsInt(r.Form["VariationAmbient"][i]),
			VariationRoad:          formValueAsInt(r.Form["VariationRoad"][i]),
		})
	}

	completeConfig := ConfigIniDefault
	completeConfig.CurrentRaceConfig = raceConfig

	entryList := EntryList{}

	// @TODO custom race needs an actual entry list.
	for i := 0; i < completeConfig.GlobalServerConfig.MaxClients; i++ {
		entryList.Add(Entrant{
			Model: cars[i%len(cars)],
		})
	}

	return rm.applyConfigAndStart(completeConfig, entryList)
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
