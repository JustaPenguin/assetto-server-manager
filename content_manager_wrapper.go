package servermanager

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/jaytaylor/html2text"
	"github.com/sirupsen/logrus"
)

const (
	// contentManagerWrapperSeparator is a special character used by Content Manager to determine which port
	// the content manager wrapper is running on. The server name shows "<server name> <separator><port>",
	// which Content Manager uses to split the string and find the port.
	contentManagerWrapperSeparator = 'ℹ'

	ContentManagerJoinLinkBase string = "https://acstuff.ru/s/q:race/online/join"
)

type ContentManagerWrapperData struct {
	ACHTTPSessionInfo
	Players ACHTTPPlayers `json:"players"`

	Description string `json:"description"`

	AmbientTemperature uint8  `json:"ambientTemperature"`
	RoadTemperature    uint8  `json:"roadTemperature"`
	WindDirection      int    `json:"windDirection"`
	WindSpeed          int    `json:"windSpeed"`
	CurrentWeatherID   string `json:"currentWeatherId"`
	Grip               int    `json:"grip"`
	GripTransfer       int    `json:"gripTransfer"`

	Assists          CMAssists `json:"assists"`
	MaxContactsPerKM int       `json:"maxContactsPerKm"`

	City             string    `json:"city"`
	PasswordChecksum [2]string `json:"passwordChecksum"`
	WrappedPort      int       `json:"wrappedPort"`

	Content   CMContent `json:"content"`
	Frequency int       `json:"frequency"`
	Until     int64     `json:"until"`
}

type CMAssists struct {
	ABSState             int  `json:"absState"`
	AllowedTyresOut      int  `json:"allowedTyresOut"`
	AutoClutchAllowed    bool `json:"autoclutchAllowed"`
	DamageMultiplier     int  `json:"damageMultiplier"`
	ForceVirtualMirror   bool `json:"forceVirtualMirror"`
	FuelRate             int  `json:"fuelRate"`
	StabilityAllowed     bool `json:"stabilityAllowed"`
	TractionControlState int  `json:"tcState"`
	TyreBlanketsAllowed  bool `json:"tyreBlanketsAllowed"`
	TyreWearRate         int  `json:"tyreWearRate"`
}

type CMContent struct {
	Cars     struct{} `json:"cars"`
	Password bool     `json:"password"`
}

type ACHTTPPlayers struct {
	Cars []*CMCar `json:"Cars"`
}

type CMCar struct {
	DriverName      string `json:"DriverName"`
	DriverNation    string `json:"DriverNation"`
	DriverTeam      string `json:"DriverTeam"`
	ID              string `json:"ID"`
	IsConnected     bool   `json:"IsConnected"`
	IsEntryList     bool   `json:"IsEntryList"`
	IsRequestedGUID bool   `json:"IsRequestedGUID"`
	Model           string `json:"Model"`
	Skin            string `json:"Skin"`
}

type ContentManagerWrapper struct {
	store      Store
	carManager *CarManager
	trackManager *TrackManager

	sessionInfo udp.SessionInfo

	reverseProxy *httputil.ReverseProxy
	serverConfig ServerConfig
	entryList    EntryList
	event        RaceEvent

	srv         *http.Server
	description string
	mutex       sync.Mutex
}

func NewContentManagerWrapper(store Store, carManager *CarManager, trackManager *TrackManager) *ContentManagerWrapper {
	return &ContentManagerWrapper{
		store:      store,
		carManager: carManager,
		trackManager: trackManager,
	}
}

func (cmw *ContentManagerWrapper) UDPCallback(message udp.Message) {
	switch m := message.(type) {
	case udp.SessionInfo:
		cmw.sessionInfo = m
	}
}

func (cmw *ContentManagerWrapper) setDescriptionText(event RaceEvent) error {
	text, err := html2text.FromString(cmw.serverConfig.GlobalServerConfig.ContentManagerWelcomeMessage)

	if err != nil {
		return err
	}

	eventDescriptionAsText, err := html2text.FromString(event.EventDescription(), html2text.Options{PrettyTables: true})

	if err != nil {
		return err
	}

	if len(text) > 0 {
		text += "\n\n"
	}

	text += eventDescriptionAsText

	if champ, ok := cmw.event.(*ActiveChampionship); ok {
		if u := champ.GetURL(); u != "" {
			text += fmt.Sprintf("\n\nView the Championship points here: %s", u)
		}
	}

	for _, carName := range strings.Split(cmw.serverConfig.CurrentRaceConfig.Cars, ";") {
		car, err := cmw.carManager.LoadCar(carName, nil)

		if err != nil {
			logrus.WithError(err).Warnf("Could not load car details for: %s, skipping attaching download URLs to Content Manager Wrapper", carName)
			continue
		}

		if car.Details.DownloadURL == "" {
			continue
		}

		text += fmt.Sprintf("\n* %s Download: %s", car.Details.Name, car.Details.DownloadURL)
	}

	track, err := cmw.trackManager.GetTrackFromName(cmw.serverConfig.CurrentRaceConfig.Track)

	if err != nil {
		logrus.WithError(err).Warnf("Could not load track: %s, skipping attaching download URL to Content Manager Wrapper", cmw.serverConfig.CurrentRaceConfig.Track)
	} else {
		err := track.LoadMetaData()

		if err != nil {
			logrus.WithError(err).Warnf("Could not load meta data for: %s, skipping attaching download URL to Content Manager Wrapper", cmw.serverConfig.CurrentRaceConfig.Track)
		} else {
			text += fmt.Sprintf("\n* %s Download: %s", track.Name, track.MetaData.DownloadURL)
		}
	}

	cmw.description = text

	return nil
}

