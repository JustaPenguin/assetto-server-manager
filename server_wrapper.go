package servermanager

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const ContentManagerSeparator = 'ℹ'

type ContentManagerWrapperData struct {
	ACHTTPSessionInfo

	// session info
	Description string `json:"description"`

	AmbientTemperature uint8  `json:"ambientTemperature"`
	RoadTemperature    uint8  `json:"roadTemperature"`
	WindDirection      int    `json:"windDirection"`
	WindSpeed          int    `json:"windSpeed"`
	CurrentWeatherID   string `json:"currentWeatherId"`
	Grip               int    `json:"grip"`
	GripTransfer       int    `json:"gripTransfer"`

	// rules
	Assists          CMAssists `json:"assists"`
	MaxContactsPerKm int       `json:"maxContactsPerKm"`

	// server info
	City             string    `json:"city"`
	PasswordChecksum [2]string `json:"passwordChecksum"`
	WrappedPort      int       `json:"wrappedPort"`

	// entrants
	Players ACHTTPPlayers `json:"players"`

	Content   CMContent `json:"content"`
	Frequency int       `json:"frequency"` // refresh frequency?
	Until     int64     `json:"until"`     // no idea
}

type CMAssists struct {
	AbsState            int  `json:"absState"`
	AllowedTyresOut     int  `json:"allowedTyresOut"`
	AutoclutchAllowed   bool `json:"autoclutchAllowed"`
	DamageMultiplier    int  `json:"damageMultiplier"`
	ForceVirtualMirror  bool `json:"forceVirtualMirror"`
	FuelRate            int  `json:"fuelRate"`
	StabilityAllowed    bool `json:"stabilityAllowed"`
	TcState             int  `json:"tcState"`
	TyreBlanketsAllowed bool `json:"tyreBlanketsAllowed"`
	TyreWearRate        int  `json:"tyreWearRate"`
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
	raceControl *RaceControl
	process     ServerProcess

	reverseProxy *httputil.ReverseProxy
	serverConfig ServerConfig
	entryList    EntryList
}

func NewContentManagerWrapper(process ServerProcess, raceControl *RaceControl) *ContentManagerWrapper {
	return &ContentManagerWrapper{
		raceControl: raceControl,
		process:     process,
	}
}

func (cmw *ContentManagerWrapper) Start(servePort int, serverConfig ServerConfig, entryList EntryList) error {
	logrus.Infof("Starting content manager wrapper server on port %d", servePort)

	u, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", serverConfig.GlobalServerConfig.HTTPPort))

	if err != nil {
		return err
	}

	cmw.serverConfig = serverConfig
	cmw.entryList = entryList
	cmw.reverseProxy = httputil.NewSingleHostReverseProxy(u)

	srv := &http.Server{Addr: fmt.Sprintf(":%d", servePort)}
	srv.Handler = cmw

	go func() {
		<-cmw.process.Done()
		logrus.Infof("Shutting down content manager wrapper server on port %d", servePort)
		err := srv.Shutdown(context.Background())

		if err != nil {
			logrus.WithError(err).Error("Could not shutdown content manager wrapper server")
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
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
	enc.SetIndent("", "    ")
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
	sessionInfo, err := cmw.getSessionInfo()

	if err != nil {
		return nil, err
	}

	for index, duration := range sessionInfo.Durations {
		// convert durations to seconds
		sessionInfo.Durations[index] = duration * 60
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

	race := cmw.serverConfig.CurrentRaceConfig
	global := cmw.serverConfig.GlobalServerConfig
	live := cmw.raceControl

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

	sessionInfo.Name = global.Name
	sessionInfo.IP = geoInfo.IP
	sessionInfo.Country = []string{geoInfo.CountryName, geoInfo.CountryCode}

	return &ContentManagerWrapperData{
		ACHTTPSessionInfo: *sessionInfo,
		Players:           *players,

		Description: "this is a test",

		AmbientTemperature: live.SessionInfo.AmbientTemp,
		RoadTemperature:    live.SessionInfo.RoadTemp,
		WindDirection:      race.WindBaseDirection,
		WindSpeed:          race.WindBaseSpeedMin,
		CurrentWeatherID:   live.SessionInfo.WeatherGraphics,
		Grip:               race.DynamicTrack.SessionStart,
		GripTransfer:       race.DynamicTrack.SessionTransfer,

		// rules
		Assists: CMAssists{
			AbsState:            race.ABSAllowed,
			AllowedTyresOut:     race.AllowedTyresOut,
			AutoclutchAllowed:   race.AutoClutchAllowed == 1,
			DamageMultiplier:    race.DamageMultiplier,
			ForceVirtualMirror:  race.ForceVirtualMirror == 1,
			FuelRate:            race.FuelRate,
			StabilityAllowed:    race.StabilityControlAllowed == 1,
			TcState:             race.TractionControlAllowed,
			TyreBlanketsAllowed: race.TyreBlanketsAllowed == 1,
			TyreWearRate:        race.TyreWearRate,
		},

		MaxContactsPerKm: race.MaxContactsPerKilometer,

		// server info
		City:             geoInfo.City,
		PasswordChecksum: passwordChecksum,
		WrappedPort:      global.ContentManagerWrapperPort,

		Content:   CMContent{}, // not supported
		Frequency: global.ClientSendIntervalInHertz,
		Until:     time.Now().Add(time.Second * time.Duration(sessionInfo.Timeleft)).Unix(),
	}, nil
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

const geoIPURL = "https://freegeoip.app/json/"

type GeoIP struct {
	City        string  `json:"city"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	IP          string  `json:"ip"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	MetroCode   int64   `json:"metro_code"`
	RegionCode  string  `json:"region_code"`
	RegionName  string  `json:"region_name"`
	TimeZone    string  `json:"time_zone"`
	ZipCode     string  `json:"zip_code"`
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
