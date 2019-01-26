package servermanager

// ConfigIniDefault is the default server config (ish) as supplied via the assetto corsa server.
var ConfigIniDefault = ServerConfig{
	Server: ServerSetupConfig{
		Name:                      "Server Managed Assetto Corsa",
		Cars:                      "bmw_m3_e30",
		TrackConfig:               "",
		Track:                     "magione",
		SunAngle:                  48,
		Password:                  "",
		AdminPassword:             "",
		UDPPort:                   9600,
		TCPPort:                   9600,
		HTTPPort:                  8081,
		PickupModeEnabled:         1,
		LoopMode:                  1,
		SleepTime:                 1,
		ClientSendIntervalInHertz: 18,
		SendBufferSize:            0,
		ReceiveBufferSize:         0,
		RaceOverTime:              180,
		KickQuorum:                85,
		VotingQuorum:              80,
		VoteDuration:              20,
		BlacklistMode:             1,
		FuelRate:                  100,
		DamageMultiplier:          100,
		TyreWearRate:              100,
		AllowedTyresOut:           2,
		ABSAllowed:                1,
		TractionControlAllowed:    1,
		StabilityControlAllowed:   0,
		AutoClutchAllowed:         0,
		TyreBlanketsAllowed:       1,
		ForceVirtualMirror:        1,
		RegisterToLobby:           1,
		MaxClients:                18,
		UDPPluginLocalPort:        0,
		UDPPluginAddress:          "",
		AuthPluginAddress:         "",
		LegalTyres:                "SV",
	},

	Sessions: map[string]SessionConfig{
		"PRACTICE": {
			Name:   "Practice",
			Time:   10,
			IsOpen: 1,
		},
		"QUALIFY": {
			Name:   "Qualify",
			Time:   10,
			IsOpen: 1,
		},
		"RACE": {
			Name:     "Race",
			IsOpen:   1,
			WaitTime: 60,
			Laps:     5,
		},
	},

	DynamicTrack: DynamicTrackConfig{
		SessionStart:    89,
		Randomness:      3,
		SessionTransfer: 80,
		LapGain:         50,
	},

	Weather: map[string]WeatherConfig{
		"WEATHER_0": {
			Graphics:               "3_clear",
			BaseTemperatureAmbient: 18,
			BaseTemperatureRoad:    6,
			VariationAmbient:       1,
			VariationRoad:          1,
		},
		"WEATHER_1": {
			Graphics:               "7_heavy_clouds",
			BaseTemperatureAmbient: 15,
			BaseTemperatureRoad:    -1,
			VariationAmbient:       1,
			VariationRoad:          1,
		},
	},
}
