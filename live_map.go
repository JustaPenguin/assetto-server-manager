package servermanager

import (
	"github.com/cj123/ini"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{}

var (
	udpMessageMutex sync.RWMutex
	udpMessageChan  = make(map[*websocket.Conn]chan udp.Message)

	websocketLastSessionInfo *udp.SessionInfo
	websocketTrackMapData    *TrackMapData

	connectedCars = make(map[udp.CarID]udp.SessionCarInfo)
)

func LiveMapHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Error(err)
		return
	}

	sendCh := make(chan udp.Message)
	udpMessageChan[c] = sendCh

	defer func() {
		logrus.Debugf("closing socket")

		close(sendCh)
		delete(udpMessageChan, c)
		c.Close()
	}()

	// new client. send them the session info if we have it

	if websocketLastSessionInfo != nil {
		err = c.WriteJSON(websocketLastSessionInfo)

		if err != nil {
			logrus.Error(err)
			return
		}
	}

	if websocketTrackMapData != nil {
		err = c.WriteJSON(websocketTrackMapData)

		if err != nil {
			logrus.Error(err)
			return
		}
	}

	for _, car := range connectedCars {
		err = c.WriteJSON(car)

		if err != nil {
			logrus.Error(err)
			return
		}
	}

	for {
		err = c.WriteJSON(<-sendCh)

		if err != nil {
			logrus.Error(err)
			return
		}
	}
}

func LiveMapCallback(message udp.Message) {
	switch m := message.(type) {

	case udp.SessionInfo:
		var err error

		websocketLastSessionInfo = &m
		websocketTrackMapData, err = LoadTrackMapData(m.Track, m.TrackConfig)

		if err != nil {
			logrus.Errorf("Could not load map data, err: %s", err)
			return
		}

		LiveMapCallback(websocketTrackMapData)

	case udp.SessionCarInfo:
		if m.Event() == udp.EventNewConnection {
			connectedCars[m.CarID] = m
		} else if m.Event() == udp.EventConnectionClosed {
			delete(connectedCars, m.CarID)
		}
	case udp.CarUpdate, *TrackMapData:
	default:
		return
	}

	for _, ch := range udpMessageChan {
		ch <- message
	}
}

type TrackMapData struct {
	Width       float64 `ini:"WIDTH"`
	Height      float64 `ini:"HEIGHT"`
	Margin      float64 `ini:"MARGIN"`
	ScaleFactor float64 `ini:"SCALE_FACTOR"`
	OffsetX     float64 `ini:"X_OFFSET"`
	OffsetZ     float64 `ini:"Z_OFFSET"`
	DrawingSize float64 `ini:"DRAWING_SIZE"`
}

func (*TrackMapData) Event() udp.Event {
	return 0
}

func LoadTrackMapData(track, trackLayout string) (*TrackMapData, error) {
	p := filepath.Join(ServerInstallPath, "content", "tracks", track)

	if trackLayout != "" {
		p = filepath.Join(p, trackLayout)
	}

	p = filepath.Join(p, "data", "map.ini")

	f, err := os.Open(p)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	i, err := ini.Load(f)

	if err != nil {
		return nil, err
	}

	s, err := i.GetSection("PARAMETERS")

	if err != nil {
		return nil, err
	}

	var mapData TrackMapData

	if err := s.MapTo(&mapData); err != nil {
		return nil, err
	}

	return &mapData, nil
}
