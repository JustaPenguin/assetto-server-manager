package servermanager

// ConfigIniDefault is the default server config (ish) as supplied via the assetto corsa server.
var ConfigIniDefault = ServerConfig{
	GlobalServerConfig: GlobalServerConfig{
		Name:                      "Server Managed Assetto Corsa",
		Password:                  "",
		AdminPassword:             "",
		UDPPort:                   9600,
		TCPPort:                   9600,
		HTTPPort:                  8081,
		ClientSendIntervalInHertz: 18,
		SendBufferSize:            0,
		ReceiveBufferSize:         0,
		KickQuorum:                85,
		VotingQuorum:              80,
		VoteDuration:              20,
		BlacklistMode:             1,
		RegisterToLobby:           1,
		UDPPluginLocalPort:        0,
		UDPPluginAddress:          "",
		AuthPluginAddress:         "",
		NumberOfThreads:           2,
		ResultScreenTime:          90,
	},

	CurrentRaceConfig: CurrentRaceConfig{
		Cars:                      "lotus_evora_gtc",
		Track:                     "ks_silverstone",
		TrackLayout:               "gp",
		SunAngle:                  48,
		PickupModeEnabled:         1,
		LoopMode:                  1,
		SleepTime:                 1,
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
		ForceVirtualMirror:        0,
		LegalTyres:                "H;M;S",
		LockedEntryList:           0,
		RacePitWindowStart:        0,
		RacePitWindowEnd:          0,
		ReversedGridRacePositions: 0,
		TimeOfDayMultiplier:       0,
		QualifyMaxWaitPercentage:  200,
		RaceGasPenaltyDisabled:    1,
		MaxBallastKilograms:       50,
		WindBaseSpeedMin:          3,
		WindBaseSpeedMax:          15,
		MaxClients:                18,
		WindBaseDirection:         30,
		WindVariationDirection:    15,
		StartRule:                 2, // drive-thru
		RaceExtraLap:              0,

		Sessions: map[SessionType]SessionConfig{
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
				Graphics:               "3_clear",
				BaseTemperatureAmbient: 26,
				BaseTemperatureRoad:    11,
				VariationAmbient:       1,
				VariationRoad:          1,
			},
		},
	},
}
