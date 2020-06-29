package servermanager

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/cj123/formulate"
)

func DefaultKissMyRankConfig() *KissMyRankConfig {
	return &KissMyRankConfig{
		EnableKissMyRank:                        false,
		MaxPing:                                 450,
		MaxPingDeviation:                        100,
		PingCheckInterval:                       15,
		WebStatsServerAddress:                   "0.0.0.0",
		WebStatsServerPort:                      7789,
		WebStatsCacheTime:                       60,
		WebAuthServerAddress:                    "0.0.0.0",
		WebAuthServerPort:                       12001,
		WebAuthCacheTime:                        60,
		WebAuthRelayTo:                          "",
		UDPReplayTo:                             "",
		CurrencySymbol:                          "$",
		ThousandSeparator:                       ",",
		MinMoney:                                0,
		RaceMinimumPlayers:                      2,
		RaceDriverEntryFee:                      0.3,
		RaceSponsorEntryFee:                     1.5,
		RaceSponsorRewardBaseLength:             40,
		RaceSponsorRewardBaseTime:               15,
		RaceSponsorCleanGainReward:              1,
		RaceSponsorCleanGainOvertakes:           5,
		RaceFastestLapPrize:                     0.3,
		LaptimeChallengeBasePrize:               0.01,
		LaptimeChallengeBaseAverageSpeed:        110,
		LaptimeChallengeLevelAverageSpeedGap:    1,
		AlltimeFastestLapPrize:                  1.5,
		DamageCostBetweenCars:                   3,
		DamageCostWithEnvironment:               3,
		HotlapProtection:                        45,
		LappingProtection:                       40,
		RelativeHotlapProtection:                0.008,
		RelativeLappingProtection:               0.007,
		WarnedCarGrace:                          21,
		MinimumDrivingStandard:                  1.1,
		MinimumDrivingStandardLaps:              12,
		MinimumDrivingStandardRechargePeriod:    86400,
		MinimumDrivingStandardMinPlayers:        12,
		CarTowingCost:                           0.05,
		CutLinesEnabled:                         1,
		SessionHistoryLength:                    30,
		SpeedUnitFormat:                         "kmh",
		ReservedSlotsGUIDList:                   "",
		ReservedSlotsAccessKey:                  "",
		ReservedSlotsBootPlayersAtRace:          0,
		MemoryMonitorEnabled:                    0,
		DatabaseSharingUniqueName:               "",
		DatabaseSharingLocalGroupPort:           0,
		DatabaseSharingRemoteListenPort:         0,
		DatabaseSharingRemoteSecretKey:          "",
		DatabaseSharingRemoteListenAddress:      "0.0.0.0",
		DatabaseSharingRemoteConnectToAddresses: "",
		DatabaseSharingRelayForNames:            "",
		WebAdminConsolePassword:                 "",
		MaxCollisionsPer100km:                   30,
		MaxCollisionsPer100kmMinDistance:        100,
		MacCollisionsPer100kmRechargeHours:      6,
		CustomChatDriverWelcomeMessages:         "Welcome to this KissMyRank+Server Manager Enabled Server!\nHave a good race!",
		ReverseGearMaxDistance:                  30,
		CollisionMinimumDamageWithEnvironment:   0.001,
		CollisionMinimumDamageBetweenCars:       0.001,
		WebStatsDriversPerPage:                  600,
		TrackBoundaryCutMaxSpeed:                30,
		TrackBoundarySameLapCutMaxSpeed:         80,
		TrackBoundarySampleLength:               3,
		CleanLapReward:                          0.05,
		TimeBasedRaceExtraLap:                   0,
		RacePodiumAnnouncement:                  1,
		RaceControlPassword:                     "",
		WebStatsInterface:                       1,
		RaceControlMaxEvents:                    180,
		WebStatsOverridePublicAddress:           "",
		WebStatsOverridePublicPort:              0,
		ACServerRestartIfInactiveForMinutes:     90,
		RaceControlCollisionSpace:               2.2,
		RaceControlCollisionTime:                9,
		RaceControlLogOvertakes:                 0,
		WebStatsResultsShowLapLog:               0,
		StartMoney:                              30.00,
		PenaltyCosts: PenaltyCostMap{
			HotLapProtection: PenaltyCostSessions{
				Practice: 0.1,
				Qualify:  0.6,
			},
			HotLappingCarCollision: PenaltyCostSessions{
				Practice: 0.1,
				Qualify:  0.6,
			},
			LappingProtection: PenaltyCostSessions{
				Race: 0,
			},
			LappingCarCollision: PenaltyCostSessions{
				Race: 0.6,
			},
			ReverseGear: PenaltyCostSessions{
				Practice: 0.1,
				Qualify:  0.6,
				Race:     0.6,
			},
			TrackBoundaryCut: PenaltyCostSessions{
				Practice: 0,
				Qualify:  0.1,
				Race:     0.05,
				Other:    0,
			},
			TrackRejoinMaxSpeed: PenaltyCostSessions{
				Practice: 0,
				Qualify:  0.1,
				Race:     0.1,
				Other:    0,
			},
			MaxInfractions: PenaltyCostSessions{
				Qualify: 0.6,
			},
			MaxCollisions: PenaltyCostSessions{
				Practice: 0.3,
				Qualify:  0.6,
				Race:     0.6,
			},
			FirstBlood: PenaltyCostSessions{
				Race: 0.6,
			},
			PitLaneSpeeding: PenaltyCostSessions{
				Qualify: 0.3,
				Race:    0.6,
			},
			PitExitLineCrossing: PenaltyCostSessions{
				Practice: 0.3,
				Qualify:  0.6,
				Race:     0.6,
			},
			CutLineYourCustomCutLine:              PenaltyCostSessions{},
			AntiCheatMaxClockDeltaConsecutiveHits: PenaltyCostSessions{},
			SpeedingUnderVirtualSafetyCar:         PenaltyCostSessions{},
			SlowingUnderVirtualSafetyCar:          PenaltyCostSessions{},
			OvertakingUnderVirtualSafety:          PenaltyCostSessions{},
			ImprovingQualifyLapTimeWithInfractions: PenaltyCostSessions{
				Qualify: 0,
			},
			ParkingNearTrack: PenaltyCostSessions{
				Practice: 0.3,
				Qualify:  0.6,
				Race:     0.6,
			},
		},
		PenaltyActions: PenaltyActionMap{
			HotLapProtection: PenaltyActionSessions{},
			LappingProtection: PenaltyActionSessions{
				Race: "DT1",
			},
			ReverseGear:         PenaltyActionSessions{},
			TrackBoundaryCut:    PenaltyActionSessions{},
			TrackRejoinMaxSpeed: PenaltyActionSessions{},
			MaxInfractions: PenaltyActionSessions{
				Race: "DT1",
			},
			MaxCollisions: PenaltyActionSessions{
				Practice: "K",
				Qualify:  "K",
				Race:     "K",
			},
			FirstBlood:               PenaltyActionSessions{},
			PitLaneSpeeding:          PenaltyActionSessions{},
			PitExitLineCrossing:      PenaltyActionSessions{},
			CutLineYourCustomCutLine: PenaltyActionSessions{},
			AntiCheatMaxClockDeltaConsecutiveHits: PenaltyActionSessions{
				Practice: "K",
				Qualify:  "K",
				Race:     "K",
			},
			SpeedingUnderVirtualSafetyCar: PenaltyActionSessions{
				Race: "K",
			},
			SlowingUnderVirtualSafetyCar: PenaltyActionSessions{
				Race: "K",
			},
			OvertakingUnderVirtualSafety: PenaltyActionSessions{
				Race: "K",
			},
			ImprovingQualifyLapTimeWithInfractions: PenaltyActionSessions{
				Qualify: "",
			},
			ParkingNearTrack: PenaltyActionSessions{
				Practice: "K",
				Qualify:  "K",
				Race:     "K",
			},
		},
		NoMoney:                                             0,
		MaxInfractions:                                      6,
		ACChatAdminGUIDList:                                 "",
		ACChatAdminPassword:                                 "",
		RaceControlCollisionReplayTime:                      6,
		RaceControlCutReplayTime:                            6,
		RaceControlOvertakeReplayTime:                       6,
		RaceControlIncludePlayersNearerThan:                 100,
		MaxCollisions:                                       6,
		TrackRejoinMaxSpeed:                                 120,
		TrackBoundaryCutMaxTime:                             9,
		DamageCostBetweenCarsBaseSpeed:                      100,
		DamageCostWithEnvironmentBaseSpeed:                  100,
		TrackBoundaryCutGainFilter:                          1,
		TrackBoundaryCutGainFilterMinLossPercent:            6,
		TrackBoundaryCutGainFilterMinAverageSpeed:           45,
		RankSortByWinStats:                                  0,
		JLPMoneyKillSwitch:                                  0,
		QualifyTopThreeBasePrize:                            0.5,
		QualifyTopThreePrizeMinPlayers:                      9,
		PitSpeedLimit:                                       80,
		DrivingLinePenaltyRepeatGrace:                       3,
		AnticheatLaptimeInvalidateMaxClockDelta:             1000,
		AnticheatPenalizeDriverMaxClockDeltaConsecutiveHits: 3,
		RollingStart:                                        0,
		VSCSpeedingMaxGrace:                                 3,
		VSCSlowingMaxGrace:                                  9,
		VSCDefaultSpeedLimit:                                120,
		VSCOvertakingMaxGrace:                               12,
		VSCFormationLapSpeedLimit:                           150,
		VSCFormationLapMinSpeed:                             30,
		VSCDefaultLeaderSlowAllowOvertakeSpeed:              30,
		VSCDefaultSlowAndFarAllowOvertakeSpeed:              40,
		VSCDefaultSlowAndFarAllowOvertakeDistance:           75,
		VSCFormationLapFarAllowOvertakeDistance:             100,
		PenaltyInfractionMap: PenaltyInfractionMap{
			HotLapProtection:       PenaltyInfractionSessions{},
			HotLappingCarCollision: PenaltyInfractionSessions{},
			LappingProtection:      PenaltyInfractionSessions{},
			LappingCarCollision:    PenaltyInfractionSessions{},
			ReverseGear:            PenaltyInfractionSessions{},
			TrackBoundaryCut: PenaltyInfractionSessions{
				Practice: 1,
				Qualify:  1,
				Race:     1,
			},
			TrackRejoinMaxSpeed: PenaltyInfractionSessions{
				Practice: 1,
				Qualify:  1,
				Race:     1,
			},
			MaxInfractions: PenaltyInfractionSessions{},
			MaxCollisions:  PenaltyInfractionSessions{},
			FirstBlood:     PenaltyInfractionSessions{},
			PitLaneSpeeding: PenaltyInfractionSessions{
				Practice: 1,
				Qualify:  1,
				Race:     1,
			},
			PitExitLineCrossing: PenaltyInfractionSessions{
				Practice: 1,
				Qualify:  1,
				Race:     1,
			},
			CutLineYourCustomCutLine: PenaltyInfractionSessions{
				Practice: 1,
				Qualify:  1,
				Race:     1,
			},
			AntiCheatMaxClockDeltaConsecutiveHits: PenaltyInfractionSessions{},
			SpeedingUnderVirtualSafetyCar:         PenaltyInfractionSessions{},
			SlowingUnderVirtualSafetyCar:          PenaltyInfractionSessions{},
			OvertakingUnderVirtualSafety:          PenaltyInfractionSessions{},
			ImprovingQualifyLapTimeWithInfractions: PenaltyInfractionSessions{
				Qualify: 0,
			},
			ParkingNearTrack: PenaltyInfractionSessions{},
		},

		RaceMassAccidentCrashedPlayersPercentage: 45,
		RaceMassAccidentCrashTime:                30,
		RaceMassAccidentMinCrashedPlayers:        4,
		RaceMassAccidentResponse: RaceMassAccidentResponse{
			FirstLap:  "VSC60",
			OtherLaps: "VSC60",
		},
		LiveTrackView:    1,
		ACAppLinkUDPPort: 12002,
		ChatDriverWelcomeMessageShowRaceControlLink:         1,
		ImprovingQualifyLaptimeWithInfractionsCutoffPercent: 107,
		WebAdminConsoleGuestPassword:                        "",
		ParkedCarMaxGrace:                                   4,
		ParkedCarSeconds:                                    6,
		ParkedCarDistance:                                   24,
		RightToBeForgottenChatCommand:                       0,
		DriveThroughNoKick:                                  0,
	}
}

