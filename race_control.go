package servermanager

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/sirupsen/logrus"
)

// @TODO rename me
var RaceControlInst *RaceControl

type Broadcaster interface {
	Send(message udp.Message) error
}

type NilBroadcaster struct{}

func (NilBroadcaster) Send(message udp.Message) error {
	logrus.WithField("message", message).Infof("Message send %d", message.Event())
	return nil
}

type DriverMap struct {
	Drivers map[udp.DriverGUID]*RaceControlDriver `json:"Drivers"`

	rwMutex sync.RWMutex
}

func NewDriverMap() *DriverMap {
	return &DriverMap{Drivers: make(map[udp.DriverGUID]*RaceControlDriver)}
}

func (d *DriverMap) Each(fn func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error) error {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	for guid, driver := range d.Drivers {
		guid := guid
		driver := driver
		err := fn(guid, driver)

		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DriverMap) Get(driverGUID udp.DriverGUID) (*RaceControlDriver, bool) {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	driver, ok := d.Drivers[driverGUID]

	return driver, ok
}

func (d *DriverMap) Add(driverGUID udp.DriverGUID, driver *RaceControlDriver) {
	d.rwMutex.Lock()
	defer d.rwMutex.Unlock()

	d.Drivers[driverGUID] = driver
}

func (d *DriverMap) Del(driverGUID udp.DriverGUID) {
	d.rwMutex.Lock()
	defer d.rwMutex.Unlock()

	delete(d.Drivers, driverGUID)
}

func (d *DriverMap) Len() int {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	return len(d.Drivers)
}

type RaceControl struct {
	SessionInfo        udp.SessionInfo `json:"SessionInfo"`
	CurrentSessionType udp.SessionType `json:"CurrentSessionType"`
	TrackMapData       *TrackMapData   `json:"TrackMapData"`
	TrackInfo          *TrackInfo      `json:"TrackInfo"`

	ConnectedDrivers    *DriverMap `json:"ConnectedDrivers"`
	DisconnectedDrivers *DriverMap `json:"DisconnectedDrivers"`

	GUIDsInPositionalOrder []udp.DriverGUID `json:"GUIDsInPositionalOrder"`
	CarIDToGUID            map[udp.CarID]udp.DriverGUID

	sessionInfoTicker  *time.Ticker
	sessionInfoContext context.Context
	sessionInfoCfn     context.CancelFunc

	broadcaster Broadcaster
}

// RaceControl piggyback's on the udp.Message interface so that the entire data can be sent to newly connected clients.
func (rc *RaceControl) Event() udp.Event {
	return 200
}

func NewRaceControlDriver(carInfo udp.SessionCarInfo) *RaceControlDriver {
	return &RaceControlDriver{
		CarInfo:      carInfo,
		PreviousCars: make(map[string]RaceControlCarBestLap),
		LastSeen:     time.Now(),
	}
}

type RaceControlDriver struct {
	CarInfo udp.SessionCarInfo `json:"CarInfo"`

	LoadedTime time.Time `json:"LoadedTime" ts:"date"`

	// Lap Info
	NumLaps              int           `json:"NumLaps"`
	LastLap              time.Duration `json:"LastLap"`
	BestLap              time.Duration `json:"BestLap"`
	LastLapCompletedTime time.Time     `json:"LastLapCompletedTime" ts:"date"`

	Position        int       `json:"Position"`
	Split           string    `json:"Split"`
	TopSpeedThisLap float64   `json:"TopSpeedThisLap"`
	TopSpeedBestLap float64   `json:"TopSpeedBestLap"`
	LastSeen        time.Time `json:"LastSeen" ts:"date"`

	Collisions []Collision `json:"Collisions"`

	// PreviousCars is a map of CarModel to the best lap of that car
	PreviousCars map[string]RaceControlCarBestLap `json:"PreviousCars"`
}

type RaceControlCarBestLap struct {
	TopSpeedBestLap float64       `json:"TopSpeedBestLap"`
	BestLap         time.Duration `json:"BestLap"`
}

type CollisionType string

const (
	CollisionWithCar         CollisionType = "with other car"
	CollisionWithEnvironment CollisionType = "with environment"
)

type Collision struct {
	Type            CollisionType  `json:"Type"`
	Time            time.Time      `json:"Time" ts:"date"`
	OtherDriverGUID udp.DriverGUID `json:"OtherDriverGUID"`
	Speed           float64        `json:"Speed"`
}

func NewRaceControl(broadcaster Broadcaster) *RaceControl {
	return &RaceControl{
		broadcaster: broadcaster,

		CarIDToGUID: make(map[udp.CarID]udp.DriverGUID),

		ConnectedDrivers:    NewDriverMap(),
		DisconnectedDrivers: NewDriverMap(),
	}
}

func (rc *RaceControl) UDPCallback(message udp.Message) {
	var err error

	sendUpdatedRaceControlStatus := false

	switch m := message.(type) {
	case udp.Version:
		err = rc.onVersion(m)
	case udp.SessionInfo:
		if m.Event() == udp.EventNewSession {
			err = rc.onNewSession(m)
		} else {
			err = rc.onSessionUpdate(m)
		}

		sendUpdatedRaceControlStatus = true
	case udp.EndSession:
		err = rc.onEndSession(m)

		sendUpdatedRaceControlStatus = true
	case udp.CarUpdate:
		sendUpdatedRaceControlStatus, err = rc.onCarUpdate(m)
	case udp.SessionCarInfo:
		if m.Event() == udp.EventNewConnection {
			err = rc.onClientConnect(m)
		} else if m.Event() == udp.EventConnectionClosed {
			err = rc.onClientDisconnect(m)
		}

		sendUpdatedRaceControlStatus = true
	case udp.ClientLoaded:
		err = rc.onClientLoaded(m)

		sendUpdatedRaceControlStatus = true
	case udp.CollisionWithCar:
		err = rc.onCollisionWithCar(m)
	case udp.CollisionWithEnvironment:
		err = rc.onCollisionWithEnvironment(m)
	case udp.LapCompleted:
		err = rc.onLapCompleted(m)

		sendUpdatedRaceControlStatus = true
	default:
		// unhandled event
		return
	}

	if err != nil {
		logrus.WithError(err).Errorf("Unable to handle event: %d", message.Event())
		return
	}

	if sendUpdatedRaceControlStatus {
		// broadcast the race control deets
		rc.broadcaster.Send(rc)
	}
}

// onVersion occurs when the Assetto Corsa Server starts up for the first time.
func (rc *RaceControl) onVersion(version udp.Version) error {
	_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
		rc.DisconnectedDrivers.Add(driverGUID, driver)

		return nil
	})

	rc.ConnectedDrivers = NewDriverMap()

	return rc.broadcaster.Send(version)
}

