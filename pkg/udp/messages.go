package udp

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/unicode/utf32"
)

// RealtimePosIntervalMs is the interval to request real time positional information.
// Set this to greater than 0 to enable.
var RealtimePosIntervalMs = -1
var CurrentRealtimePosIntervalMs = -1
var PosIntervalModifierEnabled = false

func NewServerClient(addr string, receivePort, sendPort int, forward bool, forwardAddrStr string, forwardListenPort int, callback CallbackFunc) (*AssettoServerUDP, error) {
	listener, err := net.DialUDP("udp", &net.UDPAddr{IP: net.ParseIP(addr), Port: receivePort}, &net.UDPAddr{IP: net.ParseIP(addr), Port: sendPort})

	if err != nil {
		return nil, err
	}

	if runtime.GOOS != "darwin" {
		if err := listener.SetReadBuffer(1e8); err != nil {
			logrus.WithError(err).Error("unable to set read buffer")
		}
	}

	ctx, cfn := context.WithCancel(context.Background())

	u := &AssettoServerUDP{
		ctx:      ctx,
		cfn:      cfn,
		callback: callback,
		forward:  forward,
		listener: listener,
	}

	if forward && forwardAddrStr != "" && forwardListenPort != 0 {
		forwardAddr, err := net.ResolveUDPAddr("udp", forwardAddrStr)

		if err != nil {
			return nil, err
		}

		u.forwarder, err = net.DialUDP("udp", &net.UDPAddr{IP: net.ParseIP(addr), Port: forwardListenPort}, forwardAddr)

		if err != nil {
			return nil, err
		}
	}

	go u.serve()
	go u.forwardServe()
	logrus.Debugf("Started new UDP server connection")

	return u, nil
}

type CallbackFunc func(response Message)

type AssettoServerUDP struct {
	listener  *net.UDPConn
	forwarder *net.UDPConn

	forward bool

	cfn      func()
	ctx      context.Context
	callback CallbackFunc

	closed bool
}

func (asu *AssettoServerUDP) Close() error {
	if asu.closed {
		return nil
	}

	defer func() {
		asu.closed = true
	}()

	asu.cfn()
	err := asu.listener.Close()

	if err != nil {
		return err
	}

	if asu.forwarder != nil {
		err = asu.forwarder.Close()

		if err != nil {
			return err
		}
	}

	logrus.Debugf("Closed UDP server connection")

	return nil
}

func (asu *AssettoServerUDP) forwardServe() {
	if !asu.forward || asu.forwarder == nil {
		return
	}

	for {
		select {
		case <-asu.ctx.Done():
			asu.forwarder.Close()
			return
		default:
			buf := make([]byte, 1024)

			n, _, err := asu.forwarder.ReadFromUDP(buf)

			if err != nil {
				continue
			}

			_, err = asu.listener.Write(buf[:n])

			if err != nil {
				continue
			}
		}
	}
}

