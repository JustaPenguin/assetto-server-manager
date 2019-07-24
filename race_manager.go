package servermanager

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/etcd-io/bbolt"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	ErrCustomRaceNotFound = errors.New("servermanager: custom race not found")
)

type RaceManager struct {
	process    ServerProcess
	raceStore  Store
	carManager *CarManager

	currentRace      *ServerConfig
	currentEntryList EntryList

	mutex sync.RWMutex

	// looped races
	loopedRaceSessionTypes      []SessionType
	loopedRaceWaitForSecondRace bool

	// scheduled races
	customRaceStartTimers map[string]*time.Timer
}

func NewRaceManager(
	raceStore Store,
	process ServerProcess,
	carManager *CarManager,
) *RaceManager {
	return &RaceManager{
		raceStore:  raceStore,
		process:    process,
		carManager: carManager,
	}
}

func (rm *RaceManager) CurrentRace() (*ServerConfig, EntryList) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	if !rm.process.IsRunning() {
		return nil, nil
	}

	return rm.currentRace, rm.currentEntryList
}

var ErrEntryListTooBig = errors.New("servermanager: EntryList exceeds MaxClients setting")

type RaceEvent interface {
	IsChampionship() bool
	OverrideServerPassword() bool
	ReplacementServerPassword() string
	EventName() string
	EventDescription() string
	GetURL() string
}

type normalEvent struct {
	OverridePassword    bool
	ReplacementPassword string
}

func (normalEvent) IsChampionship() bool {
	return false
}

func (normalEvent) EventName() string {
	return ""
}

func (n normalEvent) OverrideServerPassword() bool {
	return n.OverridePassword
}

func (n normalEvent) ReplacementServerPassword() string {
	return n.ReplacementPassword
}

func (n normalEvent) EventDescription() string {
	return ""
}

func (n normalEvent) GetURL() string {
	return ""
}

func (rm *RaceManager) applyConfigAndStart(config ServerConfig, entryList EntryList, loop bool, event RaceEvent) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// Reset the stored session types if this isn't a looped race
	if !loop {
		rm.clearLoopedRaceSessionTypes()
	}

	// load server opts
	serverOpts, err := rm.LoadServerOptions()

	if err != nil {
		return err
	}

	config.GlobalServerConfig = *serverOpts
	forwardingAddress := config.GlobalServerConfig.UDPPluginAddress
	forwardListenPort := config.GlobalServerConfig.UDPPluginLocalPort

	config.GlobalServerConfig.UDPPluginAddress = config.GlobalServerConfig.FreeUDPPluginAddress
	config.GlobalServerConfig.UDPPluginLocalPort = config.GlobalServerConfig.FreeUDPPluginLocalPort

	if MaxClientsOverride > 0 {
		config.CurrentRaceConfig.MaxClients = MaxClientsOverride

		if len(entryList) > MaxClientsOverride {
			return ErrEntryListTooBig
		}
	}

	// if password override turn the password off
	if event.OverrideServerPassword() {
		config.GlobalServerConfig.Password = event.ReplacementServerPassword()
	} else {
		config.GlobalServerConfig.Password = serverOpts.Password
	}

	if config.CurrentRaceConfig.HasSession(SessionTypeBooking) {
		config.CurrentRaceConfig.PickupModeEnabled = 0
	}

	// drs zones management
	err = ToggleDRSForTrack(config.CurrentRaceConfig.Track, config.CurrentRaceConfig.TrackLayout, !config.CurrentRaceConfig.DisableDRSZones)

	if err != nil {
		return err
	}

	if config.GlobalServerConfig.ShowRaceNameInServerLobby == 1 {
		// append the race name to the server name
		if name := event.EventName(); name != "" {
			config.GlobalServerConfig.Name += fmt.Sprintf(": %s", name)
		}
	}

	if config.GlobalServerConfig.EnableContentManagerWrapper == 1 && config.GlobalServerConfig.ContentManagerWrapperPort > 0 {
		config.GlobalServerConfig.Name += fmt.Sprintf(" %c%d", contentManagerWrapperSeparator, config.GlobalServerConfig.ContentManagerWrapperPort)
	}

	err = config.Write()

	if err != nil {
		return err
	}

	err = entryList.Write()

	if err != nil {
		return err
	}

	rm.currentRace = &config
	rm.currentEntryList = entryList

	if rm.process.IsRunning() {
		err := rm.process.Stop()

		if err != nil {
			return err
		}
	}

	err = rm.process.Start(config, entryList, forwardingAddress, forwardListenPort, event)

	if err != nil {
		return err
	}

	return nil
}

