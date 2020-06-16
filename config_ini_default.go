package servermanager

// ConfigIniDefault is the default server config (ish) as supplied via the assetto corsa server.
func ConfigIniDefault() ServerConfig {
	return ServerConfig{
		GlobalServerConfig: GlobalServerConfig{
			Name:                              "Assetto Corsa Server",
			Password:                          "",
			AdminPassword:                     "",
			UDPPort:                           9600,
			TCPPort:                           9600,
			HTTPPort:                          8081,
			ClientSendIntervalInHertz:         18,
			SendBufferSize:                    0,
			ReceiveBufferSize:                 0,
			KickQuorum:                        85,
			VotingQuorum:                      80,
			VoteDuration:                      20,
			BlacklistMode:                     1,
			RegisterToLobby:                   1,
			UDPPluginLocalPort:                0,
			PreventWebCrawlers:                0,
			UDPPluginAddress:                  "",
			AuthPluginAddress:                 "",
			NumberOfThreads:                   2,
			ShowRaceNameInServerLobby:         1,
			ServerNameTemplate:                defaultServerNameTemplate,
			ShowContentManagerJoinLink:        1,
			SleepTime:                         1,
			RestartEventOnServerManagerLaunch: 1,
			ContentManagerWelcomeMessage:      defaultContentManagerDescription,
			ShowEventDetailsPopup:             true,
		},

		CurrentRaceConfig: CurrentRaceConfig{
			Cars:                      "lotus_evora_gtc",
			Track:                     "ks_silverstone",
			TrackLayout:               "gp",
			SunAngle:                  48,
			PickupModeEnabled:         1,
			LoopMode:                  1,
			RaceOverTime:              180,
			FuelRate:                  100,
			DamageMultiplier:          0,
			TyreWearRate:              100,
			AllowedTyresOut:           3,
			ABSAllowed:                1,
			TractionControlAllowed:    1,
			StabilityControlAllowed:   0,
			AutoClutchAllowed:         0,
			TyreBlanketsAllowed:       1,
			ForceVirtualMirror:        1,
			LegalTyres:                "H;M;S",
			LockedEntryList:           0,
			RacePitWindowStart:        0,
			RacePitWindowEnd:          0,
			ReversedGridRacePositions: 0,
			TimeOfDayMultiplier:       0,
			QualifyMaxWaitPercentage:  200,
			RaceGasPenaltyDisabled:    1,
			MaxBallastKilograms:       50,
			MaxClients:                18,
			StartRule:                 2, // drive-thru
			RaceExtraLap:              0,
			MaxContactsPerKilometer:   -1,
			ResultScreenTime:          90,

			DriverSwapEnabled:               0,
			DriverSwapMinTime:               120,
			DriverSwapDisqualifyTime:        30,
			DriverSwapPenaltyTime:           0,
			DriverSwapMinimumNumberOfSwaps:  0,
			DriverSwapNotEnoughSwapsPenalty: 0,

			Sessions: map[SessionType]*SessionConfig{
				SessionTypePractice: {
					Name:   "Practice",
					Time:   10,
					IsOpen: 1,
				},
				SessionTypeQualifying: {
					Name:   "Qualify",
					Time:   10,
					IsOpen: 1,
				},
				SessionTypeRace: {
					Name:     "Race",
					IsOpen:   1,
					WaitTime: 60,
					Laps:     5,
				},
			},

			DynamicTrack: DynamicTrackConfig{
				SessionStart:    100,
				Randomness:      0,
				SessionTransfer: 100,
				LapGain:         10,
			},

			Weather: map[string]*WeatherConfig{
				"WEATHER_0": {
					Graphics:                    "3_clear",
					ChampionshipPracticeWeather: weatherAny,
					BaseTemperatureAmbient:      26,
					BaseTemperatureRoad:         11,
					VariationAmbient:            1,
					VariationRoad:               1,
					WindBaseSpeedMin:            3,
					WindBaseSpeedMax:            15,
					WindBaseDirection:           30,
					WindVariationDirection:      15,
				},
			},
		},
	}
}

const defaultServerNameTemplate = "{{ .ServerName }} - {{ .EventName }}"
const defaultContentManagerDescription = "{{ .Track }} {{ with .TrackLayout }}({{ . }}){{ end }} " +
	"- an event hosted by {{ .ServerName }}<br><br>{{ .EventDescription }}<br>{{ .ChampionshipPoints }}<br>{{ .CarDownloads }}<br>{{ .TrackDownload }}"
