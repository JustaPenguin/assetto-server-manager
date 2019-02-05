package servermanager

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/etcd-io/bbolt"
	"github.com/sirupsen/logrus"
)

var (
	raceManager *RaceManager

	ErrCustomRaceNotFound = errors.New("servermanager: custom race not found")
)

func init() {
	storeFileLocation := os.Getenv("STORE_LOCATION")

	bb, err := bbolt.Open(storeFileLocation, 0644, nil)

	if err != nil {
		logrus.Fatalf("could not open bbolt store at: '%s', err: %s", storeFileLocation, err)
	}

	raceManager = NewRaceManager(NewRaceStore(bb))
}

type RaceManager struct {
	currentRace *ServerConfig

	raceStore *RaceStore
}

func NewRaceManager(raceStore *RaceStore) *RaceManager {
	return &RaceManager{raceStore: raceStore}
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

	// save the custom race preset
	if err := rm.SaveCustomRace(raceConfig, entryList); err != nil {
		return err
	}

	return rm.applyConfigAndStart(completeConfig, entryList)
}

// BuildRaceOpts builds a quick race form
func (rm *RaceManager) BuildRaceOpts(r *http.Request) (map[string]interface{}, error) {
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

	templateID := r.URL.Query().Get("from")

	if templateID != "" {
		// load a from a custom race template
		customRace, err := rm.raceStore.FindCustomRaceByID(templateID)

		if err != nil {
			return nil, err
		}

		// @TODO loading entrylist
		race.CurrentRaceConfig = customRace.RaceConfig
	}

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
	Created time.Time
	Deleted time.Time
	UUID    uuid.UUID

	RaceConfig CurrentRaceConfig
	EntryList  EntryList
}

func (rm *RaceManager) ListCustomRaces() ([]CustomRace, error) {
	races, err := rm.raceStore.ListCustomRaces()

	if err == bbolt.ErrBucketNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	sort.Slice(races, func(i, j int) bool {
		return races[i].Created.After(races[j].Created)
	})

	return races, nil
}

func (rm *RaceManager) SaveCustomRace(config CurrentRaceConfig, entryList EntryList) error {
	return rm.raceStore.UpsertCustomRace(CustomRace{
		Created: time.Now(),
		UUID:    uuid.New(),

		RaceConfig: config,
		EntryList:  entryList,
	})
}

func (rm *RaceManager) StartCustomRace(uuid string) error {
	race, err := rm.raceStore.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	// @TODO eventually this will be loaded from somewhere
	cfg := ConfigIniDefault
	cfg.CurrentRaceConfig = race.RaceConfig

	return rm.applyConfigAndStart(cfg, race.EntryList)
}

func (rm *RaceManager) DeleteCustomRace(uuid string) error {
	race, err := rm.raceStore.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	return rm.raceStore.DeleteCustomRace(*race)
}