var ErrMustSubmitCar = errors.New("servermanager: you must set a car")

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

	if quickRace.CurrentRaceConfig.TrackLayout == defaultLayoutName {
		quickRace.CurrentRaceConfig.TrackLayout = ""
	}

	tyres, err := ListTyres()

	if err != nil {
		return err
	}

	quickRaceTyresMap := make(map[string]bool)

	for _, car := range cars {
		if available, ok := tyres[car]; ok {
			for tyre := range available {
				quickRaceTyresMap[tyre] = true
			}
		}
	}

	var quickRaceTyres []string

	for tyre := range quickRaceTyresMap {
		quickRaceTyres = append(quickRaceTyres, tyre)
	}

	quickRace.CurrentRaceConfig.LegalTyres = strings.Join(quickRaceTyres, ";")

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
		return ErrMustSubmitCar
	}

	entryList := EntryList{}

	var numPitboxes int

	trackInfo, err := GetTrackInfo(quickRace.CurrentRaceConfig.Track, quickRace.CurrentRaceConfig.TrackLayout)

	if err == nil {
		boxes, err := trackInfo.Pitboxes.Int64()

		if err != nil {
			numPitboxes = quickRace.CurrentRaceConfig.MaxClients
		} else {
			numPitboxes = int(boxes)
		}

	} else {
		numPitboxes = quickRace.CurrentRaceConfig.MaxClients
	}

	if numPitboxes > MaxClientsOverride && MaxClientsOverride > 0 {
		numPitboxes = MaxClientsOverride
	}

	allCars, err := rm.carManager.ListCars()

	if err != nil {
		return err
	}

	carMap := allCars.AsMap()

	for i := 0; i < numPitboxes; i++ {
		model := cars[i%len(cars)]

		var skin string

		if skins, ok := carMap[model]; ok && len(skins) > 0 {
			skin = carMap[model][rand.Intn(len(carMap[model]))]
		}

		e := NewEntrant()
		e.PitBox = i
		e.Model = model
		e.Skin = skin

		entryList.Add(e)
	}

	quickRace.CurrentRaceConfig.MaxClients = numPitboxes

	return rm.applyConfigAndStart(quickRace, entryList, false, normalEvent{})
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

func formValueAsFloat(val string) float64 {
	i, err := strconv.ParseFloat(val, 0)

	if err != nil {
		return 0
	}

	return i
}

