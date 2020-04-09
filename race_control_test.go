package servermanager

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"
)

var testStore = NewJSONStore(filepath.Join(os.TempDir(), "asm-race-store"), filepath.Join(os.TempDir(), "asm-race-store-shared"))

var drivers = []udp.SessionCarInfo{
	{
		CarID:      1,
		DriverName: "Test 1",
		DriverGUID: "7827162738272615",
		CarModel:   "ford_gt",
		CarSkin:    "red_01",
		EventType:  udp.EventNewConnection,
	},
	{
		CarID:      2,
		DriverName: "Test 2",
		DriverGUID: "7827162738272616",
		CarModel:   "ferrari_fxxk",
		CarSkin:    "purple_02",
		EventType:  udp.EventNewConnection,
	},
	{
		CarID:      3,
		DriverName: "Test 3",
		DriverGUID: "7827162738272617",
		CarModel:   "ferrari_fxxk",
		CarSkin:    "orange_33",
		EventType:  udp.EventNewConnection,
	},
	{
		CarID:      4,
		DriverName: "Test 3",
		DriverGUID: "7827162738272619",
		CarModel:   "car_model3",
		CarSkin:    "green",
		EventType:  udp.EventNewConnection,
	},
	{
		CarID:      9,
		DriverName: "Test 3",
		DriverGUID: "7827162738278889",
		CarModel:   "car_model3",
		CarSkin:    "green",
		EventType:  udp.EventNewConnection,
	},
}

func TestRaceControl_OnClientConnect(t *testing.T) {
	t.Run("Client first connect", func(t *testing.T) {
		// on first connect, a client is added to connected drivers but does not yet have a loaded time.
		// their GUID is added to the CarID -> GUID map for future lookup
		raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

		err := raceControl.OnClientConnect(drivers[0])

		if err != nil {
			t.Error(err)
			return
		}

		if guid, ok := raceControl.CarIDToGUID[drivers[0].CarID]; !ok || guid != drivers[0].DriverGUID {
			t.Errorf("Driver was not correctly added to CarID -> GUID map")
			return
		}

		driver, ok := raceControl.ConnectedDrivers.Get(drivers[0].DriverGUID)

		if !ok {
			t.Errorf("Driver was not correctly added to ConnectedDrivers")
			return
		}

		if !driver.LoadedTime.IsZero() {
			t.Errorf("Driver has loaded time when it should be zero")
			return
		}

		t.Run("Client disconnects and reconnects having done no laps", func(t *testing.T) {
			// disconnect the driver
			err := raceControl.OnClientDisconnect(drivers[0])

			if err != nil {
				t.Error(err)
				return
			}

			// assert that the driver has been disconnected
			if _, ok := raceControl.ConnectedDrivers.Get(drivers[0].DriverGUID); ok {
				t.Errorf("Driver should have been disconnected, was not. (present in ConnectedDrivers)")
				return
			}

			if _, ok := raceControl.DisconnectedDrivers.Get(drivers[0].DriverGUID); ok {
				t.Error("Driver should have been disconnected, was not. (present in DisconnectedDrivers, but no laps completed)")
				return
			}

			// reconnect!
			err = raceControl.OnClientConnect(drivers[0])

			if err != nil {
				t.Error(err)
				return
			}

			// assert that the driver has been reconnected
			if _, ok := raceControl.ConnectedDrivers.Get(drivers[0].DriverGUID); !ok {
				t.Error("Driver should have been connected, was not. (not present in ConnectedDrivers)")
				return
			}

			if _, ok := raceControl.DisconnectedDrivers.Get(drivers[0].DriverGUID); ok {
				t.Error("Driver should have been connected, was not. (present in DisconnectedDrivers)")
				return
			}
		})

		t.Run("Client disconnects and reconnects having done some laps", func(t *testing.T) {
			err := raceControl.OnLapCompleted(udp.LapCompleted{
				CarID:   drivers[0].CarID,
				LapTime: 10000,
				Cuts:    1,
			})

			if err != nil {
				t.Error(err)
				return
			}

			// disconnect the driver
			err = raceControl.OnClientDisconnect(drivers[0])

			if err != nil {
				t.Error(err)
				return
			}

			// assert that the driver has been disconnected
			if _, ok := raceControl.ConnectedDrivers.Get(drivers[0].DriverGUID); ok {
				t.Error("Driver should have been disconnected, was not. (present in ConnectedDrivers)")
				return
			}

			if _, ok := raceControl.DisconnectedDrivers.Get(drivers[0].DriverGUID); !ok {
				t.Error("Driver should have been disconnected, was not. (not present in DisconnectedDrivers, has completed laps)")
				return
			}

			driver := drivers[0]
			driver.CarModel = "newCar"

			// reconnect!
			err = raceControl.OnClientConnect(driver)

			if err != nil {
				t.Error(err)
				return
			}

			connectedDriver, connected := raceControl.ConnectedDrivers.Get(drivers[0].DriverGUID)

			// assert that the driver has been reconnected
			if !connected {
				t.Error("Driver should have been connected, was not. (not present in ConnectedDrivers)")
				return
			}

			if len(connectedDriver.Cars) != 2 {
				t.Error("Expected connected driver to have two cars")
				return
			}

			if _, ok := raceControl.DisconnectedDrivers.Get(drivers[0].DriverGUID); ok {
				t.Error("Driver should have been connected, was not. (present in DisconnectedDrivers)")
				return
			}
		})
	})

	t.Run("Client disconnects having never connected", func(t *testing.T) {
		raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

		// disconnect the driver
		driver := drivers[0]
		driver.CarID = 200 // unknown car id

		err := raceControl.OnClientDisconnect(driver)

		if err == nil {
			t.Error("Expected an error due to an unknown driver, but none was present")
			return
		}
	})
}

