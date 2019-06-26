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
	Drivers                map[udp.DriverGUID]*RaceControlDriver `json:"Drivers"`
	GUIDsInPositionalOrder []udp.DriverGUID                      `json:"GUIDsInPositionalOrder"`

	driverSortLessFunc driverSortLessFunc
	driverGroup        RaceControlDriverGroup

	rwMutex sync.RWMutex
}

type RaceControlDriverGroup int

const (
	ConnectedDrivers    RaceControlDriverGroup = 0
	DisconnectedDrivers RaceControlDriverGroup = 1
)

type driverSortLessFunc func(group RaceControlDriverGroup, driverA, driverB *RaceControlDriver) bool

func NewDriverMap(driverGroup RaceControlDriverGroup, driverSortLessFunc driverSortLessFunc) *DriverMap {
	return &DriverMap{
		Drivers:            make(map[udp.DriverGUID]*RaceControlDriver),
		driverSortLessFunc: driverSortLessFunc,
		driverGroup:        driverGroup,
	}
}

func (d *DriverMap) Each(fn func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error) error {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	for _, guid := range d.GUIDsInPositionalOrder {
		driver, ok := d.Drivers[guid]

		if !ok {
			continue
		}

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
	defer d.sort()

	d.Drivers[driverGUID] = driver

	for _, guid := range d.GUIDsInPositionalOrder {
		if guid == driverGUID {
			return
		}
	}

	d.GUIDsInPositionalOrder = append(d.GUIDsInPositionalOrder, driverGUID)
}

func (d *DriverMap) sort() {
	sort.Slice(d.GUIDsInPositionalOrder, func(i, j int) bool {
		driverA, ok := d.Drivers[d.GUIDsInPositionalOrder[i]]

		if !ok {
			return false
		}

		driverB, ok := d.Drivers[d.GUIDsInPositionalOrder[j]]

		if !ok {
			return false
		}

		return d.driverSortLessFunc(d.driverGroup, driverA, driverB)
	})

	// correct positions
	for pos, guid := range d.GUIDsInPositionalOrder {
		driver, ok := d.Drivers[guid]

		if !ok {
			continue
		}

		driver.Position = pos + 1
	}
}

func (d *DriverMap) Del(driverGUID udp.DriverGUID) {
	d.rwMutex.Lock()
	defer d.rwMutex.Unlock()

	delete(d.Drivers, driverGUID)

	for index, guid := range d.GUIDsInPositionalOrder {
		if guid == driverGUID {
			d.GUIDsInPositionalOrder = append(d.GUIDsInPositionalOrder[:index], d.GUIDsInPositionalOrder[index+1:]...)
			break
		}
	}

	d.sort()
}

func (d *DriverMap) Len() int {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	return len(d.Drivers)
}

type TrackDataGateway interface {
	TrackInfo(name, layout string) (*TrackInfo, error)
	TrackMap(name, layout string) (*TrackMapData, error)
}

type filesystemTrackData struct{}

func (filesystemTrackData) TrackMap(name, layout string) (*TrackMapData, error) {
	return LoadTrackMapData(name, layout)
}

func (filesystemTrackData) TrackInfo(name, layout string) (*TrackInfo, error) {
	return GetTrackInfo(name, layout)
}

type RaceControl struct {
	SessionInfo      udp.SessionInfo `json:"SessionInfo"`
	TrackMapData     TrackMapData    `json:"TrackMapData"`
	TrackInfo        TrackInfo       `json:"TrackInfo"`
	SessionStartTime time.Time       `json:"SessionStartTime"`

	ConnectedDrivers    *DriverMap `json:"ConnectedDrivers"`
	DisconnectedDrivers *DriverMap `json:"DisconnectedDrivers"`

	CarIDToGUID map[udp.CarID]udp.DriverGUID `json:"CarIDToGUID"`

	sessionInfoTicker  *time.Ticker
	sessionInfoContext context.Context
	sessionInfoCfn     context.CancelFunc

	broadcaster      Broadcaster
	trackDataGateway TrackDataGateway
}

// RaceControl piggyback's on the udp.Message interface so that the entire data can be sent to newly connected clients.
func (rc *RaceControl) Event() udp.Event {
	return 200
}

func NewRaceControlDriver(carInfo udp.SessionCarInfo) *RaceControlDriver {
	driver := &RaceControlDriver{
		CarInfo:  carInfo,
		Cars:     make(map[string]*RaceControlCarLapInfo),
		LastSeen: time.Now(),
	}

	driver.Cars[carInfo.CarModel] = &RaceControlCarLapInfo{}

	return driver
}

type RaceControlDriver struct {
	CarInfo      udp.SessionCarInfo `json:"CarInfo"`
	TotalNumLaps int

	LoadedTime time.Time `json:"LoadedTime" ts:"date"`

	Position int       `json:"Position"`
	Split    string    `json:"Split"`
	LastSeen time.Time `json:"LastSeen" ts:"date"`

	Collisions []Collision `json:"Collisions"`

	// Cars is a map of CarModel to the information for that car.
	Cars map[string]*RaceControlCarLapInfo `json:"Cars"`
}

func (rcd *RaceControlDriver) CurrentCar() *RaceControlCarLapInfo {
	if car, ok := rcd.Cars[rcd.CarInfo.CarModel]; ok {
		return car
	} else {
		logrus.Warnf("Could not find current car for driver: %s (current car: %s)", rcd.CarInfo.DriverGUID, rcd.CarInfo.CarModel)
		return &RaceControlCarLapInfo{}
	}
}

type RaceControlCarLapInfo struct {
	TopSpeedThisLap      float64       `json:"TopSpeedThisLap"`
	TopSpeedBestLap      float64       `json:"TopSpeedBestLap"`
	BestLap              time.Duration `json:"BestLap"`
	NumLaps              int           `json:"NumLaps"`
	LastLap              time.Duration `json:"LastLap"`
	LastLapCompletedTime time.Time     `json:"LastLapCompletedTime" ts:"date"`
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

func NewRaceControl(broadcaster Broadcaster, trackDataGateway TrackDataGateway) *RaceControl {
	rc := &RaceControl{
		broadcaster:      broadcaster,
		trackDataGateway: trackDataGateway,
	}

	rc.clearAllDrivers()

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
		} else {
			err = rc.OnSessionUpdate(m)
		}

		sendUpdatedRaceControlStatus = true
	case udp.EndSession:
		err = rc.OnEndSession(m)

		sendUpdatedRaceControlStatus = true
	case udp.CarUpdate:
		sendUpdatedRaceControlStatus, err = rc.OnCarUpdate(m)
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
	case udp.CollisionWithEnvironment:
		err = rc.OnCollisionWithEnvironment(m)
	case udp.LapCompleted:
		err = rc.OnLapCompleted(m)

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
		err = rc.broadcaster.Send(rc)

		if err != nil {
			logrus.WithError(err).Error("Unable to broadcast race control message")
			return
		}
	}
}