func (asu *AssettoServerUDP) serve() {
	messageChan := make(chan []byte, 1000)
	defer close(messageChan)

	CurrentRealtimePosIntervalMs = RealtimePosIntervalMs
	lastQueueSize := 0

	go func() {
		ticker := time.NewTicker(time.Second)

		for {
			select {
			case buf := <-messageChan:
				msg, err := asu.handleMessage(bytes.NewReader(buf))

				if err != nil {
					logrus.WithError(err).Error("could not handle UDP message")
					return
				}

				asu.callback(msg)

				if asu.forward && asu.forwarder != nil {
					// write the message to the forwarding address
					_, _ = asu.forwarder.Write(buf)
				}
			case <-ticker.C:
				if RealtimePosIntervalMs < 0 || !PosIntervalModifierEnabled {
					// there is no real time pos interval set or stracker is enabled, we don't need to check if we're keeping up with messages
					continue
				}

				currentQueueSize := len(messageChan)

				if currentQueueSize > lastQueueSize {
					logrus.Warnf("Can't keep up! queue size: %d vs %d: changed by %d", currentQueueSize, lastQueueSize, currentQueueSize-lastQueueSize)

					// update as infrequently as we can, within sensible limits
					if currentQueueSize > 5 { // at this point we are half a second behind
						CurrentRealtimePosIntervalMs += (currentQueueSize * 2) + 1

						logrus.Debugf("Adjusting real time pos interval: %d", CurrentRealtimePosIntervalMs)
						err := asu.SendMessage(NewEnableRealtimePosInterval(CurrentRealtimePosIntervalMs))

						if err != nil {
							logrus.WithError(err).Error("Could not send realtime pos interval adjustment")
						}
					}
				} else if currentQueueSize <= lastQueueSize && currentQueueSize < 5 && CurrentRealtimePosIntervalMs > RealtimePosIntervalMs {
					logrus.Debugf("Catching up, queue size: %d vs %d: changed by %d", currentQueueSize, lastQueueSize, currentQueueSize-lastQueueSize)

					if CurrentRealtimePosIntervalMs-1 >= RealtimePosIntervalMs {
						CurrentRealtimePosIntervalMs--

						logrus.Debugf("Adjusting real time pos interval, is now: %d", CurrentRealtimePosIntervalMs)
						err := asu.SendMessage(NewEnableRealtimePosInterval(CurrentRealtimePosIntervalMs))

						if err != nil {
							logrus.WithError(err).Error("Could not send realtime pos interval adjustment")
						}
					}
				}

				lastQueueSize = currentQueueSize
			case <-asu.ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	for {
		select {
		case <-asu.ctx.Done():
			asu.listener.Close()
			return
		default:
			buf := make([]byte, 1024)

			// read message from assetto
			n, _, err := asu.listener.ReadFromUDP(buf)

			if err != nil {
				logrus.WithError(err).Debug("could not read from UDP")
				continue
			}

			messageChan <- buf[:n]
		}
	}
}

func readStringW(r io.Reader) string {
	return readString(r, 4)
}

func readString(r io.Reader, sizeMultiplier int) string {
	var size uint8
	err := binary.Read(r, binary.LittleEndian, &size)

	if err != nil {
		return ""
	}

	b := make([]byte, int(size)*sizeMultiplier)

	err = binary.Read(r, binary.LittleEndian, &b)

	if err != nil {
		return ""
	}

	if sizeMultiplier == 4 {
		bs, err := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewDecoder().Bytes(b)

		if err != nil {
			return ""
		}

		return string(bs)
	}

	return string(b)
}

func (asu *AssettoServerUDP) SendMessage(message Message) error {
	switch a := message.(type) {
	case EnableRealtimePosInterval:
		if PosIntervalModifierEnabled {
			return binary.Write(asu.listener, binary.LittleEndian, a)
		}

		return nil

	case GetSessionInfo, *RestartSession, *NextSession:
		err := binary.Write(asu.listener, binary.LittleEndian, a.Event())

		if err != nil {
			return err
		}

		return err

	case *SendChat:
		buf := new(bytes.Buffer)

		if err := binary.Write(buf, binary.LittleEndian, a.EventType); err != nil {
			return err
		}

		if err := binary.Write(buf, binary.LittleEndian, a.CarID); err != nil {
			return err
		}

		if err := binary.Write(buf, binary.LittleEndian, a.Len); err != nil {
			return err
		}

		if _, err := buf.Write(a.UTF32Encoded); err != nil {
			return err
		}

		if _, err := io.Copy(asu.listener, buf); err != nil {
			return err
		}

		return nil

	case *BroadcastChat:
		buf := new(bytes.Buffer)

		if err := binary.Write(buf, binary.LittleEndian, a.EventType); err != nil {
			return err
		}

		if err := binary.Write(buf, binary.LittleEndian, a.Len); err != nil {
			return err
		}

		if _, err := buf.Write(a.UTF32Encoded); err != nil {
			return err
		}

		if _, err := io.Copy(asu.listener, buf); err != nil {
			return err
		}

		return nil

	case *AdminCommand:
		buf := new(bytes.Buffer)

		if err := binary.Write(buf, binary.LittleEndian, a.EventType); err != nil {
			return err
		}

		if err := binary.Write(buf, binary.LittleEndian, a.Len); err != nil {
			return err
		}

		if _, err := buf.Write(a.UTF32Encoded); err != nil {
			return err
		}

		if _, err := io.Copy(asu.listener, buf); err != nil {
			return err
		}

		return nil

	case *KickUser:
		buf := new(bytes.Buffer)

		if err := binary.Write(buf, binary.LittleEndian, a.EventType); err != nil {
			return err
		}

		if err := binary.Write(buf, binary.LittleEndian, a.CarID); err != nil {
			return err
		}

		if _, err := io.Copy(asu.listener, buf); err != nil {
			return err
		}

		return nil

	}

	return errors.New("udp: invalid message type")
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

		carMode := readString(r, 1)
		carSkin := readString(r, 1)

		response = SessionCarInfo{
			CarID:      carID,
			DriverName: driverName,
			DriverGUID: DriverGUID(driverGUID),
			CarModel:   carMode,
			CarSkin:    carSkin,
			EventType:  eventType,
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

		if err != nil {
			return nil, err
		}

		response = CarInfo{
			CarID:       carID,
			IsConnected: isConnected != 0,
			CarModel:    readStringW(r),
			CarSkin:     readStringW(r),
			DriverName:  readStringW(r),
			DriverTeam:  readStringW(r),
			DriverGUID:  DriverGUID(readStringW(r)),
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
		sessionInfo.Track = readString(r, 1)
		sessionInfo.TrackConfig = readString(r, 1)
		sessionInfo.Name = readString(r, 1)

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

		sessionInfo.WeatherGraphics = readString(r, 1)

		err = binary.Read(r, binary.LittleEndian, &sessionInfo.ElapsedMilliseconds)

		if err != nil {
			return nil, err
		}

		sessionInfo.EventType = eventType

		response = sessionInfo

		if RealtimePosIntervalMs > 0 && eventType == EventNewSession {
			err = asu.SendMessage(NewEnableRealtimePosInterval(RealtimePosIntervalMs))

			if err != nil {
				return nil, err
			}
		}
	case EventError:
		message := readStringW(r)

		response = ServerError{errors.New(message)}

	case EventLapCompleted:
		lapCompleted := lapCompletedInternal{}

		err := binary.Read(r, binary.LittleEndian, &lapCompleted)

		if err != nil {
			return nil, err
		}

		lc := LapCompleted{
			CarID:     lapCompleted.CarID,
			LapTime:   lapCompleted.LapTime,
			Cuts:      lapCompleted.Cuts,
			CarsCount: lapCompleted.CarsCount,
		}

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
		fmt.Printf("%x\n", buf)

		return nil, errors.New("unknown response type")
	}

	return response, nil
}
