package servermanager

type KissMyRankConfig struct {
	MaxPlayers int `json:"max_players" help:"Number of server slots."`

	// ACServer
	ACServerIP                string `json:"ac_server_ip" help:"The IP of your Assetto Corsa Server relative to the plugin. This is the IP that the plugin will use to contact the Assetto Corsa Server."`
	ACServerHTTPPort          int    `json:"ac_server_http_port" help:"The HTTP Port of the Assetto Corsa server. This should match the Assetto Corsa server_cfg.ini HTTP_PORT setting (required for ping control)."`
	ACServerPluginLocalPort   int    `json:"ac_server_plugin_local_port" help:"The plugin port of the server as set in the server_cfg.ini (UDP_PLUGIN_LOCAL_PORT)."`
	ACServerPluginAddressPort int    `json:"ac_server_plugin_address_port" help:"The port that the plugin will use (the portion after ':' in the UDP_PLUGIN_ADDRESS server_cfg.ini entry, if you set UDP_PLUGIN_ADDRESS=127.0.0.1:12000, set this to 12000)."`

	// Pings n stuff
	UpdateInterval    int `json:"update_interval" help:"How frequently should the Assetto Corsa send information about each car (lowering might increase CPU usage). 100 = 100ms."`
	MaxPing           int `json:"max_ping" help:"The maximum ping (in ms) a driver can have (the plugin will issue a warning if the instantaneous ping is over and kick the driver if the average is above the specified value for the last 4 measurements). Do not set this too low or it might affect server population. Set to 0 to disable the ping limit altogether."`
	PingCheckInterval int `json:"ping_check_interval" help:"The amount of seconds between two consecutive ping checks (decreasing this value might lead to high CPU usage). Set to 0 to disable the ping check feature altogether."`

	// WebStats
	WebStatsServerAddress string `json:"web_stats_server_address" help:"Stats Web Server Address (change this if you want to listen only on a certain interface, not recommended)."`
	WebStatsServerPort    int    `json:"web_stats_server_port" help:"Stats Web Server Port. For linux read the troubleshooting section of the readme.txt help file."`
	WebStatsCacheTime     int    `json:"web_stats_cache_time" help:"Time to cache the stats in seconds (decreasing might increase CPU usage)"`

	// WebAuth
	WebAuthServerAddress string   `json:"web_auth_server_address" help:"Stats Web Auth Address (change this if you want to listen only on a certain interface, not recommended)."`
	WebAuthServerPort    int      `json:"web_auth_server_port" help:"Stats Web Auth Port For linux don't use ports below 1024."`
	WebAuthCacheTime     int      `json:"web_auth_cache_time" help:"Time to cache the web auth result in seconds"`
	WebAuthRelayTo       []string `json:"web_auth_relay_to" help:"Use this if you wish to also block users according to other plugins. Values are the same that you would set as the AUTH_PLUGIN_ADDRESS in the Assetto Corsa server_cfg.ini for the third-party plugin (e.g. for Minorating you would replace AUTH_PLUGIN_ADDRESS_1 with plugin.minorating.com:805/minodata/auth/ABCN/?).  You can of course also use another instance of the Kissmyrank Plugin Auth to get cross server blocking using the Kissmyrank AUTH_PLUGIN_ADDRESS from the other plugin instance. Check the console at the plugin start to see if the address is parsed correctly."`

	UDPReplayTo []string `json:"udp_relay_to" help:"Use this if you wish to relay UDP traffic to other plugins (e.g. Minorating, Stracker. etc.). It works like this AC Server <-> Kissmyrank Plugin <-> Other plugins. For each plugin you need to specify the address in the  ip_address:port format (e.g. for a plugin running on 127.0.0.1 with port 12003, you would replace UDP_PLUGIN_ADDRESS_1 with 127.0.0.1:12003). In order for this to work you then need to set the other plugin like if the Kissmyrank Plugin was the Assetto Corsa Server (e.g. for the default Kissmyrank config that runs on port 12000, you would set the other plugin ac_server_plugin_port to 12000 and the ac_server_ip to the IP of the machine where you're running the Kissmyrank Plugin). Check the console at start to see if the address is parsed correctly."`

	// ACServer
	ACServerConfigIniPath   string `json:"ac_server_cfg_ini_path" help:"Path of the Assetto Corsa server_cfg.ini file to be used for track rotation. This must be the actual server_cfg.ini that acServer uses. The plugin will update this file and restart the server to rotate the track."`
	ACServerBinaryPath      string `json:"ac_server_bin_path" help:"Absolute Path of the Assetto Corsa Server executable to be used for track rotation (e.g. Windows c:/steam/acserver/acServer.exe, Linux /home/steam/acserver/acServer). The plugin will run this file to launch the Assetto Corsa server."`
	ACServerBinaryArguments string `json:"ac_server_bin_args" help:"[Not Recommended!] Assetto Corsa Binary Launch Arguments for multiple servers like ['-c path_to_server_cfg.ini', '-e path_to_entry_list.ini'] (this might not work at all on some operating systems, for multiple servers please make copies of the server folder and run each instance separately)."`
	ACServerLogPath         string `json:"ac_server_log_path" help:"The path where you wish to save the Assetto Corsa Server logs."`

	// Track Rotation is not supported with Server Manager.
	TrackList               []interface{} `json:"track_list" help:"" show:"-"`
	TrackRotationMaxPlayers int           `json:"track_rotation_max_players" help:"" show:"-"`

	// Money
	CurrencySymbol    string `json:"currency_symbol" help:"The symbol of the currency used for all drivers fines and payments (e.g. €,$,RUB)."`
	ThousandSeparator string `json:"thousand_separator" help:"The symbol to be used to separate thousands (e.g. 19,000 or 19.000)."`
	MinMoney          int    `json:"Minimum amount of money (in thousands) a driver must have to stay on the server. 1 = 1000€ (e.g. -12 -> drivers with more than 12000€ of debt will be kicked). Please also check start_money in order to make sure that this value makes sense."`

	RaceMinimumPlayers            int     `json:"race_min_players" help:"Minimum number of players required to race (must be >=2 if you're using the money system). If there are not enough players the server will skip the race session."`
	RaceDriverEntryFee            float64 `json:"race_driver_entry_fee" help:"How much a driver has to pay to enter a race session on the server (in thousands). 0.3 = 300€ (e.g. drivers will be charged 300€ at the beginning of each race)."`
	RaceSponsorEntryFee           float64 `json:"race_sponsor_entry_fee" help:"How much a sponsor pays to get the driver into the race (in thousands). 1.5 = 1500€ (e.g. this, together with the race entry fee concurs to the total competition prize, the higher, the higher the race payouts)."`
	RaceSponsorRewardBaseLength   int     `json:"race_sponsor_reward_base_length" help:"The amount of km required to pay the full sponsor fee (e.g. if the sponsor fee configuration is 1000€ and  you set this to 35, for a race of 7 laps on a 5km circuit sponsors will contribute 1000€ for each car, if the laps are 14 they will pay 2000€ for each car allowing for bigger prizes for longer races)."`
	RaceSponsorRewardBaseTime     int     `json:"race_sponsor_reward_base_time" help:"The amount of minutes required to pay the full sponsor fee (e.g. if the sponsor fee configuration is 1000€ and you set this to 15, for a race of 15 minutes sponsors will contribute 1000€ for each car, for 30 minutes they will pay 2000€ for each car allowing for bigger prizes for longer races)."`
	RaceSponsorCleanGainReward    int     `json:"race_sponsor_clean_gain_reward" help:"How much sponsors will reward a driver that makes 5 clean overtakes during the race (in thousands). 1 = 1000€. Clean overtakes are positions gained without collisions excluding gains made on drivers that disconnect during the race."`
	RaceSponsorCleanGainOvertakes int     `json:"race_sponsor_clean_gain_overtakes" help:"How many clean overtakes are required for the sponsors to issue one clean gain reward (e.g. if you set this to 5 and the driver makes 5 clean overtakes he gets 1000€, if the overtakes are 10, he gets 2000€ etc.)"`

	RaceFastestLapPrize                  float64 `json:"race_fastest_lap_prize" help:"How much a driver will be paid for the fastest lap of the race (in thousands). 0.15 = 150€."`
	LaptimeChallengeBasePrize            float64 `json:"laptime_challenge_base_prize" help:"Laptime Challenge Base Prize (in thousands). 0.01 = 10€. Reward is base*level (level 2 gets 20). Set to 0 to disable the Laptime Challenge feature."`
	LaptimeChallengeBaseAverageSpeed     int     `json:"laptime_challenge_base_average_speed" help:"Laptime Challenge Base Average Speed (km/h). 110=110km/h. A driver is Level 0 if he can drive a lap at 110 km/h average. Level 1 for 111 km/h and so on..."`
	LaptimeChallengeLevelAverageSpeedGap int     `json:"laptime_challenge_level_average_speed_gap" help:"The Average Speed Gap between two consecutive Laptime Challenge Levels in (km/h). 1=1km/h (e.g. Level 0 => 110km/h, Level 1 => 111km."`
	AlltimeFastestLapPrize               float64 `json:"alltime_fastest_lap_prize" help:"How much a driver will be paid for the fastest lap of all times  (in thousands). 1.5 = 1500€."`

	DamageCostBetweenCars int
}