func TestRaceControl_OnClientLoaded(t *testing.T) {
	raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

	for _, driverIndex := range []int{1, 2, 3} {
		err := raceControl.OnClientConnect(drivers[driverIndex])

		if err != nil {
			t.Error(err)
			return
		}
	}

	_ = raceControl.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
		switch driverGUID {
		case drivers[1].DriverGUID, drivers[2].DriverGUID, drivers[3].DriverGUID:
		default:
			t.FailNow()
			return nil
		}

		if !driver.LoadedTime.IsZero() {
			t.Errorf("Driver: %s has loaded time", driverGUID)
			return nil
		}

		return nil
	})

	err := raceControl.OnClientLoaded(udp.ClientLoaded(drivers[1].CarID))

	if err != nil {
		t.Error(err)
		return
	}

	driver, ok := raceControl.ConnectedDrivers.Get(drivers[1].DriverGUID)

	if !ok || driver.LoadedTime.IsZero() {
		t.Errorf("Driver 2 not marked as loaded")
		return
	}

	driver2, ok := raceControl.ConnectedDrivers.Get(drivers[2].DriverGUID)

	if !ok || !driver2.LoadedTime.IsZero() {
		t.Errorf("Driver 2 should not be marked as loaded")
		return
	}

	driver3, ok := raceControl.ConnectedDrivers.Get(drivers[3].DriverGUID)

	if !ok || !driver3.LoadedTime.IsZero() {
		t.Errorf("Driver 4 should not be marked as loaded")
		return
	}

	t.Run("Driver not found", func(t *testing.T) {
		err := raceControl.OnClientLoaded(udp.ClientLoaded(10))

		if err == nil {
			t.Errorf("Expected error for non-existent driver, no error reported.")
			return
		}
	})
}

type nilTrackData struct{}

func (nilTrackData) TrackInfo(name, layout string) (*TrackInfo, error) {
	return &TrackInfo{}, nil
}

func (nilTrackData) TrackMap(name, layout string) (*TrackMapData, error) {
	return &TrackMapData{}, nil
}

