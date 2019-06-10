package servermanager

import (
	"net/http"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

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

func newRaceControlMessage(message udp.Message) raceControlMessage {
	return raceControlMessage{
		EventType: message.Event(),
		Message:   message,
	}
}

type raceControlMessage struct {
	EventType udp.Event
	Message   udp.Message
}

type raceControlHub struct {
	clients   map[*raceControlClient]bool
	broadcast chan raceControlMessage
	register  chan *raceControlClient
}

func (h *raceControlHub) Send(message udp.Message) error {
	h.broadcast <- newRaceControlMessage(message)

	return nil
}

func newRaceControlHub() *raceControlHub {
	return &raceControlHub{
		broadcast: make(chan raceControlMessage),
		register:  make(chan *raceControlClient),
		clients:   make(map[*raceControlClient]bool),
	}
}

func (h *raceControlHub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.receive <- message:
				default:
					close(client.receive)
					delete(h.clients, client)
				}
			}
			/*
					@TODO find a place for me
				case <-AssettoProcess.Done():
					connectedCars.Range(func(key, value interface{}) bool {
						client, ok := value.(udp.SessionCarInfo)

						if !ok {
							return true
						}

						client.EventType = udp.EventConnectionClosed

						go func() {
							h.broadcast <- raceControlMessage{udp.EventConnectionClosed, client}
						}()

						return true
					})

					connectedCars = sync.Map{}
			*/
		}
	}
}

type raceControlClient struct {
	hub *raceControlHub

	conn    *websocket.Conn
	receive chan raceControlMessage
}

func (c *raceControlClient) writePump() {
	ticker := time.NewTicker(time.Second * 10)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.receive:
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

var mapHub *raceControlHub

func raceControlHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Error(err)
		return
	}

	client := &raceControlClient{hub: mapHub, conn: c, receive: make(chan raceControlMessage, 256)}
	client.hub.register <- client

	go client.writePump()

	/*
		// new client. send them the session info if we have it
		if websocketLastSeenSessionInfo != nil {
			client.receive <- raceControlMessage{udp.EventNewSession, websocketLastSeenSessionInfo}
		}

		if websocketTrackMapData != nil {
			client.receive <- raceControlMessage{222, websocketTrackMapData}
		}

		connectedCars.Range(func(key, value interface{}) bool {
			car, ok := value.(udp.SessionCarInfo)

			if !ok {
				return true
			}

			client.receive <- raceControlMessage{udp.EventNewConnection, car}
			client.receive <- raceControlMessage{udp.EventClientLoaded, udp.ClientLoaded(car.CarID)}

			return true
		})*/
}
