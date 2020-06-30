package servermanager

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
)

type Broadcaster interface {
	Send(message udp.Message) ([]byte, error)
}

type NilBroadcaster struct{}

func (NilBroadcaster) Send(message udp.Message) ([]byte, error) {
	logrus.WithField("message", message).Debugf("Message send %d", message.Event())
	return nil, nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func encodeRaceControlMessage(message udp.Message) ([]byte, error) {
	m := raceControlMessage{
		EventType: message.Event(),
		Message:   message,
	}

	return json.Marshal(m)
}

type raceControlMessage struct {
	EventType udp.Event
	Message   udp.Message
}

type RaceControlHub struct {
	clients   map[*raceControlClient]bool
	broadcast chan []byte
	register  chan *raceControlClient
}

func (h *RaceControlHub) Send(message udp.Message) ([]byte, error) {
	encoded, err := encodeRaceControlMessage(message)

	if err != nil {
		return nil, err
	}

	h.broadcast <- encoded

	return encoded, nil
}

func newRaceControlHub() *RaceControlHub {
	return &RaceControlHub{
		broadcast: make(chan []byte, 1000),
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
	receive chan []byte
}

func (c *raceControlClient) writePump() {
	ticker := time.NewTicker(time.Second * 10)
	defer func() {
		if rvr := recover(); rvr != nil {
			logrus.WithField("panic", rvr).Errorf("Recovered from panic")
		}
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.receive:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.TextMessage, message)

			if err != nil && !strings.HasSuffix(err.Error(), "write: broken pipe") {
				logrus.WithError(err).Errorf("Could not send websocket message")
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
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

	RaceDetails                 *CustomRace
	FrameLinks                  []string
	CSSDotSmoothing             int
	CMJoinLink                  string
	UseMPH                      bool
	IsStrackerEnabled           bool
	IsKissMyRankEnabled         bool
	KissMyRankWebStatsPublicURL string
	STrackerInterfacePublicURL  string
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

	serverOpts, err := rch.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	linkString := ""

	if serverOpts.ShowContentManagerJoinLink == 1 {
		link, err := getContentManagerJoinLink(*serverOpts)

		if err != nil {
			logrus.WithError(err).Errorf("could not get content manager join link")
		} else {
			linkString = link.String()
		}
	}

	strackerOptions, err := rch.store.LoadStrackerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load stracker options")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sTrackerPublicURL := strackerOptions.HTTPConfiguration.PublicURL

	if sTrackerPublicURL == "" {
		sTrackerPublicURL = "/stracker/mainpage"
	}

	kissMyRankOptions, err := rch.store.LoadKissMyRankOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load kissmyrank options")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rch.viewRenderer.MustLoadTemplate(w, r, "live-timing.html", &liveTimingTemplateVars{
		BaseTemplateVars: BaseTemplateVars{
			WideContainer: true,
		},
		RaceDetails:                 customRace,
		FrameLinks:                  frameLinks,
		CSSDotSmoothing:             udp.RealtimePosIntervalMs,
		CMJoinLink:                  linkString,
		UseMPH:                      serverOpts.UseMPH == 1,
		IsStrackerEnabled:           IsStrackerInstalled() && strackerOptions.EnableStracker,
		IsKissMyRankEnabled:         IsKissMyRankInstalled() && kissMyRankOptions.EnableKissMyRank,
		KissMyRankWebStatsPublicURL: kissMyRankOptions.WebStatsPublicURL,
		STrackerInterfacePublicURL:  sTrackerPublicURL,
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

	client := &raceControlClient{hub: rch.raceControlHub, conn: c, receive: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()

	// new client, send them an initial race control message.
	rch.raceControl.lastUpdateMessageMutex.Lock()
	client.receive <- rch.raceControl.lastUpdateMessage
	rch.raceControl.lastUpdateMessageMutex.Unlock()

	// send stored chat messages to new client
	rch.raceControl.ChatMessagesMutex.Lock()

	for _, message := range rch.raceControl.ChatMessages {
		encoded, err := encodeRaceControlMessage(message)

		if err != nil {
			continue
		}

		client.receive <- encoded
	}

	rch.raceControl.ChatMessagesMutex.Unlock()
}

func (rch *RaceControlHandler) broadcastChat(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		return
	}

	err := rch.raceControl.splitAndBroadcastChat(r.FormValue("broadcast-chat"), AccountFromRequest(r))

	if err != nil {
		logrus.WithError(err).Errorf("Unable to broadcast chat message")
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

	err := rch.raceControl.ConnectedDrivers.Each(func(driverGUID udp.DriverGUID, driver *RaceControlDriver) error {
		if string(driverGUID) != guid {
			return nil
		}

		command, err := udp.NewAdminCommand("/kick " + driver.CarInfo.DriverName)

		if err != nil {
			return err
		}

		return rch.serverProcess.SendUDPMessage(command)
	})

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

	err := rch.raceControl.splitAndSendChat(r.FormValue("send-chat"), guid)

	if err != nil {
		logrus.WithError(err).Errorf("Unable to send chat message to driver: %s", guid)
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

func (rch *RaceControlHandler) countdown(w http.ResponseWriter, r *http.Request) {

	// broadcast countdown
	ticker := time.NewTicker(time.Second * 3)
	i := 4

	for range ticker.C {
		var countdown string

		i--

		if i > 0 {
			countdown = strconv.Itoa(i)
		} else if i == 0 {
			countdown = "GO"
		} else {
			ticker.Stop()

			return
		}

		err := rch.raceControl.splitAndBroadcastChat(countdown, nil)

		if err != nil {
			logrus.WithError(err).Error("Unable to broadcast countdown message")
		}
	}
}