// OnVersion occurs when the Assetto Corsa Server starts up for the first time.
func (rc *RaceControl) OnVersion(version udp.Version) error {
	return rc.broadcaster.Send(version)
}

// OnCarUpdate occurs every udp.RealTimePosInterval and returns car position, speed, etc.
// drivers top speeds are recorded per lap, as well as their last seen updated.
func (rc *RaceControl) OnCarUpdate(update udp.CarUpdate) (updatedRaceControl bool, err error) {
	driver, err := rc.findConnectedDriverByCarID(update.CarID)

	if err != nil {
		return updatedRaceControl, err
	}

	speed := metersPerSecondToKilometersPerHour(
		math.Sqrt(math.Pow(float64(update.Velocity.X), 2) + math.Pow(float64(update.Velocity.Z), 2)),
	)

	if speed > driver.CurrentCar().TopSpeedThisLap {
		driver.CurrentCar().TopSpeedThisLap = speed
		updatedRaceControl = true
	}

	driver.LastSeen = time.Now()

	return updatedRaceControl, rc.broadcaster.Send(update)
}

// OnNewSession occurs every new session. If the session is the first in an event and it is not a looped practice,
// then all driver information is cleared.
func (rc *RaceControl) OnNewSession(sessionInfo udp.SessionInfo) error {
	oldSessionInfo := rc.SessionInfo
	rc.SessionInfo = sessionInfo
	rc.SessionStartTime = time.Now()

	deleteCars := true
	emptyCarInfo := false

	if sessionInfo.CurrentSessionIndex != 0 || oldSessionInfo.Track == sessionInfo.Track && oldSessionInfo.TrackConfig == sessionInfo.TrackConfig {
		// only remove cars on the first session (avoid deleting between practice/qualify/race)
		deleteCars = false
		emptyCarInfo = true
	}

	if rc.ConnectedDrivers.Len() > 0 || rc.DisconnectedDrivers.Len() > 0 && sessionInfo.Type == udp.SessionTypePractice {
		if oldSessionInfo.Type == sessionInfo.Type && oldSessionInfo.Track == sessionInfo.Track && oldSessionInfo.TrackConfig == sessionInfo.TrackConfig && oldSessionInfo.Name == sessionInfo.Name {
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

	// clear out last lap completed time each new session
	_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
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
		return err
	}

	rc.TrackMapData = *trackMapData

	logrus.Infof("New session detected: %s at %s (%s)", sessionInfo.Type.String(), sessionInfo.Track, sessionInfo.TrackConfig)

	go rc.requestSessionInfo()

	return rc.broadcaster.Send(sessionInfo)
}

// clearAllDrivers removes all known information about connected and disconnected drivers from RaceControl
func (rc *RaceControl) clearAllDrivers() {
	rc.ConnectedDrivers = NewDriverMap(ConnectedDrivers, rc.sort)
	rc.DisconnectedDrivers = NewDriverMap(DisconnectedDrivers, rc.sort)
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

			logrus.Debugf("Assetto Process completed. Disconnecting all connected drivers. Session done.")

			var drivers []*RaceControlDriver

			// the server has just stopped. send disconnect messages for all connected cars.
			_ = rc.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
				// Each takes a read lock, so we cannot call disconnectDriver (which takes a write lock) from inside it.
				// we must instead append them to a slice and disconnect them outisde the Each call.
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
		case <-rc.sessionInfoContext.Done():
			rc.sessionInfoTicker.Stop()
			return
		}
	}
}