func (rm *RaceManager) BuildEntryList(r *http.Request, start, length int) (EntryList, error) {
	entryList := EntryList{}

	allCars, err := rm.carManager.ListCars()

	if err != nil {
		return nil, err
	}

	carMap := allCars.AsMap()

	for i := start; i < start+length; i++ {
		model := r.Form["EntryList.Car"][i]
		skin := r.Form["EntryList.Skin"][i]

		if skin == "random_skin" {
			if skins, ok := carMap[model]; ok && len(skins) > 0 {
				skin = carMap[model][rand.Intn(len(carMap[model]))]
			}
		}

		e := NewEntrant()

		if r.Form["EntryList.InternalUUID"][i] != "" || r.Form["EntryList.InternalUUID"][i] != uuid.Nil.String() {
			internalUUID, err := uuid.Parse(r.Form["EntryList.InternalUUID"][i])

			if err == nil {
				e.InternalUUID = internalUUID
			}
		}

		e.Name = r.Form["EntryList.Name"][i]
		e.Team = r.Form["EntryList.Team"][i]
		e.GUID = r.Form["EntryList.GUID"][i]
		e.Model = model
		e.Skin = skin
		// Despite having the option for SpectatorMode, the server does not support it, and panics if set to 1
		// SpectatorMode: formValueAsInt(r.Form["EntryList.Spectator"][i]),
		e.Ballast = formValueAsInt(r.Form["EntryList.Ballast"][i])
		e.Restrictor = formValueAsInt(r.Form["EntryList.Restrictor"][i])
		e.FixedSetup = r.Form["EntryList.FixedSetup"][i]

		// The pit box/grid starting position
		if entrantIDs, ok := r.Form["EntryList.EntrantID"]; ok && i < len(entrantIDs) {
			e.PitBox = formValueAsInt(entrantIDs[i])
		} else {
			e.PitBox = i
		}

		if r.Form["EntryList.TransferTeamPoints"] != nil && i < len(r.Form["EntryList.TransferTeamPoints"]) && formValueAsInt(r.Form["EntryList.TransferTeamPoints"][i]) == 1 {
			e.TransferTeamPoints = true
		}

		if r.Form["EntryList.OverwriteAllEvents"] != nil && i < len(r.Form["EntryList.OverwriteAllEvents"]) && formValueAsInt(r.Form["EntryList.OverwriteAllEvents"][i]) == 1 {
			e.OverwriteAllEvents = true
		}

		entryList.Add(e)
	}

	return entryList, nil
}

