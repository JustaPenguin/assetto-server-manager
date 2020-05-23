package servermanager

import (
	"github.com/sirupsen/logrus"
	"net/http"
)

// stracker handles configuration of the stracker plugin
// https://www.racedepartment.com/downloads/stracker.3510/

func DefaultStrackerIni(serverOptions *GlobalServerConfig) *StrackerConfiguration {
	return &StrackerConfiguration{
		InstanceConfiguration: StrackerInstanceConfiguration{
			ACServerAddress:              "127.0.0.1",
			ACServerConfigIni:            "",
			ACServerWorkingDir:           "",
			AppendLogFile:                false,
			IDBasedOnDriverNames:         false,
			KeepAlivePtrackerConnections: true,
			ListeningPort:                50042,
			LogFile:                      "./stracker.log",
			LogLevel:                     "info",
			LogTimestamps:                true,
			LowerPriority:                true,
			PerformChecksumComparisons:   false,
			PtrackerConnectionMode:       "any",
			ServerName:                   serverOptions.Name,
			TeeToStdout:                  true,
		},
		SwearFilter: StrackerSwearFilter{
			Action:           "none",
			BanDuration:      30,
			NumberOfWarnings: 3,
			SwearFile:        "bad_words.txt",
			Warning:          "Please be polite and do not swear in the chat. You will be %(swear_action)s from the server after receiving %(num_warnings_left)d more warnings.",
		},
		SessionManagement: StrackerSessionManagement{
			RaceOverStrategy:      "none",
			WaitSecondsBeforeSkip: 15,
		},
		Messages: StrackerMessages{
			BestLapTimeBroadcastThreshold: 105,
			CarToCarCollisionMessage:      true,
			MessageTypesToSendOverChat:    "best_lap+welcome+race_finished",
		},
		Database: StrackerDatabase{
			DatabaseFile:         "./stracker.db3",
			DatabaseType:         "sqlite3",
			PerformBackups:       true,
			PostgresDatabaseName: "stracker",
			PostgresHostname:     "localhost",
			PostgresUsername:     "myuser",
			PostgresPassword:     "password",
		},
		DatabaseCompression: StrackerDatabaseCompression{
			Interval:         60,
			Mode:             "none",
			NeedsEmptyServer: 1,
		},
		HTTPConfiguration: StrackerHTTPConfiguration{
			Enabled:                  true,
			ListenAddress:            "0.0.0.0",
			ListenPort:               50041,
			AdminUsername:            "admin",
			AdminPassword:            "",
			TemperatureUnit:          "degc",
			VelocityUnit:             "kmh", // @TODO server options
			AuthBanAnonymisedPlayers: false,
			AuthLogFile:              "",
			Banner:                   "",
			EnableSVGGeneration:      true,
			InverseNavbar:            true,
			ItemsPerPage:             20,
			LapTimesAddColumns:       "valid+aids+laps+date",
			LogRequests:              false,
			MaximumStreamingClients:  10,
			SSL:                      false,
			SSLCertificate:           "",
			SSLPrivateKey:            "",
		},
		WelcomeMessage: StrackerWelcomeMessage{
			Line1: "Welcome to stracker %(version)s",
			Line2: "",
			Line3: "",
			Line4: "Your activities on this server are tracked. By driving on this server you give consent to store and process",
			Line5: "information like your driver name, steam GUID, chat messages and session statistics. You can anonymize this",
			Line6: "data by typing the chat message \"/st anonymize on\". You might not be able to join the server again afterwards.",
		},
		ACPlugin: StrackerACPlugin{
			ReceivePort:          0, // @TODO set this up
			SendPort:             0, // @TODO set this up
			ProxyPluginLocalPort: 0,
			ProxyPluginPort:      0,
		},
		LapValidChecks: StrackerLapValidChecks{
			InvalidateOnCarCollisions:         true,
			InvalidateOnEnvironmentCollisions: true,
			PtrackerAllowedTyresOut:           -1,
		},
	}
}

type StrackerConfiguration struct {
	InstanceConfiguration StrackerInstanceConfiguration `ini:"STRACKER_CONFIG" input:"heading"`
	SwearFilter           StrackerSwearFilter           `ini:"SWEAR_FILTER"`
	SessionManagement     StrackerSessionManagement     `ini:"SESSION_MANAGEMENT"`
	Messages              StrackerMessages              `ini:"MESSAGES"`
	Database              StrackerDatabase              `ini:"DATABASE"`
	DatabaseCompression   StrackerDatabaseCompression   `ini:"DB_COMPRESSION"`
	HTTPConfiguration     StrackerHTTPConfiguration     `ini:"HTTP_CONFIG"`
	WelcomeMessage        StrackerWelcomeMessage        `ini:"WELCOME_MSG"`
	ACPlugin              StrackerACPlugin              `ini:"ACPLUGIN"`
	LapValidChecks        StrackerLapValidChecks        `ini:"LAP_VALID_CHECKS"`
}