func (rc *RaceControl) disconnectDriver(driver *RaceControlDriver) error {
	carInfo := driver.CarInfo
	carInfo.EventType = udp.EventConnectionClosed
	return rc.OnClientDisconnect(carInfo)
}

// driverLastSeenMaxDuration is how long to wait before considering a driver 'timed out'. A timed out driver
// is force-disconnected.
var driverLastSeenMaxDuration = time.Second * 5

// OnSessionUpdate is called every sessionRequestInterval.
func (rc *RaceControl) OnSessionUpdate(sessionInfo udp.SessionInfo) error {
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

// OnEndSession is called at the end of every session.
func (rc *RaceControl) OnEndSession(sessionFile udp.EndSession) error {
	if rc.sessionInfoCfn != nil {
		rc.sessionInfoCfn()
	}

	return nil
}

// OnClientConnect stores CarID -> DriverGUID mappings. if a driver is known to have previously been in this event,
// they will be moved from DisconnectedDrivers to ConnectedDrivers.
func (rc *RaceControl) OnClientConnect(client udp.SessionCarInfo) error {
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

	if _, ok := driver.Cars[driver.CarInfo.CarModel]; !ok {
		driver.Cars[driver.CarInfo.CarModel] = &RaceControlCarLapInfo{}
	}

	driver.CurrentCar().LastLapCompletedTime = time.Now()

	rc.ConnectedDrivers.Add(driver.CarInfo.DriverGUID, driver)

	return rc.broadcaster.Send(client)
}

// OnClientDisconnect moves a client from ConnectedDrivers to DisconnectedDrivers.
func (rc *RaceControl) OnClientDisconnect(client udp.SessionCarInfo) error {
	driver, ok := rc.ConnectedDrivers.Get(client.DriverGUID)

	if !ok {
		return fmt.Errorf("racecontrol: client disconnected without ever being connected: %s (%s)", client.DriverName, client.DriverGUID)
	}

	logrus.Debugf("Driver %s (%s) disconnected", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID)

	rc.ConnectedDrivers.Del(driver.CarInfo.DriverGUID)

	if driver.TotalNumLaps > 0 {
		rc.DisconnectedDrivers.Add(driver.CarInfo.DriverGUID, driver)
	}

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

// OnClientLoaded marks a connected client as having loaded in.
func (rc *RaceControl) OnClientLoaded(loadedCar udp.ClientLoaded) error {
	driver, err := rc.findConnectedDriverByCarID(udp.CarID(loadedCar))

	if err != nil {
		return err
	}

	logrus.Debugf("Driver: %s (%s) loaded", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID)

	driver.LoadedTime = time.Now()

	return rc.broadcaster.Send(loadedCar)
}

// OnLapCompleted occurs every time a driver crosses the line. Lap information is collected for the driver
// and best lap time and top speed are calculated. OnLapCompleted also remembers the car the lap was completed in
// a PreviousCars map on the driver. This is so that lap times between different cars can be compared.
func (rc *RaceControl) OnLapCompleted(lap udp.LapCompleted) error {
	driver, err := rc.findConnectedDriverByCarID(lap.CarID)

	if err != nil {
		return err
	}

	lapDuration := lapToDuration(int(lap.LapTime))

	logrus.Debugf("Lap completed by driver: %s (%s), %s", driver.CarInfo.DriverName, driver.CarInfo.DriverGUID, lapDuration)

	driver.TotalNumLaps++
	currentCar := driver.CurrentCar()

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
						driver.Split = driverCar.LastLapCompletedTime.Sub(otherDriverCar.LastLapCompletedTime).Round(time.Millisecond).String()
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

func (rc *RaceControl) sort(driverGroup RaceControlDriverGroup, driverA, driverB *RaceControlDriver) bool {
	driverACar := driverA.CurrentCar()
	driverBCar := driverB.CurrentCar()

	if driverGroup == ConnectedDrivers {
		if rc.SessionInfo.Type == udp.SessionTypeRace {
			return driverACar.LastLapCompletedTime.Before(driverBCar.LastLapCompletedTime) && driverACar.NumLaps >= driverBCar.NumLaps
		} else {
			if driverACar.BestLap == 0 && driverBCar.BestLap == 0 {
				if driverACar.NumLaps == driverBCar.NumLaps {
					return driverACar.LastLapCompletedTime.Before(driverBCar.LastLapCompletedTime)
				} else {
					return driverACar.NumLaps > driverBCar.NumLaps
				}

			}

			if driverACar.BestLap == 0 {
				return false
			} else if driverBCar.BestLap == 0 {
				return true
			}

			return driverACar.BestLap < driverBCar.BestLap
		}
	} else if driverGroup == DisconnectedDrivers {
		// disconnected
		if rc.SessionInfo.Type == udp.SessionTypeRace {
			return driverACar.LastLapCompletedTime.After(driverBCar.LastLapCompletedTime)
		} else {
			return driverACar.BestLap < driverBCar.BestLap
		}
	} else {
		panic("unknown driver group")
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

// OnCollisionWithEnvironment registers a driver's collision with the environment.
func (rc *RaceControl) OnCollisionWithEnvironment(collision udp.CollisionWithEnvironment) error {
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

func lapToDuration(i int) time.Duration {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", i))

	return d
}