func TestRaceControl_OnNewSession(t *testing.T) {
	t.Run("New session, no previous data", func(t *testing.T) {
		raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

		if err := raceControl.OnVersion(udp.Version(4)); err != nil {
			t.Error(err)
			return
		}

		// new session
		err := raceControl.OnNewSession(udp.SessionInfo{
			Version:             4,
			SessionIndex:        0,
			CurrentSessionIndex: 0,
			SessionCount:        3,
			ServerName:          "Test Server",
			Track:               "ks_laguna_seca",
			TrackConfig:         "",
			Name:                "Test Practice Session",
			Type:                udp.SessionTypePractice,
			Time:                10,
			Laps:                0,
			WaitTime:            120,
			AmbientTemp:         12,
			RoadTemp:            16,
			WeatherGraphics:     "01_clear",
			ElapsedMilliseconds: 10,

			EventType: udp.EventNewSession,
		})

		if err != nil {
			t.Error(err)
			return
		}

		time.Sleep(time.Millisecond * 10)

		defer func() { raceControl.serverProcessStopped <- struct{}{} }()

		// this is a completely new session, connected drivers and disconnected drivers should be empty
		if raceControl.ConnectedDrivers.Len() > 0 || raceControl.DisconnectedDrivers.Len() > 0 {
			t.Errorf("Connected or disconnected drivers has entries, should be len 0")
			return
		}
	})

	t.Run("New session, drivers join, then another new session. Drivers should have lap times cleared but not be disconnected", func(t *testing.T) {
		raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

		if err := raceControl.OnVersion(udp.Version(4)); err != nil {
			t.Error(err)
			return
		}

		// new session
		err := raceControl.OnNewSession(udp.SessionInfo{
			Version:             4,
			SessionIndex:        0,
			CurrentSessionIndex: 0,
			SessionCount:        3,
			ServerName:          "Test Server",
			Track:               "ks_laguna_seca",
			TrackConfig:         "",
			Name:                "Test Practice Session",
			Type:                udp.SessionTypePractice,
			Time:                10,
			Laps:                0,
			WaitTime:            120,
			AmbientTemp:         12,
			RoadTemp:            16,
			WeatherGraphics:     "01_clear",
			ElapsedMilliseconds: 10,

			EventType: udp.EventNewSession,
		})

		if err != nil {
			t.Error(err)
			return
		}

		time.Sleep(time.Millisecond * 10)

		// stop the session info ticker
		defer func() { raceControl.serverProcessStopped <- struct{}{} }()

		// join and load all drivers
		for _, entrant := range drivers {
			if err := raceControl.OnClientConnect(entrant); err != nil {
				t.Error(err)
				return
			}

			if err := raceControl.OnClientLoaded(udp.ClientLoaded(entrant.CarID)); err != nil {
				t.Error(err)
				return
			}
		}

		if raceControl.ConnectedDrivers.Len() != len(drivers) || raceControl.DisconnectedDrivers.Len() > 0 {
			t.Errorf("Incorrect driver listings")
			return
		}

		// do some laps for each entrant
		for i := 0; i < 100; i++ {
			driver := drivers[i%len(drivers)]

			err := raceControl.OnLapCompleted(udp.LapCompleted{
				CarID:   driver.CarID,
				LapTime: uint32(rand.Intn(1000000)),
				Cuts:    0,
			})

			if err != nil {
				t.Error(err)
				return
			}
		}

		// disconnect one of the drivers
		disconnectedDriver := drivers[len(drivers)-1]
		disconnectedDriver.EventType = udp.EventConnectionClosed
		err = raceControl.OnClientDisconnect(disconnectedDriver)

		if err != nil {
			t.Error(err)
			return
		}

		err = raceControl.OnEndSession(udp.EndSession("2019_3_2_14_41_PRACTICE.json"))

		if err != nil {
			t.Error(err)
			return
		}

		// now go to the next session, lap times should be removed from all drivers, but all should still be connected.
		err = raceControl.OnNewSession(udp.SessionInfo{
			Version:             4,
			SessionIndex:        1,
			CurrentSessionIndex: 1,
			SessionCount:        3,
			ServerName:          "Test Server",
			Track:               "ks_laguna_seca",
			TrackConfig:         "",
			Name:                "Test Qualifying Session",
			Type:                udp.SessionTypeQualifying,
			Time:                10,
			Laps:                0,
			WaitTime:            120,
			AmbientTemp:         12,
			RoadTemp:            16,
			WeatherGraphics:     "02_cloudy",
			ElapsedMilliseconds: 10,

			EventType: udp.EventNewSession,
		})

		if err != nil {
			t.Error(err)
			return
		}

		if raceControl.SessionInfo.Type != udp.SessionTypeQualifying {
			t.Error("Invalid session type detected, should be qualifying")
			return
		}

		if raceControl.ConnectedDrivers.Len() != len(drivers)-1 || raceControl.DisconnectedDrivers.Len() != 0 {
			t.Error("Invalid driver list lengths. Expected all connected drivers to still be in driver lists, disconnected list to be empty.")
			return
		}

		for _, driver := range raceControl.ConnectedDrivers.Drivers {
			car := driver.CurrentCar()

			if car.BestLap != 0 || car.TopSpeedBestLap != 0 || driver.Split != "" || driver.Position != 0 || len(driver.Collisions) > 0 {
				t.Error("Connected driver data carried across from previous session")
				return
			}
		}

		for _, driver := range raceControl.DisconnectedDrivers.Drivers {
			car := driver.CurrentCar()

			if car.BestLap != 0 || car.TopSpeedBestLap != 0 || driver.Split != "" || driver.Position != 0 || len(driver.Collisions) > 0 {
				t.Error("Disconnected driver data carried across from previous session")
				return
			}
		}
	})

	t.Run("Looped practice event, all cars and session information should be kept", func(t *testing.T) {
		raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

		if err := raceControl.OnVersion(udp.Version(4)); err != nil {
			t.Error(err)
			return
		}

		// new session
		err := raceControl.OnNewSession(udp.SessionInfo{
			Version:             4,
			SessionIndex:        0,
			CurrentSessionIndex: 0,
			SessionCount:        1,
			ServerName:          "Test Server",
			Track:               "ks_laguna_seca",
			TrackConfig:         "",
			Name:                "Test Looped Practice Session",
			Type:                udp.SessionTypePractice,
			Time:                10,
			Laps:                0,
			WaitTime:            120,
			AmbientTemp:         12,
			RoadTemp:            16,
			WeatherGraphics:     "01_clear",
			ElapsedMilliseconds: 10,

			EventType: udp.EventNewSession,
		})

		if err != nil {
			t.Error(err)
			return
		}

		time.Sleep(time.Millisecond * 10)

		// stop the session info ticker
		defer func() { raceControl.serverProcessStopped <- struct{}{} }()

		// join and load all drivers
		for _, entrant := range drivers {
			if err := raceControl.OnClientConnect(entrant); err != nil {
				t.Error(err)
				return
			}

			if err := raceControl.OnClientLoaded(udp.ClientLoaded(entrant.CarID)); err != nil {
				t.Error(err)
				return
			}
		}

		if raceControl.ConnectedDrivers.Len() != len(drivers) || raceControl.DisconnectedDrivers.Len() > 0 {
			t.Error("Incorrect driver listings")
			return
		}

		// do some laps for each entrant
		for i := 0; i < 100; i++ {
			driver := drivers[i%len(drivers)]

			err := raceControl.OnLapCompleted(udp.LapCompleted{
				CarID:   driver.CarID,
				LapTime: uint32(rand.Intn(1000000)),
				Cuts:    0,
			})

			if err != nil {
				t.Error(err)
				return
			}
		}

		// disconnect one of the drivers
		disconnectedDriver := drivers[len(drivers)-1]
		disconnectedDriver.EventType = udp.EventConnectionClosed
		err = raceControl.OnClientDisconnect(disconnectedDriver)

		if err != nil {
			t.Error(err)
			return
		}

		err = raceControl.OnEndSession(udp.EndSession("2019_3_2_20_48_QUALIFY.json"))

		if err != nil {
			t.Error(err)
			return
		}

		// now go to the next session, lap times should be removed from all drivers, but all should still be connected.
		err = raceControl.OnNewSession(udp.SessionInfo{
			Version:             4,
			SessionIndex:        0,
			CurrentSessionIndex: 0,
			SessionCount:        1,
			ServerName:          "Test Server",
			Track:               "ks_laguna_seca",
			TrackConfig:         "",
			Name:                "Test Looped Practice Session",
			Type:                udp.SessionTypePractice,
			Time:                10,
			Laps:                0,
			WaitTime:            120,
			AmbientTemp:         12,
			RoadTemp:            16,
			WeatherGraphics:     "02_cloudy",
			ElapsedMilliseconds: 10,

			EventType: udp.EventNewSession,
		})

		if err != nil {
			t.Error(err)
			return
		}

		if raceControl.ConnectedDrivers.Len() != len(drivers)-1 || raceControl.DisconnectedDrivers.Len() != 1 {
			t.Error("Invalid driver list lengths. Expected all drivers to still be in driver lists.")
			return
		}

		for _, driver := range raceControl.ConnectedDrivers.Drivers {
			car := driver.CurrentCar()

			if car.BestLap == 0 || driver.Position == 0 || car.LastLap == 0 {
				t.Error("Connected driver data not carried across from previous session")
				return
			}
		}

		for _, driver := range raceControl.DisconnectedDrivers.Drivers {
			car := driver.CurrentCar()

			if car.BestLap == 0 {
				t.Error("Disconnected driver data not carried across from previous session")
				return
			}
		}
	})
}

