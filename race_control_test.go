package servermanager

import (
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"testing"
	"time"
)

var (
	drivers = []udp.SessionCarInfo{
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
	}
)

// OnVersion should move all current drivers into the disconnected driver map, and empty out the connected driver map.
func TestRaceControl_OnVersion(t *testing.T) {
	raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{})

	// add some current drivers
	for _, driverIndex := range []int{0, 2, 3} {
		err := raceControl.OnClientConnect(drivers[driverIndex])

		if err != nil {
			t.Error(err)
			return
		}
	}

	if raceControl.ConnectedDrivers.Len() != 3 {
		t.Logf("Invalid driver length: %d", raceControl.ConnectedDrivers.Len())
		t.Fail()
		return
	}

	// onversion
	err := raceControl.OnVersion(udp.Version(4))

	if err != nil {
		t.Error(err)
		return
	}

	// now we should have 0 drivers in connected, and 3 in disconnected
	if raceControl.ConnectedDrivers.Len() != 0 {
		t.Logf("Was expecting 0 connected drivers, got: %d", raceControl.ConnectedDrivers.Len())
		t.Fail()
		return
	}

	if raceControl.DisconnectedDrivers.Len() != 3 {
		t.Logf("Was expecting 3 disconnected drivers, got: %d", raceControl.DisconnectedDrivers.Len())
		t.Fail()
		return
	}
}

func TestRaceControl_OnClientConnect(t *testing.T) {
	t.Run("Client first connect", func(t *testing.T) {
		// on first connect, a client is added to connected drivers but does not yet have a loaded time.
		// their GUID is added to the CarID -> GUID map for future lookup
		raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{})

		err := raceControl.OnClientConnect(drivers[0])

		if err != nil {
			t.Error(err)
			return
		}

		if guid, ok := raceControl.CarIDToGUID[drivers[0].CarID]; !ok || guid != drivers[0].DriverGUID {
			t.Logf("Driver was not correctly added to CarID -> GUID map")
			t.Fail()
			return
		}

		driver, ok := raceControl.ConnectedDrivers.Get(drivers[0].DriverGUID)

		if !ok {
			t.Logf("Driver was not correctly added to ConnectedDrivers")
			t.Fail()
			return
		}

		if !driver.LoadedTime.IsZero() {
			t.Logf("Driver has loaded time when it should be zero")
			t.Fail()
			return
		}

		t.Run("Client disconnects and reconnects", func(t *testing.T) {
			// disconnect the driver
			err := raceControl.OnClientDisconnect(drivers[0])

			if err != nil {
				t.Error(err)
				return
			}

			// assert that the driver has been disconnected
			if _, ok := raceControl.ConnectedDrivers.Get(drivers[0].DriverGUID); ok {
				t.Log("Driver should have been disconnected, was not. (present in ConnectedDrivers)")
				t.Fail()
				return
			}

			if _, ok := raceControl.DisconnectedDrivers.Get(drivers[0].DriverGUID); !ok {
				t.Log("Driver should have been disconnected, was not. (not present in DisconnectedDrivers)")
				t.Fail()
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
				t.Log("Driver should have been connected, was not. (not present in ConnectedDrivers)")
				t.Fail()
				return
			}

			if _, ok := raceControl.DisconnectedDrivers.Get(drivers[0].DriverGUID); ok {
				t.Log("Driver should have been connected, was not. (present in DisconnectedDrivers)")
				t.Fail()
				return
			}
		})
	})
}

func TestRaceControl_OnClientLoaded(t *testing.T) {
	raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{})

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
			t.Logf("Driver: %s has loaded time", driverGUID)
			t.Fail()
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
		t.Logf("Driver 2 not marked as loaded")
		t.Fail()
		return
	}

	driver2, ok := raceControl.ConnectedDrivers.Get(drivers[2].DriverGUID)

	if !ok || !driver2.LoadedTime.IsZero() {
		t.Logf("Driver 2 should not be marked as loaded")
		t.Fail()
		return
	}

	driver3, ok := raceControl.ConnectedDrivers.Get(drivers[3].DriverGUID)

	if !ok || !driver3.LoadedTime.IsZero() {
		t.Logf("Driver 4 should not be marked as loaded")
		t.Fail()
		return
	}

	t.Run("Driver not found", func(t *testing.T) {
		err := raceControl.OnClientLoaded(udp.ClientLoaded(10))

		if err == nil {
			t.Logf("Expected error for non-existent driver, no error reported.")
			t.Fail()
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
	raceControl := NewRaceControl(NilBroadcaster{}, nilTrackData{})

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
	defer raceControl.sessionInfoCfn()

	// this is a completely new session, connected drivers and disconnected drivers should be empty
	if raceControl.ConnectedDrivers.Len() > 0 || raceControl.DisconnectedDrivers.Len() > 0 {
		t.Logf("Connected or disconnected drivers has entries, should be len 0")
		t.Fail()
		return
	}
}
