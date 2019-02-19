package servermanager

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/sirupsen/logrus"
)

// Live timing output format
type LiveTiming struct {
	// Server Info, static
	ServerName, Track, TrackConfig, Name string
	Type                                 uint8
	Time, Laps, WaitTime                 uint16
	WeatherGraphics                      string
	ElapsedMilliseconds                  int32
	SessionStarted                       int64

	Cars map[uint8]*LiveCar // map[carID]LiveCar

	// Live data
	LapNum int
}

type LiveCar struct {
	// Static Car Info
	DriverName, DriverGUID string
	CarMode, CarSkin       string

	// On disconnect
	Delete bool

	// Live Info
	LapNum int

	Loaded     bool
	LoadedTime int64

	LastLap                 string
	BestLap                 string
	BestLapTime             time.Duration
	LastLapCompleteTime     time.Time
	LastLapCompleteTimeUnix int64
	Pos                     int
	Split                   string

	Collisions []Collision
}

type LiveCarWID struct {
	Car *LiveCar
	ID  uint8
}

type Collision struct {
	Type     string
	Time     int64
	OtherCar uint8
	Speed    float32
}

var liveInfo LiveTiming

func CallbackFunc(response udp.Message) {
	currentRace, _ := raceManager.CurrentRace()

	if currentRace == nil {
		// no race live, ignore udp
		return
	}

	switch a := response.(type) {
	case udp.SessionInfo:
		if a.Event() == udp.EventNewSession {
			// New session, clear old data and create new - keep cars if necessary
			var oldCars map[uint8]*LiveCar

			if len(liveInfo.Cars) != 0 {
				oldCars = make(map[uint8]*LiveCar)
				oldCars = liveInfo.Cars
			}

			sessionT, err := time.ParseDuration(fmt.Sprintf("%dms", a.ElapsedMilliseconds))

			if err != nil {
				logrus.Error(err)
			}

			liveInfo = LiveTiming{
				ServerName:          a.ServerName,
				Track:               a.Track,
				TrackConfig:         a.TrackConfig,
				Name:                a.Name,
				Type:                a.Type,
				Time:                a.Time,
				Laps:                a.Laps,
				WaitTime:            a.WaitTime,
				WeatherGraphics:     a.WeatherGraphics,
				ElapsedMilliseconds: a.ElapsedMilliseconds,
				SessionStarted:      unixNanoToMilli(time.Now().Add(-sessionT).UnixNano()),
			}

			if len(oldCars) == 0 {
				liveInfo.Cars = make(map[uint8]*LiveCar)
			} else {
				for id, liveCar := range oldCars {
					if liveCar.Delete {
						delete(oldCars, id)
					}

					liveCar.LapNum = 0
					liveCar.BestLapTime = time.Duration(0)
					liveCar.BestLap = ""
					liveCar.LastLapCompleteTime = time.Now()
					liveCar.LastLapCompleteTimeUnix = unixNanoToMilli(time.Now().UnixNano())
					liveCar.LastLap = ""
					liveCar.Split = ""
					liveCar.Pos = 0
				}

				liveInfo.Cars = oldCars
			}
		}

	case udp.SessionCarInfo:
		if a.Event() == udp.EventNewConnection {
			liveInfo.Cars[uint8(a.CarID)] = &LiveCar{
				DriverGUID: a.DriverGUID,
				DriverName: a.DriverName,
				CarMode:    a.CarMode,
				CarSkin:    a.CarSkin,
				Delete:     false,
			}
		} else if a.Event() == udp.EventConnectionClosed {
			_, ok := liveInfo.Cars[uint8(a.CarID)]
			if ok {
				liveInfo.Cars[uint8(a.CarID)].Delete = true
			}
		}

	case udp.ClientLoaded:
		if _, ok := liveInfo.Cars[uint8(a)]; ok {
			liveInfo.Cars[uint8(a)].Loaded = true
			liveInfo.Cars[uint8(a)].LoadedTime = unixNanoToMilli(time.Now().UnixNano())
		}

	case udp.CollisionWithCar:
		if _, ok := liveInfo.Cars[uint8(a.CarID)]; ok {
			liveInfo.Cars[uint8(a.CarID)].Collisions = append(liveInfo.Cars[uint8(a.CarID)].Collisions, Collision{
				Type:     "with other car",
				Time:     unixNanoToMilli(time.Now().UnixNano()),
				OtherCar: uint8(a.OtherCarID),
				Speed:    a.ImpactSpeed,
			})
		}

	case udp.CollisionWithEnvironment:
		if _, ok := liveInfo.Cars[uint8(a.CarID)]; ok {
			liveInfo.Cars[uint8(a.CarID)].Collisions = append(liveInfo.Cars[uint8(a.CarID)].Collisions, Collision{
				Type:     "with environment",
				Time:     unixNanoToMilli(time.Now().UnixNano()),
				OtherCar: 255,
				Speed:    a.ImpactSpeed,
			})
		}

	case udp.LapCompleted:
		ID := uint8(a.LapCompletedInternal.CarID)

		if _, ok := liveInfo.Cars[ID]; ok {
			liveInfo.Cars[ID].LastLap = lapToDuration(int(a.LapCompletedInternal.LapTime)).String()
			liveInfo.Cars[ID].LapNum++
			liveInfo.Cars[ID].LastLapCompleteTime = time.Now()
			liveInfo.Cars[ID].LastLapCompleteTimeUnix = unixNanoToMilli(time.Now().UnixNano())

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
							liveInfo.Cars[ID].Split = time.Now().Sub(liveCar.LastLapCompleteTime).Round(time.Millisecond).String()
						} else {
							liveInfo.Cars[ID].Split = strconv.Itoa(liveCar.LapNum-liveInfo.Cars[ID].LapNum) + " lap(s)"
						}
					}
				}
			}
		// Qualification, Practice
		case 2, 1:
			// Create an array that can be sorted by position
			var carArray []*LiveCarWID

			for carID, liveCar := range liveInfo.Cars {
				if liveCar.BestLapTime == 0 {
					liveCar.BestLapTime = time.Duration(time.Hour * 10)
				}

				carArray = append(carArray, &LiveCarWID{
					Car: liveCar,
					ID:  carID,
				})
			}

			sort.Slice(carArray, func(i, j int) bool {
				return carArray[i].Car.BestLapTime < carArray[j].Car.BestLapTime
			})

			// Calculate splits for all other cars, they may have changed
			for i, liveCar := range carArray {
				liveInfo.Cars[liveCar.ID].Pos = i + 1

				if liveCar.Car.Pos == 1 || i == 0 {
					liveInfo.Cars[liveCar.ID].Split = time.Duration(0).String()
					continue
				}

				liveInfo.Cars[liveCar.ID].Split = (liveCar.Car.BestLapTime - carArray[i-1].Car.BestLapTime).String()
			}
		}

	}

}

func unixNanoToMilli(i int64) int64 {
	return int64(float64(i) / 1000000)
}

func lapToDuration(i int) time.Duration {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", i))

	return d
}