func TestRaceControl_OnCarUpdate(t *testing.T) {
	raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

	if err := raceControl.OnVersion(udp.Version(4)); err != nil {
		t.Error(err)
		return
	}

	// new session
	err := raceControl.OnNewSession(udp.SessionInfo{
		Version:             4,
		SessionIndex:        0,
		CurrentSessionIndex: 0,
		SessionCount:        1,
		ServerName:          "Test Server",
		Track:               "ks_laguna_seca",
		TrackConfig:         "",
		Name:                "Test Looped Practice Session",
		Type:                udp.SessionTypePractice,
		Time:                10,
		Laps:                0,
		WaitTime:            120,
		AmbientTemp:         12,
		RoadTemp:            16,
		WeatherGraphics:     "01_clear",
		ElapsedMilliseconds: 10,

		EventType: udp.EventNewSession,
	})

	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Millisecond * 10)

	// stop the session info ticker
	defer func() { raceControl.serverProcessStopped <- struct{}{} }()

	// join and load all drivers
	for _, entrant := range drivers {
		if err := raceControl.OnClientConnect(entrant); err != nil {
			t.Error(err)
			return
		}

		if err := raceControl.OnClientLoaded(udp.ClientLoaded(entrant.CarID)); err != nil {
			t.Error(err)
			return
		}
	}

	if raceControl.ConnectedDrivers.Len() != len(drivers) || raceControl.DisconnectedDrivers.Len() > 0 {
		t.Error("Incorrect driver listings")
		return
	}

	err = raceControl.handleCarUpdate(udp.CarUpdate{
		CarID:               drivers[1].CarID,
		Pos:                 udp.Vec{X: 100, Y: 20, Z: 3},
		Velocity:            udp.Vec{X: 10, Y: 20, Z: 20},
		Gear:                2,
		EngineRPM:           5000,
		NormalisedSplinePos: 0.2333,
	})

	if err != nil {
		t.Error(err)
		return
	}

	driver, ok := raceControl.ConnectedDrivers.Get(drivers[1].DriverGUID)

	if !ok {
		t.Log("Driver 1 not found")
		t.Fail()
	}

	if driver.LastPos.X == 0 && driver.LastPos.Y == 0 && driver.LastPos.Z == 0 {
		t.Log("Driver 1 has no last position")
		t.Fail()
	}

	if driver.CurrentCar().TopSpeedThisLap == 0 {
		t.Log("Driver 1 has no top speed")
		t.Fail()
	}

	t.Run("Unknown driver", func(t *testing.T) {
		err := raceControl.handleCarUpdate(udp.CarUpdate{
			CarID:               100, // unknown car
			Pos:                 udp.Vec{X: 100, Y: 20, Z: 3},
			Velocity:            udp.Vec{X: 10, Y: 20, Z: 20},
			Gear:                2,
			EngineRPM:           5000,
			NormalisedSplinePos: 0.2333,
		})

		if err == nil {
			t.Error("Error was nil, expected error")
			return
		}
	})
}

