package servermanager

import (
	"encoding/json"
	"fmt"
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"os"
	"strconv"
	"time"
)

func CreateDummy() {
	println("Create Dummy")

	time.Sleep(20 * time.Second)

	sessionStarted := udp.SessionInfo {
		Version: 10,
		ServerName: "That's a bloomin test server if I ever saw one!",
		Track: "Goomba Beach",
		TrackConfig: "Drift",
		Name: "Pizza Bab",
		Type: 3, // Is type an indicator of race/quali/etc. 0 book 1 prac 2 qual 3 race
		Time: 25,
		Laps: 0,
		WaitTime: 2,
		WeatherGraphics: "sol_23_Sand_type=24_time=0_mult=60_start=10458000",
		ElapsedMilliseconds: 267,
		EventType: udp.EventNewSession,
	}

	CallbackFunc(sessionStarted)

	time.Sleep(2 * time.Second)

	newConnectionCarInfo := udp.SessionCarInfo{
		CarID: 0,
		DriverName: "Kalom",
		DriverGUID: "101983210957",
		CarMode: "Submarine",
		CarSkin: "Blue",

		EventType: udp.EventNewConnection,
	}

	CallbackFunc(newConnectionCarInfo)

	time.Sleep(1 * time.Second)

	newConnectionCarInfo2 := udp.SessionCarInfo{
		CarID: 1,
		DriverName: "Henry",
		DriverGUID: "12309150710571",
		CarMode: "Jet Plane",
		CarSkin: "Greenish",

		EventType: udp.EventNewConnection,
	}

	CallbackFunc(newConnectionCarInfo2)

	time.Sleep(2 * time.Second)
	CallbackFunc(udp.ClientLoaded(0))
	time.Sleep(1 * time.Second)
	CallbackFunc(udp.ClientLoaded(1))

	time.Sleep(5 * time.Second)

	lapCompleted := udp.LapCompleted{
		LapCompletedInternal: udp.LapCompletedInternal{
			CarID: 0,
			LapTime: 200000,
			Cuts: 2,
			CarsCount: 2,
		},
	}

	CallbackFunc(lapCompleted)

	time.Sleep(5 * time.Second)

	lapCompleted2 := udp.LapCompleted{
		LapCompletedInternal: udp.LapCompletedInternal{
			CarID: 1,
			LapTime: 200500,
			Cuts: 1,
			CarsCount: 2,
		},
	}

	CallbackFunc(lapCompleted2)

	time.Sleep(5 * time.Second)

	lapCompleted3 := udp.LapCompleted{
		LapCompletedInternal: udp.LapCompletedInternal{
			CarID: 1,
			LapTime: 190200,
			Cuts: 1,
			CarsCount: 2,
		},
	}

	CallbackFunc(lapCompleted3)

	time.Sleep(5 * time.Second)

	lapCompleted4 := udp.LapCompleted{
		LapCompletedInternal: udp.LapCompletedInternal{
			CarID: 0,
			LapTime: 201000,
			Cuts: 2,
			CarsCount: 2,
		},
	}

	CallbackFunc(lapCompleted4)
}

// Live timing output format
type LiveTiming struct {
	// Server Info, static
	ServerName, Track, TrackConfig, Name string
	Type uint8
	Time, Laps, WaitTime uint16
	WeatherGraphics string
	ElapsedMilliseconds int32

	Cars map[uint8]*LiveCar // map[carID]LiveCar

	// Live data
	LapNum int
}

type LiveCar struct {
	// Static Car Info
	DriverName, DriverGUID string
	CarMode, CarSkin string

	// Live Info
	LapNum int
	Loaded bool
	LastLap string
	BestLap string
	BestLapTime         time.Duration
	LastLapCompleteTime time.Time
	Pos     int
	Crash   bool
	Split   string
}

var liveInfo LiveTiming

