package servermanager

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/csp"
	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"

	"github.com/google/uuid"
	"github.com/mitchellh/go-wordwrap"
	"github.com/sirupsen/logrus"
	lua "github.com/yuin/gopher-lua"
)

type RaceControl struct {
	process ServerProcess
	store   Store

	SessionInfo                udp.SessionInfo `json:"SessionInfo"`
	TrackMapData               TrackMapData    `json:"TrackMapData"`
	TrackInfo                  TrackInfo       `json:"TrackInfo"`
	SessionStartTime           time.Time       `json:"SessionStartTime"`
	CurrentRealtimePosInterval int             `json:"CurrentRealtimePosInterval"`

	ConnectedDrivers    *DriverMap `json:"ConnectedDrivers"`
	DisconnectedDrivers *DriverMap `json:"DisconnectedDrivers"`

	CarIDToGUID      map[udp.CarID]udp.DriverGUID `json:"CarIDToGUID"`
	carIDToGUIDMutex sync.RWMutex

	carUpdaters          map[udp.CarID]chan udp.CarUpdate
	serverProcessStopped chan struct{}

	broadcaster      Broadcaster
	trackDataGateway TrackDataGateway

	lastUpdateMessage      []byte
	lastUpdateMessageMutex sync.Mutex

	weatherState           *csp.WeatherConditions
	weatherIsTransitioning bool

	persistStoreDataMutex sync.Mutex
}

// RaceControl piggyback's on the udp.Message interface so that the entire data can be sent to newly connected clients.
func (rc *RaceControl) Event() udp.Event {
	return 200
}

type CollisionType string

const (
	CollisionWithCar         CollisionType = "with other car"
	CollisionWithEnvironment CollisionType = "with environment"
)

type Collision struct {
	ID              string         `json:"ID"`
	Type            CollisionType  `json:"Type"`
	Time            time.Time      `json:"Time" ts:"date"`
	OtherDriverGUID udp.DriverGUID `json:"OtherDriverGUID"`
	OtherDriverName string         `json:"OtherDriverName"`
	Speed           float64        `json:"Speed"`
}

func NewRaceControl(broadcaster Broadcaster, trackDataGateway TrackDataGateway, process ServerProcess, store Store) *RaceControl {
	rc := &RaceControl{
		broadcaster:          broadcaster,
		trackDataGateway:     trackDataGateway,
		process:              process,
		store:                store,
		carUpdaters:          make(map[udp.CarID]chan udp.CarUpdate),
		serverProcessStopped: make(chan struct{}),
	}

	process.NotifyDone(rc.serverProcessStopped)

	rc.clearAllDrivers()

	go panicCapture(rc.watchForTimedOutDrivers)

	return rc
}

func (rc *RaceControl) UDPCallback(message udp.Message) {
	var err error

	sendUpdatedRaceControlStatus := false

	switch m := message.(type) {
	case udp.Version:
		err = rc.OnVersion(m)
	case udp.SessionInfo:
		if m.Event() == udp.EventNewSession {
			err = rc.OnNewSession(m)
			sendUpdatedRaceControlStatus = true
		} else {
			sendUpdatedRaceControlStatus, err = rc.OnSessionUpdate(m)
		}

	case udp.EndSession:
		err = rc.OnEndSession(m)

		sendUpdatedRaceControlStatus = true
	case udp.CarUpdate:
		err = rc.OnCarUpdate(m)
	case udp.SessionCarInfo:
		if m.Event() == udp.EventNewConnection {
			err = rc.OnClientConnect(m)
		} else if m.Event() == udp.EventConnectionClosed {
			err = rc.OnClientDisconnect(m)
		}

		sendUpdatedRaceControlStatus = true
	case udp.ClientLoaded:
		err = rc.OnClientLoaded(m)

		sendUpdatedRaceControlStatus = true
	case udp.CollisionWithCar:
		err = rc.OnCollisionWithCar(m)
		sendUpdatedRaceControlStatus = true
	case udp.CollisionWithEnvironment:
		err = rc.OnCollisionWithEnvironment(m)
		sendUpdatedRaceControlStatus = true
	case udp.LapCompleted:
		err = rc.OnLapCompleted(m)

		sendUpdatedRaceControlStatus = true

	case udp.Chat:
		err = rc.OnChat(m)
	default:
		// unhandled event
		return
	}

	if err != nil {
		logrus.WithError(err).Errorf("Unable to handle event: %d", message.Event())
		return
	}

	if sendUpdatedRaceControlStatus {
		// update the current refresh rate
		rc.CurrentRealtimePosInterval = udp.CurrentRealtimePosIntervalMs

		lastUpdateMessage, err := rc.broadcaster.Send(rc)

		if err != nil {
			logrus.WithError(err).Error("Unable to broadcast race control message")
			return
		}

		rc.lastUpdateMessageMutex.Lock()
		rc.lastUpdateMessage = lastUpdateMessage
		rc.lastUpdateMessageMutex.Unlock()
	}
}

