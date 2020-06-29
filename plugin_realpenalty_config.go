package servermanager

import (
	"path/filepath"

	"github.com/cj123/ini"
)

const (
	realPenaltyAppConfigIniPath  = "settings.ini"
	realPenaltySettingsIniPath   = "penalty_settings.ini"
	realPenaltyACSettingsIniPath = "ac_settings.ini"
)

func DefaultRealPenaltyConfig() *RealPenaltyConfig {
	return &RealPenaltyConfig{
		RealPenaltyAppConfig:  DefaultRealPenaltyAppConfig(),
		RealPenaltySettings:   DefaultRealPenaltySettings(),
		RealPenaltyACSettings: DefaultRealPenaltyACSettings(),
	}
}

// each of these is a separate ini file
type RealPenaltyConfig struct {
	RealPenaltyAppConfig  RealPenaltyAppConfig  `show:"contents"`
	RealPenaltySettings   RealPenaltySettings   `show:"contents"`
	RealPenaltyACSettings RealPenaltyACSettings `show:"contents"`
}

func (rpc *RealPenaltyConfig) Write() error {
	if err := rpc.RealPenaltyAppConfig.write(); err != nil {
		return err
	}
	if err := rpc.RealPenaltySettings.write(); err != nil {
		return err
	}
	return rpc.RealPenaltyACSettings.write()
}

func DefaultRealPenaltyAppConfig() RealPenaltyAppConfig {
	return RealPenaltyAppConfig{
		General: RealPenaltyConfigGeneral{
			EnableRealPenalty: false,
			UDPPort:           0,
			UDPResponse:       "",
			AppTCPPort:        53000,
		},
		PluginsRelay: RealPenaltyPluginsRelay{
			OtherUDPPlugin: "",
			UDPPort:        "",
		},
	}
}

type RealPenaltyAppConfig struct {
	General      RealPenaltyConfigGeneral `ini:"General"`
	PluginsRelay RealPenaltyPluginsRelay  `ini:"Plugins_Relay" show:"open"`
}

func (rpa *RealPenaltyAppConfig) write() error {
	f := ini.NewFile([]ini.DataSource{nil}, ini.LoadOptions{
		IgnoreInlineComment: true,
	})

	// making and throwing away a default section due to the utter insanity of ini or assetto. i don't know which.
	_, err := f.NewSection("DEFAULT")

	if err != nil {
		return err
	}

	general, err := f.NewSection("General")

	if err != nil {
		return err
	}

	err = general.ReflectFrom(&rpa.General)

	if err != nil {
		return err
	}

	pluginsRelay, err := f.NewSection("Plugins_Relay")

	if err != nil {
		return err
	}

	err = pluginsRelay.ReflectFrom(&rpa.PluginsRelay)

	if err != nil {
		return err
	}

	return f.SaveTo(filepath.Join(RealPenaltyFolderPath(), realPenaltyAppConfigIniPath))
}

