package servermanager

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	CMJoinLinkBase string = "https://acstuff.ru/s/q:race/online/join"
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

type RaceControlHub struct {
	clients   map[*raceControlClient]bool
	broadcast chan raceControlMessage
	register  chan *raceControlClient
}

func (h *RaceControlHub) Send(message udp.Message) error {
	h.broadcast <- newRaceControlMessage(message)

	return nil
}

func newRaceControlHub() *RaceControlHub {
	return &RaceControlHub{
		broadcast: make(chan raceControlMessage),
		register:  make(chan *raceControlClient),
		clients:   make(map[*raceControlClient]bool),
	}
}

func (h *RaceControlHub) run() {
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
	hub *RaceControlHub

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

type RaceControlHandler struct {
	*BaseHandler
	serverProcess ServerProcess

	store          Store
	raceManager    *RaceManager
	raceControl    *RaceControl
	raceControlHub *RaceControlHub
}

func NewRaceControlHandler(baseHandler *BaseHandler, store Store, raceManager *RaceManager, raceControl *RaceControl, raceControlHub *RaceControlHub, serverProcess ServerProcess) *RaceControlHandler {
	return &RaceControlHandler{
		BaseHandler:    baseHandler,
		store:          store,
		raceManager:    raceManager,
		raceControl:    raceControl,
		raceControlHub: raceControlHub,
		serverProcess:  serverProcess,
	}
}

func (rch *RaceControlHandler) liveTiming(w http.ResponseWriter, r *http.Request) {
	currentRace, entryList := rch.raceManager.CurrentRace()

	var customRace *CustomRace

	if currentRace != nil {
		customRace = &CustomRace{EntryList: entryList, RaceConfig: currentRace.CurrentRaceConfig}
	}

	frameLinks, err := rch.store.ListPrevFrames()

	if err != nil {
		logrus.Errorf("could not get frame links, err: %s", err)
		return
	}

	linkString := ""

	if rch.serverProcess.GetServerConfig().GlobalServerConfig.ShowContentManagerJoinLink == 1 {
		link, err := rch.getCMJoinLink()

		if err != nil {
			logrus.Errorf("could not get CM join link, err: %s", err)
		}

		linkString = link.String()
	}

	rch.viewRenderer.MustLoadTemplate(w, r, "live-timing.html", map[string]interface{}{
		"RaceDetails":     customRace,
		"FrameLinks":      frameLinks,
		"CSSDotSmoothing": udp.RealtimePosIntervalMs,
		"WideContainer":   true,
		"CMJoinLink":      linkString,
	})
}

func (rch *RaceControlHandler) getCMJoinLink() (*url.URL, error) {
	// get the join link for the current session
	geoIP, err := geoIP()

	if err != nil {
		return nil, err
	}

	cmUrl, err := url.Parse(CMJoinLinkBase)

	if err != nil {
		return nil, err
	}

	queryString := cmUrl.Query()
	queryString.Set("ip", geoIP.IP)
	queryString.Set("httpPort", strconv.Itoa(rch.serverProcess.GetServerConfig().GlobalServerConfig.HTTPPort))

	cmUrl.RawQuery = queryString.Encode()

	return cmUrl, nil
}

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func (rch *RaceControlHandler) saveIFrames(w http.ResponseWriter, r *http.Request) {
	// Save the frame links from the form
	err := r.ParseForm()

	if err != nil {
		logrus.Errorf("could not load parse form, err: %s", err)
		return
	}

	err = rch.store.UpsertLiveFrames(deleteEmpty(r.Form["frame-link"]))

	if err != nil {
		logrus.Errorf("could not save frame links, err: %s", err)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (rch *RaceControlHandler) websocket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Error(err)
		return
	}

	client := &raceControlClient{hub: rch.raceControlHub, conn: c, receive: make(chan raceControlMessage, 256)}
	client.hub.register <- client

	go client.writePump()

	// new client, send them an initial race control message.
	client.receive <- newRaceControlMessage(rch.raceControl)
}
