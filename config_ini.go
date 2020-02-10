package servermanager

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/cj123/ini"
	"github.com/sirupsen/logrus"
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

	// SessionTypeSecondRace is a convenience const to allow for checking of
	// reversed grid positions signifying a second race.
	SessionTypeSecondRace SessionType = "RACEx2"

	serverConfigIniPath = "server_cfg.ini"
)

func (s SessionType) OriginalString() string {
	return string(s)
}

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
	case SessionTypeSecondRace:
		return "2nd Race"
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

var AvailableSessionsNoBooking = []SessionType{
	SessionTypeRace,
	SessionTypeQualifying,
	SessionTypePractice,
}

type ServerConfig struct {
	GlobalServerConfig GlobalServerConfig `ini:"SERVER"`
	CurrentRaceConfig  CurrentRaceConfig  `ini:"SERVER"`
}

func (sc ServerConfig) Write() error {
	// overwrite server config
	sc.GlobalServerConfig.WelcomeMessage = MOTDFilename

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
	AssettoCorsaServer FormHeading `ini:"-" json:"-" input:"heading"`

	Name                      string `ini:"NAME" help:"Server Name"`
	Password                  string `ini:"PASSWORD" input:"password" help:"Server password"`
	AdminPassword             string `ini:"ADMIN_PASSWORD" input:"password" help:"The password needed to be recognized as server administrator: you can join the server using it to be recognized automatically. Type /help in the game's chat to see the command list"`
	UDPPort                   int    `ini:"UDP_PORT" show:"open" min:"0" max:"65535" help:"UDP port number: open this port on your server's firewall"`
	TCPPort                   int    `ini:"TCP_PORT" show:"open" min:"0" max:"65535" help:"TCP port number: open this port on your server's firewall"`
	HTTPPort                  int    `ini:"HTTP_PORT" show:"open" min:"0" max:"65535" help:"Lobby port number: open these ports (both UDP and TCP) on your server's firewall"`
	UDPPluginLocalPort        int    `ini:"UDP_PLUGIN_LOCAL_PORT" show:"open" min:"0" max:"65535" help:"The port on which to listen for UDP messages from a plugin. Please note that Server Manager proxies UDP ports so that it can use them as well, for things such as Championships, Live Timings and the Map. This means that the UDP ports you see in the server_cfg.ini will be different to the ones you specify here. This is not an issue, and messages will be correctly sent/received on the UDP ports you specify here as well."`
	UDPPluginAddress          string `ini:"UDP_PLUGIN_ADDRESS" show:"open" help:"The address of the plugin to which UDP messages are sent.  Please note that Server Manager proxies UDP ports so that it can use them as well, for things such as Championships, Live Timings and the Map. This means that the UDP ports you see in the server_cfg.ini will be different to the ones you specify here. This is not an issue, and messages will be correctly sent/received on the UDP ports you specify here as well."`
	AuthPluginAddress         string `ini:"AUTH_PLUGIN_ADDRESS" show:"open" help:"The address of the auth plugin"`
	RegisterToLobby           int    `ini:"REGISTER_TO_LOBBY" show:"open" input:"checkbox" help:"Register the AC Server to the main lobby"`
	ClientSendIntervalInHertz int    `ini:"CLIENT_SEND_INTERVAL_HZ" show:"open" help:"Refresh rate of packet sending by the server. 10Hz = ~100ms. Higher number = higher MP quality = higher bandwidth resources needed. Really high values can create connection issues"`
	SendBufferSize            int    `ini:"SEND_BUFFER_SIZE" show:"open" help:""`
	ReceiveBufferSize         int    `ini:"RECV_BUFFER_SIZE" show:"open" help:""`
	KickQuorum                int    `ini:"KICK_QUORUM" help:"Percentage that is required for the kick vote to pass"`
	VotingQuorum              int    `ini:"VOTING_QUORUM" min:"0" max:"100" help:"Percentage that is required for the session vote to pass"`
	VoteDuration              int    `ini:"VOTE_DURATION" min:"0" help:"Vote length in seconds"`
	BlacklistMode             int    `ini:"BLACKLIST_MODE" min:"0" max:"2" help:"0 = normal kick, kicked player can rejoin; 1 = kicked player cannot rejoin until server restart; 2 = kick player and add to blacklist.txt, kicked player can not rejoin unless removed from blacklist (Better to use ban_id command rather than set this)."`
	NumberOfThreads           int    `ini:"NUM_THREADS" show:"open" min:"1" help:"Number of threads to run on"`
	WelcomeMessage            string `ini:"WELCOME_MESSAGE" show:"-" help:"Path to the file that contains the server welcome message"`

	SleepTime int `ini:"SLEEP_TIME" help:"The use of this setting is not fully known. Leave the value as 1 unless you really know what you're doing. (Values other than 1 cause excessive CPU usage)"`

	FreeUDPPluginLocalPort int    `ini:"-" show:"-"`
	FreeUDPPluginAddress   string `ini:"-" show:"-"`

	// ACSR
	AssettoCorsaSkillRating FormHeading `ini:"-" json:"-" show:"premium" input:"heading"`
	EnableACSR              bool        `ini:"-" show:"premium" input:"checkbox" help:"Enable ACSR integration. <a href='https://acsr.assettocorsaservers.com'>You can read more about ACSR here</a>."`
	ACSRAccountID           string      `ini:"-" show:"premium" help:"Your ACSR account ID. You can <a href='https://acsr.assettocorsaservers.com/account'>request an ACSR key here</a>."`
	ACSRAPIKey              string      `ini:"-" show:"premium" help:"Your ACSR API Key. You can <a href='https://acsr.assettocorsaservers.com/account'>request an ACSR key here</a>."`

	ServerName                FormHeading `ini:"-" json:"-" input:"heading"`
	ShowRaceNameInServerLobby int         `ini:"-" input:"checkbox" help:"When on, this option will make Server Manager append the Custom Race or Championship name to the Server name in the lobby."`
	ServerNameTemplate        string      `ini:"-" help:"You can enter anything you like in here. If you put <code>{{ .ServerName }}</code> in, the Server Name will replace it. If you put <code>{{ .EventName }}</code>, then the Event Name will replace it. Note this only works if 'Show Race Name In Server Lobby' (above) is enabled. You can <a href='https://github.com/JustaPenguin/assetto-server-manager/wiki/Server-Name-Template-Examples'>view some examples</a> on the Server Manager Wiki!"`

	Theme     FormHeading `ini:"-" json:"-" input:"heading"`
	DarkTheme int         `ini:"-" input:"checkbox" help:"Enable Server Manager's Dark Theme by default"`
	CustomCSS string      `ini:"-" input:"textarea" help:"Customise the style of Server Manager! You can <a href='https://github.com/JustaPenguin/assetto-server-manager/wiki/Custom-CSS-Examples'>view some examples</a> on the Server Manager Wiki!"`
	OGImage   string      `ini:"-" show:"premium" help:"Link to an image on the web here to set it as your default Open Graph image (will show in links)"`

	ContentManagerIntegration   FormHeading `ini:"-" json:"-" input:"heading"`
	EnableContentManagerWrapper int         `ini:"-" input:"checkbox" help:"When on, this option makes Server Manager provide extra information to Content Manager. This includes more detail about connected clients, event descriptions and download links. A side-effect of this is that your server name will contain a new piece of information (an 'i' character followed by a port - which Content Manager requires). Also - if enabled - this wrapper uses a GeoIP functionality provided by <a href='https://freegeoip.app''>freegeoip.app</a>."`
	ContentManagerWrapperPort   int         `ini:"-" min:"0" max:"65535" help:"The port on which to serve Content Manager with the above information. Please make sure this port is open on your firewall."`
	ShowContentManagerJoinLink  int         `ini:"-" input:"checkbox" help:"When on, this option will make Server Manager display Content Manager join links on the Live Timing page and (if enabled) in Discord race start notifications."`
	ContentManagerIPOverride    string      `ini:"-" show:"open" help:"When set, this overrides the IP address detected by the GeoIP service used for the Content Manager join link. This must be an IPv4 address."`
	//ContentManagerWrapperContentRequiresPassword int         `ini:"-" input:"checkbox" help:"When on a user will require the server password in order to download linked content through the Content Manager Wrapper."`

	Miscellaneous                     FormHeading `ini:"-" json:"-" input:"heading"`
	UseShortenedDriverNames           int         `ini:"-" input:"checkbox" help:"When on, this option will make Server Manager hide driver's last names, for example 'John Smith' becomes 'John S.'"`
	FallBackResultsSorting            int         `ini:"-" input:"checkbox" help:"When on results will use a fallback method of sorting. Only enable this if you are experiencing results that are in the wrong order in the json file."`
	UseMPH                            int         `ini:"-" input:"checkbox" help:"When on, this option will make Server Manager use MPH instead of Km/h for all speed values."`
	PreventWebCrawlers                int         `ini:"-" input:"checkbox" help:"When on, robots will be prohibited from indexing this manager by the robots.txt. Please note this will only deter well behaved bots, and not malware/spam bots etc."`
	RestartEventOnServerManagerLaunch int         `ini:"-" input:"checkbox" help:"When on, if Server Manager is stopped while there is an event in progress, Server Manager will try to restart the event when Server Manager is restarted."`
	LogACServerOutputToFile           bool        `ini:"-" input:"checkbox" help:"When on, Server Manager will output each Assetto Corsa session into a log file in the logs folder."`
	NumberOfACServerLogsToKeep        int         `ini:"-" help:"The number of AC Server logs to keep in the logs folder. (Oldest files will be deleted first. 0 = keep all files)"`

	// Discord Integration
	DiscordIntegration FormHeading `ini:"-" json:"-" input:"heading"`
	DiscordAPIToken    string      `ini:"-" help:"If set, will enable race start and scheduled reminder messages to the Discord channel ID specified below.  Use your bot's user token, not the OAuth token."`
	DiscordChannelID   string      `ini:"-" help:"If Discord is enabled, this is the channel ID it will send messages to.  To find the channel ID, enable Developer mode in Discord (user settings, Appearance), then Server Settings, Roles, and right click on the channel and Copy ID."`
	DiscordRoleID      string      `ini:"-" help:"If set, this role will be mentioned in all Discord notifications.  Any users with this role and access to the channel will be pinged.  To find the role ID, enable Developer mode (see above)), then Server Settings, Roles, right click on the role and Copy ID."`
	DiscordRoleCommand string      `ini:"-" help:"If the Discord Role ID is set, you can optionally specify a command string here, like \"notify\" (no ! prefix), which if run as a ! command by a user (on a line by itself) in Discord will cause this server to attempt to add the configured role to the user.  If you run multiple servers with Discord enabled, only set this on one of them.  In order for this to work your bot must have the \"Manage Roles\" permission."`

	NotificationReminderTimer   int    `ini:"-"  show:"-" min:"0" max:"65535" help:"This setting has been deprecated and will be removed in the next release.  Use Notification Reminder Timers instead."`
	NotificationReminderTimers  string `ini:"-" help:"If Discord is enabled, a reminder will be sent this many minutes prior to race start.  If 0 or empty, only race start messages will be sent.  You may schedule multiple reminders by using a comma separated list like 120,15."`
	ShowPasswordInNotifications int    `ini:"-" input:"checkbox" help:"Show the server password in race start notifications."`
	NotifyWhenScheduled         int    `ini:"-" input:"checkbox" help:"Send a notification when a race is scheduled (or cancelled)."`

	// Messages
	ContentManagerWelcomeMessage string `ini:"-" show:"-"`
	ServerJoinMessage            string `ini:"-" show:"-"`
}