type RealPenaltyConfigGeneral struct {
	EnableRealPenalty bool `ini:"-" help:"Turn Real Penalty on or off"`

	ACServerPath    string `ini:"AC_SERVER_PATH" show:"-" help:"Path to the AC Server folder"`
	ACCFGFile       string `ini:"AC_CFG_FILE" show:"-" help:"Path and file of the cfg file of the AC server. If not found the plugin sets it automatic from AC_SERVER_PATH"`
	ACTracksFolder  string `ini:"AC_TRACKS_FOLDER" show:"-" help:"Path of the AC server tracks folder. If not found the plugin sets it automatic from AC_SERVER_PATH"`
	ACWeatherFolder string `ini:"AC_WEATHER_FOLDER" show:"-" help:"Path of the AC server weather folder. If not found the plugin sets it automatic from AC_SERVER_PATH"`

	UDPPort     int    `ini:"UDP_PORT" show:"-" help:"Listening UDP port - Set the same port (without IP) of cfg of the server, UDP_PLUGIN_ADDRESS "`
	UDPResponse string `ini:"UDP_RESPONSE" show:"-" help:"Destination IP and UDP port for response - Set the same port in the cfg of the AC server, UDP_PLUGIN_LOCAL_PORT"`
	AppTCPPort  int    `ini:"APP_TCP_PORT" show:"open" help:"Listening UDP port from AC app (to open in firewall/router). Must be one of 53000, 53001, 53002 to 53020, or the port of cfg AC server, HTTP_PORT + 27. The app will try all these ports on the ac server's ip address (until the right connection is found)"`

	AppFile      string `ini:"APP_FILE" show:"-" help:"Path and file names of the app from the plugin package"`
	ImagesFile   string `ini:"IMAGES_FILE" show:"-" help:"Path and file names of the images from the plugin package"`
	SoundsFile   string `ini:"SOUNDS_FILE" show:"-" help:"Path and file names of the sounds from the plugin package"`
	TracksFolder string `ini:"TRACKS_FOLDER" show:"-" help:"Folder of the additional .ini track files"`
}

type RealPenaltyPluginsRelay struct {
	OtherUDPPlugin string `ini:"OTHER_UDP_PLUGIN" show:"open" help:"List of all other UDP plugins connected to AC Server (format: IP:Ports - separated by a semicolon)"`
	UDPPort        string `ini:"UDP_PORT" show:"open" help:"List of listening UDP ports for all other plugins connected (separated by a semicolon)"`
}

type RealPenaltySettings struct {
	General   RealPenaltySettingsGeneral   `ini:"General"`
	Cutting   RealPenaltySettingsCutting   `ini:"Cutting"`
	Speeding  RealPenaltySettingsSpeeding  `ini:"Speeding"`
	Crossing  RealPenaltySettingsCrossing  `ini:"Crossing"`
	JumpStart RealPenaltySettingsJumpStart `ini:"Jump_Start"`
	DRS       RealPenaltySettingsDRS       `ini:"Drs"`
	BlueFlag  RealPenaltySettingsBlueFlag  `ini:"Blue_Flag"`
}

func (rps *RealPenaltySettings) write() error {
	f := ini.NewFile([]ini.DataSource{nil}, ini.LoadOptions{
		IgnoreInlineComment: true,
	})

	// making and throwing away a default section due to the utter insanity of ini or assetto. i don't know which.
	_, err := f.NewSection("DEFAULT")

	if err != nil {
		return err
	}

	general, err := f.NewSection("General")

	if err != nil {
		return err
	}

	err = general.ReflectFrom(&rps.General)

	if err != nil {
		return err
	}

	cutting, err := f.NewSection("Cutting")

	if err != nil {
		return err
	}

	err = cutting.ReflectFrom(&rps.Cutting)

	if err != nil {
		return err
	}

	speeding, err := f.NewSection("Speeding")

	if err != nil {
		return err
	}

	err = speeding.ReflectFrom(&rps.Speeding)

	if err != nil {
		return err
	}

	crossing, err := f.NewSection("Crossing")

	if err != nil {
		return err
	}

	err = crossing.ReflectFrom(&rps.Crossing)

	if err != nil {
		return err
	}

	jumpStart, err := f.NewSection("Jump_Start")

	if err != nil {
		return err
	}

	err = jumpStart.ReflectFrom(&rps.JumpStart)

	if err != nil {
		return err
	}

	drs, err := f.NewSection("DRS")

	if err != nil {
		return err
	}

	err = drs.ReflectFrom(&rps.DRS)

	if err != nil {
		return err
	}

	blueFlag, err := f.NewSection("Blue_Flag")

	if err != nil {
		return err
	}

	err = blueFlag.ReflectFrom(&rps.BlueFlag)

	if err != nil {
		return err
	}

	return f.SaveTo(filepath.Join(RealPenaltyFolderPath(), realPenaltySettingsIniPath))
}