// onCarUpdate occurs every udp.RealTimePosInterval and returns car position, speed, etc.
// drivers top speeds are recorded per lap, as well as their last seen updated.
func (rc *RaceControl) onCarUpdate(update udp.CarUpdate) (updatedRaceControl bool, err error) {
	driver, err := rc.findConnectedDriverByCarID(update.CarID)

	if err != nil {
		return updatedRaceControl, err
	}

	speed := metersPerSecondToKilometersPerHour(
		math.Sqrt(math.Pow(float64(update.Velocity.X), 2) + math.Pow(float64(update.Velocity.Z), 2)),
	)

	if speed > driver.TopSpeedThisLap {
		driver.TopSpeedThisLap = speed
		updatedRaceControl = true
	}

	driver.LastSeen = time.Now()

	return updatedRaceControl, rc.broadcaster.Send(update)
}

// onNewSession occurs every new session. If the session is the first in an event and it is not a looped practice,
// then all driver information is cleared.
func (rc *RaceControl) onNewSession(sessionInfo udp.SessionInfo) error {
	oldSessionInfo := rc.SessionInfo
	rc.SessionInfo = sessionInfo

	deleteCars := false
	emptyCarInfo := false

	if sessionInfo.CurrentSessionIndex != 0 || oldSessionInfo.Track == sessionInfo.Track && oldSessionInfo.TrackConfig == sessionInfo.TrackConfig {
		// only remove cars on the first session (avoid deleting between practice/qualify/race)
		deleteCars = false
		emptyCarInfo = true
	}

	if rc.ConnectedDrivers.Len() > 0 || rc.DisconnectedDrivers.Len() > 0 && sessionInfo.SessionType == udp.SessionTypePractice {
		if oldSessionInfo.SessionType == sessionInfo.SessionType && oldSessionInfo.Track == sessionInfo.Track && oldSessionInfo.TrackConfig == sessionInfo.TrackConfig && oldSessionInfo.Name == sessionInfo.Name {
			// this is a looped practice event, keep the cars
			deleteCars = false
			emptyCarInfo = false
		}
	}

	if deleteCars {
		rc.clearAllDrivers()
	}

	if emptyCarInfo {
		_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
			*driver = *NewRaceControlDriver(driver.CarInfo)

			return nil
		})

		_ = rc.DisconnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
			*driver = *NewRaceControlDriver(driver.CarInfo)

			return nil
		})
	}

	var err error

	rc.TrackInfo, err = GetTrackInfo(sessionInfo.Track, sessionInfo.TrackConfig)

	if err != nil {
		return err
	}

	rc.TrackMapData, err = LoadTrackMapData(sessionInfo.Track, sessionInfo.TrackConfig)

	if err != nil {
		return err
	}

	logrus.Infof("New session detected: %s at %s (%s)", sessionInfo.SessionType.String(), sessionInfo.Track, sessionInfo.TrackConfig)

	go rc.requestSessionInfo()

	return rc.broadcaster.Send(sessionInfo)
}