type StrackerInstanceConfiguration struct {
	InstanceConfiguration FormHeading `ini:"-" input:"heading"`

	ACServerAddress              string `ini:"ac_server_address" help:"Server ip address or name used to poll results from. You should not touch the default value: 127.0.0.1"`
	ACServerConfigIni            string `ini:"ac_server_cfg_ini" help:"Path to configuration file of ac server. Note: whenever the server is restarted, it is required to restart stracker as well"`
	ACServerWorkingDir           string `ini:"ac_server_working_dir" help:"Working directory of the ac server, needed to read the race result json files. If empty, the directory is deduced from the ac_server_cfg_ini path assuming the default directory structure"`
	AppendLogFile                bool   `ini:"append_log_file" help:"Set to ON, if you want to append to log files rather than overwriting them. Only meaningful with an external log file rotation system."`
	IDBasedOnDriverNames         bool   `ini:"guids_based_on_driver_names" help:"You normally want to leave this at the default (OFF). Use case for this is an environment where the same steam account is used by different drivers."`
	KeepAlivePtrackerConnections bool   `ini:"keep_alive_ptracker_conns" help:"Set to OFF if you want to disable the TCP keep_alive option (that was the behaviour pre 3.1.7)."`
	ListeningPort                int    `ini:"listening_port" help:"Listening port for incoming connections of ptracker. Must be one of 50042, 50043, 54242, 54243, 60023, 60024, 62323, 62324, 42423, 42424, 23232, 23233, <AC udp port>+42; ptracker will try all these ports on the ac server's ip address (until a better solution is found...)"`
	LogFile                      string `ini:"log_file" help:"Name of the stracker log file (utf-8 encoded), all messages go into there"`
	LogLevel                     string `ini:"log_level" help:"Valid values are 'info', 'debug' and 'dump'. Use 'dump' only for problem analysis, log files can get very big."`
	LogTimestamps                bool   `ini:"log_timestamps" help:"Set to ON if you want the log messages to be prefixed with a timestamp"`
	LowerPriority                bool   `ini:"lower_priority" help:"Set to ON if you want stracker to reduce its priority. Will use BELOW_NORMAL on windows and nice(5) on linux."`
	PerformChecksumComparisons   bool   `ini:"perform_checksum_comparisons" help:"Set to ON if you want stracker to compare the players checksums."`
	PtrackerConnectionMode       string `ini:"ptracker_connection_mode" help:"Configure which ptracker instances shall be allowed to connect: Valid values are 'any', 'newer' or 'none'."`
	ServerName                   string `ini:"server_name" help:"Name for the server; sessions in the database will be tagged with that name; useful when more than one server is running in parallel on the same database"`
	TeeToStdout                  bool   `ini:"tee_to_stdout" help:"Set to ON if you want the messages appear on stdout (in Server Manager's plugin logs)"`
}

type StrackerSwearFilter struct {
	SwearFilter FormHeading `ini:"-" input:"heading"`

	Action           string `ini:"action" help:"Valid values are 'none', 'kick' and 'ban'"`
	BanDuration      int    `ini:"ban_duration" help:"The number of days to ban a player for (if the Action is 'ban')"`
	NumberOfWarnings int    `ini:"num_warnings" help:"The number of warnings issued before the player is kicked"`
	SwearFile        string `ini:"swear_file" help:"A file with bad words to be used for filtering"`
	Warning          string `ini:"warning" help:"The message sent to a player after swear detection"`
}

type StrackerSessionManagement struct {
	SessionManagement FormHeading `ini:"-" input:"heading"`

	RaceOverStrategy      string `ini:"race_over_strategy" help:"What to do when the race is over and no player is actively racing. Valid values are: 'none' or 'skip'."`
	WaitSecondsBeforeSkip int    `ini:"wait_secs_before_skip" help:"Number of seconds to wait before the session skip is executed (if Race Over Strategy is set to 'skip')"`
}