func (rm *RaceManager) BuildCustomRaceFromForm(r *http.Request) (*CurrentRaceConfig, error) {
	cars := r.Form["Cars"]
	isSol := r.FormValue("Sol.Enabled") == "1"

	gasPenaltyDisabled := formValueAsInt(r.FormValue("RaceGasPenaltyDisabled"))
	lockedEntryList := formValueAsInt(r.FormValue("LockedEntryList"))

	if gasPenaltyDisabled == 0 {
		gasPenaltyDisabled = 1
	} else {
		gasPenaltyDisabled = 0
	}

	if lockedEntryList == 0 {
		lockedEntryList = 1
	} else {
		lockedEntryList = 0
	}

	trackLayout := r.FormValue("TrackLayout")

	if trackLayout == "<default>" {
		trackLayout = ""
	}

	raceConfig := &CurrentRaceConfig{
		// general race config
		Cars:        strings.Join(cars, ";"),
		Track:       r.FormValue("Track"),
		TrackLayout: trackLayout,

		// assists
		ABSAllowed:              formValueAsInt(r.FormValue("ABSAllowed")),
		TractionControlAllowed:  formValueAsInt(r.FormValue("TractionControlAllowed")),
		StabilityControlAllowed: formValueAsInt(r.FormValue("StabilityControlAllowed")),
		AutoClutchAllowed:       formValueAsInt(r.FormValue("AutoClutchAllowed")),
		TyreBlanketsAllowed:     formValueAsInt(r.FormValue("TyreBlanketsAllowed")),

		// weather
		IsSol:                  formValueAsInt(r.FormValue("Sol.Enabled")),
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
		PickupModeEnabled:         lockedEntryList,
		RacePitWindowStart:        formValueAsInt(r.FormValue("RacePitWindowStart")),
		RacePitWindowEnd:          formValueAsInt(r.FormValue("RacePitWindowEnd")),
		ReversedGridRacePositions: formValueAsInt(r.FormValue("ReversedGridRacePositions")),
		QualifyMaxWaitPercentage:  formValueAsInt(r.FormValue("QualifyMaxWaitPercentage")),
		RaceGasPenaltyDisabled:    gasPenaltyDisabled,
		MaxBallastKilograms:       formValueAsInt(r.FormValue("MaxBallastKilograms")),
		AllowedTyresOut:           formValueAsInt(r.FormValue("AllowedTyresOut")),
		LoopMode:                  formValueAsInt(r.FormValue("LoopMode")),
		SleepTime:                 formValueAsInt(r.FormValue("SleepTime")),
		RaceOverTime:              formValueAsInt(r.FormValue("RaceOverTime")),
		StartRule:                 formValueAsInt(r.FormValue("StartRule")),
		MaxClients:                formValueAsInt(r.FormValue("MaxClients")),
		RaceExtraLap:              formValueAsInt(r.FormValue("RaceExtraLap")),
		MaxContactsPerKilometer:   formValueAsInt(r.FormValue("MaxContactsPerKilometer")),
		ResultScreenTime:          formValueAsInt(r.FormValue("ResultScreenTime")),
		DisableDRSZones:           formValueAsInt(r.FormValue("DisableDRSZones")) == 1,
	}

	if isSol {
		raceConfig.SunAngle = 0
		raceConfig.TimeOfDayMultiplier = 0
	}

	for _, session := range AvailableSessions {
		sessName := session.String()

		if r.FormValue(sessName+".Enabled") != "1" {
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
		weatherName := r.Form["Graphics"][i]

		WFXType, err := getWeatherType(weatherName)

		// if WFXType can't be found due to an error, default to non-sol weather.
		if !isSol || err != nil {
			raceConfig.AddWeather(&WeatherConfig{
				Graphics:               weatherName,
				BaseTemperatureAmbient: formValueAsInt(r.Form["BaseTemperatureAmbient"][i]),
				BaseTemperatureRoad:    formValueAsInt(r.Form["BaseTemperatureRoad"][i]),
				VariationAmbient:       formValueAsInt(r.Form["VariationAmbient"][i]),
				VariationRoad:          formValueAsInt(r.Form["VariationRoad"][i]),
			})
		} else {
			startTime, err := time.ParseInLocation("2006-01-02T15:04", r.Form["DateUnix"][i], time.UTC)

			if err != nil {
				return nil, err
			}

			startTimeZoned := startTime.In(time.FixedZone("UTC+10", 10*60*60))
			timeMulti := r.Form["TimeMulti"][i]
			timeMultiInt := formValueAsInt(timeMulti)

			// This is probably a bit hacky, and may need removing with a future Sol update
			startTimeFinal := startTimeZoned.Add(-(time.Duration(timeMultiInt) * 5 * time.Hour))

			raceConfig.AddWeather(&WeatherConfig{
				Graphics: weatherName + "_type=" + strconv.Itoa(WFXType) + "_time=0_mult=" +
					timeMulti + "_start=" + strconv.Itoa(int(startTimeFinal.Unix())),
				BaseTemperatureAmbient: formValueAsInt(r.Form["BaseTemperatureAmbient"][i]),
				BaseTemperatureRoad:    formValueAsInt(r.Form["BaseTemperatureRoad"][i]),
				VariationAmbient:       formValueAsInt(r.Form["VariationAmbient"][i]),
				VariationRoad:          formValueAsInt(r.Form["VariationRoad"][i]),

				CMGraphics:          weatherName,
				CMWFXType:           WFXType,
				CMWFXUseCustomTime:  1,
				CMWFXTime:           0,
				CMWFXTimeMulti:      timeMultiInt,
				CMWFXUseCustomDate:  1,
				CMWFXDate:           int(startTimeFinal.Unix()),
				CMWFXDateUnModified: int(startTime.Unix()),
			})
		}
	}

	return raceConfig, nil
}

func (rm *RaceManager) SetupCustomRace(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	raceConfig, err := rm.BuildCustomRaceFromForm(r)

	if err != nil {
		return err
	}

	var entryList EntryList

	if !raceConfig.HasSession(SessionTypeBooking) {
		entryList, err = rm.BuildEntryList(r, 0, len(r.Form["EntryList.Name"]))

		if err != nil {
			return err
		}
	}

	completeConfig := ConfigIniDefault
	completeConfig.CurrentRaceConfig = *raceConfig

	overridePassword := r.FormValue("OverridePassword") == "1"
	replacementPassword := r.FormValue("ReplacementPassword")

	if customRaceID := r.FormValue("Editing"); customRaceID != "" {
		// we are editing the race. load the previous one and overwrite it with this one
		customRace, err := rm.raceStore.FindCustomRaceByID(customRaceID)

		if err != nil {
			return err
		}

		customRace.OverridePassword = overridePassword
		customRace.ReplacementPassword = replacementPassword

		customRace.Name = r.FormValue("CustomRaceName")
		customRace.EntryList = entryList
		customRace.RaceConfig = *raceConfig

		return rm.raceStore.UpsertCustomRace(customRace)
	} else {
		saveAsPresetWithoutStartingRace := r.FormValue("action") == "justSave"
		schedule := r.FormValue("action") == "schedule"

		// save the custom race preset
		race, err := rm.SaveCustomRace(r.FormValue("CustomRaceName"), overridePassword, replacementPassword, *raceConfig, entryList, saveAsPresetWithoutStartingRace)

		if err != nil {
			return err
		}

		if schedule {
			dateString := r.FormValue("CustomRaceScheduled")
			timeString := r.FormValue("CustomRaceScheduledTime")
			timezone := r.FormValue("CustomRaceScheduledTimezone")

			location, err := time.LoadLocation(timezone)

			if err != nil {
				logrus.WithError(err).Errorf("could not find location: %s", location)
				location = time.Local
			}

			// Parse time in correct time zone
			date, err := time.ParseInLocation("2006-01-02-15:04", dateString+"-"+timeString, location)

			if err != nil {
				return err
			}

			err = rm.ScheduleRace(race.UUID.String(), date, "add")

			if err != nil {
				return err
			}

			return nil
		}

		if saveAsPresetWithoutStartingRace {
			return nil
		}

		return rm.applyConfigAndStart(completeConfig, entryList, false, race)
	}
}

// applyCurrentRaceSetupToOptions takes current values in race which require more detailed configuration
// and applies them to the template opts.
func (rm *RaceManager) applyCurrentRaceSetupToOptions(opts map[string]interface{}, race CurrentRaceConfig) error {
	tyres, err := ListTyres()

	if err != nil {
		return err
	}

	deselectedTyres := make(map[string]bool)

	for _, car := range varSplit(race.Cars) {
		tyresForCar, ok := tyres[car]

		if !ok {
			continue
		}

		for carTyre := range tyresForCar {
			found := false

			for _, t := range varSplit(race.LegalTyres) {
				if carTyre == t {
					found = true
					break
				}
			}

			if !found {
				deselectedTyres[carTyre] = true
			}
		}
	}

	opts["Tyres"] = tyres
	opts["DeselectedTyres"] = deselectedTyres

	return nil
}

func (rm *RaceManager) ListAutoFillEntrants() ([]*Entrant, error) {
	entrants, err := rm.raceStore.ListEntrants()

	if err != nil {
		return nil, err
	}

	sort.Slice(entrants, func(i, j int) bool {
		return entrants[i].Name < entrants[j].Name
	})

	return entrants, nil
}

// BuildRaceOpts builds a quick race form
func (rm *RaceManager) BuildRaceOpts(r *http.Request) (map[string]interface{}, error) {
	_, cars, err := rm.carManager.Search(r.Context(), "", 0, 100000)

	if err != nil {
		return nil, err
	}

	tracks, err := ListTracks()

	if err != nil {
		return nil, err
	}

	weather, err := ListWeather()

	if err != nil {
		return nil, err
	}

	race := ConfigIniDefault

	templateID := r.URL.Query().Get("from")

	var entrants EntryList

	if templateID != "" {
		// load a from a custom race template
		customRace, err := rm.raceStore.FindCustomRaceByID(templateID)

		if err != nil {
			return nil, err
		}

		race.CurrentRaceConfig = customRace.RaceConfig
		entrants = customRace.EntryList
	}

	templateIDForEditing := chi.URLParam(r, "uuid")
	isEditing := templateIDForEditing != ""
	var customRaceName, replacementPassword string
	var overridePassword bool

	if isEditing {
		customRace, err := rm.raceStore.FindCustomRaceByID(templateIDForEditing)

		if err != nil {
			return nil, err
		}

		customRaceName = customRace.Name
		race.CurrentRaceConfig = customRace.RaceConfig
		entrants = customRace.EntryList
		overridePassword = customRace.OverrideServerPassword()
		replacementPassword = customRace.ReplacementServerPassword()
	}

	possibleEntrants, err := rm.ListAutoFillEntrants()

	if err != nil {
		return nil, err
	}

	fixedSetups, err := ListAllSetups()

	if err != nil {
		return nil, err
	}

	solIsInstalled := false

	for availableWeather := range weather {
		if strings.HasPrefix(availableWeather, "sol_") {
			solIsInstalled = true
			break
		}
	}

	// default sol time to now
	for _, weather := range race.CurrentRaceConfig.Weather {
		if weather.CMWFXDate == 0 {
			weather.CMWFXDate = int(time.Now().Unix())
			weather.CMWFXDateUnModified = int(time.Now().Unix())
		}
	}

	opts := map[string]interface{}{
		"CarOpts":             cars,
		"TrackOpts":           tracks,
		"AvailableSessions":   AvailableSessions,
		"Weather":             weather,
		"SolIsInstalled":      solIsInstalled,
		"Current":             race.CurrentRaceConfig,
		"CurrentEntrants":     entrants,
		"PossibleEntrants":    possibleEntrants,
		"FixedSetups":         fixedSetups,
		"IsChampionship":      false, // this flag is overridden by championship setup
		"IsEditing":           isEditing,
		"EditingID":           templateIDForEditing,
		"CustomRaceName":      customRaceName,
		"SurfacePresets":      DefaultTrackSurfacePresets,
		"OverridePassword":    overridePassword,
		"ReplacementPassword": replacementPassword,
	}

	err = rm.applyCurrentRaceSetupToOptions(opts, race.CurrentRaceConfig)

	if err != nil {
		return nil, err
	}

	return opts, nil
}

const maxRecentRaces = 30

func (rm *RaceManager) ListCustomRaces() (recent, starred, looped, scheduled []*CustomRace, err error) {
	recent, err = rm.raceStore.ListCustomRaces()

	if err == bbolt.ErrBucketNotFound {
		return nil, nil, nil, nil, nil
	} else if err != nil {
		return nil, nil, nil, nil, err
	}

	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Created.After(recent[j].Created)
	})

	var filteredRecent []*CustomRace

	for _, race := range recent {
		if race.Loop {
			looped = append(looped, race)
		}

		if race.Starred {
			starred = append(starred, race)
		}

		if race.Scheduled.After(time.Now()) {
			scheduled = append(scheduled, race)
		}

		if !race.Starred && !race.Loop && !race.Scheduled.After(time.Now()) {
			filteredRecent = append(filteredRecent, race)
		}
	}

	if len(filteredRecent) > maxRecentRaces {
		filteredRecent = filteredRecent[:maxRecentRaces]
	}

	return filteredRecent, starred, looped, scheduled, nil
}