type KissMyRankConfig struct {
	EnableKissMyRank bool

	// Pings n stuff
	ConnectionOptions FormHeading `ini:"-" json:"-"`
	UpdateInterval    int         `json:"update_interval" show:"-" help:"How frequently should the Assetto Corsa send information about each car (lowering might increase CPU usage). 100 = 100ms."`
	MaxPing           int         `json:"max_ping" help:"The maximum ping (in ms) a driver can have (the plugin will issue a warning if the instantaneous ping is over and kick the driver if the average is above the specified value for the last 4 measurements). Do not set this too low or it might affect server population. Set to 0 to disable the ping limit altogether."`
	MaxPingDeviation  int         `json:"max_ping_deviation" help:"The maximum ping deviation (in ms) drivers can have before their connection is considered unstable (the plugin will kick the driver if the standard ping deviation is above the specified value for the last 4 measurements). This is a measure of the connection instability which is important for MP racing. Set to 0 to disable."`
	PingCheckInterval int         `json:"ping_check_interval" help:"The amount of seconds between two consecutive ping checks (decreasing this value might lead to high CPU usage). Set to 0 to disable the ping check feature altogether."`

	// WebStats
	WebInterface                  FormHeading          `ini:"-" json:"-"`
	WebStatsInterface             formulate.BoolNumber `json:"web_stats_interface" help:"Whether to enable the web stats interface (required for Race Control and Race Director collision moderation)."`
	WebStatsServerAddress         string               `json:"web_stats_server_address" show:"open" help:"Stats Web Server Address (change this if you want to listen only on a certain interface, not recommended)."`
	WebStatsServerPort            int                  `json:"web_stats_server_port" show:"open" help:"Stats Web Server Port. For linux read the troubleshooting section of the readme.txt help file."`
	WebStatsPublicURL             string               `json:"web_stats_public_url" show:"open" help:"<strong>Please set up this value if you are using the web stats port!</strong> This should be set to the URL that anyone can access. This link will be displayed on Server Manager's Live Timings page so that anybody can view the KMR interface. If this is left blank, no link will be shown on the Live Timings page!"`
	WebStatsDriversPerPage        int                  `json:"web_stats_drivers_per_page" help:"Web Stats Drivers per page. Sets how many drivers per page will show on the Web Stats Drivers list."`
	WebStatsCacheTime             int                  `json:"web_stats_cache_time" help:"Time to cache the stats in seconds (decreasing might increase CPU usage)"`
	WebStatsOverridePublicAddress string               `json:"web_stats_override_public_address" show:"open" help:"Set this to your server public Web Stats address if you wish to override the plugin public address autodetection."`
	WebStatsOverridePublicPort    int                  `json:"web_stats_override_public_port" show:"open" help:"Set this to your server public Web Stats port if you wish to override the default setting (e.g. in case of a forward, proxy, redirect etc.)."`
	WebStatsResultsShowLapLog     formulate.BoolNumber `json:"web_stats_results_show_lap_log" help:"Set this to 1 if you wish the Web Stats Race Results viewer to include the lap log (which shows the times recorded for each player, the tyres he was on etc. on a per lap basis)."`
	WebAdminConsolePassword       string               `json:"web_admin_console_password" help:"Web Admin Console Password. The password (min 12 characters or the Kissmyrank Web Admin Interface will be forcefully disabled) that you will use to login to your Kissmyrank Web Admin Console at http://yourserver/kissmyrank_admin (when you open the page, if your password is 'yourcomplexpassword' you need to type 'login yourcomplexpassword' in order to gain access). Leave this key empty if you don't want to use the Kissmyrank Web Admin Console."`
	WebAdminConsoleGuestPassword  string               `json:"web_admin_console_guest_password" help:"Set this password to at least 8 characters if you wish to give view-only access to the Web Admin console. Please keep in mind that web_admin_console_password needs to be set to activate the console in first place. Furthermore please only give this password to trusted member as they will see everything that appears in the Kissmyrank console."`
	LiveTrackView                 formulate.BoolNumber `json:"live_track_view" help:"Whether to enable the Live Track View (e.g. switch off if you wish to disable the Web Stats Live Map option)."`

	// WebAuth
	WebAuth              FormHeading          `ini:"-" json:"-" show:"open"`
	WebAuthServerAddress string               `json:"web_auth_server_address" show:"open" help:"Stats Web Auth Address (change this if you want to listen only on a certain interface, not recommended)."`
	WebAuthServerPort    int                  `json:"web_auth_server_port" show:"open" help:"Stats Web Auth Port For linux don't use ports below 1024."`
	WebAuthCacheTime     int                  `json:"web_auth_cache_time" show:"open" help:"Time to cache the web auth result in seconds"`
	WebAuthRelayTo       NewLineSeparatedList `json:"web_auth_relay_to" elem:"textarea" show:"open" help:"One per line. Use this if you wish to also block users according to other plugins. Values are the same that you would set as the AUTH_PLUGIN_ADDRESS in the Assetto Corsa server_cfg.ini for the third-party plugin (e.g. for Minorating you would replace AUTH_PLUGIN_ADDRESS_1 with plugin.minorating.com:805/minodata/auth/ABCN/?).  You can of course also use another instance of the Kissmyrank Plugin Auth to get cross server blocking using the Kissmyrank AUTH_PLUGIN_ADDRESS from the other plugin instance. Check the console at the plugin start to see if the address is parsed correctly."`

	UDPRelay    FormHeading          `ini:"-" json:"-" show:"open"`
	UDPReplayTo NewLineSeparatedList `json:"udp_relay_to" elem:"textarea" show:"open" help:"One per line. Use this if you wish to relay UDP traffic to other plugins (e.g. Minorating, Stracker. etc.). It works like this AC Server <-> Kissmyrank Plugin <-> Other plugins. For each plugin you need to specify the address in the  ip_address:port format (e.g. for a plugin running on 127.0.0.1 with port 12003, you would replace UDP_PLUGIN_ADDRESS_1 with 127.0.0.1:12003). In order for this to work you then need to set the other plugin like if the Kissmyrank Plugin was the Assetto Corsa Server (e.g. for the default Kissmyrank config that runs on port 12000, you would set the other plugin ac_server_plugin_port to 12000 and the ac_server_ip to the IP of the machine where you're running the Kissmyrank Plugin). Check the console at start to see if the address is parsed correctly."`

	// Money
	Money                                FormHeading          `ini:"-" json:"-"`
	CurrencySymbol                       string               `json:"currency_symbol" help:"The symbol of the currency used for all drivers fines and payments (e.g. €,$,RUB)."`
	ThousandSeparator                    string               `json:"thousand_separator" help:"The symbol to be used to separate thousands (e.g. 19,000 or 19.000)."`
	StartMoney                           float64              `json:"start_money" help:"The money (in thousands) that any unranked player is given when he joins the server the first time (30=30000€). Please also check min_money in order to make sure that this value makes sense."`
	MinMoney                             int                  `json:"min_money" help:"Minimum amount of money (in thousands) a driver must have to stay on the server. 1 = 1000€ (e.g. -12 -> drivers with more than 12000€ of debt will be kicked). Please also check start_money in order to make sure that this value makes sense."`
	NoMoney                              formulate.BoolNumber `json:"no_money" help:"Set this to ON if you wish to use points instead of money (no money, no party :D)."`
	RaceMinimumPlayers                   int                  `json:"race_min_players" help:"Minimum number of players required to race (must be >=2 if you're using the money system). If there are not enough players the server will skip the race session."`
	RaceDriverEntryFee                   float64              `json:"race_driver_entry_fee" help:"How much a driver has to pay to enter a race session on the server (in thousands). 0.3 = 300€ (e.g. drivers will be charged 300€ at the beginning of each race)."`
	RaceSponsorEntryFee                  float64              `json:"race_sponsor_entry_fee" help:"How much a sponsor pays to get the driver into the race (in thousands). 1.5 = 1500€ (e.g. this, together with the race entry fee concurs to the total competition prize, the higher, the higher the race payouts)."`
	RaceSponsorRewardBaseLength          int                  `json:"race_sponsor_reward_base_length" help:"The amount of km required to pay the full sponsor fee (e.g. if the sponsor fee configuration is 1000€ and  you set this to 35, for a race of 7 laps on a 5km circuit sponsors will contribute 1000€ for each car, if the laps are 14 they will pay 2000€ for each car allowing for bigger prizes for longer races)."`
	RaceSponsorRewardBaseTime            int                  `json:"race_sponsor_reward_base_time" help:"The amount of minutes required to pay the full sponsor fee (e.g. if the sponsor fee configuration is 1000€ and you set this to 15, for a race of 15 minutes sponsors will contribute 1000€ for each car, for 30 minutes they will pay 2000€ for each car allowing for bigger prizes for longer races)."`
	RaceSponsorCleanGainReward           int                  `json:"race_sponsor_clean_gain_reward" help:"How much sponsors will reward a driver that makes 5 clean overtakes during the race (in thousands). 1 = 1000€. Clean overtakes are positions gained without collisions excluding gains made on drivers that disconnect during the race."`
	RaceSponsorCleanGainOvertakes        int                  `json:"race_sponsor_clean_gain_overtakes" help:"How many clean overtakes are required for the sponsors to issue one clean gain reward (e.g. if you set this to 5 and the driver makes 5 clean overtakes he gets 1000€, if the overtakes are 10, he gets 2000€ etc.)"`
	RaceFastestLapPrize                  float64              `json:"race_fastest_lap_prize" help:"How much a driver will be paid for the fastest lap of the race (in thousands). 0.15 = 150€."`
	LaptimeChallengeBasePrize            float64              `json:"laptime_challenge_base_prize" help:"Laptime Challenge Base Prize (in thousands). 0.01 = 10€. Reward is base*level (level 2 gets 20). Set to 0 to disable the Laptime Challenge feature."`
	LaptimeChallengeBaseAverageSpeed     int                  `json:"laptime_challenge_base_average_speed" help:"Laptime Challenge Base Average Speed (km/h). 110=110km/h. A driver is Level 0 if he can drive a lap at 110 km/h average. Level 1 for 111 km/h and so on..."`
	LaptimeChallengeLevelAverageSpeedGap int                  `json:"laptime_challenge_level_average_speed_gap" help:"The Average Speed Gap between two consecutive Laptime Challenge Levels in (km/h). 1=1km/h (e.g. Level 0 => 110km/h, Level 1 => 111km."`
	AlltimeFastestLapPrize               float64              `json:"alltime_fastest_lap_prize" help:"How much a driver will be paid for the fastest lap of all times  (in thousands). 1.5 = 1500€."`
	DamageCostBetweenCars                int                  `json:"damage_cost_between_cars" help:"The cost (in thousands) of the damage of a collision between cars at the damage_cost_between_cars_base_speed. 3 = 3000€ (e.g. drivers will be charged 3000€ if they collide at the specified relative base speed). Please notice that damage goes with the square of the relative velocity."`
	DamageCostWithEnvironment            int                  `json:"damage_cost_with_environment" help:"The cost (in thousands) of the damage of a collision with the environment at damage_cost_with_environment_base_speed. 3 = 3000€ (e.g. drivers will be charged 3000€ if they collide with the environment at the specified relative base speed). Please notice that damage goes with the square of the velocity."`
	DamageCostBetweenCarsBaseSpeed       int                  `json:"damage_cost_between_cars_base_speed" help:"The base speed (in km/h) for damage_cost_between_cars."`
	DamageCostWithEnvironmentBaseSpeed   int                  `json:"damage_cost_with_environment_base_speed" help:"The base speed (in km/h) for damage_cost_with_environment."`
	CarTowingCost                        float64              `json:"car_towing_cost" help:"Car Towing Cost per Km. How much (in thousands) drivers will pay to tow their cars to the pits per Km. 0.05 = 50€ per Km (e.g. when they go back to the pits, a driver will pay 50€ for every km. Set to 0 to disable the feature altogether."`
	JLPMoneyKillSwitch                   formulate.BoolNumber `json:"jlp_money_kill_switch" help:"Set this to 1 if, just like Captain Jean-Luc Picard, you do not want to use the money/point system at all (e.g. leagues, private servers and all people that are only interested in Kissmyrank Race Control and Tracking features). If you change this, you have to reset the database (e.g. perform a fresh install of the plugin)."`
	QualifyTopThreeBasePrize             float64              `json:"qualify_top_three_base_prize" help:"How much (in thousands) should the first three drivers be rewarded, if more than the specified number of players posted a valid qualify time. 0.5 = 500€ (e.g. first gets 1500, second gets 1000, third gets 500€)."`
	QualifyTopThreePrizeMinPlayers       int                  `json:"qualify_top_three_prize_min_players" help:"How many players need to post a time before the pole prizes are assigned."`

	Protections                                         FormHeading          `ini:"-" json:"-"`
	HotlapProtection                                    int                  `json:"hotlap_protection" help:"At what gap (in meters) should a car on an outlap receive the 'Let the other car pass' warning when it's about to get passed."`
	LappingProtection                                   int                  `json:"lapping_protection" help:"At what gap (in meters) should a car that is being lapped get the 'Blue flag' warning when it's about to get passed."`
	RelativeHotlapProtection                            float64              `json:"relative_hotlap_protection" help:"At what gap (as a fraction of overall track length) should a car on an outlap get the 'Let the other car pass' warning when it's about to get passed. Only used if track length is missing. 1 is the full track length (0.008 is 8/1000 of the full track length)."`
	RelativeLappingProtection                           float64              `json:"relatve_lapping_protection" help:"At what gap (as a fraction of overall track length) should a car that is being lapped get the 'Blue flag' warning when it's about to get passed. Only used if track length is missing. 1 is the full track length (0.007 is 7/1000 of the full track length)."`
	WarnedCarGrace                                      int                  `json:"warned_car_grace" help:"The time in seconds every car is given to make room after the first warning. Increase to give more time to a warned driver to let the other car pass."`
	MinimumDrivingStandard                              float64              `json:"minimum_driving_standard" help:"Maximum allowed leaderboard time as a fraction of the fastest time. 1.1 = 110% of the fastest time (e.g. a driver that is not able to set a time that is within 110% of the fastest lap time within minimum_driving_standard_laps laps will not be allowed in the server."`
	MinimumDrivingStandardLaps                          int                  `json:"minimum_driving_standard_laps" help:"The number of valid laps that a driver has to complete in order to reach the minimum driving standard. A driver is given a second chance after a few days according to minimum_driving_standard_recharge_period."`
	MinimumDrivingStandardRechargePeriod                int                  `json:"minimum_driving_standard_recharge_period" help:"The number of seconds to recharge one lap to a driver that doesn't meet the standards (e.g. if minimum_driving_standard_laps is set to 12, this value sets the amount of time a player with 13 laps and a poor time has to wait before he can join the server again)."`
	MinimumDrivingStandardMinPlayers                    int                  `json:"minimum_driving_standard_min_players" help:"Minimum number of players over which to enforce the minimum driving standards. If set to anything that is not 0, the driving standard policy will be enforced only if more than the specified number of players is present on server."`
	CutLinesEnabled                                     formulate.BoolNumber `json:"cut_lines_enabled" help:"Enable cut lines. Set this to 0 if you wish to disable the cut_lines feature. Keep in mind that cut lines will only work if they are defined using the cut line drawer. To check the current cut lines, type cut_line_list in the console.. For more, please check readme.txt."`
	MaxCollisionsPer100km                               int                  `json:"max_collisions_per_100km" help:"The maximum amount of collisions per 100 km a driver is allowed on the server. This applies to drivers that have driven more than max_collisions_per_100km_min_distance."`
	MaxCollisionsPer100kmMinDistance                    int                  `json:"max_collisions_per_100km_min_distance" help:"The driven distance (in km) over which the max_collisions_per_100km will start to apply."`
	MacCollisionsPer100kmRechargeHours                  int                  `json:"max_collisions_per_100km_recharge_hours" help:"The base time (hours per collision unit) a driver exceeding max_collisions_per_100km has to wait before he can rejoin the server (e.g. if max_collisions_per_100km is set to 30, a driver with 31 collision per 100km will have to wait 1h form the last time he left the server, if 32, he will have to wait 2 hours and so on)."`
	ReverseGearMaxDistance                              int                  `json:"reverse_gear_max_distance" help:"The maximum distance in meters a driver is allowed to reverse on track (set to 0 to disable the penalty)."`
	CollisionMinimumDamageWithEnvironment               float64              `json:"collision_minimum_damage_with_environment" help:"The minimum damage cost (in thousands) under which collisions with the environment will not be logged. Set this to 0 to log all collisions. 0.001 = do not log collisions with the environment below 1€."`
	CollisionMinimumDamageBetweenCars                   float64              `json:"collision_minimum_damage_between_cars" help:"The minimum damage cost (in thousands) under which collisions between cars will not be logged. Set this to 0 to log all collisions. 0.001 = do not log collisions with the environment below 1€."`
	TrackBoundaryCutMaxSpeed                            int                  `json:"track_boundary_cut_max_speed" help:"The maximum speed (in km/h) at which a driver is allowed to cross the track boundary."`
	TrackBoundarySameLapCutMaxSpeed                     int                  `json:"track_boundary_same_lap_cut_max_speed" help:"The maximum speed (in km/h) at which a driver is allowed to cross the track boundary after the first violation (this both applies to track re-entry and to further cuts)."`
	TrackBoundarySampleLength                           int                  `json:"track_boundary_sample_length" help:"The length (in meters) of a single track boundary length (max recommended value is 3). Increase this if you wish to save disk space at the expense of accuracy. Please notice that this value only applies to new data sets."`
	CleanLapReward                                      float64              `json:"clean_lap_reward" help:"How much (in thousands) a driver will be paid for a clean lap (e.g. lap without any cut). You can use this on practice servers (since race rewards will not work there)."`
	AnticheatLaptimeInvalidateMaxClockDelta             int                  `json:"anticheat_laptime_invalidate_max_clock_delta" help:"The maximum difference (in milliseconds) between the laptimes reported by the Assetto Corsa Server and those measured by the Plugin (with the hosting machine clock) before a laptime is rejected (e.g. to prevent the abuse where one can manipulate Assetto Corsa laptimes by acting on the CPU clock). Please keep in mind that this value needs to account for the natural delays in the communication between the plugin and the server. Do not set it too low or it might penalize drivers when your server is under stress. Use it only if your server has enough resources to process packets with steady delays. Set to 0 to disable."`
	AnticheatPenalizeDriverMaxClockDeltaConsecutiveHits int                  `json:"anticheat_penalize_driver_max_clock_delta_consecutive_hits" help:"The number of consecutive times a driver has to hit the max_clock_delta before getting the penalty defined in the penalty cost and penalty action maps. Set to 0 to disable."`
	MaxInfractions                                      int                  `json:"max_infractions" help:"The maximum number of times a driver is allowed to violate the server rules (cut track, speeding, pit exit line crossing etc.) before receiving the penalties set in the Penalty Costs and Penalty Actions."`
	ParkedCarMaxGrace                                   int                  `json:"parked_car_max_grace" help:"The amount of times a driver can trigger the parked car detection before he gets a penalty. 4 = 4 times (set to 0 to disable this detection)."`
	ParkedCarSeconds                                    int                  `json:"parked_car_seconds" help:"The number of seconds for the car parked detection. 6 = 6s (e.g. a driver that is near the track and doesn't move more than the parked_car_distance in 6s will trigger the detection once)."`
	ParkedCarDistance                                   int                  `json:"parked_car_distance" help:"The distance (in meters) a car must travel in parked_car_max_seconds not to trigger the detection. 24 = 24 meters (a car on the track that moves more than 24 meters in parked_car_seconds will not trigger the detection)."`
	RightToBeForgottenChatCommand                       formulate.BoolNumber `json:"right_to_be_forgotten_chat_command" help:"Whether to enable the 'kmr erase_personal_data_and_ban_myself' chat command which allows drivers to make use of their right to be forgotten and get all of their stats removed."`
	DriveThroughNoKick                                  formulate.BoolNumber `json:"drive_through_no_kick" help:"Whether to disable kick penalty when a Drive-Through is not cleared or when two Drive-Through events occur in a row."`

	Rules                                               FormHeading          `ini:"-" json:"-"`
	TimeBasedRaceExtraLap                               formulate.BoolNumber `json:"time_based_race_extra_lap" help:"Whether in a time based race the extra lap is enabled in the Assetto Corsa server_cfg.ini. If track rotation is active, the plugin will read the setting directly from the server_cfg.ini and ignore this value."`
	RacePodiumAnnouncement                              formulate.BoolNumber `json:"race_podium_announcement" help:"Whether to announce the first three drivers at the end of the race."`
	MaxCollisions                                       int                  `json:"max_collisions" help:"The maximum number of collision any user can be involved before receiving the penalties set in the penalty_cost and penalty_action maps."`
	TrackRejoinMaxSpeed                                 int                  `json:"track_rejoin_max_speed" help:"The maximum speed at which a driver is allowed to rejoin the track after going off track in an excluded area or stopping the car outside the track."`
	TrackBoundaryCutMaxTime                             int                  `json:"track_boundary_cut_max_time" help:"The maximum time (in seconds) between the cut start and the cut end (cuts outside this range will be regarded as out-of-track moments)."`
	TrackBoundaryCutGainFilter                          formulate.BoolNumber `json:"track_boundary_cut_gain_filter" help:"Whether to enable the Track Boundary Cut Gain filter which compares the cut advantage against the fastest lap of the session and doesn't give the penalty if the driver loses time against it."`
	TrackBoundaryCutGainFilterMinLossPercent            int                  `json:"track_boundary_cut_gain_filter_min_loss_percent" help:"The minimum acceptable time loss (in percent) for the Track Boundary Cut Gain filter to ignore the cut. 6=6% (e.g. a driving losing more than 6% compared to the fastest laptime of the session will not incur in a penalty)."`
	TrackBoundaryCutGainFilterMinAverageSpeed           int                  `json:"track_boundary_cut_gain_filter_min_average_speed" help:"This speed (expressed in km/h) sets the floor speed of the Track Boundary Cut Gain filter. 45 = 45km/h (e.g. if a player during the cut wastes more time than traveling the interested track section at 45km/h the cut will be disregarded). This is only useful in the first laps when the comparison lap has not been acquired yet."`
	DrivingLinePenaltyRepeatGrace                       int                  `json:"driving_line_penalty_repeat_grace" help:"The minimum time (in seconds) between two consecutive cutting penalties (boundaries and cut lines). 6 = 6s (e.g. if a driver cuts twice in less than 6s he will be only penalized once). Do not lower this too much to avoid penalty spam."`
	PitSpeedLimit                                       int                  `json:"pit_speed_limit" help:"The pit speed limit (in km/h). Above this speed the drive-through will be aborted. 80 = 80km/h (you might want to use 81 if you face the speed limiter bug). This value is required, please do not set to 0."`
	RollingStart                                        formulate.BoolNumber `json:"rolling_start" help:"If you wish to use Kissmyrank Rolling Start at the beginning of a race."`
	ImprovingQualifyLaptimeWithInfractionsCutoffPercent int                  `json:"improving_qualify_laptime_with_infractions_cutoff_percent" help:"The laptime (in percent of the session best) over which an improved time with cuts will not be considered an abuse. 107 = 107% (e.g. a driver posting a time with Kissmyrank infractions but no AC infractions that is above 107% of the fastest lap of the session will not trigger the penalty)."`

	VirtualSafetyCar                          FormHeading              `ini:"-" json:"-" `
	VSCSpeedingMaxGrace                       int                      `json:"vsc_speeding_max_grace" help:"The number of seconds a driver is allowed to drive over the speed limit during a Virtual Safety Car or the Rolling Start Formation Lap."`
	VSCSlowingMaxGrace                        int                      `json:"vsc_slowing_max_grace" help:"The number of seconds a driver is allowed to slow down under the minimum speed during the Rolling Start Formation Lap."`
	VSCDefaultSpeedLimit                      int                      `json:"vsc_default_speed_limit" help:"The maximum speed (in km/h) a player is allowed to drive during a Virtual Safety Car."`
	VSCOvertakingMaxGrace                     int                      `json:"vsc_overtaking_max_grace" help:"The number of seconds a driver has to give the position back when overtaking another car during a Virtual Safety Car or the Rolling Start Formation Lap."`
	VSCFormationLapSpeedLimit                 int                      `json:"vsc_formation_lap_speed_limit" help:"The maximum speed (in km/h) a player is allowed to drive during the Rolling Start Formation Lap."`
	VSCFormationLapMinSpeed                   int                      `json:"vsc_formation_lap_min_speed" help:"The minimum speed (in km/h) a driver must drive above during the Rolling Start Formation Lap (set to 0 to disable)."`
	VSCDefaultLeaderSlowAllowOvertakeSpeed    int                      `json:"vsc_default_leader_slow_allow_overtake_speed" help:"This sets the speed in km/h under which the leader can be overtaken during Virtual Safety Car or the Rolling Start Formation Lap (e.g. not allowing a parked car to block everyone)."`
	VSCDefaultSlowAndFarAllowOvertakeSpeed    int                      `json:"vsc_default_slow_and_far_allow_overtake_speed" help:"This sets the speed in km/h under which a driver can be overtaken during the Virtual Safety Car or the Rolling Start Formation Lap if he's farther than vsc_defalut_slow_and_far_allow_overtake_distance."`
	VSCDefaultSlowAndFarAllowOvertakeDistance int                      `json:"vsc_default_slow_and_far_allow_overtake_distance" help:"This sets the distance from the previous car over which a driver can be overtaken during the Virtual Safety Car or the Rolling Start Formation Lap if he's slower than vsc_defalut_slow_and_far_allow_overtake_speed."`
	VSCFormationLapFarAllowOvertakeDistance   int                      `json:"vsc_formation_lap_far_allow_overtake_distance" help:"This sets the distance from the previous car over which a driver can be overtaken during the Rolling Start Formation Lap (as during the formation lap cars need to be near to each other)."`
	RaceMassAccidentCrashedPlayersPercentage  int                      `json:"race_mass_accident_crashed_players_percentage" help:"The percentage of players that have to be involved in a collision within the specified mass_accident_crash_time before the plugin triggers the mass accident response (e.g. Virtual Safety Car or Automatic-Restart). 45 = 45%."`
	RaceMassAccidentCrashTime                 int                      `json:"race_mass_accident_crash_time" help:"The time in seconds over which to evaluate race_mass_accident_crashed_players_percentage. 30 = 30s."`
	RaceMassAccidentMinCrashedPlayers         int                      `json:"race_mass_accident_min_crashed_players" help:"The minimum number of players that have to crash for the mass accident response (e.g. to prevent the Virtual Safety Car coming out too often in servers with few players)."`
	RaceMassAccidentResponse                  RaceMassAccidentResponse `json:"race_mass_accident_response" help:"What to do when a mass accident occurs. Use VSC# where # = seconds for the Virtual Safety Car (e.g. VSC60 for 60 seconds) and AR for the automatic race restart. Leave empty to disable."`

	RaceControl                         FormHeading          `ini:"-" json:"-" `
	RaceControlPassword                 string               `json:"race_control_password" help:"The password that Race Directors need to use to judge collisions via the Web Stats Page (needs to be at least 8 characters long or the feature will be disabled)."`
	RaceControlMaxEvents                int                  `json:"race_control_max_events" help:"The maximum number of events that race control should display."`
	RaceControlCollisionSpace           float64              `json:"race_control_collision_space" help:"Defines the maximum distance in meters between the two positions reported by the Assetto Corsa server for the same collision (to account for client-side discrepancies due to lag). Increase if collisions are reported twice."`
	RaceControlCollisionTime            int                  `json:"race_control_collision_time" help:"Defines the maximum number of seconds that the Assetto Corsa server has to provide collision data for both cars. Increase if collisions are reported twice."`
	RaceControlLogOvertakes             formulate.BoolNumber `json:"race_control_log_overtakes" help:"Set this to 1 if you wish the plugin to log overtakes and add them to the Race Control viewer (this might use your memory and clutter the view to show events that are not strictly required for Race Control)."`
	RaceControlCollisionReplayTime      int                  `json:"race_control_collision_replay_time" help:"Defines the number of seconds that the plugin should show for each car in the Race Control Collsion Replay."`
	RaceControlCutReplayTime            int                  `json:"race_control_cut_replay_time" help:"Defines the number of seconds that the plugin should show for each car in the Race Control Cut Replay."`
	RaceControlOvertakeReplayTime       int                  `json:"race_control_overtake_replay_time" help:"Defines the number of seconds that the plugin should show for each car in the Race Control Overtake Replay."`
	RaceControlIncludePlayersNearerThan int                  `json:"race_control_include_players_nearer_than" help:"Cars within this distance (in meters) from the car originating the Race Control Event will be included in the replay. 100 = 100m (e.g. if a car collides with another, all the drivers within 100 meters will be included in the replay)."`

	Database                                FormHeading          `ini:"-" json:"-" show:"open"`
	DatabaseSharingUniqueName               string               `json:"database_sharing_unique_name" show:"open" help:"Database Sharing Unique Name. Set a unique name for this plugin to better identify it in the logs. This is only cosmetic and completely optional."`
	DatabaseSharingLocalGroupPort           int                  `json:"database_sharing_local_group_port" show:"open" help:"Local Database Sharing Port for sharing the database with Kissmyrank instances running on the same machine. Set this to a valid port number If you wish to share the same database across different instances of the Kissmyrank plugin running on the same machine. You must set the port to the same value for all plugins that are members of the same sharing group (e.g. let's say that you have 4 Kissmyrank instances A,B,C,D and you want to use the same Kissmyrank database in couples with A sharing the same db with B and C and D sharing a separate one, you then set database_sharing_group_port config.json on KMR A and KMR B to 4567 and for KMR C and KMR D to 4568)."`
	DatabaseSharingRemoteListenPort         int                  `json:"database_sharing_remote_listen_port" show:"open" help:"Remote Database Sharing Listen Port. Only use this if Local Database Sharing is not possible. Set this to a valid port number if you want to share the Database with a plugin running on a remote machine. Remote sharing might introduce lag so I recommend using Local Database Sharing whenever it's possible."`
	DatabaseSharingRemoteSecretKey          string               `json:"database_sharing_remote_secret_key" show:"open" help:"Remote Database Sharing Secret Key. Only needed for remote sharing. This must be set to the same for all the hosts using Remote Database Sharing."`
	DatabaseSharingRemoteListenAddress      string               `json:"database_remote_listen_address" show:"open" help:"Remote Database Sharing Listen Address (change this only if you want to listen only on a specific interface, not recommended)."`
	DatabaseSharingRemoteConnectToAddresses NewLineSeparatedList `json:"database_sharing_remote_connect_to_addresses" elem:"textarea" show:"open" help:"One per line. Remote Database Sharing Connect to Addresses. The addresses of the remote plugins in the usual host:port format (where host is the IP/Address of the server that runs the plugin and port is the database_sharing_remote_listen_port that you set for that plugin). If you wish plugin A to share the database with the plugin B, you need to add hostA:portA to plugin B config.json and hostB:portB to plugin A config.json. For each plugin add all the ones that it should communicate with (excluding of course self or you'll start a loop). Don't connect two plugins with both local and remote sharing."`
	DatabaseSharingRelayForNames            NewLineSeparatedList `json:"database_sharing_relay_for_names" show:"open" elem:"textarea" help:"One per line. Remote Database Sharing Relay For Names. Warning: this key is not required for standard Database Sharing, it's only required if you have lonely plugins (e.g. plugins connected to one instance and not to the others). Let's say that you want to share the database between 3 plugins (A,B and C) and that you set database_sharing_unique_name to plugin_a, plugin_b, plugin_c respectively. Let's say that you can only connect A<->B and B<->C but you can't connect A with C directly. After properly setting all the other keys for (A<->B and B<->C), you then edit plugin_b config.json and add plugin_a and plugin_c to database_sharing_relay_for_names key. This will tell plugin B to relay information to the other plugin allowing you to do A<->B<->C. Keep in mind that in this case if you take down B, you have to also take down A and C or the synchronization might not fully apply. This feature works for both local and remote connections as long as you set the unique name. Be careful not to create double links between plugins or you might get double updates."`

	Miscellaneous                               FormHeading          `ini:"-" json:"-"`
	SessionHistoryLength                        int                  `json:"session_history_length" help:"Session history length. How many sessions we should keep in the memory. Setting this to a higher value will use more memory."`
	MemoryMonitorEnabled                        formulate.BoolNumber `json:"memory_monitor_enabled" show:"open" help:"Whether to enable the memory monitor feature (on Linux it requires libc6 2.14+). Enable this if you wish the plugin to monitor the memory usage (good to troubleshoot problems that might be related to the memory allocation)."`
	SpeedUnitFormat                             string               `json:"speed_unit_format" help:"The speed unit to be used when formatting data for public view. Allowed values are kmh and mph. Please note that this is only cosmetic for the end user and that you still need to use km/h for all the config purposes."`
	ReservedSlotsGUIDList                       NewLineSeparatedList `json:"reserved_slots_guid_list" elem:"textarea" help:"One per line. A list of the GUIDs for the drivers that have a reserved slot on the server (requires the Kissmyrank Multiplayer Launcher Mod). If the server is full and one of these GUIDs attempts to join, a player will be kicked to make room."`
	ReservedSlotsAccessKey                      string               `json:"reserved_slots_access_key" help:"The key that VIP players need to use in the Kissmyrank Multiplayer Launcher Mod to access their reserved slot. Drivers with a reserved slot need to type this key in the Kissmyrank Multiplayer Launcher Mod input box (near the red question mark button on top of the server list) before they can access their reserved slot. If this configuration entry is not set the reserved slots feature will be disabled altogether."`
	ReservedSlotsBootPlayersAtRace              int                  `json:"reserved_slots_boot_players_at_race" help:"Whether to disable the Reserved Slots Player Booting feature during the race. Slots are freed per car starting from the bottom of the grid (so on a full grid it will most likely boot players that late joined or that are several laps down). You can set this to 1 if you wish to disable the reserved slot during the race session altogether. In this case, VIP players will have to wait for a slot to free up or for the next session."`
	RankSortByWinStats                          formulate.BoolNumber `json:"rank_sort_by_win_stats" help:"Sort rank by winning stats instead of money. If you change this you have to reset the database."`
	ACAppLinkUDPPort                            int                  `json:"ac_app_link_udp_port" show:"open" help:"The UDP port to be used to relay Kissmyrank events to compatible Assetto Corsa Game Apps (set to 0 to disable this feature). AC apps might connect to this port. Full documentation is available under /applink/doc. Demo app available under /applink/demo/."`
	CustomChatDriverWelcomeMessages             NewLineSeparatedList `json:"custom_chat_driver_welcome_messages" elem:"textarea" help:"The custom chat messages that you wish to deliver when a driver connects. One per line."`
	ChatDriverWelcomeMessageShowRaceControlLink formulate.BoolNumber `json:"chat_driver_welcome_message_show_race_control_link" help:"Whether to show the Race Control link when a driver joins."`
	ACChatAdminPassword                         string               `json:"ac_chat_admin_password" help:"The admin password to be used to login from the Assetto Corsa chat (type '/kmr login password' in the game chat to login)."`
	ACChatAdminGUIDList                         NewLineSeparatedList `json:"ac_chat_admin_guid_list" elem:"textarea" help:"One GUID per line. A list of the GUIDs for the drivers that can send Kissmyrank Admin Commands via the Assetto Corsa Chat (type '/kmr login password' in the chat to login and '/kmr command' to launch one of the supported commands)."`

	PenaltyInfractionMap PenaltyInfractionMap `json:"penalty_infraction_map"`
	PenaltyCosts         PenaltyCostMap       `json:"penalty_cost_map"`
	PenaltyActions       PenaltyActionMap     `json:"penalty_action_map"`

	// Hidden values below here...
	// ACServer
	ACServerIP                          string `json:"ac_server_ip" show:"-" help:"The IP of your Assetto Corsa Server relative to the plugin. This is the IP that the plugin will use to contact the Assetto Corsa Server."`
	ACServerHTTPPort                    int    `json:"ac_server_http_port" show:"-" help:"The HTTP Port of the Assetto Corsa server. This should match the Assetto Corsa server_cfg.ini HTTP_PORT setting (required for ping control)."`
	ACServerPluginLocalPort             int    `json:"ac_server_plugin_local_port" show:"-" help:"The plugin port of the server as set in the server_cfg.ini (UDP_PLUGIN_LOCAL_PORT)."`
	ACServerPluginAddressPort           int    `json:"ac_server_plugin_address_port" show:"-" help:"The port that the plugin will use (the portion after ':' in the UDP_PLUGIN_ADDRESS server_cfg.ini entry, if you set UDP_PLUGIN_ADDRESS=127.0.0.1:12000, set this to 12000)."`
	AfterACServerStartRunPath           string `json:"after_ac_server_start_run_path" show:"-" help:"Path to a shell script or to a program to run after the plugin launches the Assetto Corsa Server (e.g track rotation). This can be used to automate the restart of other plugins or if you wish to perform some other task when the Assetto Corsa Server starts. This setting will only apply if the Kissmyrank Track Rotation Feature is enabled."`
	BeforeACServerStartRunPath          string `json:"before_ac_server_start_run_path" show:"-" help:"Path to a shell script or program to run before the plugin launches the Assetto Corsa Server (e.g track rotation). This can be used to automate tasks before the Assetto Corsa Server starts. Please note that the Assetto Corsa Server will not be launched until this program terminates, so please use a shell script to launch programs that are required to run at the same time as the server (or use after_ac_server_start_run_path which is run after the server launch without further waiting). This setting will only apply if the Kissmyrank Track Rotation Feature is enabled."`
	ACServerRestartIfInactiveForMinutes int    `json:"ac_server_restart_if_inactive_for_minutes" show:"-" help:"Restarts the Assetto Corsa server if no activity is recorded for more than the specified amount of minutes (minimum 30 minutes, this setting only works if the track rotation is enabled). Please note that the new session packet is considered as valid server activity and will reset the counter."`
	ACServerResultsBasePath             string `json:"ac_server_results_base_path" show:"-" help:"This is the parent of the Assetto Corsa results folder (e.g. this entry should be normally set to the Assetto Corsa Server Root Folder path). If you already set ac_server_bin_path, you can leave this empty as the plugin will detect the base path automatically. Use this only if you wish to force a specific base path for the results (e.g. if you don't use the track rotation but you still want to collect the results for Web View)."`
	ACServerConfigIniPath               string `json:"ac_server_cfg_ini_path" show:"-" help:"Path of the Assetto Corsa server_cfg.ini file to be used for track rotation. This must be the actual server_cfg.ini that acServer uses. The plugin will update this file and restart the server to rotate the track."`
	ACServerBinaryPath                  string `json:"ac_server_bin_path" show:"-" help:"Absolute Path of the Assetto Corsa Server executable to be used for track rotation (e.g. Windows c:/steam/acserver/acServer.exe, Linux /home/steam/acserver/acServer). The plugin will run this file to launch the Assetto Corsa server."`
	ACServerBinaryArguments             string `json:"ac_server_bin_args" show:"-" help:"[Not Recommended!] Assetto Corsa Binary Launch Arguments for multiple servers like ['-c path_to_server_cfg.ini', '-e path_to_entry_list.ini'] (this might not work at all on some operating systems, for multiple servers please make copies of the server folder and run each instance separately)."`
	ACServerLogPath                     string `json:"ac_server_log_path" show:"-" help:"The path where you wish to save the Assetto Corsa Server logs."`

	MaxPlayers int `json:"max_players" show:"-" help:"Number of server slots."`

	// Track Rotation is not supported with Server Manager.
	TrackList               []interface{} `json:"track_list" help:"" show:"-"`
	TrackRotationMaxPlayers int           `json:"track_rotation_max_players" help:"" show:"-"`

	TrackRotationVoteMinPercent int `json:"track_rotation_vote_min_percent" show:"-" help:"The minimum amount of votes required to trigger a track change (in %). 60 = 60% of the drivers online must type kmr vote_track to initiate the track change to the most voted track. Set to 0 to disable track voting altogether."`
	TrackRotationVoteMinVotes   int `json:"track_rotation_vote_min_votes" show:"-" help:"The minimum total amount of votes (for or against) required to initiate the track change. 4 = minimum 4 votes are required to change the track. Use this if you wish to prevent lonely players from changing the track."`
}