func (gsc GlobalServerConfig) GetName() string {
	split := strings.Split(gsc.Name, fmt.Sprintf(" %c", contentManagerWrapperSeparator))

	if len(split) > 0 {
		return split[0]
	}

	return gsc.Name
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
	RacePitWindowStart        int    `ini:"RACE_PIT_WINDOW_START" help:"pit window opens at lap/minute specified"`
	RacePitWindowEnd          int    `ini:"RACE_PIT_WINDOW_END" help:"pit window closes at lap/minute specified"`
	ReversedGridRacePositions int    `ini:"REVERSED_GRID_RACE_POSITIONS" help:" 0 = no additional race, 1toX = only those position will be reversed for the next race, -1 = all the position will be reversed (Retired players will be on the last positions)"`
	TimeOfDayMultiplier       int    `ini:"TIME_OF_DAY_MULT" help:"multiplier for the time of day"`
	QualifyMaxWaitPercentage  int    `ini:"QUALIFY_MAX_WAIT_PERC" help:"The factor to calculate the remaining time in a qualify session after the session is ended: 120 means that 120% of the session fastest lap remains to end the current lap."`
	RaceGasPenaltyDisabled    int    `ini:"RACE_GAS_PENALTY_DISABLED" input:"checkbox" help:"0 = any cut will be penalized with the gas cut message; 1 = no penalization will be forced, but cuts will be saved in the race result json."`
	MaxBallastKilograms       int    `ini:"MAX_BALLAST_KG" help:"the max total of ballast that can be added to an entrant in the entry list or through the admin command"`
	RaceExtraLap              int    `ini:"RACE_EXTRA_LAP" input:"checkbox" help:"If the race is timed, force an extra lap after the leader has crossed the line"`
	MaxContactsPerKilometer   int    `ini:"MAX_CONTACTS_PER_KM" help:"Maximum number times you can make contact with another car in 1 kilometer."`
	ResultScreenTime          int    `ini:"RESULT_SCREEN_TIME" help:"Seconds of result screen between racing sessions"`

	PickupModeEnabled int `ini:"PICKUP_MODE_ENABLED" input:"checkbox" help:"if 0 the server start in booking mode (do not use it). Warning: in pickup mode you have to list only a circuit under TRACK and you need to list a least one car in the entry_list"`
	LockedEntryList   int `ini:"LOCKED_ENTRY_LIST" input:"checkbox" help:"Only players already included in the entry list can join the server"`
	LoopMode          int `ini:"LOOP_MODE" input:"checkbox" help:"the server restarts from the first track, to disable this set it to 0"`

	MaxClients   int `ini:"MAX_CLIENTS" help:"max number of clients (must be <= track's number of pits)"`
	RaceOverTime int `ini:"RACE_OVER_TIME" help:"time remaining in seconds to finish the race from the moment the first one passes on the finish line"`
	StartRule    int `ini:"START_RULE" min:"0" max:"2" help:"0 is car locked until start;   1 is teleport   ; 2 is drive-through (if race has 3 or less laps then the Teleport penalty is enabled)"`

	IsSol int `ini:"-" help:"Allows for 24 hour time cycles. The server treats time differently if enabled. Clients also require Sol and Content Manager"`

	WindBaseSpeedMin       int `ini:"WIND_BASE_SPEED_MIN" help:"Min speed of the session possible"`
	WindBaseSpeedMax       int `ini:"WIND_BASE_SPEED_MAX" help:"Max speed of session possible (max 40)"`
	WindBaseDirection      int `ini:"WIND_BASE_DIRECTION" help:"base direction of the wind (wind is pointing at); 0 = North, 90 = East etc"`
	WindVariationDirection int `ini:"WIND_VARIATION_DIRECTION" help:"variation (+ or -) of the base direction"`

	DisableDRSZones bool `ini:"-"`

	DynamicTrack DynamicTrackConfig `ini:"-"`

	Sessions Sessions                  `ini:"-"`
	Weather  map[string]*WeatherConfig `ini:"-"`
}