func (rm *RaceManager) SaveEntrantsForAutoFill(entryList EntryList) error {
	for _, entrant := range entryList {
		if entrant.Name == "" {
			continue // only save entrants that have a name
		}

		err := rm.raceStore.UpsertEntrant(*entrant)

		if err != nil {
			return err
		}
	}

	return nil
}

func (rm *RaceManager) SaveCustomRace(name string, overridePassword bool, replacementPassword string,
	config CurrentRaceConfig, entryList EntryList, starred bool) (*CustomRace, error) {

	hasCustomRaceName := true

	if name == "" {
		var trackLayout string

		if config.TrackLayout != "" {
			trackLayout = prettifyName(config.TrackLayout, true)
		}

		name = fmt.Sprintf("%s (%s) in %s (%d entrants)",
			prettifyName(config.Track, false),
			trackLayout,
			carList(config.Cars),
			len(entryList),
		)

		hasCustomRaceName = false
	}

	if err := rm.SaveEntrantsForAutoFill(entryList); err != nil {
		return nil, err
	}

	race := &CustomRace{
		Name:                name,
		HasCustomName:       hasCustomRaceName,
		OverridePassword:    overridePassword,
		ReplacementPassword: replacementPassword,
		Created:             time.Now(),
		UUID:                uuid.New(),
		Starred:             starred,

		RaceConfig: config,
		EntryList:  entryList,
	}

	err := rm.raceStore.UpsertCustomRace(race)

	if err != nil {
		return nil, err
	}

	return race, nil
}