func (kmr *KissMyRankConfig) Write() error {
	f, err := os.Create(KissMyRankConfigPath())

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	return enc.Encode(kmr)
}

type RaceMassAccidentResponse struct {
	FirstLap  string
	OtherLaps string
}

type NewLineSeparatedList string

func (c NewLineSeparatedList) MarshalJSON() ([]byte, error) {
	var out []string

	for _, val := range strings.Split(string(c), "\n") {
		out = append(out, strings.TrimSpace(val))
	}

	return json.Marshal(out)
}

func (c *NewLineSeparatedList) UnmarshalJSON(data []byte) error {
	var out []string

	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}

	*c = NewLineSeparatedList(strings.Join(out, "\n"))

	return nil
}

type PenaltyCostSessions struct {
	Practice float64 `json:"practice" help:"The money penalties (in thousands) that you would like to give for any given situation."`
	Qualify  float64 `json:"qualify" help:"The money penalties (in thousands) that you would like to give for any given situation."`
	Race     float64 `json:"race" help:"The money penalties (in thousands) that you would like to give for any given situation."`
	Other    float64 `json:"other" help:"The money penalties (in thousands) that you would like to give for any given situation."`
}

type PenaltyCostMap struct {
	HotLapProtection                       PenaltyCostSessions `json:"hotlap_protection"`
	HotLappingCarCollision                 PenaltyCostSessions `json:"hotlapping_car_collision"`
	LappingProtection                      PenaltyCostSessions `json:"lapping_protection"`
	LappingCarCollision                    PenaltyCostSessions `json:"lapping_car_collision"`
	ReverseGear                            PenaltyCostSessions `json:"reverse_gear"`
	TrackBoundaryCut                       PenaltyCostSessions `json:"track_boundary_cut"`
	TrackRejoinMaxSpeed                    PenaltyCostSessions `json:"track_rejoin_max_speed"`
	MaxInfractions                         PenaltyCostSessions `json:"max_infractions"`
	MaxCollisions                          PenaltyCostSessions `json:"max_collisions"`
	FirstBlood                             PenaltyCostSessions `json:"first_blood"`
	PitLaneSpeeding                        PenaltyCostSessions `json:"pit_lane_speeding"`
	PitExitLineCrossing                    PenaltyCostSessions `json:"pit_exit_line_crossing"`
	CutLineYourCustomCutLine               PenaltyCostSessions `json:"cut_line_your_custom_cut_line"`
	AntiCheatMaxClockDeltaConsecutiveHits  PenaltyCostSessions `json:"anticheat_max_clock_delta_consecutive_hits"`
	SpeedingUnderVirtualSafetyCar          PenaltyCostSessions `json:"speeding_under_vsc"`
	SlowingUnderVirtualSafetyCar           PenaltyCostSessions `json:"slowing_under_vsc"`
	OvertakingUnderVirtualSafety           PenaltyCostSessions `json:"overtaking_under_vsc"`
	ImprovingQualifyLapTimeWithInfractions PenaltyCostSessions `json:"improving_qualify_laptime_with_infractions"`
	ParkingNearTrack                       PenaltyCostSessions `json:"parking_near_track"`
}