var driverTimeout = time.Minute * 5

func (rc *RaceControl) watchForTimedOutDrivers() {
	ticker := time.NewTicker(time.Minute)

	for range ticker.C {
		var driversToDisconnect []*RaceControlDriver

		_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
			driver.mutex.Lock()
			defer driver.mutex.Unlock()

			if !driver.LastSeen.IsZero() && time.Since(driver.LastSeen) > driverTimeout || driver.LastSeen.IsZero() && time.Since(driver.ConnectedTime) > driverTimeout {
				driversToDisconnect = append(driversToDisconnect, driver)
			}

			return nil
		})

		for _, driver := range driversToDisconnect {
			logrus.Debugf("Driver: %s (%s) has not been seen in 5 minutes, disconnecting", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID)
			err := rc.disconnectDriver(driver)

			if err != nil {
				logrus.WithError(err).Errorf("Could not disconnect driver: %s (%s)", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID)
				continue
			}
		}

		if len(driversToDisconnect) > 0 {
			_, err := rc.broadcaster.Send(rc)

			if err != nil {
				logrus.WithError(err).Error("Could not broadcast driver disconnect message")
			}
		}
	}
}

// OnVersion occurs when the Assetto Corsa Server starts up for the first time.
func (rc *RaceControl) OnVersion(version udp.Version) error {
	go panicCapture(rc.requestSessionInfo)

	_, err := rc.broadcaster.Send(version)

	return err
}

// OnCarUpdate occurs every udp.RealTimePosInterval and returns car position, speed, etc.
// drivers top speeds are recorded per lap, as well as their last seen updated.
func (rc *RaceControl) OnCarUpdate(update udp.CarUpdate) error {
	if ch, ok := rc.carUpdaters[update.CarID]; !ok || ch == nil {
		rc.carUpdaters[update.CarID] = make(chan udp.CarUpdate, 1000)

		go panicCapture(func() {
			for update := range rc.carUpdaters[update.CarID] {
				err := rc.handleCarUpdate(update)

				if err != nil {
					logrus.WithError(err).Error("Could not handle car update")
				}
			}
		})
	}

	rc.carUpdaters[update.CarID] <- update

	return nil
}