type StrackerMessages struct {
	Messages FormHeading `ini:"-" input:"heading"`

	BestLapTimeBroadcastThreshold int  `ini:"best_lap_time_broadcast_threshold" help:"Lap times below this threshold (in percent of the best time) will be broadcasted as best laps. Lap times above this will be whispered to the player achieving it."`
	CarToCarCollisionMessage      bool `ini:"car_to_car_collision_msg" help:"Set to ON to enable car to car private messages."`
	// @TODO can a multiselect be used for this one?
	MessageTypesToSendOverChat string `ini:"message_types_to_send_over_chat" help:"Available message types are 'enter_leave','best_lap','checksum_errors','welcome','race_finished' and 'collision'. Connect them using a + sign without spaces."`
}

type StrackerDatabase struct {
	Database FormHeading `ini:"-" input:"heading"`

	DatabaseFile         string `ini:"database_file" help:"Only relevant if database_type=sqlite3. Path to the stracker database. If a relative path is given, it is relative to the <stracker> executable"`
	DatabaseType         string `ini:"database_type" help:"Valid values are 'sqlite3' and 'postgres'. Selects the database to be used."`
	PerformBackups       bool   `ini:"perform_backups" help:"Set to OFF if you do not want stracker to backup the database before migrating to a new db version. Note: The backups will be created as sqlite3 db in the current working directory."`
	PostgresDatabaseName string `ini:"postgres_db" help:"The name of the postgres database"`
	PostgresHostname     string `ini:"postgres_host" help:"Name of the host running the postgresql server."`
	PostgresUsername     string `ini:"postgres_user" help:"Name of the postgresql user"`
	PostgresPassword     string `ini:"postgres_pwd" help:"Postgresql user password"`
}

type StrackerDatabaseCompression struct {
	DatabaseCompression FormHeading `ini:"-" input:"heading"`

	Interval         int    `ini:"interval" help:"Interval of database compression in minutes"`
	Mode             string `ini:"mode" help:"Various options to minimize database size. Valid values are 'none' (no compression, save all available infos), 'remove_slow_laps' (save detailed infos for fast laps only) and 'remove_all' (save no detailed lap info)."`
	NeedsEmptyServer int    `ini:"needs_empty_server" input:"checkbox" help:"If set to ON database compression will only take place if the server is empty."`
}

// @TODO webui is enabled, accessible thru reverse proxy in server manager. Hide as many config options as possible
// @TODO make mph/kmh equal to the same value as set in server manager

type StrackerHTTPConfiguration struct {
	HTTPConfiguration FormHeading `ini:"-" input:"heading"`

	Enabled       bool   `ini:"enabled"`
	ListenAddress string `ini:"listen_addr" help:"Listening address of the http server (normally there is no need to change the default value 0.0.0.0 which means that the whole internet can connect to the server)"`
	ListenPort    int    `ini:"listen_port" help:"TCP listening port of the http server"`
	AdminUsername string `ini:"admin_username" help:"Username for the stracker admin pages. Leaving empty results in disabled admin pages"`
	AdminPassword string `ini:"admin_password" input:"password" help:"Password for the stracker admin pages. Leaving empty results in disabled admin pages"`

	TemperatureUnit string `ini:"temperature_unit" help:"Valid values are 'degc' or 'degf'"`
	VelocityUnit    string `ini:"velocity_unit" help:"Valid values are 'kmh' or 'mph'"`

	AuthBanAnonymisedPlayers bool   `ini:"auth_ban_anonymized_players" help:"Add anonymized players to blacklist."`
	AuthLogFile              string `ini:"auth_log_file" help:"Set to a file to be used for logging http authentication requests. Useful to prevent attacks with external program (e.g., fail2ban)."`
	Banner                   string `ini:"banner" help:"Icon to be used in webpages (leave empty for default Assetto Corsa icon)"`
	EnableSVGGeneration      bool   `ini:"enable_svg_generation" help:"Set to OFF if you do not want svg graphs in the http output (for saving bandwidth)"`
	InverseNavbar            bool   `ini:"inverse_navbar" help:"Set to true to get the navbar inverted (i.e., dark instead of bright)"`
	ItemsPerPage             int    `ini:"items_per_page" help:"Number of items displayed per page"`
	// @TODO can a multiselect be used for this one?
	LapTimesAddColumns      string `ini:"lap_times_add_columns" help:"Additional columns to be displayed in LapTimes table (seperated by a + sign). Columns can be 'valid', 'aids', 'laps', 'date', 'grip', 'cuts', 'collisions', 'tyres', 'temps', 'ballast' and 'vmax'. Note that too many displayed columns might cause problems on some browsers."`
	LogRequests             bool   `ini:"log_requests" help:"If set to ON, http requests will be logged in stracker.log. Otherwise they are not logged."`
	MaximumStreamingClients int    `ini:"max_streaming_clients" help:"Maximum number of streaming clients (LiveMap/Log users) allowed to connect to this server in parallel. The number of threads allocated for http serving will be max(10, max_streaming_clients + 5)"`

	// @TODO these probably won't be useful
	SSL            bool   `ini:"ssl" help:"Set to true if you want to use https. Note that you need a SSL certificate and key. If you enable this option, you can reach stracker at https://ip:port/ instead of http://ip:port/"`
	SSLCertificate string `ini:"ssl_certificate" help:"Path to the SSL certificate for https. Only used when ssl is True. A self-signed certificate can be generated with 'openssl req -new -x509 -days 365 -key privkey.pem -out cert.pem'"`
	SSLPrivateKey  string `ini:"ssl_private_key" help:"ath to the SSL private key for https. Only used when ssl is True. A private key can be generated with 'openssl genrsa -out privkey.pem 2048'"`
}