func DefaultRealPenaltySettings() RealPenaltySettings {
	return RealPenaltySettings{
		General: RealPenaltySettingsGeneral{
			EnableCuttingPenalties:  true,
			EnableSpeedingPenalties: true,
			EnableCrossingPenalties: true,
			EnableDRSPenalties:      true,
			LapsToTakePenalty:       2,
			PenaltySeconds:          20,
			LastTimeWithoutPenalty:  360,
			LastLapsWithoutPenalty:  1,
		},
		Cutting: RealPenaltySettingsCutting{
			EnabledDuringSafetyCar:    true,
			TotalCutWarnings:          3,
			EnableTyresDirtLevel:      false,
			WheelsOut:                 3,
			MinSpeed:                  40,
			SecondsBetweenCuts:        3,
			MaxCutTime:                3,
			MinSlowDownRatio:          0.9,
			QualifySlowDownSpeed:      40,
			QualifyMaxSectorOutSpeed:  150,
			QualifySlowDownSpeedRatio: 0.99,
			PostCuttingTime:           1,
			PenaltyType:               "dt",
		},
		Speeding: RealPenaltySettingsSpeeding{
			PitLaneSpeed:       82,
			PenaltyType0:       "dt",
			SpeedLimitPenalty0: 100,
			PenaltyType1:       "sg10",
			SpeedLimitPenalty1: 200,
			PenaltyType2:       "dsq",
			SpeedLimitPenalty2: 9999,
		},
		Crossing: RealPenaltySettingsCrossing{
			PenaltyType: "dt",
		},
		JumpStart: RealPenaltySettingsJumpStart{
			PenaltyType0:       "dt",
			SpeedLimitPenalty0: 50,
			PenaltyType1:       "sg10",
			SpeedLimitPenalty1: 200,
			PenaltyType2:       "dsq",
			SpeedLimitPenalty2: 9999,
		},
		DRS: RealPenaltySettingsDRS{
			PenaltyType:      "dt",
			Gap:              1.0,
			EnabledAfterLaps: 0,
			MinSpeed:         50,
			BonusTime:        0.8,
			MaxIllegalUses:   2,
		},
		BlueFlag: RealPenaltySettingsBlueFlag{
			QualifyTimeThreshold: 2.5,
			RaceTimeThreshold:    2.5,
		},
	}
}

type RealPenaltySettingsGeneral struct {
	EnableCuttingPenalties  bool `ini:"ENABLE_CUTTING_PENALTIES" help:"Set to true to enable cuttings penalties; false to issue warnings only"`
	EnableSpeedingPenalties bool `ini:"ENABLE_SPEEDING_PENALTIES" help:"Set to true to enable pit lane speeding penalties"`
	EnableCrossingPenalties bool `ini:"ENABLE_CROSSING_PENALTIES" help:"Set to true to enable exit pit lane crossing line penalty"`
	EnableDRSPenalties      bool `ini:"ENABLE_DRS_PENALTIES" help:"Set to true to enable DRS penalty (only for car with DRS)"`

	LapsToTakePenalty      int `ini:"LAPS_TO_TAKE_PENALTY" help:"How many laps a driver has to take a penalty in. If LAPS_TO_TAKE_PENALTY > 2 --> Jump start is always 2 for AC limit!"`
	PenaltySeconds         int `ini:"PENALTY_SECONDS" help:"Number of seconds to add manually to the final race result for all penalties not taken during the race"`
	LastTimeWithoutPenalty int `ini:"LAST_TIME_WITHOUT_PENALTY" help:"Only for Time Race - Seconds at and of race to end race time (remember the optional +1 lap is extra) without mandatory penalty. All penalties not taken ----> Penalty seconds to add to the final race time (in chat and log). Default 360 (6 minutes). IMPORTANT! Adjust this value in according to the track, the cars and the additional lap (ca. 3/5 laps but in according to LAPS_TO_TAKE_PENALTY)"`
	LastLapsWithoutPenalty int `ini:"LAST_LAPS_WITHOUT_PENALTY" help:"If a divers gets a penalty in the last N laps (N = LAPS_TO_TAKE_PENALTY + LAST_LAPS_WITHOUT_PENALTY) can drive to the end of the race without taking it but receive time penalty! (Drive through = PENALTY_SECONDS, Stop & GO 10 = PENALTY_SECONDS + 10, .......)"`
}