// clearAllDrivers removes all known information about connected and disconnected drivers from RaceControl
func (rc *RaceControl) clearAllDrivers() {
	rc.ConnectedDrivers = NewDriverMap()
	rc.DisconnectedDrivers = NewDriverMap()
	rc.GUIDsInPositionalOrder = []udp.DriverGUID{}
	rc.CarIDToGUID = make(map[udp.CarID]udp.DriverGUID)
}

var sessionInfoRequestInterval = time.Second

// requestSessionInfo sends a request every sessionInfoRequestInterval to get information about temps, etc in the session.
func (rc *RaceControl) requestSessionInfo() {
	if rc.sessionInfoTicker != nil {
		rc.sessionInfoTicker.Stop()
	}

	rc.sessionInfoTicker = time.NewTicker(sessionInfoRequestInterval)
	rc.sessionInfoContext, rc.sessionInfoCfn = context.WithCancel(context.Background())

	for {
		select {
		case <-rc.sessionInfoTicker.C:
			err := AssettoProcess.SendUDPMessage(udp.GetSessionInfo{})

			if err == ErrNoOpenUDPConnection {
				logrus.WithError(err).Errorf("Couldn't send session info udp request. Breaking loop.")
				rc.sessionInfoTicker.Stop()
				return
			} else if err != nil {
				logrus.WithError(err).Errorf("Couldn't send session info udp request")
			}

		case <-AssettoProcess.Done():
			rc.sessionInfoTicker.Stop()

			// the server has just stopped. send disconnect messages for all connected cars.
			_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
				// disconnect the driver
				err := rc.disconnectDriver(driver)

				if err != nil {
					logrus.WithError(err).Errorf("Could not disconnect driver: %s (%s)", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID)
					return nil
				}

				return nil
			})
		case <-rc.sessionInfoContext.Done():
			rc.sessionInfoTicker.Stop()
			return
		}
	}
}

func (rc *RaceControl) disconnectDriver(driver *RaceControlDriver) error {
	carInfo := driver.CarInfo
	carInfo.EventType = udp.EventConnectionClosed
	return rc.onClientDisconnect(carInfo)
}

// driverLastSeenMaxDuration is how long to wait before considering a driver 'timed out'. A timed out driver
// is force-disconnected.
var driverLastSeenMaxDuration = time.Second * 5

