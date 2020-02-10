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

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"
	"github.com/JustaPenguin/assetto-server-manager/pkg/when"

	"4d63.com/tz"
	"github.com/etcd-io/bbolt"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var ErrCustomRaceNotFound = errors.New("servermanager: custom race not found")

type RaceManager struct {
	process             ServerProcess
	store               Store
	carManager          *CarManager
	trackManager        *TrackManager
	raceControl         *RaceControl
	notificationManager NotificationDispatcher

	currentRace      *ServerConfig
	currentEntryList EntryList

	mutex sync.RWMutex

	forceStopTimer *when.Timer

	// looped races
	loopedRaceSessionTypes      []SessionType
	loopedRaceWaitForSecondRace bool

	// scheduled races
	customRaceStartTimers    map[string]*when.Timer
	customRaceReminderTimers map[string]*when.Timer
}

func NewRaceManager(
	store Store,
	process ServerProcess,
	carManager *CarManager,
	trackManager *TrackManager,
	notificationManager NotificationDispatcher,
	raceControl *RaceControl,
) *RaceManager {
	return &RaceManager{
		store:               store,
		process:             process,
		carManager:          carManager,
		trackManager:        trackManager,
		notificationManager: notificationManager,
		raceControl:         raceControl,
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

func (rm *RaceManager) applyConfigAndStart(event RaceEvent) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// Reset the stored session types if this isn't a looped race
	if !event.IsLooping() {
		rm.clearLoopedRaceSessionTypes()
	}

	// load server opts
	serverOpts, err := rm.LoadServerOptions()

	if err != nil {
		return err
	}

	if serverOpts.RestartEventOnServerManagerLaunch == 1 {
		if err := rm.store.ClearLastRaceEvent(); err != nil {
			logrus.WithError(err).Errorf("Could not clear last race event")
		}
	}

	raceConfig := event.GetRaceConfig()
	entryList := event.GetEntryList()

	if config.Lua.Enabled && Premium() {
		err = eventStartPlugin(&raceConfig, serverOpts, &entryList)

		if err != nil {
			logrus.WithError(err).Error("event start plugin script failed")
		}
	}

	// the server won't start if an entrant has a larger ballast than is set as the max, correct if necessary
	greatestBallast := entryList.FindGreatestBallast()

	if greatestBallast > raceConfig.MaxBallastKilograms {
		raceConfig.MaxBallastKilograms = greatestBallast
	}

	config := ServerConfig{
		CurrentRaceConfig:  raceConfig,
		GlobalServerConfig: *serverOpts,
	}

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

	// filter out "AnyCarModel"
	finalCars := make([]string, 0)

	for _, car := range strings.Split(config.CurrentRaceConfig.Cars, ";") {
		if car == AnyCarModel {
			continue
		}

		finalCars = append(finalCars, car)
	}

	config.CurrentRaceConfig.Cars = strings.Join(finalCars, ";")

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
			config.GlobalServerConfig.Name = buildFinalServerName(config.GlobalServerConfig.ServerNameTemplate, event, config)
		}
	}

	if config.GlobalServerConfig.EnableContentManagerWrapper == 1 && config.GlobalServerConfig.ContentManagerWrapperPort > 0 {
		config.GlobalServerConfig.Name += fmt.Sprintf(" %c%d", contentManagerWrapperSeparator, config.GlobalServerConfig.ContentManagerWrapperPort)
	}

	err = config.Write()

	if err != nil {
		return err
	}

	// any available car should make sure you have one of each before randomising (#678)
	for _, car := range finalCars {
		for _, entrant := range entryList {
			if entrant.Model == AnyCarModel {
				entrant.Model = car
				entrant.Skin = rm.carManager.RandomSkin(entrant.Model)
				break
			}
		}
	}

	for _, entrant := range entryList {
		if entrant.Model == AnyCarModel {
			// cars with 'any car model' become random in the entry list.
			entrant.Model = finalCars[rand.Intn(len(finalCars))]
			entrant.Skin = rm.carManager.RandomSkin(entrant.Model)
		}
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

	err = rm.process.Start(event, config.GlobalServerConfig.UDPPluginAddress, config.GlobalServerConfig.UDPPluginLocalPort, forwardingAddress, forwardListenPort)

	if err != nil {
		return err
	}

	if !event.IsLooping() {
		_ = rm.notificationManager.SendRaceStartMessage(config, event)
	}

	// existing timer needs to be stopped in all cases
	if rm.forceStopTimer != nil {
		rm.forceStopTimer.Stop()
	}

	if event.GetForceStopTime() != 0 {
		// initiate force stop timer
		var withDrivers string

		if !event.GetForceStopWithDrivers() {
			withDrivers = "(unless there are drivers on the server at that time)"
		} else {
			withDrivers = "(even if there are drivers on the server at that time)"
		}

		logrus.Infof("Force Stop timer initialised, the server will be forcibly stopped after %.2f minutes %s.", event.GetForceStopTime().Minutes(), withDrivers)

		rm.forceStopTimer, err = when.When(time.Now().Add(event.GetForceStopTime()), func() {
			if rm.process.IsRunning() {

				if (event.GetForceStopWithDrivers()) || (rm.raceControl.ConnectedDrivers.Len() == 0) {
					err := rm.process.Stop()

					if err != nil {
						logrus.WithError(err).Error("couldn't forcibly stop the server!")
						return
					}

					logrus.Infof("Force Stop time expired, the server has been successfully stopped.")
				} else {
					logrus.Infof("Force Stop time expired, but %d drivers are on the server! Force stop aborted. "+
						"The server should stop automatically on event completion.", rm.raceControl.ConnectedDrivers.Len())
				}

			}
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func eventStartPlugin(raceConfig *CurrentRaceConfig, serverOpts *GlobalServerConfig, entryList *EntryList) error {
	p := &LuaPlugin{}

	newRaceConfig, newServerOpts, newEntryList := &CurrentRaceConfig{}, &GlobalServerConfig{}, &EntryList{}

	p.Inputs(raceConfig, serverOpts, entryList).Outputs(newRaceConfig, newServerOpts, newEntryList)
	err := p.Call("./plugins/events.lua", "onEventStart")

	if err != nil {
		return err
	}

	*raceConfig, *serverOpts, *entryList = *newRaceConfig, *newServerOpts, *newEntryList

	return nil
}

var ErrMustSubmitCar = errors.New("servermanager: you must set a car")

func (rm *RaceManager) SetupQuickRace(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	// load default config values
	quickRace := ConfigIniDefault().CurrentRaceConfig

	cars := r.Form["Cars"]

	quickRace.Cars = strings.Join(cars, ";")
	quickRace.Track = r.Form.Get("Track")
	quickRace.TrackLayout = r.Form.Get("TrackLayout")

	if quickRace.TrackLayout == defaultLayoutName {
		quickRace.TrackLayout = ""
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

	quickRace.LegalTyres = strings.Join(quickRaceTyres, ";")

	quickRace.Sessions = make(map[SessionType]*SessionConfig)

	qualifyingTime, err := strconv.ParseInt(r.Form.Get("Qualifying.Time"), 10, 0)

	if err != nil {
		return err
	}

	quickRace.AddSession(SessionTypeQualifying, &SessionConfig{
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

	quickRace.AddSession(SessionTypeRace, &SessionConfig{
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

	trackInfo, err := GetTrackInfo(quickRace.Track, quickRace.TrackLayout)

	if err == nil {
		boxes, err := trackInfo.Pitboxes.Int64()

		if err != nil {
			numPitboxes = quickRace.MaxClients
		} else {
			numPitboxes = int(boxes)
		}

	} else {
		numPitboxes = quickRace.MaxClients
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

	quickRace.MaxClients = numPitboxes

	return rm.applyConfigAndStart(&QuickRace{
		RaceConfig: quickRace,
		EntryList:  entryList,
	})
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
	pickupModeEnabled := formValueAsInt(r.FormValue("PickupModeEnabled"))

	if gasPenaltyDisabled == 0 {
		gasPenaltyDisabled = 1
	} else {
		gasPenaltyDisabled = 0
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
		PickupModeEnabled:         pickupModeEnabled,
		LockedEntryList:           lockedEntryList,
		RacePitWindowStart:        formValueAsInt(r.FormValue("RacePitWindowStart")),
		RacePitWindowEnd:          formValueAsInt(r.FormValue("RacePitWindowEnd")),
		ReversedGridRacePositions: formValueAsInt(r.FormValue("ReversedGridRacePositions")),
		QualifyMaxWaitPercentage:  formValueAsInt(r.FormValue("QualifyMaxWaitPercentage")),
		RaceGasPenaltyDisabled:    gasPenaltyDisabled,
		MaxBallastKilograms:       formValueAsInt(r.FormValue("MaxBallastKilograms")),
		AllowedTyresOut:           formValueAsInt(r.FormValue("AllowedTyresOut")),
		LoopMode:                  formValueAsInt(r.FormValue("LoopMode")),
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

		raceConfig.AddSession(session, &SessionConfig{
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

			timeMulti := r.Form["TimeMulti"][i]
			timeMultiInt := formValueAsInt(timeMulti)

			// This is probably a bit hacky, and may need removing with a future Sol update
			startTimeFinal := startTime.Add(-(time.Duration(timeMultiInt) * 5 * time.Hour))

			unixTime := time.Unix(0, 0)

			if startTimeFinal.Before(unixTime) {
				startTimeFinal = unixTime
			}

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

	if err := rm.SaveEntrantsForAutoFill(entryList); err != nil {
		return err
	}

	completeConfig := ConfigIniDefault()
	completeConfig.CurrentRaceConfig = *raceConfig

	overridePassword := r.FormValue("OverridePassword") == "1"
	replacementPassword := r.FormValue("ReplacementPassword")

	forceStopTime := formValueAsInt(r.FormValue("ForceStopTime"))
	forceStopWithDrivers := r.FormValue("ForceStopWithDrivers") == "1"

	if customRaceID := r.FormValue("Editing"); customRaceID != "" {
		// we are editing the race. load the previous one and overwrite it with this one
		customRace, err := rm.store.FindCustomRaceByID(customRaceID)

		if err != nil {
			return err
		}

		customRace.OverridePassword = overridePassword
		customRace.ReplacementPassword = replacementPassword

		customRace.ForceStopTime = forceStopTime
		customRace.ForceStopWithDrivers = forceStopWithDrivers

		customRace.Name = r.FormValue("CustomRaceName")
		customRace.EntryList = entryList
		customRace.RaceConfig = *raceConfig

		return rm.store.UpsertCustomRace(customRace)
	}

	saveAsPresetWithoutStartingRace := r.FormValue("action") == "justSave"
	schedule := r.FormValue("action") == "schedule"

	// save the custom race preset
	race, err := rm.SaveCustomRace(r.FormValue("CustomRaceName"), overridePassword, replacementPassword, *raceConfig, entryList, saveAsPresetWithoutStartingRace, forceStopTime, forceStopWithDrivers)

	if err != nil {
		return err
	}

	if schedule {
		dateString := r.FormValue("CustomRaceScheduled")
		timeString := r.FormValue("CustomRaceScheduledTime")
		timezone := r.FormValue("CustomRaceScheduledTimezone")

		location, err := tz.LoadLocation(timezone)

		if err != nil {
			logrus.WithError(err).Errorf("could not find location: %s", location)
			location = time.Local
		}

		// Parse time in correct time zone
		date, err := time.ParseInLocation("2006-01-02-15:04", dateString+"-"+timeString, location)

		if err != nil {
			return err
		}

		err = rm.ScheduleRace(race.UUID.String(), date, "add", r.FormValue("event-schedule-recurrence"))

		if err != nil {
			return err
		}

		return nil
	}

	if saveAsPresetWithoutStartingRace {
		return nil
	}

	return rm.applyConfigAndStart(race)
}

// applyCurrentRaceSetupToOptions takes current values in race which require more detailed configuration
// and applies them to the template opts.
func (rm *RaceManager) applyCurrentRaceSetupToOptions(opts *RaceTemplateVars, race CurrentRaceConfig) error {
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

	opts.Tyres = tyres
	opts.DeselectedTyres = deselectedTyres

	return nil
}

func (rm *RaceManager) ListAutoFillEntrants() ([]*Entrant, error) {
	entrants, err := rm.store.ListEntrants()

	if err != nil {
		return nil, err
	}

	sort.Slice(entrants, func(i, j int) bool {
		return entrants[i].Name < entrants[j].Name
	})

	return entrants, nil
}

type RaceTemplateVars struct {
	BaseTemplateVars

	CarOpts             Cars
	TrackOpts           []Track
	AvailableSessions   []SessionType
	Weather             Weather
	SolIsInstalled      bool
	Current             CurrentRaceConfig
	CurrentEntrants     EntryList
	PossibleEntrants    []*Entrant
	FixedSetups         CarSetups
	IsEditing           bool
	EditingID           string
	CustomRaceName      string
	SurfacePresets      []TrackSurfacePreset
	OverridePassword    bool
	ReplacementPassword string
	Tyres               Tyres
	DeselectedTyres     map[string]bool

	ForceStopTime        int
	ForceStopWithDrivers bool

	IsChampionship                 bool
	Championship                   *Championship
	ChampionshipHasAtLeastOnceRace bool

	IsRaceWeekend                   bool
	RaceWeekend                     *RaceWeekend
	RaceWeekendSession              *RaceWeekendSession
	RaceWeekendHasAtLeastOneSession bool

	ShowOverridePasswordCard bool
}

// BuildRaceOpts builds a quick race form
func (rm *RaceManager) BuildRaceOpts(r *http.Request) (*RaceTemplateVars, error) {
	_, cars, err := rm.carManager.Search(r.Context(), "", 0, 100000)

	if err != nil {
		return nil, err
	}

	tracks, err := rm.trackManager.ListTracks()

	if err != nil {
		return nil, err
	}

	weather, err := ListWeather()

	if err != nil {
		return nil, err
	}

	race := ConfigIniDefault()

	templateID := r.URL.Query().Get("from")

	var entrants EntryList

	if templateID != "" {
		// load a from a custom race template
		customRace, err := rm.store.FindCustomRaceByID(templateID)

		if err != nil {
			return nil, err
		}

		race.CurrentRaceConfig = customRace.RaceConfig
		entrants = customRace.EntryList
	}

	templateIDForEditing := chi.URLParam(r, "uuid")
	isEditing := templateIDForEditing != ""
	var customRaceName, replacementPassword string
	var overridePassword, forceStopWithDrivers bool
	var forceStopTime int

	if isEditing {
		customRace, err := rm.store.FindCustomRaceByID(templateIDForEditing)

		if err != nil {
			return nil, err
		}

		customRaceName = customRace.Name
		race.CurrentRaceConfig = customRace.RaceConfig
		entrants = customRace.EntryList
		overridePassword = customRace.OverrideServerPassword()
		replacementPassword = customRace.ReplacementServerPassword()

		forceStopTime = customRace.ForceStopTime
		forceStopWithDrivers = customRace.ForceStopWithDrivers
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
		if weather.CMWFXDate <= 0 {
			weather.CMWFXDate = int(time.Now().Unix())
			weather.CMWFXDateUnModified = int(time.Now().Unix())
		}
	}

	opts := &RaceTemplateVars{
		CarOpts:                  cars,
		TrackOpts:                tracks,
		AvailableSessions:        AvailableSessions,
		Weather:                  weather,
		SolIsInstalled:           solIsInstalled,
		Current:                  race.CurrentRaceConfig,
		CurrentEntrants:          entrants,
		PossibleEntrants:         possibleEntrants,
		FixedSetups:              fixedSetups,
		IsChampionship:           false, // this flag is overridden by championship setup
		IsRaceWeekend:            false, // this flag is overridden by race weekend setup
		IsEditing:                isEditing,
		EditingID:                templateIDForEditing,
		CustomRaceName:           customRaceName,
		SurfacePresets:           DefaultTrackSurfacePresets,
		OverridePassword:         overridePassword,
		ReplacementPassword:      replacementPassword,
		ShowOverridePasswordCard: true,
		ForceStopTime:            forceStopTime,
		ForceStopWithDrivers:     forceStopWithDrivers,
	}

	err = rm.applyCurrentRaceSetupToOptions(opts, race.CurrentRaceConfig)

	if err != nil {
		return nil, err
	}

	return opts, nil
}

const maxRecentRaces = 30

func (rm *RaceManager) ListCustomRaces() (recent, starred, looped, scheduled []*CustomRace, err error) {
	recent, err = rm.store.ListCustomRaces()

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
		if race.IsLooping() {
			looped = append(looped, race)
		}

		if race.Starred {
			starred = append(starred, race)
		}

		if race.Scheduled.After(time.Now()) {
			scheduled = append(scheduled, race)
		}

		if !race.Starred && !race.IsLooping() && !race.Scheduled.After(time.Now()) {
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

		err := rm.store.UpsertEntrant(*entrant)

		if err != nil {
			return err
		}
	}

	return nil
}

func (rm *RaceManager) SaveCustomRace(
	name string,
	overridePassword bool,
	replacementPassword string,
	config CurrentRaceConfig,
	entryList EntryList,
	starred bool,
	forceStopTime int,
	forceStopWithDrivers bool,
) (*CustomRace, error) {
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

	race := &CustomRace{
		Name:                name,
		HasCustomName:       hasCustomRaceName,
		OverridePassword:    overridePassword,
		ReplacementPassword: replacementPassword,
		Created:             time.Now(),
		UUID:                uuid.New(),
		Starred:             starred,

		ForceStopTime:        forceStopTime,
		ForceStopWithDrivers: forceStopWithDrivers,

		RaceConfig: config,
		EntryList:  entryList,
	}

	err := rm.store.UpsertCustomRace(race)

	if err != nil {
		return nil, err
	}

	return race, nil
}

func (rm *RaceManager) StartCustomRace(uuid string, forceRestart bool) error {
	race, err := rm.store.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	// Required for our nice auto loop stuff
	if forceRestart {
		race.RaceConfig.LoopMode = 1
	}

	if race.LoopServer != nil {
		race.LoopServer[serverID] = forceRestart
	}

	return rm.applyConfigAndStart(race)
}

func (rm *RaceManager) ScheduleRace(uuid string, date time.Time, action string, recurrence string) error {
	race, err := rm.store.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	originalDate := race.Scheduled
	race.Scheduled = date
	race.ScheduledServerID = serverID

	// if there is an existing schedule timer for this event stop it
	if timer := rm.customRaceStartTimers[race.UUID.String()]; timer != nil {
		timer.Stop()
	}

	if timer := rm.customRaceReminderTimers[race.UUID.String()]; timer != nil {
		timer.Stop()
	}

	if action == "add" {
		if race.Scheduled.IsZero() {
			return ErrScheduledTimeIsZero
		}

		// add a scheduled event on date
		if recurrence != "already-set" {
			if recurrence != "" {
				err := race.SetRecurrenceRule(recurrence)

				if err != nil {
					return err
				}

				// only set once when the event is first scheduled
				race.ScheduledInitial = race.Scheduled
			} else {
				race.ClearRecurrenceRule()
			}
		}

		if config.Lua.Enabled && Premium() {
			err = eventSchedulePlugin(race)

			if err != nil {
				logrus.WithError(err).Error("event schedule plugin script failed")
			}
		}

		rm.customRaceStartTimers[race.UUID.String()], err = when.When(race.Scheduled, func() {
			err := rm.StartScheduledRace(race)

			if err != nil {
				logrus.WithError(err).Errorf("Couldn't start scheduled race: %s, %s.", race.Name, race.UUID.String())
			}
		})

		if err != nil {
			return err
		}

		if rm.notificationManager.HasNotificationReminders() {
			_ = rm.notificationManager.SendRaceScheduledMessage(race, race.Scheduled)

			for _, timer := range rm.notificationManager.GetNotificationReminders() {
				thisTimer := timer

				rm.customRaceReminderTimers[race.UUID.String()], err = when.When(race.Scheduled.Add(time.Duration(0-timer)*time.Minute), func() {
					_ = rm.notificationManager.SendRaceReminderMessage(race, thisTimer)
				})

				if err != nil {
					logrus.WithError(err).Error("Could not set up race reminder timer")
				}
			}
		}

	} else {
		_ = rm.notificationManager.SendRaceCancelledMessage(race, originalDate)
		race.ClearRecurrenceRule()
	}

	return rm.store.UpsertCustomRace(race)
}

func eventSchedulePlugin(race *CustomRace) error {
	p := &LuaPlugin{}

	newRace := &CustomRace{}

	p.Inputs(race).Outputs(newRace)
	err := p.Call("./plugins/events.lua", "onEventSchedule")

	if err != nil {
		return err
	}

	*race = *newRace

	return nil
}

func (rm *RaceManager) StartScheduledRace(race *CustomRace) error {
	err := rm.StartCustomRace(race.UUID.String(), false)

	if err != nil {
		return err
	}

	if race.HasRecurrenceRule() {
		// this function carries out a save
		return rm.ScheduleNextFromRecurrence(race)
	}

	race.Scheduled = time.Time{}

	return rm.store.UpsertCustomRace(race)
}

func (rm *RaceManager) ScheduleNextFromRecurrence(race *CustomRace) error {
	// set the scheduled time to the next iteration of the recurrence rule
	nextRecurrence := rm.FindNextRecurrence(race, race.Scheduled)

	if nextRecurrence.IsZero() {
		// no recurrence was found (likely the recurrence had an UNTIL date)
		race.ClearRecurrenceRule()
		return rm.store.UpsertCustomRace(race)
	}

	return rm.ScheduleRace(race.UUID.String(), nextRecurrence, "add", "already-set")
}

func (rm *RaceManager) FindNextRecurrence(race *CustomRace, start time.Time) time.Time {
	rule, err := race.GetRecurrenceRule()

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't get recurrence rule for race: %s, %s", race.Name, race.Recurrence)
		return time.Time{}
	}

	next := rule.After(start, false)

	if next.After(time.Now()) {
		return next
	} else if next.IsZero() {
		return next
	} else {
		return rm.FindNextRecurrence(race, next)
	}
}

func (rm *RaceManager) DeleteCustomRace(uuid string) error {
	race, err := rm.store.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	if !race.Scheduled.IsZero() {
		err := rm.ScheduleRace(uuid, time.Time{}, "remove", "")

		if err != nil {
			return err
		}
	}

	return rm.store.DeleteCustomRace(race)
}

func (rm *RaceManager) ToggleStarCustomRace(uuid string) error {
	race, err := rm.store.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	race.Starred = !race.Starred

	return rm.store.UpsertCustomRace(race)
}

func (rm *RaceManager) ToggleLoopCustomRace(uuid string) error {
	race, err := rm.store.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	if race.LoopServer == nil {
		race.LoopServer = make(map[ServerID]bool)
	}

	race.LoopServer[serverID] = !race.LoopServer[serverID]

	return rm.store.UpsertCustomRace(race)
}

func (rm *RaceManager) SaveServerOptions(newServerOpts *GlobalServerConfig) error {
	oldServerOpts, err := rm.store.LoadServerOptions()

	if err != nil {
		return err
	}

	err = rm.store.UpsertServerOptions(newServerOpts)

	if err != nil {
		return err
	}

	err = rm.notificationManager.SaveServerOptions(oldServerOpts, newServerOpts)

	if err != nil {
		return err
	}

	err = rm.RescheduleNotifications(oldServerOpts, newServerOpts)

	if err != nil {
		return err
	}

	return nil
}

func (rm *RaceManager) LoadServerOptions() (*GlobalServerConfig, error) {
	serverOpts, err := rm.store.LoadServerOptions()

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
			logrus.WithError(err).Errorf("couldn't list custom races")
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
				logrus.WithError(err).Errorf("couldn't start auto loop custom race")
				return
			}

			i++
		}
	}
}

// callback check for udp end session, load result file, check session type against sessionTypes
// if session matches last session in sessionTypes then stop server and clear sessionTypes
func (rm *RaceManager) LoopCallback(message udp.Message) {
	if a, ok := message.(udp.EndSession); ok {
		if rm.loopedRaceSessionTypes == nil {
			logrus.Infof("Session types == nil. ignoring end session callback")
			return
		}

		filename := filepath.Base(string(a))

		results, err := LoadResult(filename)

		if err != nil {
			logrus.WithError(err).Errorf("Could not read session results for %s", filename)
			return
		}

		var endSession SessionType

		// If this is a race, and there is a second race configured
		// then wait for the second race to happen.
		if results.Type == SessionTypeRace {
			for _, session := range rm.loopedRaceSessionTypes {
				if session == SessionTypeSecondRace {
					if !rm.loopedRaceWaitForSecondRace {
						rm.loopedRaceWaitForSecondRace = true
						return
					}

					rm.loopedRaceWaitForSecondRace = false
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

		if results.Type == endSession {
			logrus.Infof("Event end detected, stopping looped session.")

			rm.clearLoopedRaceSessionTypes()

			err := rm.process.Stop()

			if err != nil {
				logrus.WithError(err).Errorf("Could not stop server")
				return
			}
		}
	}
}

func (rm *RaceManager) clearLoopedRaceSessionTypes() {
	rm.loopedRaceSessionTypes = nil
}

func (rm *RaceManager) InitScheduledRaces() error {
	rm.customRaceStartTimers = make(map[string]*when.Timer)
	rm.customRaceReminderTimers = make(map[string]*when.Timer)

	races, err := rm.store.ListCustomRaces()

	if err != nil {
		return err
	}

	for _, race := range races {
		race := race

		if race.ScheduledServerID != serverID {
			continue
		}

		if race.Scheduled.After(time.Now()) {
			// add a scheduled event on date
			rm.customRaceStartTimers[race.UUID.String()], err = when.When(race.Scheduled, func() {
				err := rm.StartScheduledRace(race)

				if err != nil {
					logrus.WithError(err).Errorf("Couldn't start scheduled race: %s, %s", race.Name, race.UUID.String())
				}
			})

			if err != nil {
				logrus.WithError(err).Error("Could not set up scheduled race timer")
			}

			if rm.notificationManager.HasNotificationReminders() {
				for _, timer := range rm.notificationManager.GetNotificationReminders() {
					if race.Scheduled.Add(time.Duration(0-timer) * time.Minute).After(time.Now()) {
						// add reminder
						thisTimer := timer

						rm.customRaceReminderTimers[race.UUID.String()], err = when.When(race.Scheduled.Add(time.Duration(0-timer)*time.Minute), func() {
							_ = rm.notificationManager.SendRaceReminderMessage(race, thisTimer)
						})

						if err != nil {
							logrus.WithError(err).Error("Could not set up scheduled race reminder timer")
						}
					}
				}
			}
		} else {
			if race.HasRecurrenceRule() {
				emptyTime := time.Time{}
				if race.Scheduled != emptyTime {
					logrus.Infof("Looks like the server was offline whilst a recurring scheduled event was meant to start!"+
						" Start time: %s. The schedule has been cleared, and the next recurrence time has been set."+
						" Start the event manually if you wish to run it.", race.Scheduled.String())
				}

				err := rm.ScheduleNextFromRecurrence(race)

				if err != nil {
					logrus.WithError(err).Errorf("Couldn't schedule next recurring race: %s, %s, %s", race.Name, race.UUID.String(), race.Recurrence)
				}
			} else {
				emptyTime := time.Time{}
				if race.Scheduled != emptyTime {
					logrus.Infof("Looks like the server was offline whilst a scheduled event was meant to start!"+
						" Start time: %s. The schedule has been cleared. Start the event manually if you wish to run it.", race.Scheduled.String())

					race.Scheduled = emptyTime

					err := rm.store.UpsertCustomRace(race)

					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// reschedule notifications if notification timer changed
func (rm *RaceManager) RescheduleNotifications(oldServerOpts *GlobalServerConfig, newServerOpts *GlobalServerConfig) error {
	if newServerOpts.NotificationReminderTimers == oldServerOpts.NotificationReminderTimers {
		return nil
	}

	// stop all existing timers
	for _, timer := range rm.customRaceReminderTimers {
		timer.Stop()
	}

	// rebuild the timers
	rm.customRaceReminderTimers = make(map[string]*when.Timer)

	if rm.notificationManager.HasNotificationReminders() {
		races, err := rm.store.ListCustomRaces()

		if err != nil {
			return err
		}

		for _, timer := range rm.notificationManager.GetNotificationReminders() {
			for _, race := range races {
				race := race

				if race.Scheduled.After(time.Now()) && race.Scheduled.Add(time.Duration(0-timer)*time.Minute).After(time.Now()) {
					// add reminder
					thisTimer := timer

					rm.customRaceReminderTimers[race.UUID.String()], err = when.When(race.Scheduled.Add(time.Duration(0-timer)*time.Minute), func() {
						_ = rm.notificationManager.SendRaceReminderMessage(race, thisTimer)
					})

					if err != nil {
						logrus.WithError(err).Error("Could not set up scheduled race reminder timer")
					}
				}
			}
		}
	}

	return nil
}