type driverLapResult struct {
	Driver        int
	LapTime       int
	ExpectedPos   int
	ExpectedSplit string
}

var raceLapTest = []driverLapResult{ // value in comments is 'total lap time (across all laps) for driver thus far'
	{Driver: 1, LapTime: 1, ExpectedPos: 1, ExpectedSplit: "0s"},  // 1
	{Driver: 2, LapTime: 2, ExpectedPos: 2, ExpectedSplit: "1ms"}, // 2
	{Driver: 3, LapTime: 3, ExpectedPos: 3, ExpectedSplit: "1ms"}, // 3

	{Driver: 1, LapTime: 1, ExpectedPos: 1, ExpectedSplit: "0s"},  // 2
	{Driver: 3, LapTime: 3, ExpectedPos: 2, ExpectedSplit: "4ms"}, // 6
	{Driver: 2, LapTime: 5, ExpectedPos: 3, ExpectedSplit: "1ms"}, // 7

	{Driver: 3, LapTime: 4, ExpectedPos: 1, ExpectedSplit: "0s"},  // 10
	{Driver: 2, LapTime: 5, ExpectedPos: 2, ExpectedSplit: "2ms"}, // 12
	// driver 1 has a bad lap, does not complete on lead lap

	{Driver: 3, LapTime: 4, ExpectedPos: 1, ExpectedSplit: "0s"},   // 14
	{Driver: 1, LapTime: 13, ExpectedPos: 3, ExpectedSplit: "3ms"}, // 15
	{Driver: 2, LapTime: 4, ExpectedPos: 2, ExpectedSplit: "2ms"},  // 16

	{Driver: 3, LapTime: 3, ExpectedPos: 1, ExpectedSplit: "0s"},    // 17
	{Driver: 2, LapTime: 4, ExpectedPos: 2, ExpectedSplit: "3ms"},   // 20
	{Driver: 1, LapTime: 7, ExpectedPos: 3, ExpectedSplit: "1 lap"}, // 22

	{Driver: 2, LapTime: 1, ExpectedPos: 1, ExpectedSplit: "0s"},  // 21
	{Driver: 3, LapTime: 5, ExpectedPos: 2, ExpectedSplit: "1ms"}, // 22
	// driver 1 has another bad lap, will be 2 laps down at crossing the line...

	{Driver: 2, LapTime: 3, ExpectedPos: 1, ExpectedSplit: "0s"},     // 24
	{Driver: 3, LapTime: 3, ExpectedPos: 2, ExpectedSplit: "1ms"},    // 25
	{Driver: 1, LapTime: 7, ExpectedPos: 3, ExpectedSplit: "2 laps"}, // 29

	// now driver 1 is setting personal bests, and unlaps himself *Ocon moment*
	{Driver: 2, LapTime: 3, ExpectedPos: 1, ExpectedSplit: "0s"},     // 27
	{Driver: 3, LapTime: 4, ExpectedPos: 2, ExpectedSplit: "2ms"},    // 29
	{Driver: 1, LapTime: 1, ExpectedPos: 3, ExpectedSplit: "2 laps"}, // 30

	{Driver: 2, LapTime: 3, ExpectedPos: 1, ExpectedSplit: "0s"},    // 30
	{Driver: 1, LapTime: 1, ExpectedPos: 3, ExpectedSplit: "1 lap"}, // 31 - speedy boy
	{Driver: 3, LapTime: 3, ExpectedPos: 2, ExpectedSplit: "2ms"},   // 32
}

