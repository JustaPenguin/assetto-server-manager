package udp

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

func NewServerClient(ctx context.Context, addr string, receivePort, sendPort int, callback CallbackFunc) (*AssettoServerUDP, error) {
	u := &AssettoServerUDP{
		ctx:      ctx,
		callback: callback,
	}
	err := u.listen(addr, receivePort, sendPort)

	if err != nil {
		return nil, err
	}

	return u, nil
}

type CallbackFunc func(response Message)

type AssettoServerUDP struct {
	ctx      context.Context
	callback CallbackFunc
}

func (asu *AssettoServerUDP) listen(hostname string, receivePort, sendPort int) error {
	errCh := make(chan error)

	go func() {
		serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: []byte{0, 0, 0, 0}, Port: receivePort, Zone: ""})

		if err != nil {
			errCh <- err
			return
		} else {
			errCh <- nil
		}

		defer serverConn.Close()

		buf := make([]byte, 1024)

		for {
			_, _, err := serverConn.ReadFromUDP(buf)

			if err != nil {
				asu.callback(ServerError{err})
				continue
			}

			msg, err := asu.handleMessage(bytes.NewReader(buf))

			if err != nil {
				asu.callback(ServerError{err})
				continue
			}

			asu.callback(msg)

			select {
			case <-asu.ctx.Done():
				return
			default:
			}
		}
	}()

	return <-errCh
}

func readStringW(r io.Reader) string {
	var size uint8
	err := binary.Read(r, binary.LittleEndian, &size)

	if err != nil {
		return ""
	}

	s := make([]byte, size*4)

	err = binary.Read(r, binary.LittleEndian, &s)

	return string(bytes.Replace(s, []byte("\x00"), nil, -1))
}

func readString(r io.Reader) string {
	var size uint8
	err := binary.Read(r, binary.LittleEndian, &size)

	if err != nil {
		return ""
	}

	s := make([]byte, size)

	err = binary.Read(r, binary.LittleEndian, &s)

	return string(bytes.Replace(s, []byte("\x00"), nil, -1))
}

func (asu *AssettoServerUDP) handleMessage(r io.Reader) (Message, error) {
	var messageType uint8

	err := binary.Read(r, binary.LittleEndian, &messageType)

	if err != nil {
		return nil, err
	}

	eventType := Event(messageType)

	var response Message

	switch eventType {
	case EventNewConnection, EventConnectionClosed:
		driverName := readStringW(r)
		driverGUID := readStringW(r)

		var carID CarID

		err = binary.Read(r, binary.LittleEndian, &carID)

		if err != nil {
			return nil, err
		}

		carMode := readString(r)
		carSkin := readStringW(r)

		response = SessionCarInfo{
			CarID:      carID,
			DriverName: driverName,
			DriverGUID: driverGUID,
			CarMode:    carMode,
			CarSkin:    carSkin,
			event:      eventType,
		}

	case EventCarUpdate:
		carUpdate := CarUpdate{}

		err := binary.Read(r, binary.LittleEndian, &carUpdate)

		if err != nil {
			return nil, err
		}

		response = carUpdate
	case EventCarInfo:
		var carID CarID

		err = binary.Read(r, binary.LittleEndian, &carID)

		if err != nil {
			return nil, err
		}

		var isConnected uint8

		err = binary.Read(r, binary.LittleEndian, &isConnected)

		response = CarInfo{
			CarID:       carID,
			IsConnected: isConnected != 0,
			CarModel:    readStringW(r),
			CarSkin:     readStringW(r),
			DriverName:  readStringW(r),
			DriverTeam:  readStringW(r),
			DriverGUID:  readStringW(r),
		}
	case EventEndSession:
		filename := readStringW(r)

		response = EndSession(filename)
	case EventVersion:
		var version uint8

		err = binary.Read(r, binary.LittleEndian, &version)

		if err != nil {
			return nil, err
		}

		response = Version(version)
	case EventChat:
		var carID CarID

		err := binary.Read(r, binary.LittleEndian, &carID)

		if err != nil {
			return nil, err
		}

		message := readStringW(r)

		response = Chat{
			CarID:   carID,
			Message: message,
		}
	case EventClientLoaded:
		var carID CarID

		err := binary.Read(r, binary.LittleEndian, &carID)

		if err != nil {
			return nil, err
		}

		response = ClientLoaded(carID)
	case EventNewSession, EventSessionInfo:
		sessionInfo := SessionInfo{}

		err := binary.Read(r, binary.LittleEndian, &sessionInfo.Version)

		if err != nil {
			return nil, err
		}

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.SessionIndex)

		if err != nil {
			return nil, err
		}

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.CurrentSessionIndex)

		if err != nil {
			return nil, err
		}

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.SessionCount)

		if err != nil {
			return nil, err
		}

		sessionInfo.ServerName = readStringW(r)
		sessionInfo.Track = readString(r)
		sessionInfo.TrackConfig = readString(r)
		sessionInfo.Name = readString(r)

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.Type)

		if err != nil {
			return nil, err
		}

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.Time)

		if err != nil {
			return nil, err
		}

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.Laps)

		if err != nil {
			return nil, err
		}

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.WaitTime)

		if err != nil {
			return nil, err
		}

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.AmbientTemp)

		if err != nil {
			return nil, err
		}

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.RoadTemp)

		if err != nil {
			return nil, err
		}

		sessionInfo.WeatherGraphics = readString(r)

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.ElapsedMilliseconds)

		if err != nil {
			return nil, err
		}

		sessionInfo.event = eventType

		response = sessionInfo
	case EventError:
		message := readStringW(r)

		response = ServerError{errors.New(message)}

	case EventLapCompleted:
		lapCompleted := LapCompletedInternal{}

		err := binary.Read(r, binary.LittleEndian, &lapCompleted)

		if err != nil {
			return nil, err
		}

		lc := LapCompleted{LapCompletedInternal: lapCompleted}

		for i := uint8(0); i < lapCompleted.CarsCount; i++ {
			var car LapCompletedCar
			err := binary.Read(r, binary.LittleEndian, &car)

			if err != nil {
				return nil, err
			}

			lc.Cars = append(lc.Cars, &car)
		}

		response = lc
	case EventClientEvent:
		var collisionType uint8

		err := binary.Read(r, binary.LittleEndian, &collisionType)

		if err != nil {
			return nil, err
		}

		if Event(collisionType) == EventCollisionWithCar {
			collision := CollisionWithCar{}

			err := binary.Read(r, binary.LittleEndian, &collision)

			if err != nil {
				return nil, err
			}

			response = collision
		} else if Event(collisionType) == EventCollisionWithEnv {
			collision := CollisionWithEnvironment{}

			err := binary.Read(r, binary.LittleEndian, &collision)

			if err != nil {
				return nil, err
			}

			response = collision
		}

	default:
		buf := new(bytes.Buffer)

		_, err = buf.ReadFrom(r)

		if err != nil {
			return nil, err
		}

		fmt.Println("Unknown response type", eventType)
		fmt.Println(buf.String())

		return nil, errors.New("unknown response type")
	}

	return response, nil
}