func (rm *RaceManager) StartCustomRace(uuid string, forceRestart bool) error {
	race, err := rm.raceStore.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	cfg := ConfigIniDefault
	cfg.CurrentRaceConfig = race.RaceConfig

	// Required for our nice auto loop stuff
	if forceRestart {
		cfg.CurrentRaceConfig.LoopMode = 1
	}

	return rm.applyConfigAndStart(cfg, race.EntryList, forceRestart, race)
}

func (rm *RaceManager) ScheduleRace(uuid string, date time.Time, action string) error {
	race, err := rm.raceStore.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	race.Scheduled = date

	// if there is an existing schedule timer for this event stop it
	if timer := rm.customRaceStartTimers[race.UUID.String()]; timer != nil {
		timer.Stop()
	}

	if action == "add" {
		// add a scheduled event on date
		duration := time.Until(date)

		race.Scheduled = date
		rm.customRaceStartTimers[race.UUID.String()] = time.AfterFunc(duration, func() {
			err := rm.StartCustomRace(race.UUID.String(), false)

			if err != nil {
				logrus.Errorf("couldn't start scheduled race, err: %s", err)
			}
		})
	}

	return rm.raceStore.UpsertCustomRace(race)
}

func (rm *RaceManager) DeleteCustomRace(uuid string) error {
	race, err := rm.raceStore.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	return rm.raceStore.DeleteCustomRace(race)
}