func CallbackFunc (response udp.Message) {
	/*currentRace, _ := raceManager.CurrentRace()

	if currentRace == nil {
		// no race live, ignore udp
		return
	}*/

	switch a := response.(type) {
	case udp.SessionInfo:
		if a.Event() == udp.EventNewSession {
			// New session, clear old data and create new
			liveInfo = LiveTiming {
				ServerName: a.ServerName,
				Track: a.Track,
				TrackConfig: a.TrackConfig,
				Name: a.Name,
				Type: a.Type,
				Time: a.Time,
				Laps: a.Laps,
				WaitTime: a.WaitTime,
				WeatherGraphics: a.WeatherGraphics,
				ElapsedMilliseconds: a.ElapsedMilliseconds,

				Cars: make(map[uint8]*LiveCar),
			}
		}

	case udp.SessionCarInfo:
		if a.Event() == udp.EventNewConnection {
			liveInfo.Cars[uint8(a.CarID)] = &LiveCar {
				DriverGUID: a.DriverGUID,
				DriverName: a.DriverName,
				CarMode: a.CarMode,
				CarSkin: a.CarSkin,
			}
		}

	case udp.ClientLoaded:
		if _, ok := liveInfo.Cars[uint8(a)]; ok {
			liveInfo.Cars[uint8(a)].Loaded = true
		}

	case udp.LapCompleted:
		ID := uint8(a.LapCompletedInternal.CarID)

		if _, ok := liveInfo.Cars[ID]; ok {
			liveInfo.Cars[ID].LastLap = lapToDuration(int(a.LapCompletedInternal.LapTime)).String()
			liveInfo.Cars[ID].LapNum ++
			liveInfo.Cars[ID].LastLapCompleteTime = time.Now()

			if lapToDuration(int(a.LapCompletedInternal.LapTime)) < liveInfo.Cars[ID].BestLapTime || liveInfo.Cars[ID].BestLapTime == 0 {
				liveInfo.Cars[ID].BestLapTime = lapToDuration(int(a.LapCompletedInternal.LapTime))
				liveInfo.Cars[ID].BestLap = liveInfo.Cars[ID].BestLapTime.String()
			}
		} else {
			return
		}

		var pos = 1

		switch liveInfo.Type {

			// Race
			case 3:
				for carID, liveCar := range liveInfo.Cars {
					if carID == ID {
						continue
					}

					if liveCar.LastLapCompleteTime.Before(liveInfo.Cars[ID].LastLapCompleteTime) &&
						liveCar.LapNum >= liveInfo.Cars[ID].LapNum {

						pos++
					}

				}

				liveInfo.Cars[ID].Pos = pos

				if liveInfo.Cars[ID].Pos == 1 {
					liveInfo.Cars[ID].Split = time.Duration(0).String()
				} else {
					for _, liveCar := range liveInfo.Cars {
						if liveCar.Pos == liveInfo.Cars[ID].Pos-1 {
							if liveCar.LapNum == liveInfo.Cars[ID].LapNum {
								liveInfo.Cars[ID].Split = time.Now().Sub(liveCar.LastLapCompleteTime).String()
							} else {
								liveInfo.Cars[ID].Split = strconv.Itoa(liveCar.LapNum-liveInfo.Cars[ID].LapNum) + " lap(s)"
							}
						}
					}
				}
			// Qualification, Practice
			case 2, 1:
				// @TODO no point doing this, just sort the map by bestlaptime in js
				for carID, liveCar := range liveInfo.Cars {
					if carID == ID {
						return
					}

					if liveCar.BestLapTime > liveInfo.Cars[ID].BestLapTime {
						pos++
					}
				}

				liveInfo.Cars[ID].Pos = pos

				if liveInfo.Cars[ID].Pos == 1 {
					liveInfo.Cars[ID].Split = time.Duration(0).String()
				} else {
					for _, liveCar := range liveInfo.Cars {
						if liveCar.Pos == liveInfo.Cars[ID].Pos-1 {
							liveInfo.Cars[ID].Split = (liveInfo.Cars[ID].BestLapTime - liveCar.BestLapTime).String()
						}
					}
				}
		}

	}

	err := json.NewEncoder(os.Stdout).Encode(liveInfo)

	if err != nil {
		//@TODO
		panic(err)
	}

}

func lapToDuration(i int) time.Duration {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", i))

	return d
}