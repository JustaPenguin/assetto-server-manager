package replay

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/sirupsen/logrus"
)

var entries []Entry

type Entry struct {
	Received  time.Time
	EventType udp.Event

	Data udp.Message
}

func (e *Entry) UnmarshalJSON(b []byte) error {
	var rawData map[string]json.RawMessage

	if err := json.Unmarshal(b, &rawData); err != nil {
		return err
	}

	eventType, ok := rawData["EventType"]

	if !ok {
		return errors.New("event type not specified")
	}

	if err := json.Unmarshal(eventType, &e.EventType); err != nil {
		return err
	}

	received, ok := rawData["Received"]

	if !ok {
		return errors.New("received time not specified")
	}

	if err := json.Unmarshal(received, &e.Received); err != nil {
		return err
	}

	msg := rawData["Data"]

	switch e.EventType {
	case udp.EventNewConnection, udp.EventConnectionClosed:
		var data *udp.SessionCarInfo

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = *data
	case udp.EventCarUpdate:
		var data *udp.CarUpdate

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = *data
	case udp.EventCarInfo:
		var data *udp.CarInfo

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = *data
	case udp.EventEndSession:
		var data udp.EndSession

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = data
	case udp.EventVersion:
		var data udp.Version

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = data
	case udp.EventChat:
		var data *udp.Chat

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = *data
	case udp.EventClientLoaded:
		var data udp.ClientLoaded

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = data
	case udp.EventNewSession, udp.EventSessionInfo:
		var data *udp.SessionInfo

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = *data
	case udp.EventError:
		var data *udp.ServerError

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = *data
	case udp.EventLapCompleted:
		var data *udp.LapCompleted

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = *data
	case udp.EventCollisionWithCar:
		var data *udp.CollisionWithCar

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = *data
	case udp.EventCollisionWithEnv:
		var data *udp.CollisionWithEnvironment

		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		e.Data = *data
	}

	return nil
}

func RecordUDPMessages(filename string) (callbackFunc udp.CallbackFunc) {
	return func(message udp.Message) {
		entries = append(entries, Entry{
			Received:  time.Now(),
			EventType: message.Event(),
			Data:      message,
		})

		f, err := os.Create(filename)

		if err != nil {
			panic(err)
		}

		defer f.Close()

		encoder := json.NewEncoder(f)
		encoder.SetIndent("", "  ")

		err = encoder.Encode(entries)

		if err != nil {
			fmt.Println("err encoding", err)
		}
	}
}

func ReplayUDPMessages(filename string, multiplier int, callbackFunc udp.CallbackFunc, async bool) error {
	var loadedEntries []*Entry

	f, err := os.Open(filename)

	if err != nil {
		return err
	}

	defer f.Close()

	if err := json.NewDecoder(f).Decode(&loadedEntries); err != nil {
		return err
	}

	if len(loadedEntries) == 0 {
		return nil
	}

	timeStart := loadedEntries[0].Received

	for _, entry := range loadedEntries {
		tickDuration := entry.Received.Sub(timeStart) / time.Duration(multiplier)

		logrus.Debugf("next tick occurs in: %s", tickDuration)

		if tickDuration > 0 {
			tickWhenEventOccurs := time.Tick(entry.Received.Sub(timeStart) / time.Duration(multiplier))
			<-tickWhenEventOccurs
		}

		if async {
			go callbackFunc(entry.Data)
		} else {
			callbackFunc(entry.Data)
		}

		timeStart = entry.Received
	}

	return nil
}
