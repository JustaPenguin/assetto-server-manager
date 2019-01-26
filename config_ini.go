package servermanager

import (
	"fmt"
	"path/filepath"

	"gopkg.in/ini.v1"
)

func init() {
	// assetto seems to take very unkindly to 'pretty formatted' ini files. disable them.
	ini.PrettyFormat = false
}

type SessionType string

const (
	SessionTypeBooking    SessionType = "BOOKING"
	SessionTypePractice   SessionType = "PRACTICE"
	SessionTypeQualifying SessionType = "QUALIFY"
	SessionTypeRace       SessionType = "RACE"

	serverConfigIniPath = "server_cfg.ini"
)

func (s SessionType) String() string {
	return string(s)
}

type ServerConfig struct {
	Server       ServerSetupConfig  `ini:"SERVER"`
	DynamicTrack DynamicTrackConfig `ini:"DYNAMIC_TRACK"`

	Sessions map[string]SessionConfig
	Weather  map[string]WeatherConfig
}

func (sc ServerConfig) Write() error {
	f := ini.Empty()

	server, err := f.NewSection("SERVER")

	if err != nil {
		return err
	}

	err = server.ReflectFrom(&sc.Server)

	if err != nil {
		return err
	}

	for k, v := range sc.Sessions {
		sess, err := f.NewSection(k)

		if err != nil {
			return err
		}

		err = sess.ReflectFrom(&v)

		if err != nil {
			return err
		}
	}

	dynamicTrack, err := f.NewSection("DYNAMIC_TRACK")

	if err != nil {
		return err
	}

	err = dynamicTrack.ReflectFrom(&sc.Server)

	if err != nil {
		return err
	}

	for k, v := range sc.Weather {
		weather, err := f.NewSection(k)

		if err != nil {
			return err
		}

		err = weather.ReflectFrom(&v)

		if err != nil {
			return err
		}
	}

	return f.SaveTo(filepath.Join(ServerInstallPath, ServerConfigPath, serverConfigIniPath))
}

func (sc ServerConfig) AddSession(sessionType SessionType, config SessionConfig) {
	sc.Sessions[sessionType.String()] = config
}

func (sc ServerConfig) RemoveSession(sessionType SessionType) {
	delete(sc.Sessions, sessionType.String())
}

func (sc ServerConfig) AddWeather(weather WeatherConfig) {
	sc.Weather[fmt.Sprintf("WEATHER_%d", len(sc.Weather))] = weather
}

func (sc ServerConfig) RemoveWeather(weather WeatherConfig) {
	for k, v := range sc.Weather {
		if v == weather {
			delete(sc.Weather, k)
			return
		}
	}
}

type ServerSetupConfig struct {
	Name                    string `ini:"NAME"`
	Cars                    string `ini:"CARS"`
	TrackConfig             string `ini:"CONFIG_TRACK"`
	Track                   string `ini:"TRACK"`
	SunAngle                int    `ini:"SUN_ANGLE"`
	LegalTyres              string `ini:"LEGAL_TYRES"`
	FuelRate                int    `ini:"FUEL_RATE"`
	DamageMultiplier        int    `ini:"DAMAGE_MULTIPLIER"`
	TyreWearRate            int    `ini:"TYRE_WEAR_RATE"`
	AllowedTyresOut         int    `ini:"ALLOWED_TYRES_OUT"`
	ABSAllowed              int    `ini:"ABS_ALLOWED"`
	TractionControlAllowed  int    `ini:"TC_ALLOWED"`
	StabilityControlAllowed int    `ini:"STABILITY_ALLOWED"`
	AutoClutchAllowed       int    `ini:"AUTOCLUTCH_ALLOWED"`
	TyreBlanketsAllowed     int    `ini:"TYRE_BLANKETS_ALLOWED"`
	ForceVirtualMirror      int    `ini:"FORCE_VIRTUAL_MIRROR"`

	Password                  string `ini:"PASSWORD"`
	AdminPassword             string `ini:"ADMIN_PASSWORD"`
	UDPPort                   int    `ini:"UDP_PORT"`
	TCPPort                   int    `ini:"TCP_PORT"`
	HTTPPort                  int    `ini:"HTTP_PORT"`
	UDPPluginLocalPort        int    `ini:"UDP_PLUGIN_LOCAL_PORT"`
	UDPPluginAddress          string `ini:"UDP_PLUGIN_ADDRESS"`
	AuthPluginAddress         string `ini:"AUTH_PLUGIN_ADDRESS"`
	RegisterToLobby           int    `ini:"REGISTER_TO_LOBBY"`
	ClientSendIntervalInHertz int    `ini:"CLIENT_SEND_INTERVAL_HZ"`
	SendBufferSize            int    `ini:"SEND_BUFFER_SIZE"`
	ReceiveBufferSize         int    `ini:"RECV_BUFFER_SIZE"`
	MaxClients                int    `ini:"MAX_CLIENTS"`

	PickupModeEnabled int `ini:"PICKUP_MODE_ENABLED"`
	LoopMode          int `ini:"LOOP_MODE"`

	SleepTime     int `ini:"SLEEP_TIME"`
	RaceOverTime  int `ini:"RACE_OVER_TIME"`
	KickQuorum    int `ini:"KICK_QUORUM"`
	VotingQuorum  int `ini:"VOTING_QUORUM"`
	VoteDuration  int `ini:"VOTE_DURATION"`
	BlacklistMode int `ini:"BLACKLIST_MODE"`
}

type SessionConfig struct {
	Name     string `ini:"NAME"`
	Time     int    `ini:"TIME"`
	Laps     int    `ini:"LAPS"`
	IsOpen   int    `ini:"IS_OPEN"`
	WaitTime int    `ini:"WAIT_TIME"`
}

type DynamicTrackConfig struct {
	SessionStart    int `ini:"SESSION_START"`
	Randomness      int `ini:"RANDOMNESS"`
	SessionTransfer int `ini:"SESSION_TRANSFER"`
	LapGain         int `ini:"LAP_GAIN"`
}

type WeatherConfig struct {
	Graphics               string `ini:"GRAPHICS"`
	BaseTemperatureAmbient int    `ini:"BASE_TEMPERATURE_AMBIENT"` // 0-36
	BaseTemperatureRoad    int    `ini:"BASE_TEMPERATURE_ROAD"`    // 0-36
	VariationAmbient       int    `ini:"VARIATION_AMBIENT"`
	VariationRoad          int    `ini:"VARIATION_ROAD"`
}