// onSessionUpdate is called every sessionRequestInterval.
func (rc *RaceControl) onSessionUpdate(sessionInfo udp.SessionInfo) error {
	rc.SessionInfo = sessionInfo

	if udp.RealtimePosIntervalMs > 0 {
		var driversToDisconnect []*RaceControlDriver

		_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
			if time.Now().Sub(driver.LastSeen) > driverLastSeenMaxDuration {
				driversToDisconnect = append(driversToDisconnect, driver)
			}

			return nil
		})

		for _, driver := range driversToDisconnect {
			logrus.Infof("Driver: %s (%s) has not been seen in %s. Forcing a disconnect message.", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID, driverLastSeenMaxDuration)

			// disconnect the driver
			err := rc.disconnectDriver(driver)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// onEndSession is called at the end of every session.
func (rc *RaceControl) onEndSession(sessionFile udp.EndSession) error {
	rc.sessionInfoCfn()

	return nil
}

// onClientConnect stores CarID -> DriverGUID mappings. if a driver is known to have previously been in this event,
// they will be moved from DisconnectedDrivers to ConnectedDrivers.
func (rc *RaceControl) onClientConnect(client udp.SessionCarInfo) error {
	rc.CarIDToGUID[client.CarID] = client.DriverGUID

	var driver *RaceControlDriver

	client.DriverName = driverName(client.DriverName)

	if disconnectedDriver, ok := rc.DisconnectedDrivers.Get(client.DriverGUID); ok {
		driver = disconnectedDriver
		driver.CarInfo = client
		logrus.Debugf("Driver %s (%s) reconnected in %s (car id: %d)", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID, driver.CarInfo.CarModel, client.CarID)
		rc.DisconnectedDrivers.Del(client.DriverGUID)
	} else {
		driver = NewRaceControlDriver(client)
		logrus.Debugf("Driver %s (%s) connected in %s (car id: %d)", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID, driver.CarInfo.CarModel, client.CarID)
	}

	rc.ConnectedDrivers.Add(driver.CarInfo.DriverGUID, driver)

	return rc.broadcaster.Send(client)
}

// onClientDisconnect moves a client from ConnectedDrivers to DisconnectedDrivers.
func (rc *RaceControl) onClientDisconnect(client udp.SessionCarInfo) error {
	driver, ok := rc.ConnectedDrivers.Get(client.DriverGUID)

	if !ok {
		return fmt.Errorf("racecontrol: client disconnected without ever being connected: %s (%s)", client.DriverName, client.DriverGUID)
	}

	logrus.Debugf("Driver %s (%s) disconnected", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID)

	rc.ConnectedDrivers.Del(driver.CarInfo.DriverGUID)
	rc.DisconnectedDrivers.Add(driver.CarInfo.DriverGUID, driver)

	return rc.broadcaster.Send(client)
}

// findConnectedDriverByCarID looks for a driver in ConnectedDrivers by their CarID. This is the only place CarID
// is used for a look-up, and it uses the CarIDToGUID map to perform the lookup.
func (rc *RaceControl) findConnectedDriverByCarID(carID udp.CarID) (*RaceControlDriver, error) {
	driverGUID, ok := rc.CarIDToGUID[carID]

	if !ok {
		return nil, fmt.Errorf("racecontrol: could not find DriverGUID for CarID: %d", carID)
	}

	driver, ok := rc.ConnectedDrivers.Get(driverGUID)

	if !ok {
		return nil, fmt.Errorf("racecontrol: could not find connected driver for DriverGUID: %s", driverGUID)
	}

	return driver, nil
}

// onClientLoaded marks a connected client as having loaded in.
func (rc *RaceControl) onClientLoaded(loadedCar udp.ClientLoaded) error {
	driver, err := rc.findConnectedDriverByCarID(udp.CarID(loadedCar))

	if err != nil {
		return err
	}

	driver.LoadedTime = time.Now()

	return rc.broadcaster.Send(loadedCar)
}

// onLapCompleted occurs every time a driver crosses the line. Lap information is collected for the driver
// and best lap time and top speed are calculated. onLapCompleted also remembers the car the lap was completed in
// a PreviousCars map on the driver. This is so that lap times between different cars can be compared.
func (rc *RaceControl) onLapCompleted(lap udp.LapCompleted) error {
	driver, err := rc.findConnectedDriverByCarID(lap.CarID)

	if err != nil {
		return err
	}

	lapDuration := lapToDuration(int(lap.LapTime))

	driver.LastLap = lapDuration
	driver.NumLaps++
	driver.LastLapCompletedTime = time.Now()

	if lap.Cuts == 0 && (lapDuration < driver.BestLap || driver.BestLap == 0) {
		driver.BestLap = lapDuration
		driver.TopSpeedBestLap = driver.TopSpeedThisLap

		previousCar, ok := driver.PreviousCars[driver.CarInfo.CarModel]

		if ok && lapDuration < previousCar.BestLap {
			previousCar.BestLap = driver.BestLap
			previousCar.TopSpeedBestLap = driver.TopSpeedBestLap
		} else if !ok {
			driver.PreviousCars[driver.CarInfo.CarModel] = RaceControlCarBestLap{
				BestLap:         driver.BestLap,
				TopSpeedBestLap: driver.TopSpeedBestLap,
			}
		}
	}

	driver.TopSpeedThisLap = 0

	if rc.CurrentSessionType == udp.SessionTypeRace {
		// calculate driver position
		position := 1

		_ = rc.ConnectedDrivers.Each(func(otherDriverGUID udp.DriverGUID, otherDriver *RaceControlDriver) error {
			if otherDriverGUID == driver.CarInfo.DriverGUID {
				return nil // continue
			}

			if otherDriver.LastLapCompletedTime.Before(driver.LastLapCompletedTime) && otherDriver.NumLaps >= driver.NumLaps {
				position++
			}

			return nil
		})

		driver.Position = position

		// calculate split
		if driver.Position == 1 {
			driver.Split = time.Duration(0).String()
		} else {
			_ = rc.ConnectedDrivers.Each(func(otherDriverGUID udp.DriverGUID, otherDriver *RaceControlDriver) error {
				if otherDriver.Position == driver.Position-1 {
					driver.Split = time.Since(otherDriver.LastLapCompletedTime).Round(time.Millisecond).String()
				} else {
					lapDifference := otherDriver.NumLaps - driver.NumLaps

					if lapDifference == 1 {
						driver.Split = "1 lap"
					} else {
						driver.Split = fmt.Sprintf("%d laps", lapDifference)
					}
				}

				return nil
			})
		}
	}

	rc.GUIDsInPositionalOrder = []udp.DriverGUID{}

	// sort all driver GUIDs every lap complete by position or best lap, depending on session type

	_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
		rc.GUIDsInPositionalOrder = append(rc.GUIDsInPositionalOrder, driverGUID)
		return nil
	})

	sort.Slice(rc.GUIDsInPositionalOrder, func(i, j int) bool {
		driverA, driverAOK := rc.ConnectedDrivers.Get(rc.GUIDsInPositionalOrder[i])
		driverB, driverBOK := rc.ConnectedDrivers.Get(rc.GUIDsInPositionalOrder[j])

		if rc.CurrentSessionType == udp.SessionTypeRace {
			return driverAOK && driverBOK && driverA.Position < driverB.Position
		} else {
			return driverAOK && driverBOK && driverA.BestLap < driverB.BestLap
		}
	})

	if rc.CurrentSessionType == udp.SessionTypeQualifying || rc.CurrentSessionType == udp.SessionTypePractice {
		for pos, driverGUID := range rc.GUIDsInPositionalOrder {
			driver, ok := rc.ConnectedDrivers.Get(driverGUID)

			if !ok {
				continue
			}

			driver.Position = pos + 1

			if pos == 0 || driver.Position == 1 {
				driver.Split = time.Duration(0).String()
			} else {
				driverAhead, ok := rc.ConnectedDrivers.Get(rc.GUIDsInPositionalOrder[pos-1])

				if !ok {
					continue
				}

				driver.Split = (driver.BestLap - driverAhead.BestLap).String()
			}
		}
	}

	return nil
}