type PenaltyActionSessions struct {
	Practice string `json:"practice" help:"DT0 for a drive through before the end of the lap. DT1 for a drive through before the end of the following lap and so on. DT given during qualify and practice will have to be cleared during the following race. K to kick immediately. TB30 to issue a temporary ban for 30 minutes. TB60 to issue a temporary ban for 60 minutes."`
	Qualify  string `json:"qualify" help:"DT0 for a drive through before the end of the lap. DT1 for a drive through before the end of the following lap and so on. DT given during qualify and practice will have to be cleared during the following race. K to kick immediately. TB30 to issue a temporary ban for 30 minutes. TB60 to issue a temporary ban for 60 minutes."`
	Race     string `json:"race" help:"DT0 for a drive through before the end of the lap. DT1 for a drive through before the end of the following lap and so on. DT given during qualify and practice will have to be cleared during the following race. K to kick immediately. TB30 to issue a temporary ban for 30 minutes. TB60 to issue a temporary ban for 60 minutes."`
}

type PenaltyActionMap struct {
	HotLapProtection                       PenaltyActionSessions `json:"hotlap_protection"`
	HotLappingCarCollision                 PenaltyActionSessions `json:"hotlapping_car_collision"`
	LappingProtection                      PenaltyActionSessions `json:"lapping_protection"`
	LappingCarCollision                    PenaltyActionSessions `json:"lapping_car_collision"`
	ReverseGear                            PenaltyActionSessions `json:"reverse_gear"`
	TrackBoundaryCut                       PenaltyActionSessions `json:"track_boundary_cut"`
	TrackRejoinMaxSpeed                    PenaltyActionSessions `json:"track_rejoin_max_speed"`
	MaxInfractions                         PenaltyActionSessions `json:"max_infractions"`
	MaxCollisions                          PenaltyActionSessions `json:"max_collisions"`
	FirstBlood                             PenaltyActionSessions `json:"first_blood"`
	PitLaneSpeeding                        PenaltyActionSessions `json:"pit_lane_speeding"`
	PitExitLineCrossing                    PenaltyActionSessions `json:"pit_exit_line_crossing"`
	CutLineYourCustomCutLine               PenaltyActionSessions `json:"cut_line_your_custom_cut_line"`
	AntiCheatMaxClockDeltaConsecutiveHits  PenaltyActionSessions `json:"anticheat_max_clock_delta_consecutive_hits"`
	SpeedingUnderVirtualSafetyCar          PenaltyActionSessions `json:"speeding_under_vsc"`
	SlowingUnderVirtualSafetyCar           PenaltyActionSessions `json:"slowing_under_vsc"`
	OvertakingUnderVirtualSafety           PenaltyActionSessions `json:"overtaking_under_vsc"`
	ImprovingQualifyLapTimeWithInfractions PenaltyActionSessions `json:"improving_qualify_laptime_with_infractions"`
	ParkingNearTrack                       PenaltyActionSessions `json:"parking_near_track"`
}

