package servermanager

import (
	"context"
	"errors"
	"fmt"
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/davecgh/go-spew/spew"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/etcd-io/bbolt"
	"github.com/google/uuid"
)

var (
	raceManager *RaceManager

	ErrCustomRaceNotFound = errors.New("servermanager: custom race not found")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func SetupRaceManager(storeFileLocation string) error {
	bb, err := bbolt.Open(storeFileLocation, 0644, nil)

	if err != nil {
		return err
	}

	raceManager = NewRaceManager(NewRaceStore(bb))

	return nil
}

type RaceManager struct {
	currentRace      *ServerConfig
	currentEntryList EntryList

	raceStore *RaceStore

	udpListenerContext context.Context
	udpListenerCfn     func()
	udpServerConn *udp.AssettoServerUDP
}

func NewRaceManager(raceStore *RaceStore) *RaceManager {
	ctx, cfn := context.WithCancel(context.Background())

	return &RaceManager{
		raceStore: raceStore,
		udpListenerContext: ctx,
		udpListenerCfn: cfn,
	}
}

func (rm *RaceManager) CurrentRace() (*ServerConfig, EntryList) {
	if !AssettoProcess.IsRunning() {
		return nil, nil
	}

	return rm.currentRace, rm.currentEntryList
}

func (rm *RaceManager) startUDPListener(cfg ServerConfig) error {
	// close old udp listener
	rm.udpListenerCfn()
	rm.udpListenerContext, rm.udpListenerCfn = context.WithCancel(context.Background())

	var err error

	rm.udpServerConn, err = udp.NewServerClient(rm.udpListenerContext, "127.0.0.1", 12000, 11000, rm.UDPCallback)

	if err != nil {
		return err
	}

	return nil
}

func (rm *RaceManager) UDPCallback(message udp.Message) {
	spew.Dump(message)
}

func (rm *RaceManager) applyConfigAndStart(config ServerConfig, entryList EntryList) error {
	// load server opts
	serverOpts, err := rm.LoadServerOptions()

	if err != nil {
		return err
	}

	config.GlobalServerConfig = *serverOpts

	err = config.Write()

	if err != nil {
		return err
	}

	err = entryList.Write()

	if err != nil {
		return err
	}

	err = rm.startUDPListener(config)

	if err != nil {
		return err
	}

	rm.currentRace = &config
	rm.currentEntryList = entryList

	if AssettoProcess.IsRunning() {
		return AssettoProcess.Restart()
	}

	err = AssettoProcess.Start()

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
		}

		numPitboxes = int(boxes)
	} else {
		numPitboxes = quickRace.CurrentRaceConfig.MaxClients
	}

	allCars, err := ListCars()

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

		entryList.Add(Entrant{
			Model: model,
			Skin:  skin,
		})
	}

	quickRace.CurrentRaceConfig.MaxClients = numPitboxes

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
		MaxClients:                formValueAsInt(r.FormValue("MaxClients")),
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

	allCars, err := ListCars()

	if err != nil {
		return err
	}

	carMap := allCars.AsMap()

	for i := 0; i < len(r.Form["EntryList.Name"]); i++ {
		model := r.Form["EntryList.Car"][i]
		skin := r.Form["EntryList.Skin"][i]

		if skin == "random_skin" {
			if skins, ok := carMap[model]; ok && len(skins) > 0 {
				skin = carMap[model][rand.Intn(len(carMap[model]))]
			}
		}

		entryList.Add(Entrant{
			Name:  r.Form["EntryList.Name"][i],
			Team:  r.Form["EntryList.Team"][i],
			GUID:  r.Form["EntryList.GUID"][i],
			Model: model,
			Skin:  skin,
			// Despite having the option for SpectatorMode, the server does not support it, and panics if set to 1
			// SpectatorMode: formValueAsInt(r.Form["EntryList.Spectator"][i]),
			Ballast:    formValueAsInt(r.Form["EntryList.Ballast"][i]),
			Restrictor: formValueAsInt(r.Form["EntryList.Restrictor"][i]),
		})
	}

	saveAsPresetWithoutStartingRace := r.FormValue("action") == "justSave"

	// save the custom race preset
	if err := rm.SaveCustomRace(r.FormValue("CustomRaceName"), raceConfig, entryList, saveAsPresetWithoutStartingRace); err != nil {
		return err
	}

	if saveAsPresetWithoutStartingRace {
		return nil
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

	var trackNames, trackLayouts []string

	tyres, err := ListTyres()

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

	possibleEntrants, err := rm.raceStore.ListEntrants()

	if err != nil {
		return nil, err
	}

	deselectedTyres := make(map[string]bool)

	for _, car := range varSplit(race.CurrentRaceConfig.Cars) {
		tyresForCar, ok := tyres[car]

		if !ok {
			continue
		}

		for carTyre := range tyresForCar {
			found := false

			for _, t := range varSplit(race.CurrentRaceConfig.LegalTyres) {
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

	return map[string]interface{}{
		"CarOpts":           cars,
		"TrackOpts":         trackNames,
		"TrackLayoutOpts":   trackLayouts,
		"AvailableSessions": AvailableSessions,
		"Tyres":             tyres,
		"DeselectedTyres":   deselectedTyres,
		"Weather":           weather,
		"Current":           race.CurrentRaceConfig,
		"CurrentEntrants":   entrants,
		"PossibleEntrants":  possibleEntrants,
	}, nil
}

const maxRecentRaces = 30

func (rm *RaceManager) ListCustomRaces() (recent, starred []CustomRace, err error) {
	recent, err = rm.raceStore.ListCustomRaces()

	if err == bbolt.ErrBucketNotFound {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Created.After(recent[j].Created)
	})

	var filteredRecent []CustomRace

	for _, race := range recent {
		if race.Starred {
			starred = append(starred, race)
		} else {
			filteredRecent = append(filteredRecent, race)
		}
	}

	if len(filteredRecent) > maxRecentRaces {
		filteredRecent = filteredRecent[:maxRecentRaces]
	}

	return filteredRecent, starred, nil
}

func (rm *RaceManager) SaveCustomRace(name string, config CurrentRaceConfig, entryList EntryList, starred bool) error {
	if name == "" {
		name = fmt.Sprintf("%s (%s) in %s (%d entrants)",
			prettifyName(config.Track, false),
			prettifyName(config.TrackLayout, true),
			carList(config.Cars),
			len(entryList),
		)
	}

	for _, entrant := range entryList {
		if entrant.Name == "" {
			continue // only save entrants that have a name
		}

		err := rm.raceStore.UpsertEntrant(entrant)

		if err != nil {
			return err
		}
	}

	return rm.raceStore.UpsertCustomRace(CustomRace{
		Name:    name,
		Created: time.Now(),
		UUID:    uuid.New(),
		Starred: starred,

		RaceConfig: config,
		EntryList:  entryList,
	})
}

func (rm *RaceManager) StartCustomRace(uuid string) error {
	race, err := rm.raceStore.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

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

func (rm *RaceManager) ToggleStarCustomRace(uuid string) error {
	race, err := rm.raceStore.FindCustomRaceByID(uuid)

	if err != nil {
		return err
	}

	race.Starred = !race.Starred

	return rm.raceStore.UpsertCustomRace(*race)
}

func (rm *RaceManager) SaveServerOptions(so *GlobalServerConfig) error {
	return rm.raceStore.UpsertServerOptions(so)
}

func (rm *RaceManager) LoadServerOptions() (*GlobalServerConfig, error) {
	serverOpts, err := rm.raceStore.LoadServerOptions()

	if err != nil {
		return nil, err
	}
/*
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

	serverOpts.UDPPluginAddress = fmt.Sprintf("127.0.0.1:%d", udpSendPort)
	serverOpts.UDPPluginRemotePort = udpSendPort
	serverOpts.UDPPluginLocalPort = udpListenPort
*/
	return serverOpts, nil
}