func (cmw *ContentManagerWrapper) Start(process ServerProcess, servePort int, serverConfig ServerConfig, entryList EntryList, event RaceEvent) error {
	cmw.mutex.Lock()
	defer cmw.mutex.Unlock()

	logrus.Infof("Starting content manager wrapper server on port %d", servePort)

	u, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", serverConfig.GlobalServerConfig.HTTPPort))

	if err != nil {
		return err
	}

	cmw.serverConfig = serverConfig
	cmw.entryList = entryList
	cmw.event = event
	cmw.reverseProxy = httputil.NewSingleHostReverseProxy(u)

	if err := cmw.setDescriptionText(event); err != nil {
		logrus.WithError(err).Warn("could not set description text")
	}

	cmw.srv = &http.Server{Addr: fmt.Sprintf(":%d", servePort)}
	cmw.srv.Handler = cmw

	if err := cmw.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (cmw *ContentManagerWrapper) Stop() {
	if cmw.srv == nil {
		return
	}

	logrus.Infof("Shutting down content manager wrapper server")
	err := cmw.srv.Shutdown(context.Background())

	if err != nil {
		logrus.WithError(err).Error("Could not shutdown content manager wrapper server")
	}
}

func (cmw *ContentManagerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/details" {
		cmw.reverseProxy.ServeHTTP(w, r)
		return
	}

	details, err := cmw.buildContentManagerDetails(r.URL.Query().Get("guid"))

	if err != nil {
		logrus.WithError(err).Error("could not build content manager details")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	if Debug {
		enc.SetIndent("", "    ")
	}
	_ = enc.Encode(details)
}

type ACHTTPSessionInfo struct {
	Cars         []string    `json:"cars"`
	Clients      int64       `json:"clients"`
	Country      []string    `json:"country"`
	HTTPPort     int64       `json:"cport"`
	Durations    []int64     `json:"durations"`
	Extra        bool        `json:"extra"`
	Inverted     int64       `json:"inverted"`
	IP           string      `json:"ip"`
	JSON         interface{} `json:"json"`
	L            bool        `json:"l"`
	MaxClients   int64       `json:"maxclients"`
	Name         string      `json:"name"`
	Pass         bool        `json:"pass"`
	Pickup       bool        `json:"pickup"`
	Pit          bool        `json:"pit"`
	Port         int64       `json:"port"`
	Session      int64       `json:"session"`
	Sessiontypes []int64     `json:"sessiontypes"`
	Timed        bool        `json:"timed"`
	Timeleft     int64       `json:"timeleft"`
	Timeofday    int64       `json:"timeofday"`
	Timestamp    int64       `json:"timestamp"`
	TCPPort      int64       `json:"tport"`
	Track        string      `json:"track"`
}

func (cmw *ContentManagerWrapper) getSessionInfo() (*ACHTTPSessionInfo, error) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/INFO", cmw.serverConfig.GlobalServerConfig.HTTPPort))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var sessionInfo *ACHTTPSessionInfo

	if err := json.NewDecoder(resp.Body).Decode(&sessionInfo); err != nil {
		return nil, err
	}

	return sessionInfo, nil
}

func (cmw *ContentManagerWrapper) getPlayers(guid string) (*ACHTTPPlayers, error) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/JSON|%s", cmw.serverConfig.GlobalServerConfig.HTTPPort, guid))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var clients *ACHTTPPlayers

	if err := json.NewDecoder(resp.Body).Decode(&clients); err != nil {
		return nil, err
	}

	return clients, nil
}

