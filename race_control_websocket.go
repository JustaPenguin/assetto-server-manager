package servermanager

import (
	"net/http"
	"strings"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
)

type Broadcaster interface {
	Send(message udp.Message) error
}

type NilBroadcaster struct{}

func (NilBroadcaster) Send(message udp.Message) error {
	logrus.WithField("message", message).Infof("Message send %d", message.Event())
	return nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

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
		if rvr := recover(); rvr != nil {
			logrus.WithField("panic", rvr).Errorf("Recovered from panic")
		}
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

			if err != nil && !strings.HasSuffix(err.Error(), "write: broken pipe") {
				logrus.WithError(err).Errorf("Could not send websocket message")
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

var raceControlWebsocketHub *raceControlHub

func raceControlWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Error(err)
		return
	}

	client := &raceControlClient{hub: raceControlWebsocketHub, conn: c, receive: make(chan raceControlMessage, 256)}
	client.hub.register <- client

	go client.writePump()

	// new client, send them an initial race control message.
	client.receive <- newRaceControlMessage(ServerRaceControl)
}
