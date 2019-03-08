package servermanager

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cj123/ini"
)

func init() {
	// assetto seems to take very unkindly to 'pretty formatted' ini files. disable them.
	ini.PrettyFormat = false
}

type SessionType string

const (
	SessionTypeBooking    SessionType = "BOOK"
	SessionTypePractice   SessionType = "PRACTICE"
	SessionTypeQualifying SessionType = "QUALIFY"
	SessionTypeRace       SessionType = "RACE"

	// SessionTypeRacex2 is a convenience const to allow for checking of
	// reversed grid positions signifying a second race.
	SessionTypeRacex2     SessionType = "RACEx2"

	serverConfigIniPath = "server_cfg.ini"
)

func (s SessionType) String() string {
	switch s {
	case SessionTypeBooking:
		return "Booking"
	case SessionTypePractice:
		return "Practice"
	case SessionTypeQualifying:
		return "Qualifying"
	case SessionTypeRace:
		return "Race"
	default:
		return strings.Title(strings.ToLower(string(s)))
	}
}

var AvailableSessions = []SessionType{
	SessionTypeRace,
	SessionTypeQualifying,
	SessionTypePractice,
	SessionTypeBooking,
}

type ServerConfig struct {
	GlobalServerConfig GlobalServerConfig `ini:"SERVER"`
	CurrentRaceConfig  CurrentRaceConfig  `ini:"SERVER"`
}

