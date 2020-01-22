package servermanager

import (
	"net/http"
	"strings"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"
	"github.com/mitchellh/go-wordwrap"

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
		broadcast: make(chan raceControlMessage, 1000),
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

type liveTimingTemplateVars struct {
	BaseTemplateVars

	RaceDetails       *CustomRace
	FrameLinks        []string
	CSSDotSmoothing   int
	CMJoinLink        string
	UseMPH            bool
	IsStrackerEnabled bool
}

func (rch *RaceControlHandler) liveTiming(w http.ResponseWriter, r *http.Request) {
	currentRace, entryList := rch.raceManager.CurrentRace()

	var customRace *CustomRace

	if currentRace != nil {
		customRace = &CustomRace{EntryList: entryList, RaceConfig: currentRace.CurrentRaceConfig}
	}

	frameLinks, err := rch.store.ListPrevFrames()

	if err != nil {
		logrus.WithError(err).Errorf("could not get frame links")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	linkString := ""

	if rch.serverProcess.GetServerConfig().GlobalServerConfig.ShowContentManagerJoinLink == 1 {
		link, err := getContentManagerJoinLink(rch.serverProcess.GetServerConfig())

		if err != nil {
			logrus.WithError(err).Errorf("could not get content manager join link")
		} else {
			linkString = link.String()
		}
	}

	serverOpts, err := rch.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	strackerOptions, err := rch.store.LoadStrackerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rch.viewRenderer.MustLoadTemplate(w, r, "live-timing.html", &liveTimingTemplateVars{
		BaseTemplateVars: BaseTemplateVars{
			WideContainer: true,
		},
		RaceDetails:       customRace,
		FrameLinks:        frameLinks,
		CSSDotSmoothing:   udp.RealtimePosIntervalMs,
		CMJoinLink:        linkString,
		UseMPH:            serverOpts.UseMPH == 1,
		IsStrackerEnabled: IsStrackerInstalled() && strackerOptions.EnableStracker,
	})
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
		logrus.WithError(err).Errorf("could not load parse form")
		return
	}

	err = rch.store.UpsertLiveFrames(deleteEmpty(r.Form["frame-link"]))

	if err != nil {
		logrus.WithError(err).Errorf("could not save frame links")
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

func (rch *RaceControlHandler) broadcastChat(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		return
	}

	wrapped := strings.Split(wordwrap.WrapString(
		r.FormValue("broadcast-chat"),
		60,
	), "\n")

	for _, msg := range wrapped {
		broadcastMessage, err := udp.NewBroadcastChat(msg)

		if err == nil {
			err := rch.serverProcess.SendUDPMessage(broadcastMessage)

			if err != nil {
				logrus.WithError(err).Errorf("Unable to broadcast chat message")
			}
		} else {
			logrus.WithError(err).Errorf("Unable to build chat message")
		}
	}
}

func (rch *RaceControlHandler) adminCommand(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		return
	}

	adminCommand, err := udp.NewAdminCommand(r.FormValue("admin-command"))

	if err == nil {
		err := rch.serverProcess.SendUDPMessage(adminCommand)

		if err != nil {
			logrus.WithError(err).Errorf("Unable to send admin command")
		}
	} else {
		logrus.WithError(err).Errorf("Unable to build admin command")
	}
}

func (rch *RaceControlHandler) kickUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		return
	}

	guid := r.FormValue("kick-user")

	if (guid == "") || (guid == "default-driver-spacer") {
		return
	}

	var carID uint8

	for id, rangeGuid := range rch.raceControl.CarIDToGUID {
		if string(rangeGuid) == guid {
			carID = uint8(id)
			break
		}
	}

	kickUser := udp.NewKickUser(carID)

	err := rch.serverProcess.SendUDPMessage(kickUser)

	if err != nil {
		logrus.WithError(err).Errorf("Unable to send kick command")
	}
}

func (rch *RaceControlHandler) sendChat(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		return
	}

	guid := r.FormValue("chat-user")

	if (guid == "") || (guid == "default-driver-spacer") {
		return
	}

	var carID uint8

	for id, rangeGuid := range rch.raceControl.CarIDToGUID {
		if string(rangeGuid) == guid {
			carID = uint8(id)
			break
		}
	}

	wrapped := strings.Split(wordwrap.WrapString(
		r.FormValue("send-chat"),
		60,
	), "\n")

	for _, msg := range wrapped {
		welcomeMessage, err := udp.NewSendChat(udp.CarID(carID), msg)

		if err == nil {
			err := rch.serverProcess.SendUDPMessage(welcomeMessage)

			if err != nil {
				logrus.WithError(err).Errorf("Unable to send chat message to car: %d", carID)
			}
		} else {
			logrus.WithError(err).Errorf("Unable to build chat message to car: %d", carID)
		}
	}
}

func (rch *RaceControlHandler) restartSession(w http.ResponseWriter, r *http.Request) {
	err := rch.serverProcess.SendUDPMessage(&udp.RestartSession{})

	if err != nil {
		logrus.WithError(err).Errorf("Unable to restart session")

		AddErrorFlash(w, r, "The server was unable to restart the session!")
	}

	http.Redirect(w, r, "/live-timing", http.StatusFound)
}

func (rch *RaceControlHandler) nextSession(w http.ResponseWriter, r *http.Request) {
	err := rch.serverProcess.SendUDPMessage(&udp.NextSession{})

	if err != nil {
		logrus.WithError(err).Errorf("Unable to move to next session")

		AddErrorFlash(w, r, "The server was unable to move to the next session!")
	}

	http.Redirect(w, r, "/live-timing", http.StatusFound)
}