func (rc *RaceControl) handleCarUpdate(update udp.CarUpdate) error {
	driver, err := rc.findConnectedDriverByCarID(update.CarID)

	if err != nil {
		return err
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	speed := metersPerSecondToKilometersPerHour(
		math.Sqrt(math.Pow(float64(update.Velocity.X), 2) + math.Pow(float64(update.Velocity.Z), 2)),
	)

	if speed > driver.CurrentCar().TopSpeedThisLap {
		driver.CurrentCar().TopSpeedThisLap = speed
	}

	driver.LastSeen = time.Now()
	driver.LastPos = update.Pos

	_, err = rc.broadcaster.Send(update)

	return err
}

var emptyCarInfoMutex = sync.Mutex{}

// OnNewSession occurs every new session. If the session is the first in an event and it is not a looped practice,
// then all driver information is cleared.
func (rc *RaceControl) OnNewSession(sessionInfo udp.SessionInfo) error {
	oldSessionInfo := rc.SessionInfo
	rc.SessionInfo = sessionInfo
	rc.SessionStartTime = time.Now()

	emptyCarInfo := true

	if (rc.ConnectedDrivers.Len() > 0 || rc.DisconnectedDrivers.Len() > 0) && sessionInfo.Type == udp.SessionTypePractice {
		if oldSessionInfo.Type == sessionInfo.Type && oldSessionInfo.Track == sessionInfo.Track && oldSessionInfo.TrackConfig == sessionInfo.TrackConfig && oldSessionInfo.Name == sessionInfo.Name {
			// this is a looped event, keep the cars
			emptyCarInfo = false
		}
	}

	if emptyCarInfo {
		_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
			emptyCarInfoMutex.Lock()
			defer emptyCarInfoMutex.Unlock()

			*driver = *NewRaceControlDriver(driver.CarInfo)

			return nil
		})

		// all disconnected drivers are removed when car info is emptied, otherwise we are just showing empty entries in
		// the disconnected drivers table, which is pointless.
		rc.DisconnectedDrivers = NewDriverMap(DisconnectedDrivers, rc.SortDrivers)
	}

	// clear out last lap completed time each new session
	_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
		driver.mutex.Lock()
		defer driver.mutex.Unlock()

		driver.CurrentCar().LastLapCompletedTime = time.Now()

		return nil
	})

	var err error

	trackInfo, err := rc.trackDataGateway.TrackInfo(sessionInfo.Track, sessionInfo.TrackConfig)

	if err != nil {
		return err
	}

	rc.TrackInfo = *trackInfo

	trackMapData, err := rc.trackDataGateway.TrackMap(sessionInfo.Track, sessionInfo.TrackConfig)

	if err != nil {
		logrus.WithError(err).Errorf("Could not load track map data")
	} else {
		rc.TrackMapData = *trackMapData
	}

	logrus.Debugf("New session detected: %s at %s (%s) [emptyCarInfo: %t]", sessionInfo.Type.String(), sessionInfo.Track, sessionInfo.TrackConfig, emptyCarInfo)

	// look for live timings stored previously
	persistedInfo, err := rc.store.LoadLiveTimingsData()

	if err == nil && persistedInfo != nil {
		if persistedInfo.SessionType == rc.SessionInfo.Type &&
			persistedInfo.Track == rc.SessionInfo.Track &&
			persistedInfo.TrackLayout == rc.SessionInfo.TrackConfig &&
			persistedInfo.SessionName == rc.SessionInfo.Name {

			for guid, driver := range persistedInfo.Drivers {
				_, driverPresentInDisconnectedList := rc.DisconnectedDrivers.Get(guid)
				_, driverPresentInConnectedList := rc.ConnectedDrivers.Get(guid)

				if !driverPresentInConnectedList && !driverPresentInDisconnectedList {
					rc.DisconnectedDrivers.Add(guid, driver)
				}
			}

			logrus.Infof("Loaded previous Live Timings data for %s (%s), num drivers: %d", persistedInfo.Track, persistedInfo.TrackLayout, len(persistedInfo.Drivers))
		}
	} else {
		logrus.WithError(err).Debugf("Could not load persisted live timings practice data")
	}

	_, err = rc.broadcaster.Send(sessionInfo)

	return err
}

// clearAllDrivers removes all known information about connected and disconnected drivers from RaceControl
func (rc *RaceControl) clearAllDrivers() {
	rc.ConnectedDrivers = NewDriverMap(ConnectedDrivers, rc.SortDrivers)
	rc.DisconnectedDrivers = NewDriverMap(DisconnectedDrivers, rc.SortDrivers)
	rc.carIDToGUIDMutex.Lock()
	rc.CarIDToGUID = make(map[udp.CarID]udp.DriverGUID)
	rc.carIDToGUIDMutex.Unlock()
}

var sessionInfoRequestInterval = time.Second * 30

// requestSessionInfo sends a request every sessionInfoRequestInterval to get information about temps, etc in the session.
func (rc *RaceControl) requestSessionInfo() {
	sessionInfoTicker := time.NewTicker(sessionInfoRequestInterval)

	for {
		select {
		case <-sessionInfoTicker.C:
			err := rc.process.SendUDPMessage(udp.GetSessionInfo{})

			if err == ErrNoOpenUDPConnection {
				logrus.WithError(err).Warnf("Couldn't send session info udp request. Breaking loop.")
				sessionInfoTicker.Stop()
				return
			} else if err != nil {
				logrus.WithError(err).Errorf("Couldn't send session info udp request")
			}

		case <-rc.serverProcessStopped:
			logrus.Debugf("Assetto Process completed. Disconnecting all connected drivers. Session done.")
			sessionInfoTicker.Stop()

			var drivers []*RaceControlDriver

			rc.persistTimingData()

			// the server has just stopped. send disconnect messages for all connected cars.
			_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
				// Each takes a read lock, so we cannot call disconnectDriver (which takes a write lock) from inside it.
				// we must instead append them to a slice and disconnect them outside the Each call.
				drivers = append(drivers, driver)

				return nil
			})

			for _, driver := range drivers {
				// disconnect the driver
				err := rc.disconnectDriver(driver)

				if err != nil {
					logrus.WithError(err).Errorf("Could not disconnect driver: %s (%s)", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID)
					continue
				}
			}

			if _, err := rc.broadcaster.Send(rc); err != nil {
				logrus.WithError(err).Errorf("Couldn't broadcast race control")
			}

			return
		}
	}
}