func TestRaceControl_OnLapCompleted(t *testing.T) {
	raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

	if err := raceControl.OnVersion(udp.Version(4)); err != nil {
		t.Error(err)
		return
	}

	// new session
	err := raceControl.OnNewSession(udp.SessionInfo{
		Version:             4,
		SessionIndex:        0,
		CurrentSessionIndex: 0,
		SessionCount:        1,
		ServerName:          "Test Server",
		Track:               "ks_laguna_seca",
		TrackConfig:         "",
		Name:                "Test Looped Practice Session",
		Type:                udp.SessionTypeRace,
		Time:                10,
		Laps:                0,
		WaitTime:            120,
		AmbientTemp:         12,
		RoadTemp:            16,
		WeatherGraphics:     "01_clear",
		ElapsedMilliseconds: 10,

		EventType: udp.EventNewSession,
	})

	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Millisecond * 10)

	// stop the session info ticker
	defer func() { raceControl.serverProcessStopped <- struct{}{} }()

	driversOnFirstLap := raceLapTest[0:3]

	// join and load all drivers
	for _, driver := range driversOnFirstLap {
		if err := raceControl.OnClientConnect(drivers[driver.Driver]); err != nil {
			t.Error(err)
			return
		}

		if err := raceControl.OnClientLoaded(udp.ClientLoaded(drivers[driver.Driver].CarID)); err != nil {
			t.Error(err)
			return
		}
	}

	if raceControl.ConnectedDrivers.Len() != len(driversOnFirstLap) || raceControl.DisconnectedDrivers.Len() > 0 {
		t.Error("Incorrect driver listings")
		return
	}

	for _, driver := range raceLapTest {
		t.Logf("Driver: %d just crossed the line with a %d", driver.Driver, driver.LapTime)

		err = raceControl.OnLapCompleted(udp.LapCompleted{
			CarID:   drivers[driver.Driver].CarID,
			LapTime: uint32(driver.LapTime),
			Cuts:    0,
		})

		if err != nil {
			t.Error(err)
			return
		}

		rcDriver, ok := raceControl.ConnectedDrivers.Get(drivers[driver.Driver].DriverGUID)

		if !ok {
			t.Fail()
			return
		}

		if rcDriver.Position != driver.ExpectedPos {
			t.Errorf("Expected driver %d's position to be %d, was actually: %d", driver.Driver, driver.ExpectedPos, rcDriver.Position)
		}

		if rcDriver.Split != driver.ExpectedSplit {
			t.Errorf("Expected driver %d's split to be %s, was actually: %s", driver.Driver, driver.ExpectedSplit, rcDriver.Split)
		}
	}

	t.Run("Driver not found", func(t *testing.T) {
		err := raceControl.OnLapCompleted(udp.LapCompleted{
			CarID:   110,
			LapTime: 434683,
			Cuts:    0,
		})

		if err == nil {
			t.Error("Expected error on lap completed, none found (invalid driver expected)")
		}
	})
}