func (cmw *ContentManagerWrapper) buildContentManagerDetails(guid string) (*ContentManagerWrapperData, error) {
	race := cmw.serverConfig.CurrentRaceConfig
	global := cmw.serverConfig.GlobalServerConfig
	live := cmw.sessionInfo

	sessionInfo, err := cmw.getSessionInfo()

	if err != nil {
		return nil, err
	}

	for index, duration := range sessionInfo.Durations {
		// convert durations to seconds
		if !(race.HasSession(SessionTypeRace) && index == len(sessionInfo.Durations)-1 && race.Sessions[SessionTypeRace].Laps > 0) {
			sessionInfo.Durations[index] = duration * 60
		}
	}

	players, err := cmw.getPlayers(guid)

	if err != nil {
		return nil, err
	}

	for entrantNum, entrant := range cmw.entryList.AsSlice() {
		if entrantNum < len(players.Cars) {
			players.Cars[entrantNum].ID = contentManagerIDChecksum(entrant.GUID)
		}
	}
	var passwordChecksum [2]string

	if global.Password != "" {
		passwordChecksum[0] = contentManagerPasswordChecksum(global.Name, global.Password)
		passwordChecksum[1] = contentManagerPasswordChecksum(global.Name, global.AdminPassword)
	}

	geoInfo, err := geoIP()

	if err != nil {
		logrus.WithError(err).Warn("could not get geo IP data")
		geoInfo = &GeoIP{}
	}

	serverNameSplit := strings.Split(sessionInfo.Name, fmt.Sprintf(" %c", contentManagerWrapperSeparator))

	if len(serverNameSplit) > 0 {
		sessionInfo.Name = serverNameSplit[0]
	}

	sessionInfo.IP = geoInfo.IP
	sessionInfo.Country = []string{geoInfo.CountryName, geoInfo.CountryCode}

	var description string

	if champ, ok := cmw.event.(*ActiveChampionship); ok {
		championship, err := cmw.store.LoadChampionship(champ.ChampionshipID.String())

		if err == nil {
			description = championship.GetPlayerSummary(guid) + "\n\n"
		} else {
			logrus.WithError(err).Warn("can't load championship info")
		}
	}

	description += cmw.description

	return &ContentManagerWrapperData{
		ACHTTPSessionInfo: *sessionInfo,
		Players:           *players,

		Description: description,

		AmbientTemperature: live.AmbientTemp,
		RoadTemperature:    live.RoadTemp,
		WindDirection:      race.WindBaseDirection,
		WindSpeed:          race.WindBaseSpeedMin,
		CurrentWeatherID:   getSolWeatherPrettyName(live.WeatherGraphics),
		Grip:               race.DynamicTrack.SessionStart,
		GripTransfer:       race.DynamicTrack.SessionTransfer,

		// rules
		Assists: CMAssists{
			ABSState:             race.ABSAllowed,
			AllowedTyresOut:      race.AllowedTyresOut,
			AutoClutchAllowed:    race.AutoClutchAllowed == 1,
			DamageMultiplier:     race.DamageMultiplier,
			ForceVirtualMirror:   race.ForceVirtualMirror == 1,
			FuelRate:             race.FuelRate,
			StabilityAllowed:     race.StabilityControlAllowed == 1,
			TractionControlState: race.TractionControlAllowed,
			TyreBlanketsAllowed:  race.TyreBlanketsAllowed == 1,
			TyreWearRate:         race.TyreWearRate,
		},

		MaxContactsPerKM: race.MaxContactsPerKilometer,

		// server info
		PasswordChecksum: passwordChecksum,
		WrappedPort:      global.ContentManagerWrapperPort,

		Content:   CMContent{}, // not supported
		Frequency: global.ClientSendIntervalInHertz,
		Until:     time.Now().Add(time.Second * time.Duration(sessionInfo.Timeleft)).Unix(),
	}, nil
}

func getSolWeatherPrettyName(weatherName string) string {
	if !strings.HasPrefix(weatherName, "sol_") {
		return weatherName
	}

	parts := strings.Split(weatherName, "_type")

	if len(parts) == 0 {
		return weatherName
	}

	// remove sol_ prefix and transform to lower case
	solName := strings.ToLower(strings.TrimPrefix(parts[0], "sol_"))

	// remove underscores, convert to title case
	solName = strings.Title(strings.Replace(solName, "_", " ", -1))

	return "Sol: " + solName
}

func contentManagerPasswordChecksum(serverName, password string) string {
	h := sha1.New()
	h.Write([]byte("apatosaur" + serverName + password))

	return hex.EncodeToString(h.Sum(nil))
}

func contentManagerIDChecksum(guid string) string {
	h := sha1.New()
	h.Write([]byte("antarcticfurseal" + guid))

	return hex.EncodeToString(h.Sum(nil))
}

var geoIPData *GeoIP

const geoIPURL = "https://geoip.cj.workers.dev"

type GeoIP struct {
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	IP          string `json:"ip"`
}

func geoIP() (*GeoIP, error) {
	if geoIPData != nil {
		return geoIPData, nil
	}

	resp, err := http.Get(geoIPURL)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&geoIPData); err != nil {
		return nil, err
	}

	return geoIPData, nil
}

func getContentManagerJoinLink(config ServerConfig) (*url.URL, error) {
	geoIP, err := geoIP()

	if err != nil {
		return nil, err
	}

	cmUrl, err := url.Parse(ContentManagerJoinLinkBase)

	if err != nil {
		return nil, err
	}

	queryString := cmUrl.Query()
	queryString.Set("ip", geoIP.IP)
	queryString.Set("httpPort", strconv.Itoa(config.GlobalServerConfig.HTTPPort))

	cmUrl.RawQuery = queryString.Encode()

	return cmUrl, nil
}