func (rc *RaceControl) disconnectDriver(driver *RaceControlDriver) error {
	driver.mutex.Lock()
	carInfo := driver.CarInfo
	carInfo.EventType = udp.EventConnectionClosed
	driver.mutex.Unlock()

	return rc.OnClientDisconnect(carInfo)
}

// OnSessionUpdate is called every sessionRequestInterval.
func (rc *RaceControl) OnSessionUpdate(sessionInfo udp.SessionInfo) (bool, error) {
	oldSessionInfo := rc.SessionInfo

	// we can't just copy over the session information, we must copy individual
	// parts of it, as the session type is incorrect.
	rc.SessionInfo.AmbientTemp = sessionInfo.AmbientTemp
	rc.SessionInfo.RoadTemp = sessionInfo.RoadTemp
	rc.SessionInfo.WeatherGraphics = sessionInfo.WeatherGraphics
	rc.SessionInfo.ElapsedMilliseconds = sessionInfo.ElapsedMilliseconds

	sessionHasChanged := oldSessionInfo.AmbientTemp != rc.SessionInfo.AmbientTemp || oldSessionInfo.RoadTemp != rc.SessionInfo.RoadTemp || oldSessionInfo.WeatherGraphics != rc.SessionInfo.WeatherGraphics

	return sessionHasChanged, nil
}

// OnEndSession is called at the end of every session.
func (rc *RaceControl) OnEndSession(sessionFile udp.EndSession) error {
	return nil
}