type RealPenaltySettingsCutting struct {
	EnabledDuringSafetyCar bool    `ini:"ENABLED_DURING_SAFETY_CAR" help:"Cutting penalty enable during Safety Car or Virtual Safety Car"`
	TotalCutWarnings       int     `ini:"TOTAL_CUT_WARNINGS" help:"How many warnings are allowed before a penalty is given"`
	EnableTyresDirtLevel   bool    `ini:"ENABLE_TYRES_DIRT_LEVEL" help:"Tyres dirt for cutting"`
	WheelsOut              int     `ini:"WHEELS_OUT" help:"The maximum number of wheels that are allowed to go off track"`
	MinSpeed               int     `ini:"MIN_SPEED" help:"The minimum speed, in kph, that will trigger a cut"`
	SecondsBetweenCuts     int     `ini:"SECONDS_BETWEEN_CUTS" help:"You won't get a cut until this many seconds after the last one"`
	MaxCutTime             int     `ini:"MAX_CUT_TIME" help:"The maximum time cut (in seconds) to trigger a warning. If you make this too long, off-track accidents may trigger cut warnings"`
	MinSlowDownRatio       float64 `ini:"MIN_SLOW_DOWN_RATIO" help:"The minimum ratio of speed at leaving the track to speed at re-entering the track that will trigger a warning. A speed ratio under this means that the car has slowed, and negated any advantage gained, and a warning won't be triggered. 0.9 means the car is re-entering the track at 90% of the speed at which it left it. A value < 1 means the car has slowed during the cut; a value > 1 means the car has sped up"`

	QualifySlowDownSpeed      int     `ini:"QUAL_SLOW_DOWN_SPEED" help:"Slowing down to this speed (kph) will make the INVALID LAP, SLOW DOWN message in qualifying go away"`
	QualifyMaxSectorOutSpeed  int     `ini:"QUALIFY_MAX_SECTOR_OUT_SPEED" help:"Initial qualify max allowed speed on the start line after cutting in the last corner. IMPORTANT! Adjust this value in according to the track and the cars!"`
	QualifySlowDownSpeedRatio float64 `ini:"QUALIFY_SLOW_DOWN_SPEED_RATIO" help:"Ratio of the QUALIFY_MAX_SECTOR_OUT_SPEED. The max allowed speed on the start line after cutting = QUALIFY_MAX_SECTOR_OUT_SPEED (other new max speed of driver) * QUALIFY_SLOW_DOWN_SPEED_RATIO"`

	PostCuttingTime int    `ini:"POST_CUTTING_TIME" help:"Bonus time after reentering on the track. If speed decreases in this time --> No cutting! Set to 0 to have cutting checks similar to PLP (no check after re-entering the track)"`
	PenaltyType     string `ini:"PENALTY_TYPE" help:"Penalty type for cutting. dt = drive through, sgn = stop and go n seconds, n = n seconds to add at the end of the race"`
}

