package servermanager

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/cj123/ini"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
)

var (
	websocketLastSessionInfo *udp.SessionInfo
	websocketTrackMapData    *TrackMapData
	connectedCars            = make(map[udp.CarID]udp.SessionCarInfo)
)

type liveMapHub struct {
	clients   map[*liveMapClient]bool
	broadcast chan udp.Message
	register  chan *liveMapClient
}

func newLiveMapHub() *liveMapHub {
	return &liveMapHub{
		broadcast: make(chan udp.Message),
		register:  make(chan *liveMapClient),
		clients:   make(map[*liveMapClient]bool),
	}
}

func (h *liveMapHub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

type liveMapClient struct {
	hub *liveMapHub

	conn *websocket.Conn
	send chan udp.Message
}

func (c *liveMapClient) writePump() {
	ticker := time.NewTicker(time.Second * 10)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteJSON(message)

			if err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var mapHub = newLiveMapHub()

func init() {
	go mapHub.run()
}

func LiveMapHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Error(err)
		return
	}

	client := &liveMapClient{hub: mapHub, conn: c, send: make(chan udp.Message, 256)}
	client.hub.register <- client

	go client.writePump()

	// new client. send them the session info if we have it
	if websocketLastSessionInfo != nil {
		client.send <- websocketLastSessionInfo
	}

	if websocketTrackMapData != nil {
		client.send <- websocketTrackMapData
	}

	for _, car := range connectedCars {
		client.send <- car
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
	case udp.CarUpdate, *TrackMapData, udp.CollisionWithEnvironment, udp.CollisionWithCar:
	default:
		return
	}

	mapHub.broadcast <- message
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