// OnClientConnect stores CarID -> DriverGUID mappings. if a driver is known to have previously been in this event,
// they will be moved from DisconnectedDrivers to ConnectedDrivers.
func (rc *RaceControl) OnClientConnect(client udp.SessionCarInfo) error {
	rc.carIDToGUIDMutex.Lock()
	rc.CarIDToGUID[client.CarID] = client.DriverGUID
	rc.carIDToGUIDMutex.Unlock()

	client.DriverInitials = driverInitials(client.DriverName)
	client.DriverName = driverName(client.DriverName)

	var driver *RaceControlDriver

	if disconnectedDriver, ok := rc.DisconnectedDrivers.Get(client.DriverGUID); ok {
		driver = disconnectedDriver
		logrus.Debugf("Driver %s (%s) reconnected in %s (car id: %d)", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID, driver.CarInfo.CarModel, client.CarID)
		rc.DisconnectedDrivers.Del(client.DriverGUID)
	} else {
		driver = NewRaceControlDriver(client)
		logrus.Debugf("Driver %s (%s) connected in %s (car id: %d)", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID, driver.CarInfo.CarModel, client.CarID)
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	driver.CarInfo = client

	if _, ok := driver.Cars[driver.CarInfo.CarModel]; !ok {
		driver.Cars[driver.CarInfo.CarModel] = &RaceControlCarLapInfo{}
	}

	driver.ConnectedTime = time.Now()
	driver.LastSeen = time.Time{}
	driver.CurrentCar().LastLapCompletedTime = time.Now()

	rc.ConnectedDrivers.Add(driver.CarInfo.DriverGUID, driver)

	_, err := rc.broadcaster.Send(client)

	return err
}

func (rc *RaceControl) OnChat(chat udp.Chat) error {
	fmt.Println("received chat message", chat.Message)

	_, err := csp.FromChatMessage(chat.Message)

	if err != nil {
		return err
	}

	return nil
}

// OnClientDisconnect moves a client from ConnectedDrivers to DisconnectedDrivers.
func (rc *RaceControl) OnClientDisconnect(client udp.SessionCarInfo) error {
	if ch, ok := rc.carUpdaters[client.CarID]; ok && ch != nil {
		delete(rc.carUpdaters, client.CarID)
	}

	driver, ok := rc.ConnectedDrivers.Get(client.DriverGUID)

	if !ok {
		return fmt.Errorf("racecontrol: client disconnected without ever being connected: %s (%s)", client.DriverName, client.DriverGUID)
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	logrus.Debugf("Driver %s (%s) disconnected", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID)

	rc.ConnectedDrivers.Del(driver.CarInfo.DriverGUID)

	if driver.TotalNumLaps > 0 {
		rc.DisconnectedDrivers.Add(driver.CarInfo.DriverGUID, driver)
	}

	_, err := rc.broadcaster.Send(client)

	return err
}

// findConnectedDriverByCarID looks for a driver in ConnectedDrivers by their CarID. This is the only place CarID
// is used for a look-up, and it uses the CarIDToGUID map to perform the lookup.
func (rc *RaceControl) findConnectedDriverByCarID(carID udp.CarID) (*RaceControlDriver, error) {
	rc.carIDToGUIDMutex.RLock()
	driverGUID, ok := rc.CarIDToGUID[carID]
	rc.carIDToGUIDMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("racecontrol: could not find DriverGUID for CarID: %d", carID)
	}

	driver, ok := rc.ConnectedDrivers.Get(driverGUID)

	if !ok {
		return nil, fmt.Errorf("racecontrol: could not find connected driver for DriverGUID: %s", driverGUID)
	}

	return driver, nil
}

var ErrWeatherMidTransition = errors.New("servermanager: weather is currently transitioning, cannot start new transition")

func (rc *RaceControl) nextWeather(timeStamp, timeToApply, nextWeather int) error {
	var date uint64
	var current csp.Weather

	cfg := rc.process.Event().GetRaceConfig()

	for _, weather := range cfg.Weather {
		if weather != nil {
			date = uint64(weather.CMWFXDate)
			current = csp.Weather(weather.CMWFXType)
		}
		break
	}

	if timeStamp != 0 {
		date = uint64(timeStamp)
	}

	if rc.weatherIsTransitioning {
		return ErrWeatherMidTransition
	}

	if rc.weatherState == nil {
		// first transition
		rc.weatherState = &csp.WeatherConditions{
			Timestamp:   date,
			Current:     current,
			Next:        csp.Weather(nextWeather),
			Transition:  0,
			TimeToApply: float32(timeToApply),
		}
	} else {
		// nth transition
		rc.weatherState.Timestamp = date
		rc.weatherState.Current = rc.weatherState.Next
		rc.weatherState.Next = csp.Weather(nextWeather)
		rc.weatherState.Transition = 0
		rc.weatherState.TimeToApply = float32(timeToApply)
	}

	go func() {

		rc.weatherIsTransitioning = true

		for i := float32(0); i <= 1.1; i += 0.1 {
			rc.weatherState.Transition = i

			var wg sync.WaitGroup

			for _, d := range rc.ConnectedDrivers.Drivers {
				go func(driver *RaceControlDriver) {
					wg.Add(1)

					message, err := csp.ToChatMessage(driver.CarInfo.CarID, rc.weatherState)

					if err != nil {
						logrus.WithError(err).Errorf("couldn't build weather change message to driver ID: %d", driver.CarInfo.CarID)
						return
					}

					err = rc.process.SendUDPMessage(message)

					if err != nil {
						logrus.WithError(err).Errorf("couldn't send weather change message to driver ID: %d", driver.CarInfo.CarID)
						return
					}

					wg.Done()

				}(d)
			}

			wg.Wait()

			time.Sleep(time.Duration(rc.weatherState.TimeToApply/10) * time.Second)
		}

		rc.weatherIsTransitioning = false
	}()

	logrus.Debug("successfully sent next weather messages")

	return nil
}

func (rc *RaceControl) testWeather(timeStamp, timeToApply, currentWeather, nextWeather int, transition float64) error {

	rc.weatherState = &csp.WeatherConditions{
		Timestamp:   uint64(timeStamp),
		Current:     csp.Weather(currentWeather),
		Next:        csp.Weather(nextWeather),
		Transition:  float32(transition),
		TimeToApply: float32(timeToApply),
	}

	go func() {
		var wg sync.WaitGroup

		for _, driver := range rc.ConnectedDrivers.Drivers {
			go func(driver *RaceControlDriver) {
				wg.Add(1)

				message, err := csp.ToChatMessage(driver.CarInfo.CarID, rc.weatherState)

				if err != nil {
					logrus.WithError(err).Errorf("couldn't build weather change message to driver ID: %d", driver.CarInfo.CarID)
					return
				}

				err = rc.process.SendUDPMessage(message)

				if err != nil {
					logrus.WithError(err).Errorf("couldn't send weather change message to driver ID: %d", driver.CarInfo.CarID)
					return
				}

				wg.Done()
			}(driver)
		}

		wg.Wait()
	}()

	logrus.Debug("successfully sent test next weather messages")

	return nil
}

// OnClientLoaded marks a connected client as having loaded in.
func (rc *RaceControl) OnClientLoaded(loadedCar udp.ClientLoaded) error {
	driver, err := rc.findConnectedDriverByCarID(udp.CarID(loadedCar))

	if err != nil {
		return err
	}

	serverConfig, err := rc.store.LoadServerOptions()

	if err != nil {
		return err
	}

	solWarning := ""
	liveLink := ""

	/*go func() {
		time.Sleep(time.Second * 10)

		current := csp.ScatteredClouds
		next := csp.HeavyRain

		cfg := rc.process.Event().GetRaceConfig()

		var w *WeatherConfig

		for _, weather := range cfg.Weather {
			w = weather
			break
		}

		secondsToApply := 5

		weatherChange := csp.WeatherConditions{
			Timestamp:   uint64(w.CMWFXDate),
			Current:     current,
			Next:        next,
			Transition:  0,
			TimeToApply: float32(secondsToApply),
		}

		i := float32(0.1)

		for ; i <= 1.1; i += 0.1 {
			time.Sleep(time.Duration(secondsToApply) * time.Second)
			fmt.Println(i)
			weatherChange.Transition = i

			message, err := csp.ToChatMessage(driver.CarInfo.CarID, weatherChange)

			if err != nil {
				fmt.Println(err)
				continue
			}

			err = rc.process.SendUDPMessage(message)

			if err != nil {
				fmt.Println(err)
				continue
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 30)

		current := csp.HeavyRain
		next := csp.Clear

		cfg := rc.process.Event().GetRaceConfig()

		var w *WeatherConfig

		for _, weather := range cfg.Weather {
			w = weather
			break
		}
		secondsToApply := 5

		weatherChange := csp.WeatherConditions{
			Timestamp:   uint64(w.CMWFXDate),
			Current:     current,
			Next:        next,
			Transition:  0,
			TimeToApply: float32(secondsToApply),
		}

		for i := float32(0); i <= 1.1; i += 0.1 {
			fmt.Println(i)
			weatherChange.Transition = i

			message, err := csp.ToChatMessage(driver.CarInfo.CarID, weatherChange)

			if err != nil {
				fmt.Println(err)
				continue
			}

			err = rc.process.SendUDPMessage(message)

			if err != nil {
				fmt.Println(err)
				continue
			}

			time.Sleep(time.Duration(secondsToApply) * time.Second)
		}*/
		/*
			current = csp.OvercastClouds
			next = csp.HeavyRain

			weatherChange = csp.WeatherConditions{
				Timestamp:   uint64(w.CMWFXDate),
				Current:     current,
				Next:        next,
				Transition:  0,
				TimeToApply: float32(secondsToApply),
			}

			for ; i <= 1; i += 0.03 {
				fmt.Println(i)
				weatherChange.Transition = i

				message, err := csp.ToChatMessage(driver.CarInfo.CarID, weatherChange)

				if err != nil {
					fmt.Println(err)
					continue
				}

				err = rc.process.SendUDPMessage(message)

				if err != nil {
					fmt.Println(err)
					continue
				}

				time.Sleep(time.Duration(secondsToApply) * time.Second)
			}
	}()*/

	if rc.process.Event().GetRaceConfig().IsSol == 1 {
		solWarning = "This server is running Sol. For the best experience please install Sol, and remember the other drivers may be driving in night conditions."
	}

	if config != nil && config.HTTP.BaseURL != "" {
		liveLink = fmt.Sprintf("You can view live timings for this event at %s", config.HTTP.BaseURL+"/live-timing")
	}

	wrapped := strings.Split(wordwrap.WrapString(
		fmt.Sprintf(
			"Hi, %s! Welcome to the %s server! %s %s Make this race count! %s\n",
			driver.CarInfo.DriverName,
			serverConfig.GetName(),
			serverConfig.ServerJoinMessage,
			solWarning,
			liveLink,
		),
		60,
	), "\n")

	for _, msg := range wrapped {
		welcomeMessage, err := udp.NewSendChat(driver.CarInfo.CarID, msg)

		if err == nil {
			err := rc.process.SendUDPMessage(welcomeMessage)

			if err != nil {
				logrus.WithError(err).Errorf("Unable to send welcome message to: %s", driver.CarInfo.DriverName)
			}
		} else {
			logrus.WithError(err).Errorf("Unable to build welcome message to: %s", driver.CarInfo.DriverName)
		}
	}

	logrus.Debugf("Driver: %s (%s) loaded", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID)

	driver.LoadedTime = time.Now()

	_, err = rc.broadcaster.Send(loadedCar)

	return err
}

// OnLapCompleted occurs every time a driver crosses the line. Lap information is collected for the driver
// and best lap time and top speed are calculated. OnLapCompleted also remembers the car the lap was completed in
// a PreviousCars map on the driver. This is so that lap times between different cars can be compared.
func (rc *RaceControl) OnLapCompleted(lap udp.LapCompleted) error {
	driver, err := rc.findConnectedDriverByCarID(lap.CarID)

	if err != nil {
		return err
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	lapDuration := lapToDuration(int(lap.LapTime))

	logrus.Debugf("Lap completed by driver: %s (%s), %s", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID, lapDuration)

	driver.TotalNumLaps++
	currentCar := driver.CurrentCar()

	currentCar.TotalLapTime += lapDuration
	currentCar.LastLap = lapDuration
	currentCar.NumLaps++
	currentCar.LastLapCompletedTime = time.Now()

	if lap.Cuts == 0 && (lapDuration < currentCar.BestLap || currentCar.BestLap == 0) {
		currentCar.BestLap = lapDuration
		currentCar.TopSpeedBestLap = currentCar.TopSpeedThisLap
	}

	currentCar.TopSpeedThisLap = 0

	rc.ConnectedDrivers.sort()

	if rc.SessionInfo.Type == udp.SessionTypeRace {
		// calculate split
		if driver.Position == 1 {
			driver.Split = time.Duration(0).String()
		} else {
			_ = rc.ConnectedDrivers.Each(func(otherDriverGUID udp.DriverGUID, otherDriver *RaceControlDriver) error {
				if otherDriver.Position == driver.Position-1 {
					driverCar := driver.CurrentCar()
					otherDriverCar := otherDriver.CurrentCar()

					lapDifference := otherDriverCar.NumLaps - driverCar.NumLaps

					if lapDifference <= 0 {
						driver.Split = (driverCar.TotalLapTime - otherDriverCar.TotalLapTime).Round(time.Millisecond).String()
					} else if lapDifference == 1 {
						driver.Split = "1 lap"
					} else {
						driver.Split = fmt.Sprintf("%d laps", lapDifference)
					}
				}

				return nil
			})
		}
	} else {
		var previousCar *RaceControlCarLapInfo

		// gaps are calculated vs best lap
		_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
			if previousCar == nil {
				driver.Split = "0s"
			} else {
				car := driver.CurrentCar()

				if car.BestLap >= previousCar.BestLap && car.BestLap != 0 {
					driver.Split = (car.BestLap - previousCar.BestLap).String()
				} else {
					driver.Split = ""
				}
			}

			previousCar = driver.CurrentCar()

			return nil
		})
	}

	return nil
}