type RealPenaltySettingsSpeeding struct {
	PitLaneSpeed int `ini:"PIT_LANE_SPEED" help:"The speed, in kph, above which you will be deemed to be speeding in pits. Make this higher to give more leniency on pit lane entry"`

	PenaltyType0       string `ini:"PENALTY_TYPE_0" help:"Penalty type for speeding. dt = drive through, sgn = stop and go n seconds, n = n seconds to add at the end of the race, dsq = disqualification --> kick"`
	SpeedLimitPenalty0 int    `ini:"SPEED_LIMIT_PENALTY_0" help:""`

	PenaltyType1       string `ini:"PENALTY_TYPE_1" help:"Penalty type for speeding. dt = drive through, sgn = stop and go n seconds, n = n seconds to add at the end of the race, dsq = disqualification --> kick"`
	SpeedLimitPenalty1 int    `ini:"SPEED_LIMIT_PENALTY_1" help:""`

	PenaltyType2       string `ini:"PENALTY_TYPE_2" help:"Penalty type for speeding. dt = drive through, sgn = stop and go n seconds, n = n seconds to add at the end of the race, dsq = disqualification --> kick"`
	SpeedLimitPenalty2 int    `ini:"SPEED_LIMIT_PENALTY_2" help:""`
}

type RealPenaltySettingsCrossing struct {
	PenaltyType string `ini:"PENALTY_TYPE" help:"Penalty type for crossing. dt = drive through, sgn = stop and go n seconds, n = n seconds to add at the end of the race"`
}

type RealPenaltySettingsJumpStart struct {
	PenaltyType0       string `ini:"PENALTY_TYPE_0" help:"Penalty type for jump start. Don't set seconds penalty! dt = drive through, sgn = stop and go n seconds, dsq = disqualification --> kick"`
	SpeedLimitPenalty0 int    `ini:"SPEED_LIMIT_PENALTY_0" help:""`

	PenaltyType1       string `ini:"PENALTY_TYPE_0" help:""`
	SpeedLimitPenalty1 int    `ini:"SPEED_LIMIT_PENALTY_0" help:"Penalty type for jump start. Don't set seconds penalty! dt = drive through, sgn = stop and go n seconds, dsq = disqualification --> kick"`

	PenaltyType2       string `ini:"PENALTY_TYPE_0" help:""`
	SpeedLimitPenalty2 int    `ini:"SPEED_LIMIT_PENALTY_0" help:"Penalty type for jump start. Don't set seconds penalty! dt = drive through, sgn = stop and go n seconds, dsq = disqualification --> kick"`
}

type RealPenaltySettingsDRS struct {
	PenaltyType      string  `ini:"PENALTY_TYPE" help:"Penalty type for illegal DRS use. dt = drive through, sgn = stop and go n seconds, n = n seconds to add at the end of the race"`
	Gap              float64 `ini:"GAP" help:"Max gap in seconds from front car"`
	EnabledAfterLaps int     `ini:"ENABLED_AFTER_LAPS" help:"DRS enabled after N lap from start (+1 if rolling start with SC - file ac_settings.ini)"`
	MinSpeed         int     `ini:"MIN_SPEED" help:"If the car speed < MIN SPEED no penalty for illegal DRS use"`
	BonusTime        float64 `ini:"BONUS_TIME" help:"How many seconds the DRS can remain open during each illegal use before the penalty"`
	MaxIllegalUses   int     `ini:"MAX_ILLEGAL_USES" help:"How many time the driver can open the illegal DRS in each sector before the penalty"`
}

type RealPenaltySettingsBlueFlag struct {
	QualifyTimeThreshold float64 `ini:"QUALIFY_TIME_THRESHOLD" help:"Time distance (seconds) to show the blue flag in qualifying"`
	RaceTimeThreshold    float64 `ini:"RACE_TIME_THRESHOLD" help:"Time distance (seconds) to show the blue flag in the race"`
}

type RealPenaltyACSettings struct {
	General   RealPenaltyACSettingsGeneral   `ini:"General" help:""`
	App       RealPenaltyACSettingsApp       `ini:"App" help:""`
	Sol       RealPenaltyACSettingsSol       `ini:"Sol" help:""`
	SafetyCar RealPenaltyACSettingsSafetyCar `ini:"Safety_Car" help:""`
	NoPenalty RealPenaltyACSettingsNoPenalty `ini:"No_Penalty" help:""`
	Admin     RealPenaltyACSettingsAdmin     `ini:"Admin" help:""`
	Helicorsa RealPenaltyACSettingsHelicorsa `ini:"Helicorsa" help:""`
}