func metersPerSecondToKilometersPerHour(mps float64) float64 {
	return mps * 3.6
}

// onCollisionWithCar registers a driver's collision with another car.
func (rc *RaceControl) onCollisionWithCar(collision udp.CollisionWithCar) error {
	driver, err := rc.findConnectedDriverByCarID(collision.CarID)

	if err != nil {
		return err
	}

	c := Collision{
		Type:  CollisionWithCar,
		Time:  time.Now(),
		Speed: metersPerSecondToKilometersPerHour(float64(collision.ImpactSpeed)),
	}

	otherDriver, err := rc.findConnectedDriverByCarID(collision.OtherCarID)

	if err == nil {
		c.OtherDriverGUID = otherDriver.CarInfo.DriverGUID
	}

	driver.Collisions = append(driver.Collisions, c)

	return rc.broadcaster.Send(collision)
}

// onCollisionWithEnvironment registers a driver's collision with the environment.
func (rc *RaceControl) onCollisionWithEnvironment(collision udp.CollisionWithEnvironment) error {
	driver, err := rc.findConnectedDriverByCarID(collision.CarID)

	if err != nil {
		return err
	}

	driver.Collisions = append(driver.Collisions, Collision{
		Type:  CollisionWithEnvironment,
		Time:  time.Now(),
		Speed: metersPerSecondToKilometersPerHour(float64(collision.ImpactSpeed)),
	})

	return rc.broadcaster.Send(collision)
}