func (sc ServerConfig) Write() error {
	f := ini.NewFile([]ini.DataSource{nil}, ini.LoadOptions{
		IgnoreInlineComment: true,
	})

	// making and throwing away a default section due to the utter insanity of ini or assetto. i don't know which.
	_, err := f.NewSection("DEFAULT")

	if err != nil {
		return err
	}

	server, err := f.NewSection("SERVER")

	if err != nil {
		return err
	}

	err = server.ReflectFrom(&sc)

	if err != nil {
		return err
	}

	for k, v := range sc.CurrentRaceConfig.Sessions {
		sess, err := f.NewSection(string(k))

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

	err = dynamicTrack.ReflectFrom(&sc.CurrentRaceConfig.DynamicTrack)

	if err != nil {
		return err
	}

	for k, v := range sc.CurrentRaceConfig.Weather {
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

type GlobalServerConfig struct {
	Name                      string `ini:"NAME" help:"Server Name"`
	Password                  string `ini:"PASSWORD" input:"password" help:"Server password"`
	AdminPassword             string `ini:"ADMIN_PASSWORD" input:"password" help:"The password needed to be recognized as server administrator: you can join the server using it to be recognized automatically. Type /help in the game's chat to see the command list"`
	UDPPort                   int    `ini:"UDP_PORT" min:"0" max:"65535" help:"UDP port number: open this port on your server's firewall"`
	TCPPort                   int    `ini:"TCP_PORT" min:"0" max:"65535" help:"TCP port number: open this port on your server's firewall"`
	HTTPPort                  int    `ini:"HTTP_PORT" min:"0" max:"65535" help:"Lobby port number: open these ports (both UDP and TCP) on your server's firewall"`
	UDPPluginLocalPort        int    `ini:"UDP_PLUGIN_LOCAL_PORT" min:"0" max:"65535" help:"The port on which to listen for UDP messages from a plugin"`
	UDPPluginAddress          string `ini:"UDP_PLUGIN_ADDRESS" help:"The address of the plugin to which UDP messages are sent"`
	AuthPluginAddress         string `ini:"AUTH_PLUGIN_ADDRESS" help:"The address of the auth plugin"`
	RegisterToLobby           int    `ini:"REGISTER_TO_LOBBY" input:"checkbox" help:"Register the AC Server to the main lobby"`
	ClientSendIntervalInHertz int    `ini:"CLIENT_SEND_INTERVAL_HZ" help:"Refresh rate of packet sending by the server. 10Hz = ~100ms. Higher number = higher MP quality = higher bandwidth resources needed. Really high values can create connection issues"`
	SendBufferSize            int    `ini:"SEND_BUFFER_SIZE" help:""`
	ReceiveBufferSize         int    `ini:"RECV_BUFFER_SIZE" help:""`
	KickQuorum                int    `ini:"KICK_QUORUM" help:"Percentage that is required for the kick vote to pass"`
	VotingQuorum              int    `ini:"VOTING_QUORUM" min:"0" max:"100" help:"Percentage that is required for the session vote to pass"`
	VoteDuration              int    `ini:"VOTE_DURATION" min:"0" help:"Vote length in seconds"`
	BlacklistMode             int    `ini:"BLACKLIST_MODE" min:"0" max:"2" help:"0 = normal kick, kicked player can rejoin; 1 = kicked player cannot rejoin until server restart; 2 = kick player and add to blacklist.txt, kicked player can not rejoin unless removed from blacklist (Better to use ban_id command rather than set this)."`
	NumberOfThreads           int    `ini:"NUM_THREADS" min:"1" help:"Number of threads to run on"`
	WelcomeMessage            string `ini:"WELCOME_MESSAGE" help:"Path to the file that contains the server welcome message"`
	ResultScreenTime          int    `ini:"RESULT_SCREEN_TIME" help:"Seconds of result screen between racing sessions"`

	FreeUDPPluginLocalPort int    `ini:"-" show:"-"`
	FreeUDPPluginAddress   string `ini:"-" show:"-"`
}

type CurrentRaceConfig struct {
	Cars                      string `ini:"CARS" show:"quick" input:"multiSelect" formopts:"CarOpts" help:"Models of cars allowed in the server"`
	Track                     string `ini:"TRACK" show:"quick" input:"dropdown" formopts:"TrackOpts" help:"Track name"`
	TrackLayout               string `ini:"CONFIG_TRACK" show:"quick" input:"dropdown" formopts:"TrackLayoutOpts" help:"Track layout. Some raceSetup don't have this."`
	SunAngle                  int    `ini:"SUN_ANGLE" help:"Angle of the position of the sun"`
	LegalTyres                string `ini:"LEGAL_TYRES" help:"List of tyres short names that are allowed"`
	FuelRate                  int    `ini:"FUEL_RATE" min:"0" help:"Fuel usage from 0 (no fuel usage) to XXX (100 is the realistic one)"`
	DamageMultiplier          int    `ini:"DAMAGE_MULTIPLIER" min:"0" max:"100" help:"Damage from 0 (no damage) to 100 (full damage)"`
	TyreWearRate              int    `ini:"TYRE_WEAR_RATE" min:"0" help:"Tyre wear from 0 (no tyre wear) to XXX (100 is the realistic one)"`
	AllowedTyresOut           int    `ini:"ALLOWED_TYRES_OUT" help:"TODO: I have no idea"`
	ABSAllowed                int    `ini:"ABS_ALLOWED" min:"0" max:"2" help:"0 -> no car can use ABS, 1 -> only car provided with ABS can use it; 2-> any car can use ABS"`
	TractionControlAllowed    int    `ini:"TC_ALLOWED" min:"0" max:"2" help:"0 -> no car can use TC, 1 -> only car provided with TC can use it; 2-> any car can use TC"`
	StabilityControlAllowed   int    `ini:"STABILITY_ALLOWED" input:"checkbox" help:"Stability assist 0 -> OFF; 1 -> ON"`
	AutoClutchAllowed         int    `ini:"AUTOCLUTCH_ALLOWED" input:"checkbox" help:"Autoclutch assist 0 -> OFF; 1 -> ON"`
	TyreBlanketsAllowed       int    `ini:"TYRE_BLANKETS_ALLOWED" input:"checkbox" help:"at the start of the session or after the pitstop the tyre will have the the optimal temperature"`
	ForceVirtualMirror        int    `ini:"FORCE_VIRTUAL_MIRROR" input:"checkbox" help:"1 virtual mirror will be enabled for every client, 0 for mirror as optional"`
	LockedEntryList           int    `ini:"LOCKED_ENTRY_LIST" input:"checkbox" help:"Only players already included in the entry list can join the server"`
	RacePitWindowStart        int    `ini:"RACE_PIT_WINDOW_START" help:"pit window opens at lap/minute specified"`
	RacePitWindowEnd          int    `ini:"RACE_PIT_WINDOW_END" help:"pit window closes at lap/minute specified"`
	ReversedGridRacePositions int    `ini:"REVERSED_GRID_RACE_POSITIONS" help:" 0 = no additional race, 1toX = only those position will be reversed for the next race, -1 = all the position will be reversed (Retired players will be on the last positions)"`
	TimeOfDayMultiplier       int    `ini:"TIME_OF_DAY_MULT" help:"multiplier for the time of day"`
	QualifyMaxWaitPercentage  int    `ini:"QUALIFY_MAX_WAIT_PERC" help:"The factor to calculate the remaining time in a qualify session after the session is ended: 120 means that 120% of the session fastest lap remains to end the current lap."`
	RaceGasPenaltyDisabled    int    `ini:"RACE_GAS_PENALTY_DISABLED" input:"checkbox" help:"0 = any cut will be penalized with the gas cut message; 1 = no penalization will be forced, but cuts will be saved in the race result json."`
	MaxBallastKilograms       int    `ini:"MAX_BALLAST_KG" help:"the max total of ballast that can be added through the admin command"`
	RaceExtraLap              int    `ini:"RACE_EXTRA_LAP" input:"checkbox" help:"If the race is timed, force an extra lap after the leader has crossed the line"`

	PickupModeEnabled int `ini:"PICKUP_MODE_ENABLED" help:"if 0 the server start in booking mode (do not use it). Warning: in pickup mode you have to list only a circuit under TRACK and you need to list a least one car in the entry_list"`
	LoopMode          int `ini:"LOOP_MODE" input:"checkbox" help:"the server restarts from the first track, to disable this set it to 0"`

	MaxClients   int `ini:"MAX_CLIENTS" help:"max number of clients (must be <= track's number of pits)"`
	SleepTime    int `ini:"SLEEP_TIME" help:"TODO"`
	RaceOverTime int `ini:"RACE_OVER_TIME" help:"time remaining in seconds to finish the race from the moment the first one passes on the finish line"`
	StartRule    int `ini:"START_RULE" min:"0" max:"2" help:"0 is car locked until start;   1 is teleport   ; 2 is drive-through (if race has 3 or less laps then the Teleport penalty is enabled)"`

	IsSol int `ini:"-" help:"Allows for 24 hour time cycles. The server treats time differently if enabled. Clients also require Sol and Content Manager"`

	WindBaseSpeedMin       int `ini:"WIND_BASE_SPEED_MIN" help:"Min speed of the session possible"`
	WindBaseSpeedMax       int `ini:"WIND_BASE_SPEED_MAX" help:"Max speed of session possible (max 40)"`
	WindBaseDirection      int `ini:"WIND_BASE_DIRECTION" help:"base direction of the wind (wind is pointing at); 0 = North, 90 = East etc"`
	WindVariationDirection int `ini:"WIND_VARIATION_DIRECTION" help:"variation (+ or -) of the base direction"`

	DynamicTrack DynamicTrackConfig `ini:"-"`

	Sessions map[SessionType]SessionConfig `ini:"-"`
	Weather  map[string]*WeatherConfig     `ini:"-"`
}

func (c CurrentRaceConfig) HasSession(sess SessionType) bool {
	_, ok := c.Sessions[sess]

	return ok
}

func (c *CurrentRaceConfig) AddSession(sessionType SessionType, config SessionConfig) {
	if c.Sessions == nil {
		c.Sessions = make(map[SessionType]SessionConfig)
	}

	c.Sessions[sessionType] = config
}

func (c *CurrentRaceConfig) RemoveSession(sessionType SessionType) {
	delete(c.Sessions, sessionType)
}

func (c *CurrentRaceConfig) AddWeather(weather *WeatherConfig) {
	if c.Weather == nil {
		c.Weather = make(map[string]*WeatherConfig)
	}

	c.Weather[fmt.Sprintf("WEATHER_%d", len(c.Weather))] = weather
}

func (c *CurrentRaceConfig) RemoveWeather(weather *WeatherConfig) {
	for k, v := range c.Weather {
		if v == weather {
			delete(c.Weather, k)
			return
		}
	}
}

type SessionConfig struct {
	Name     string `ini:"NAME" show:"quick"`
	Time     int    `ini:"TIME" show:"quick" help:"session length in minutes"`
	Laps     int    `ini:"LAPS" show:"quick" help:"number of laps in the race"`
	IsOpen   int    `ini:"IS_OPEN" input:"checkbox" help:"0 = no join, 1 = free join, 2 = free join until 20 seconds to the green light"`
	WaitTime int    `ini:"WAIT_TIME" help:"seconds before the start of the session"`
}

type DynamicTrackConfig struct {
	SessionStart    int `ini:"SESSION_START" help:"% level of grip at session start"`
	Randomness      int `ini:"RANDOMNESS" help:"level of randomness added to the start grip"`
	SessionTransfer int `ini:"SESSION_TRANSFER"  help:"how much of the gained grip is to be added to the next session 100 -> all the gained grip. Example: difference between starting (90) and ending (96) grip in the session = 6%, with session_transfer = 50 then the next session is going to start with 93."`
	LapGain         int `ini:"LAP_GAIN" help:"how many laps are needed to add 1% grip"`
}

type WeatherConfig struct {
	Graphics               string `ini:"GRAPHICS" help:"exactly one of the folder names that you find in the 'content\\weather'' directory"`
	BaseTemperatureAmbient int    `ini:"BASE_TEMPERATURE_AMBIENT" help:"ambient temperature"`                                                                                                                                                               // 0-36
	BaseTemperatureRoad    int    `ini:"BASE_TEMPERATURE_ROAD" help:"Relative road temperature: this value will be added to the final ambient temp. In this example the road temperature will be between 22 (16 + 6) and 26 (20 + 6). It can be negative."` // 0-36
	VariationAmbient       int    `ini:"VARIATION_AMBIENT" help:"variation of the ambient's temperature. In this example final ambient's temperature can be 16 or 20"`
	VariationRoad          int    `ini:"VARIATION_ROAD" help:"variation of the road's temperature. Like the ambient one"`

	CMGraphics          string `ini:"__CM_GRAPHICS" help:"Graphics folder name"`
	CMWFXType           int    `ini:"__CM_WFX_TYPE" help:"Weather ini file number, inside weather.ini"`
	CMWFXUseCustomTime  int    `ini:"__CM_WFX_USE_CUSTOM_TIME" help:"If Sol is active then this should be too"`
	CMWFXTime           int    `ini:"__CM_WFX_TIME" help:"Seconds after 12 noon, usually leave at 0 and use unix timestamp instead"`
	CMWFXTimeMulti      int    `ini:"__CM_WFX_TIME_MULT" help:"Time speed multiplier, default to 1x"`
	CMWFXUseCustomDate  int    `ini:"__CM_WFX_USE_CUSTOM_DATE" help:"If Sol is active then this should be too"`
	CMWFXDate           int    `ini:"__CM_WFX_DATE" help:"Unix timestamp (UTC + 10)"`
	CMWFXDateUnModified int    `ini:"__CM_WFX_DATE_UNMODIFIED" help:"Unix timestamp (UTC + 10), without multiplier correction"`
}

func (w WeatherConfig) UnixToTime(unix int) time.Time {
	return time.Unix(int64(unix), 0)
}

func (w WeatherConfig) TrimName(name string) string {
	// Should not clash with normal weathers, but required for Sol weather setup
	return strings.TrimSuffix(strings.Split(name, "=")[0], "_type")
}