func (rm *RaceManager) ToggleStarCustomRace(uuid string) error {
	race, err := rm.raceStore.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	race.Starred = !race.Starred

	return rm.raceStore.UpsertCustomRace(race)
}

func (rm *RaceManager) ToggleLoopCustomRace(uuid string) error {
	race, err := rm.raceStore.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	race.Loop = !race.Loop

	return rm.raceStore.UpsertCustomRace(race)
}

func (rm *RaceManager) SaveServerOptions(so *GlobalServerConfig) error {
	return rm.raceStore.UpsertServerOptions(so)
}

func (rm *RaceManager) LoadServerOptions() (*GlobalServerConfig, error) {
	serverOpts, err := rm.raceStore.LoadServerOptions()

	if err != nil {
		return nil, err
	}

	udpListenPort, udpSendPort := 0, 0

	for udpListenPort == udpSendPort {
		udpListenPort, err = FreeUDPPort()

		if err != nil {
			return nil, err
		}

		udpSendPort, err = FreeUDPPort()

		if err != nil {
			return nil, err
		}
	}

	serverOpts.FreeUDPPluginAddress = fmt.Sprintf("127.0.0.1:%d", udpSendPort)
	serverOpts.FreeUDPPluginLocalPort = udpListenPort

	return serverOpts, nil
}