type StrackerWelcomeMessage struct {
	WelcomeMessage FormHeading `ini:"-" input:"heading"`

	Line1 string `ini:"line1"`
	Line2 string `ini:"line2"`
	Line3 string `ini:"line3"`
	Line4 string `ini:"line4"`
	Line5 string `ini:"line5"`
	Line6 string `ini:"line6"`
}

type StrackerACPlugin struct {
	AssettoCorsaPlugin FormHeading `ini:"-" input:"heading"`

	ReceivePort int `ini:"rcvPort" help:"UDP port the plugins receives from. -1 means to use the AC servers setting UDP_PLUGIN_ADDRESS"`
	SendPort    int `ini:"sendPort" help:"UDP port the plugins sends to. -1 means to use the AC servers setting UDP_PLUGIN_LOCAL_PORT"`

	// probably not needed
	ProxyPluginLocalPort int `ini:"proxyPluginLocalPort" help:"Proxy the AC server protocol on these ports, so multiple plugins may be chained (this is equivalent to UDP_PLUGIN_LOCAL_PORT in server_cfg.ini)"`
	ProxyPluginPort      int `ini:"proxyPluginPort" help:"Proxy the AC server protocol on these ports, so multiple plugins may be chained (this is equivalent to UDP_PLUGIN_ADDRESS in server_cfg.ini)"`
}

type StrackerLapValidChecks struct {
	LapValidChecks FormHeading `ini:"-" input:"heading"`

	InvalidateOnCarCollisions         bool `ini:"invalidateOnCarCollisions" help:"If ON, collisions with other cars will invalidate laps"`
	InvalidateOnEnvironmentCollisions bool `ini:"invalidateOnEnvCollisions" help:"If ON, collisions with environment objects will invalidate laps"`
	PtrackerAllowedTyresOut           int  `ini:"ptrackerAllowedTyresOut" help:"If -1: use server penalty setting, if available, otherwise use 2. All other values are passed to ptracker."`
}

type StrackerHandler struct {
	*BaseHandler

	store Store
}

func NewStrackerHandler(baseHandler *BaseHandler, store Store) *StrackerHandler {
	return &StrackerHandler{BaseHandler: baseHandler, store: store}
}

type strackerConfigurationTemplateVars struct {
	BaseTemplateVars

	Form *Form
}

func (sth *StrackerHandler) options(w http.ResponseWriter, r *http.Request) {
	serverOpts, err := sth.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
		return
	}

	stracker := DefaultStrackerIni(serverOpts)

	form := NewForm(stracker, nil, "", AccountFromRequest(r).Name == "admin")
	/*
		if r.Method == http.MethodPost {
			err := form.Submit(r)

			if err != nil {
				logrus.WithError(err).Errorf("couldn't submit form")
			}

			UseShortenedDriverNames = serverOpts.UseShortenedDriverNames == 1
			UseFallBackSorting = serverOpts.FallBackResultsSorting == 1

			// save the config
			err = sah.store.SaveServerOptions(serverOpts)

			if err != nil {
				logrus.WithError(err).Errorf("couldn't save config")
				AddErrorFlash(w, r, "Failed to save server options")
			} else {
				AddFlash(w, r, "Server options successfully saved!")
			}
		}
	*/
	sth.viewRenderer.MustLoadTemplate(w, r, "server/stracker-options.html", &strackerConfigurationTemplateVars{
		Form: form,
	})
}
