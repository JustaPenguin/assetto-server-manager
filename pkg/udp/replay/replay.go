package replay

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"

	"github.com/etcd-io/bbolt"
	"github.com/sirupsen/logrus"
)

type Entries []*Entry

func (e Entries) Len() int {
	return len(e)
}

func (e Entries) Less(i, j int) bool {
	return e[i].Received.Before(e[j].Received)
}

func (e Entries) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

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
	default:
	}

	return nil
}

var BucketName = []byte("sessions")

func RecordUDPMessages(db *bbolt.DB) (callbackFunc udp.CallbackFunc) {
	return func(message udp.Message) {
		e := Entry{
			Received:  time.Now(),
			EventType: message.Event(),
			Data:      message,
		}

		buf := new(bytes.Buffer)

		encoder := json.NewEncoder(buf)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(e)

		if err != nil {
			logrus.WithError(err).Errorf("could not encode json")
			return
		}

		err = db.Update(func(tx *bbolt.Tx) error {
			bkt, err := tx.CreateBucketIfNotExists(BucketName)

			if err != nil {
				return err
			}

			return bkt.Put([]byte(e.Received.Format(time.RFC3339Nano)), buf.Bytes())
		})

		if err != nil {
			logrus.WithError(err).Errorf("could not save to bucket")
		}
	}
}

func UDPMessages(db *bbolt.DB, multiplier int, callbackFunc udp.CallbackFunc, waitTime time.Duration) error {
	var loadedEntries Entries

	var wg sync.WaitGroup

	err := db.View(func(tx *bbolt.Tx) error {
		err := tx.Bucket(BucketName).ForEach(func(k, v []byte) error {
			var entry *Entry
			err := json.Unmarshal(v, &entry)

			if err != nil {
				return err
			}

			loadedEntries = append(loadedEntries, entry)

			return err
		})

		if err != nil {
			return err
		}

		if len(loadedEntries) == 0 {
			return nil
		}

		sort.Sort(loadedEntries)

		timeStart := loadedEntries[0].Received

		for _, entry := range loadedEntries {
			entry := entry

			tickDuration := entry.Received.Sub(timeStart) / time.Duration(multiplier)

			if tickDuration > waitTime {
				tickDuration = waitTime
			}

			if tickDuration > 0 {
				tickWhenEventOccurs := time.NewTicker(tickDuration)
				<-tickWhenEventOccurs.C
				tickWhenEventOccurs.Stop()
			}

			wg.Add(1)

			go func() {
				callbackFunc(entry.Data)
				wg.Done()
			}()

			timeStart = entry.Received
		}

		return nil
	})

	wg.Wait()

	return err
}