func (rc *RaceControl) SortDrivers(driverGroup RaceControlDriverGroup, driverA, driverB *RaceControlDriver) bool {
	driverACar := driverA.CurrentCar()
	driverBCar := driverB.CurrentCar()

	if rc.SessionInfo.Type == udp.SessionTypeRace {
		if driverGroup == ConnectedDrivers {
			if driverACar.NumLaps == driverBCar.NumLaps {
				return driverACar.TotalLapTime < driverBCar.TotalLapTime
			}

			return driverACar.NumLaps > driverBCar.NumLaps
		} else if driverGroup == DisconnectedDrivers {
			return driverACar.LastLapCompletedTime.After(driverBCar.LastLapCompletedTime)
		} else {
			panic("unknown driver group")
		}
	} else {
		if driverACar.BestLap == 0 && driverBCar.BestLap == 0 {
			if driverACar.NumLaps == driverBCar.NumLaps {
				return driverACar.LastLapCompletedTime.Before(driverBCar.LastLapCompletedTime)
			}

			return driverACar.NumLaps > driverBCar.NumLaps
		}

		if driverACar.BestLap == 0 {
			return false
		} else if driverBCar.BestLap == 0 {
			return true
		}

		return driverACar.BestLap < driverBCar.BestLap
	}
}