func (rpac *RealPenaltyACSettings) write() error {
	f := ini.NewFile([]ini.DataSource{nil}, ini.LoadOptions{
		IgnoreInlineComment: true,
	})

	// making and throwing away a default section due to the utter insanity of ini or assetto. i don't know which.
	_, err := f.NewSection("DEFAULT")

	if err != nil {
		return err
	}

	general, err := f.NewSection("General")

	if err != nil {
		return err
	}

	err = general.ReflectFrom(&rpac.General)

	if err != nil {
		return err
	}

	app, err := f.NewSection("App")

	if err != nil {
		return err
	}

	err = app.ReflectFrom(&rpac.App)

	if err != nil {
		return err
	}

	sol, err := f.NewSection("Sol")

	if err != nil {
		return err
	}

	err = sol.ReflectFrom(&rpac.Sol)

	if err != nil {
		return err
	}

	safetyCar, err := f.NewSection("Safety_Car")

	if err != nil {
		return err
	}

	err = safetyCar.ReflectFrom(&rpac.SafetyCar)

	if err != nil {
		return err
	}

	noPenalty, err := f.NewSection("No_Penalty")

	if err != nil {
		return err
	}

	err = noPenalty.ReflectFrom(&rpac.NoPenalty)

	if err != nil {
		return err
	}

	admin, err := f.NewSection("Admin")

	if err != nil {
		return err
	}

	err = admin.ReflectFrom(&rpac.Admin)

	if err != nil {
		return err
	}

	helicorsa, err := f.NewSection("Helicorsa")

	if err != nil {
		return err
	}

	err = helicorsa.ReflectFrom(&rpac.Helicorsa)

	if err != nil {
		return err
	}

	return f.SaveTo(filepath.Join(RealPenaltyFolderPath(), realPenaltyACSettingsIniPath))
}

func DefaultRealPenaltyACSettings() RealPenaltyACSettings {
	return RealPenaltyACSettings{
		General: RealPenaltyACSettingsGeneral{
			FirstCheckTime:  5,
			CockpitCamera:   false,
			TrackChecksum:   false,
			WeatherChecksum: false,
		},
		App: RealPenaltyACSettingsApp{
			Mandatory:      "true",
			CheckFrequency: 60,
		},
		Sol: RealPenaltyACSettingsSol{
			Mandatory:              false,
			PerformanceModeAllowed: true,
			CheckFrequency:         60,
		},
		SafetyCar: RealPenaltyACSettingsSafetyCar{
			CarModel:                   "",
			RaceStartBehindSC:          false,
			NormalizedLightOffPosition: 0.5,
			NormalizedStartPosition:    0.95,
			GreenLightDelay:            5.0,
		},
		NoPenalty: RealPenaltyACSettingsNoPenalty{
			GUIDs: "76000000000000000;76000000000000001",
		},
		Admin: RealPenaltyACSettingsAdmin{
			GUIDs: "76000000000000000;76000000000000001",
		},
		Helicorsa: RealPenaltyACSettingsHelicorsa{
			Mandatory:         true,
			DistanceThreshold: 30.0,
			WorldZoom:         5.0,
			OpacityThreshold:  8.0,
			FrontFadeOutArc:   90.0,
			FrontFadeAngle:    10.0,
			CarLength:         4.3,
			CarWidth:          1.8,
		},
	}
}

type RealPenaltyACSettingsGeneral struct {
	FirstCheckTime int `ini:"FIRST_CHECK_TIME" help:"Delay (seconds) after connection of new driver for the first check (app + sol)"`

	CockpitCamera bool `ini:"COCKPIT_CAMERA" help:"Set to true for mandatory cockpit visual for all drivers. For race Direction or live transmission insert the GUIDs under [No_Penalty] section"`

	TrackChecksum   bool `ini:"TRACK_CHECKSUM" help:"Set to true if you want an additional checksum of the track (model + kn5 files if they exist on the server)"`
	WeatherChecksum bool `ini:"WEATHER_CHECKSUM" help:"Set to true if you want the weather checksum (if the weather exists on the server)"`
}