func (c CurrentRaceConfig) Tyres() map[string]bool {
	tyres := make(map[string]bool)

	for _, tyre := range strings.Split(c.LegalTyres, ";") {
		tyres[tyre] = true
	}

	return tyres
}

type Sessions map[SessionType]*SessionConfig

func (s Sessions) AsSlice() []*SessionConfig {
	var out []*SessionConfig

	if x, ok := s[SessionTypeBooking]; ok {
		out = append(out, x)
	}

	if x, ok := s[SessionTypePractice]; ok {
		out = append(out, x)
	}

	if x, ok := s[SessionTypeQualifying]; ok {
		out = append(out, x)
	}

	if x, ok := s[SessionTypeRace]; ok {
		out = append(out, x)
	}

	return out
}

func (c CurrentRaceConfig) HasMultipleRaces() bool {
	return c.HasSession(SessionTypeRace) && c.ReversedGridRacePositions != 0
}

func (c CurrentRaceConfig) HasSession(sess SessionType) bool {
	_, ok := c.Sessions[sess]

	return ok
}

func (c CurrentRaceConfig) GetSession(sessionType SessionType) *SessionConfig {
	sess, ok := c.Sessions[sessionType]

	if !ok {
		return &SessionConfig{}
	}

	return sess
}

func (c *CurrentRaceConfig) AddSession(sessionType SessionType, config *SessionConfig) {
	if c.Sessions == nil {
		c.Sessions = make(map[SessionType]*SessionConfig)
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

type serverNameTemplateOpts struct {
	GlobalServerConfig
	CurrentRaceConfig
	RaceEvent

	ServerName string
}

func buildFinalServerName(userTemplate string, event RaceEvent, config ServerConfig) string {
	t, err := template.New("serverName").Parse(userTemplate)

	if err != nil {
		logrus.WithError(err).Errorf("could not parse user server name template.")
		return config.GlobalServerConfig.Name
	}

	out := new(bytes.Buffer)

	err = t.Execute(out, serverNameTemplateOpts{
		ServerName:         config.GlobalServerConfig.Name,
		CurrentRaceConfig:  config.CurrentRaceConfig,
		GlobalServerConfig: config.GlobalServerConfig,
		RaceEvent:          event,
	})

	if err != nil {
		logrus.WithError(err).Errorf("could not execute user server name template.")
		return config.GlobalServerConfig.Name
	}

	return out.String()
}