func (rm *RaceManager) LoopRaces() {
	var i int
	ticker := time.NewTicker(30 * time.Second)

	for range ticker.C {
		currentRace, _ := rm.CurrentRace()

		if currentRace != nil {
			continue
		}

		_, _, looped, _, err := rm.ListCustomRaces()

		if err != nil {
			logrus.Errorf("couldn't list custom races, err: %s", err)
			return
		}

		if looped != nil {
			if i >= len(looped) {
				i = 0
			}

			// Reset the stored session types
			rm.loopedRaceSessionTypes = []SessionType{}

			for sessionID := range looped[i].RaceConfig.Sessions {
				rm.loopedRaceSessionTypes = append(rm.loopedRaceSessionTypes, sessionID)
			}

			if looped[i].RaceConfig.ReversedGridRacePositions != 0 {
				rm.loopedRaceSessionTypes = append(rm.loopedRaceSessionTypes, SessionTypeSecondRace)
			}

			err := rm.StartCustomRace(looped[i].UUID.String(), true)

			if err != nil {
				logrus.Errorf("couldn't start auto loop custom race, err: %s", err)
				return
			}

			i++
		}
	}
}

// callback check for udp end session, load result file, check session type against sessionTypes
// if session matches last session in sessionTypes then stop server and clear sessionTypes
func (rm *RaceManager) LoopCallback(message udp.Message) {
	switch a := message.(type) {
	case udp.EndSession:
		if rm.loopedRaceSessionTypes == nil {
			logrus.Infof("Session types == nil. ignoring end session callback")
			return
		}

		filename := filepath.Base(string(a))

		results, err := LoadResult(filename)

		if err != nil {
			logrus.Errorf("Could not read session results for %s, err: %s", filename, err)
			return
		}

		var endSession SessionType

		// If this is a race, and there is a second race configured
		// then wait for the second race to happen.
		if results.Type == string(SessionTypeRace) {
			for _, session := range rm.loopedRaceSessionTypes {
				if session == SessionTypeSecondRace {
					if !rm.loopedRaceWaitForSecondRace {
						rm.loopedRaceWaitForSecondRace = true
						return
					} else {
						rm.loopedRaceWaitForSecondRace = false
					}
				}
			}
		}

		for _, session := range rm.loopedRaceSessionTypes {
			if session == SessionTypeRace {
				endSession = SessionTypeRace
				break
			} else if session == SessionTypeQualifying {
				endSession = SessionTypeQualifying
			} else if session == SessionTypePractice && endSession != SessionTypeQualifying {
				endSession = SessionTypePractice
			} else if session == SessionTypeBooking && (endSession != SessionTypeQualifying && endSession != SessionTypePractice) {
				endSession = SessionTypeBooking
			}
		}

		logrus.Infof("results type: %s, endSession: %s", results.Type, string(endSession))

		if results.Type == string(endSession) {
			logrus.Infof("Event end detected, stopping looped session.")

			rm.clearLoopedRaceSessionTypes()

			err := rm.process.Stop()

			if err != nil {
				logrus.Errorf("Could not stop server, err: %s", err)
				return
			}
		}
	}
}

func (rm *RaceManager) clearLoopedRaceSessionTypes() {
	rm.loopedRaceSessionTypes = nil
}

func (rm *RaceManager) InitScheduledRaces() error {
	rm.customRaceStartTimers = make(map[string]*time.Timer)

	races, err := rm.raceStore.ListCustomRaces()

	if err != nil {
		return err
	}

	for _, race := range races {
		race := race

		if race.Scheduled.After(time.Now()) {
			// add a scheduled event on date
			duration := time.Until(race.Scheduled)

			rm.customRaceStartTimers[race.UUID.String()] = time.AfterFunc(duration, func() {
				err := rm.StartCustomRace(race.UUID.String(), false)

				if err != nil {
					logrus.Errorf("couldn't start scheduled race, err: %s", err)
				}
			})

			err := rm.raceStore.UpsertCustomRace(race)

			if err != nil {
				return err
			}
		} else {
			emptyTime := time.Time{}
			if race.Scheduled != emptyTime {
				logrus.Infof("Looks like the server was offline whilst a scheduled event was meant to start!"+
					" Start time: %s. The schedule has been cleared. Start the event manually if you wish to run it.", race.Scheduled.String())

				race.Scheduled = emptyTime

				err := rm.raceStore.UpsertCustomRace(race)

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