type RealPenaltyACSettingsApp struct {
	Mandatory      boolString `ini:"MANDATORY" help:"on = Real Penalty app is mandatory, off = Real Penalty app is not mandatory"`
	CheckFrequency int        `ini:"CHECK_FREQUENCY" help:"Frequency (seconds) for app check"`
}

type RealPenaltyACSettingsSol struct {
	Mandatory              bool `ini:"MANDATORY" help:"Set to true if the event is with mod Sol day to night transition"`
	PerformanceModeAllowed bool `ini:"PERFORMACE_MODE_ALLOWED" help:"Set to true if Sol performance mode is allowed"` // misspelt in real penalty INI
	CheckFrequency         int  `ini:"CHECK_FREQUENCY" help:"Frequency (seconds) for Sol check"`
}

type RealPenaltyACSettingsSafetyCar struct {
	CarModel                   string  `ini:"CAR_MODEL" help:"Car model of safety car"`
	RaceStartBehindSC          bool    `ini:"RACE_START_BEHIND_SC" help:"Set to true if race starts after the first lap behind Safety Car (it works with or without a real Safety Car on the server)"`
	NormalizedLightOffPosition float64 `ini:"NORMALIZED_LIGHT_OFF_POSITION" help:"Rolling start: normalized position (0 = start, 1 = end) of the first driver during rolling start when the app switches off the SC signal"`
	NormalizedStartPosition    float64 `ini:"NORMALIZED_START_POSITION" help:"Rolling start: normalized position (0 = start, 1 = end) of the first driver during rolling start when the app switches on the red signal. START_NORMALIZED_POSITION must be greater than LIGHT_OFF_NORMALIZED_POSITION"`
	GreenLightDelay            float64 `ini:"GREEN_LIGHT_DELAY" help:"Rolling start: delay of the green light after red signal (seconds)"`
}

type RealPenaltyACSettingsNoPenalty struct {
	GUIDs string `ini:"GUIDs" name:"GUIDs" help:"List of Steam GUIDs (separated by a semicolon) that can connect to the server without the app and sol (for example 'Race Direction' or for 'Live')"`
}

type RealPenaltyACSettingsAdmin struct {
	GUIDs string `ini:"GUIDs" name:"GUIDs" help:"List of Steam GUIDs (separated by a semicolon) that can send commands to the server via chat"`
}

type RealPenaltyACSettingsHelicorsa struct {
	Mandatory         bool    `ini:"MANDATORY" help:"Set to true if the Helicorsa app is mandatory for the race"`
	DistanceThreshold float64 `ini:"DISTANCE_THRESHOLD" help:"How far away are the cars we paint?"`
	WorldZoom         float64 `ini:"WORLD_ZOOM" help:"World coordinates zoom or how big the bars are"`
	OpacityThreshold  float64 `ini:"OPACITY_THRESHOLD" help:"Opacity threshold: At which distance (in meters) should the cars start to fade?"`
	FrontFadeOutArc   float64 `ini:"FRONT_FADE_OUT_ARC" help:"Fade out cars in front of the player in an arc of X degrees (0 to disable)"`
	FrontFadeAngle    float64 `ini:"FRONT_FADE_ANGLE" help:"If a car in front is faded out, how soft should it fade? (again in degrees, 0 to disable = on/off)"`

	CarLength float64 `ini:"CAR_LENGHT" help:"How long are the cars (in meters). Sorry, no data from AC available"` // this is misspelt in the example ac_settings.ini
	CarWidth  float64 `ini:"CAR_WIDTH" help:"How wide are the cars (in meters). Sorry, no data from AC available"`
}