func TestRaceControl_SortDrivers(t *testing.T) {
	t.Run("Race, connected drivers", func(t *testing.T) {
		rc := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))
		rc.SessionInfo.Type = udp.SessionTypeRace

		d0 := NewRaceControlDriver(drivers[0])
		d0.CurrentCar().NumLaps = 10
		d0.CurrentCar().TotalLapTime = 100

		rc.ConnectedDrivers.Add(d0.CarInfo.DriverGUID, d0)

		d1 := NewRaceControlDriver(drivers[1])
		d1.CurrentCar().NumLaps = 10
		d1.CurrentCar().TotalLapTime = 88

		rc.ConnectedDrivers.Add(d1.CarInfo.DriverGUID, d1)

		d2 := NewRaceControlDriver(drivers[2])
		d2.CurrentCar().NumLaps = 7
		d2.CurrentCar().TotalLapTime = 30

		rc.ConnectedDrivers.Add(d2.CarInfo.DriverGUID, d2)

		rc.ConnectedDrivers.sort()

		if rc.ConnectedDrivers.GUIDsInPositionalOrder[0] != drivers[1].DriverGUID {
			t.Error("Driver 1 should be in first")
		}

		if rc.ConnectedDrivers.GUIDsInPositionalOrder[1] != drivers[0].DriverGUID {
			t.Error("Driver 0 should be in second")
		}

		if rc.ConnectedDrivers.GUIDsInPositionalOrder[2] != drivers[2].DriverGUID {
			t.Error("Driver 2 should be in third")
		}
	})

	t.Run("Non-race, connected drivers", func(t *testing.T) {
		t.Run("Two drivers with valid laps, two without", func(t *testing.T) {
			rc := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))
			rc.SessionInfo.Type = udp.SessionTypePractice

			d0 := NewRaceControlDriver(drivers[0])
			d0.CurrentCar().NumLaps = 10
			d0.CurrentCar().BestLap = 0

			rc.ConnectedDrivers.Add(d0.CarInfo.DriverGUID, d0)

			d1 := NewRaceControlDriver(drivers[1])
			d1.CurrentCar().NumLaps = 1
			d1.CurrentCar().BestLap = 88

			rc.ConnectedDrivers.Add(d1.CarInfo.DriverGUID, d1)

			d2 := NewRaceControlDriver(drivers[2])
			d2.CurrentCar().NumLaps = 7
			d2.CurrentCar().BestLap = 0

			rc.ConnectedDrivers.Add(d2.CarInfo.DriverGUID, d2)

			d3 := NewRaceControlDriver(drivers[3])
			d3.CurrentCar().NumLaps = 11
			d3.CurrentCar().BestLap = 89

			rc.ConnectedDrivers.Add(d3.CarInfo.DriverGUID, d3)

			rc.ConnectedDrivers.sort()

			if rc.ConnectedDrivers.GUIDsInPositionalOrder[0] != drivers[1].DriverGUID {
				t.Error("Driver 1 should be in first")
			}

			if rc.ConnectedDrivers.GUIDsInPositionalOrder[1] != drivers[3].DriverGUID {
				t.Error("Driver 3 should be in second")
			}

			if rc.ConnectedDrivers.GUIDsInPositionalOrder[2] != drivers[0].DriverGUID {
				t.Error("Driver 0 should be in third")
			}

			if rc.ConnectedDrivers.GUIDsInPositionalOrder[3] != drivers[2].DriverGUID {
				t.Error("Driver 2 should be in fourth")
			}
		})
	})

	t.Run("Race, disconnected drivers", func(t *testing.T) {
		rc := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))
		rc.SessionInfo.Type = udp.SessionTypeRace

		d0 := NewRaceControlDriver(drivers[0])
		d0.CurrentCar().LastLapCompletedTime = time.Now().Add(-10 * time.Minute)
		rc.DisconnectedDrivers.Add(d0.CarInfo.DriverGUID, d0)

		d1 := NewRaceControlDriver(drivers[1])
		d1.CurrentCar().LastLapCompletedTime = time.Now().Add(-30 * time.Minute)
		rc.DisconnectedDrivers.Add(d1.CarInfo.DriverGUID, d1)

		d2 := NewRaceControlDriver(drivers[2])
		d2.CurrentCar().LastLapCompletedTime = time.Now().Add(time.Minute)
		rc.DisconnectedDrivers.Add(d2.CarInfo.DriverGUID, d2)

		d3 := NewRaceControlDriver(drivers[3])
		d3.CurrentCar().LastLapCompletedTime = time.Now()
		rc.DisconnectedDrivers.Add(d3.CarInfo.DriverGUID, d3)

		rc.DisconnectedDrivers.sort()

		if rc.DisconnectedDrivers.GUIDsInPositionalOrder[0] != drivers[2].DriverGUID {
			t.Error("Driver 2 should be in first")
		}

		if rc.DisconnectedDrivers.GUIDsInPositionalOrder[1] != drivers[3].DriverGUID {
			t.Error("Driver 3 should be in second")
		}

		if rc.DisconnectedDrivers.GUIDsInPositionalOrder[2] != drivers[0].DriverGUID {
			t.Error("Driver 0 should be in third")
		}

		if rc.DisconnectedDrivers.GUIDsInPositionalOrder[3] != drivers[1].DriverGUID {
			t.Error("Driver 1 should be in fourth")
		}
	})

	t.Run("Non-Race, disconnected drivers", func(t *testing.T) {
		rc := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))
		rc.SessionInfo.Type = udp.SessionTypeQualifying

		d0 := NewRaceControlDriver(drivers[0])
		d0.CurrentCar().BestLap = 2000
		rc.DisconnectedDrivers.Add(d0.CarInfo.DriverGUID, d0)

		d1 := NewRaceControlDriver(drivers[1])
		d1.CurrentCar().BestLap = 40
		rc.DisconnectedDrivers.Add(d1.CarInfo.DriverGUID, d1)

		d2 := NewRaceControlDriver(drivers[2])
		d2.CurrentCar().BestLap = 600
		rc.DisconnectedDrivers.Add(d2.CarInfo.DriverGUID, d2)

		d3 := NewRaceControlDriver(drivers[3])
		d3.CurrentCar().BestLap = 0
		rc.DisconnectedDrivers.Add(d3.CarInfo.DriverGUID, d3)

		rc.DisconnectedDrivers.sort()

		if rc.DisconnectedDrivers.GUIDsInPositionalOrder[0] != drivers[1].DriverGUID {
			t.Error("Driver 1 should be in first")
		}

		if rc.DisconnectedDrivers.GUIDsInPositionalOrder[1] != drivers[2].DriverGUID {
			t.Error("Driver 2 should be in second")
		}

		if rc.DisconnectedDrivers.GUIDsInPositionalOrder[2] != drivers[0].DriverGUID {
			t.Error("Driver 0 should be in third")
		}

		if rc.DisconnectedDrivers.GUIDsInPositionalOrder[3] != drivers[3].DriverGUID {
			t.Error("Driver 3 should be in fourth (no lap time)")
		}
	})
}

