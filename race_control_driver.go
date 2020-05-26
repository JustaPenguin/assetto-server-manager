package servermanager

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"

	"github.com/sirupsen/logrus"
)

func NewRaceControlDriver(carInfo udp.SessionCarInfo) *RaceControlDriver {
	driver := &RaceControlDriver{
		CarInfo:  carInfo,
		Cars:     make(map[string]*RaceControlCarLapInfo),
		LastSeen: time.Now(),
	}

	driver.Cars[carInfo.CarModel] = NewRaceControlCarLapInfo(carInfo.CarModel)

	return driver
}

func NewRaceControlCarLapInfo(carModel string) *RaceControlCarLapInfo {
	return &RaceControlCarLapInfo{
		CarName: prettifyName(carModel, true),
	}
}

type RaceControlDriver struct {
	CarInfo      udp.SessionCarInfo `json:"CarInfo"`
	TotalNumLaps int                `json:"TotalNumLaps"`

	ConnectedTime time.Time `json:"ConnectedTime" ts:"date"`
	LoadedTime    time.Time `json:"LoadedTime" ts:"date"`

	Position int       `json:"Position"`
	Split    string    `json:"Split"`
	LastSeen time.Time `json:"LastSeen" ts:"date"`
	LastPos  udp.Vec   `json:"LastPos"`

	Collisions []Collision `json:"Collisions"`

	driverSwapContext context.Context
	driverSwapCfn     context.CancelFunc

	// Cars is a map of CarModel to the information for that car.
	Cars map[string]*RaceControlCarLapInfo `json:"Cars"`

	mutex sync.Mutex
}

func (rcd *RaceControlDriver) CurrentCar() *RaceControlCarLapInfo {
	if car, ok := rcd.Cars[rcd.CarInfo.CarModel]; ok {
		return car
	}

	logrus.Warnf("Could not find current car for driver: %s (current car: %s)", rcd.CarInfo.DriverGUID, rcd.CarInfo.CarModel)
	return &RaceControlCarLapInfo{}
}

type RaceControlCarLapInfo struct {
	TopSpeedThisLap      float64       `json:"TopSpeedThisLap"`
	TopSpeedBestLap      float64       `json:"TopSpeedBestLap"`
	BestLap              time.Duration `json:"BestLap"`
	NumLaps              int           `json:"NumLaps"`
	LastLap              time.Duration `json:"LastLap"`
	LastLapCompletedTime time.Time     `json:"LastLapCompletedTime" ts:"date"`
	TotalLapTime         time.Duration `json:"TotalLapTime"`
	CarName              string        `json:"CarName"`
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