type PenaltyInfractionSessions struct {
	Practice float64 `json:"practice" help:"How many infractions you would like to add to the infraction counter for each penalty during each session."`
	Qualify  float64 `json:"qualify" help:"How many infractions you would like to add to the infraction counter for each penalty during each session."`
	Race     float64 `json:"race" help:"How many infractions you would like to add to the infraction counter for each penalty during each session."`
}

type PenaltyInfractionMap struct {
	HotLapProtection                       PenaltyInfractionSessions `json:"hotlap_protection"`
	HotLappingCarCollision                 PenaltyInfractionSessions `json:"hotlapping_car_collision"`
	LappingProtection                      PenaltyInfractionSessions `json:"lapping_protection"`
	LappingCarCollision                    PenaltyInfractionSessions `json:"lapping_car_collision"`
	ReverseGear                            PenaltyInfractionSessions `json:"reverse_gear"`
	TrackBoundaryCut                       PenaltyInfractionSessions `json:"track_boundary_cut"`
	TrackRejoinMaxSpeed                    PenaltyInfractionSessions `json:"track_rejoin_max_speed"`
	MaxInfractions                         PenaltyInfractionSessions `json:"max_infractions"`
	MaxCollisions                          PenaltyInfractionSessions `json:"max_collisions"`
	FirstBlood                             PenaltyInfractionSessions `json:"first_blood"`
	PitLaneSpeeding                        PenaltyInfractionSessions `json:"pit_lane_speeding"`
	PitExitLineCrossing                    PenaltyInfractionSessions `json:"pit_exit_line_crossing"`
	CutLineYourCustomCutLine               PenaltyInfractionSessions `json:"cut_line_your_custom_cut_line"`
	AntiCheatMaxClockDeltaConsecutiveHits  PenaltyInfractionSessions `json:"anticheat_max_clock_delta_consecutive_hits"`
	SpeedingUnderVirtualSafetyCar          PenaltyInfractionSessions `json:"speeding_under_vsc"`
	SlowingUnderVirtualSafetyCar           PenaltyInfractionSessions `json:"slowing_under_vsc"`
	OvertakingUnderVirtualSafety           PenaltyInfractionSessions `json:"overtaking_under_vsc"`
	ImprovingQualifyLapTimeWithInfractions PenaltyInfractionSessions `json:"improving_qualify_laptime_with_infractions"`
	ParkingNearTrack                       PenaltyInfractionSessions `json:"parking_near_track"`
}
