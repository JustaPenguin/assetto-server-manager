package servermanager

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"
)

var LaunchTime = time.Now()

type HealthCheck struct {
	raceControl *RaceControl
	process     ServerProcess
	store       Store
}

func NewHealthCheck(raceControl *RaceControl, store Store, process ServerProcess) *HealthCheck {
	return &HealthCheck{
		store:       store,
		raceControl: raceControl,
		process:     process,
	}
}

type HealthCheckResponse struct {
	OK        bool
	Version   string
	IsPremium bool
	IsHosted  bool

	OS            string
	NumCPU        int
	NumGoroutines int
	Uptime        string
	GoVersion     string

	AssettoIsInstalled  bool
	StrackerIsInstalled bool

	CarDirectoryIsWritable     bool
	TrackDirectoryIsWritable   bool
	WeatherDirectoryIsWritable bool
	SetupsDirectoryIsWritable  bool
	ConfigDirectoryIsWritable  bool
	ResultsDirectoryIsWritable bool

	ServerName          string
	EventInProgress     bool
	EventIsCritical     bool
	EventIsChampionship bool
	EventIsRaceWeekend  bool
	EventIsPractice     bool
	NumConnectedDrivers int
	MaxClientsOverride  int
}

func (h *HealthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	event := h.process.Event()
	opts, err := h.store.LoadServerOptions()

	var serverName string

	if err == nil {
		serverName = opts.Name
	}

	_ = json.NewEncoder(w).Encode(HealthCheckResponse{
		OK:                 true,
		OS:                 runtime.GOOS + "/" + runtime.GOARCH,
		Version:            BuildVersion,
		IsPremium:          Premium(),
		IsHosted:           IsHosted,
		MaxClientsOverride: MaxClientsOverride,
		NumCPU:             runtime.NumCPU(),
		NumGoroutines:      runtime.NumGoroutine(),
		Uptime:             time.Since(LaunchTime).String(),
		GoVersion:          runtime.Version(),

		ServerName:          serverName,
		EventInProgress:     h.raceControl.process.IsRunning(),
		EventIsCritical:     !event.IsPractice() && (event.IsChampionship() || event.IsRaceWeekend() || h.raceControl.SessionInfo.Type == udp.SessionTypeRace || h.raceControl.SessionInfo.Type == udp.SessionTypeQualifying),
		EventIsChampionship: event.IsChampionship(),
		EventIsRaceWeekend:  event.IsRaceWeekend(),
		EventIsPractice:     event.IsPractice(),
		NumConnectedDrivers: h.raceControl.ConnectedDrivers.Len(),
		AssettoIsInstalled:  IsAssettoInstalled(),
		StrackerIsInstalled: IsStrackerInstalled(),

		ConfigDirectoryIsWritable:  IsDirWriteable(filepath.Join(ServerInstallPath, "cfg")) == nil,
		CarDirectoryIsWritable:     IsDirWriteable(filepath.Join(ServerInstallPath, "content", "cars")) == nil,
		TrackDirectoryIsWritable:   IsDirWriteable(filepath.Join(ServerInstallPath, "content", "tracks")) == nil,
		WeatherDirectoryIsWritable: IsDirWriteable(filepath.Join(ServerInstallPath, "content", "weather")) == nil,
		SetupsDirectoryIsWritable:  IsDirWriteable(filepath.Join(ServerInstallPath, "setups")) == nil,
		ResultsDirectoryIsWritable: IsDirWriteable(filepath.Join(ServerInstallPath, "results")) == nil,
	})
}

func IsDirWriteable(dir string) error {
	file := filepath.Join(dir, ".test-write")

	if err := ioutil.WriteFile(file, []byte(""), 0600); err != nil {
		return err
	}

	return os.Remove(file)
}