func TestRaceControl_OnSessionUpdate(t *testing.T) {
	t.Run("Session update", func(t *testing.T) {
		raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

		if err := raceControl.OnVersion(udp.Version(4)); err != nil {
			t.Error(err)
			return
		}

		sessionInfo := udp.SessionInfo{
			Version:             4,
			SessionIndex:        0,
			CurrentSessionIndex: 0,
			SessionCount:        1,
			ServerName:          "Test Server",
			Track:               "ks_laguna_seca",
			TrackConfig:         "",
			Name:                "Test Looped Practice Session",
			Type:                udp.SessionTypePractice,
			Time:                10,
			Laps:                0,
			WaitTime:            120,
			AmbientTemp:         12,
			RoadTemp:            16,
			WeatherGraphics:     "01_clear",
			ElapsedMilliseconds: 10,

			EventType: udp.EventNewSession,
		}

		// new session
		err := raceControl.OnNewSession(sessionInfo)

		if err != nil {
			t.Error(err)
			return
		}

		time.Sleep(time.Millisecond * 10)

		// stop the session info ticker
		defer func() { raceControl.serverProcessStopped <- struct{}{} }()

		// join and load all drivers
		for _, entrant := range drivers {
			if err := raceControl.OnClientConnect(entrant); err != nil {
				t.Error(err)
				return
			}

			if err := raceControl.OnClientLoaded(udp.ClientLoaded(entrant.CarID)); err != nil {
				t.Error(err)
				return
			}
		}

		sessionInfo.EventType = udp.EventSessionInfo
		sessionInfo.AmbientTemp = 100

		_, err = raceControl.OnSessionUpdate(sessionInfo)

		if err != nil {
			t.Error(err)
			return
		}

		if raceControl.SessionInfo.AmbientTemp != 100 {
			t.Errorf("Expected ambient temp of 100, got %d", raceControl.SessionInfo.AmbientTemp)
			return
		}
	})
}

func TestRaceControl_Event(t *testing.T) {
	rc := NewRaceControl(NilBroadcaster{}, nilTrackData{}, dummyServerProcess{}, testStore, NewPenaltiesManager(testStore))

	if rc.Event() != 200 {
		t.Error("Expected Race Control event to be 200")
		return
	}
}