func metersPerSecondToKilometersPerHour(mps float64) float64 {
	return mps * 3.6
}

// OnCollisionWithCar registers a driver's collision with another car.
func (rc *RaceControl) OnCollisionWithCar(collision udp.CollisionWithCar) error {
	driver, err := rc.findConnectedDriverByCarID(collision.CarID)

	if err != nil {
		return err
	}

	c := Collision{
		ID:    uuid.New().String(),
		Type:  CollisionWithCar,
		Time:  time.Now(),
		Speed: metersPerSecondToKilometersPerHour(float64(collision.ImpactSpeed)),
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	otherDriver, err := rc.findConnectedDriverByCarID(collision.OtherCarID)

	if err == nil {
		c.OtherDriverGUID = otherDriver.CarInfo.DriverGUID
		c.OtherDriverName = otherDriver.CarInfo.DriverName
	}

	driver.Collisions = append(driver.Collisions, c)

	_, err = rc.broadcaster.Send(collision)

	return err
}

// OnCollisionWithEnvironment registers a driver's collision with the environment.
func (rc *RaceControl) OnCollisionWithEnvironment(collision udp.CollisionWithEnvironment) error {
	driver, err := rc.findConnectedDriverByCarID(collision.CarID)

	if err != nil {
		return err
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	driver.Collisions = append(driver.Collisions, Collision{
		ID:    uuid.New().String(),
		Type:  CollisionWithEnvironment,
		Time:  time.Now(),
		Speed: metersPerSecondToKilometersPerHour(float64(collision.ImpactSpeed)),
	})

	_, err = rc.broadcaster.Send(collision)

	return err
}

type LiveTimingsPersistedData struct {
	SessionType udp.SessionType
	Track       string
	TrackLayout string
	SessionName string

	Drivers map[udp.DriverGUID]*RaceControlDriver
}

func (rc *RaceControl) persistTimingData() {
	rc.persistStoreDataMutex.Lock()
	defer rc.persistStoreDataMutex.Unlock()

	data := &LiveTimingsPersistedData{
		SessionType: rc.SessionInfo.Type,
		Track:       rc.SessionInfo.Track,
		TrackLayout: rc.SessionInfo.TrackConfig,
		SessionName: rc.SessionInfo.Name,

		Drivers: rc.AllLapTimes(),
	}

	err := rc.store.UpsertLiveTimingsData(data)

	if err != nil {
		logrus.WithError(err).Errorf("Could not save live timings data")
	}

	logrus.Debug("successfully persisted live timing data")
}

func (rc *RaceControl) AllLapTimes() map[udp.DriverGUID]*RaceControlDriver {
	out := make(map[udp.DriverGUID]*RaceControlDriver)

	_ = rc.DisconnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
		out[driverGUID] = driver

		return nil
	})

	_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
		out[driverGUID] = driver

		return nil
	})

	return out
}

func (rc *RaceControl) LuaBroadcastChat(L *lua.LState) int {
	message := L.ToString(1)

	err := rc.splitAndBroadcastChat(message)

	if err != nil {
		logrus.WithError(err).Errorf("Unable to broadcast chat message")
		L.Push(lua.LBool(false))
	} else {
		L.Push(lua.LBool(true))
	}

	return 1
}

func (rc *RaceControl) LuaSendChat(L *lua.LState) int {
	message := L.ToString(1)
	guid := L.ToString(2)

	err := rc.splitAndSendChat(message, guid)

	if err != nil {
		logrus.WithError(err).Errorf("Unable to broadcast chat message")
		L.Push(lua.LBool(false))
	} else {
		L.Push(lua.LBool(true))
	}

	return 1
}

func (rc *RaceControl) splitAndBroadcastChat(message string) error {
	wrapped := strings.Split(wordwrap.WrapString(
		message,
		60,
	), "\n")

	for _, msg := range wrapped {
		broadcastMessage, err := udp.NewBroadcastChat(msg)

		if err == nil {
			err := rc.process.SendUDPMessage(broadcastMessage)

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (rc *RaceControl) splitAndSendChat(message, guid string) error {
	var carID uint8

	for id, rangeGUID := range rc.CarIDToGUID {
		if string(rangeGUID) == guid {
			carID = uint8(id)
			break
		}
	}

	wrapped := strings.Split(wordwrap.WrapString(
		message,
		60,
	), "\n")

	for _, msg := range wrapped {
		welcomeMessage, err := udp.NewSendChat(udp.CarID(carID), msg)

		if err == nil {
			err := rc.process.SendUDPMessage(welcomeMessage)

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func lapToDuration(i int) time.Duration {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", i))

	return d
}
