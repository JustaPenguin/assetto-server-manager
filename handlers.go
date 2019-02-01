package servermanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var (
	ViewRenderer *Renderer
	logOutput    = new(bytes.Buffer)
)

func init() {
	logrus.SetOutput(io.MultiWriter(os.Stdout, logOutput))
}

func Router() *mux.Router {
	r := mux.NewRouter()

	// pages
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/server-options", globalServerOptionsHandler)
	r.HandleFunc("/quick", quickRaceHandler)
	r.Methods(http.MethodPost).Path("/quick/submit").HandlerFunc(quickRaceSubmitHandler)
	r.HandleFunc("/race-options", raceOptionsHandler)
	r.HandleFunc("/logs", serverLogsHandler)
	r.HandleFunc("/process/{action}", serverProcessHandler)

	// endpoints
	r.HandleFunc("/api/logs", apiServerLogHandler)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))

	return r
}

// homeHandler serves content to /
func homeHandler(w http.ResponseWriter, r *http.Request) {
	ViewRenderer.MustLoadTemplate(w, r, "home.html", map[string]interface{}{
		"CurrentRace": raceManager.CurrentRace(),
	})
}

// serverProcessHandler modifies the server process.
func serverProcessHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	switch mux.Vars(r)["action"] {
	case "start":
		err = AssettoProcess.Start()
	case "stop":
		err = AssettoProcess.Stop()
	case "restart":
		err = AssettoProcess.Restart()
	}

	if err != nil {
		// @TODO err
		panic(err)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func globalServerOptionsHandler(w http.ResponseWriter, r *http.Request) {
	form := NewForm(&ConfigIniDefault.Server.GlobalServerConfig, nil, "")

	if r.Method == http.MethodPost {
		err := form.Submit(r)

		if err != nil {
			logrus.Errorf("couldn't submit form, err: %s", err)
		}

		// save the config
		err = ConfigIniDefault.Write()

		if err != nil {
			logrus.Errorf("couldn't save config, err: %s", err)
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, "global_server_options.html", map[string]interface{}{
		"form": form,
	})
}

func raceOptionsHandler(w http.ResponseWriter, r *http.Request) {
	cars, err := ListCars()

	if err != nil {
		logrus.Fatalf("could not get car list, err: %s", err)
	}

	tracks, err := ListTracks()

	if err != nil {
		logrus.Fatalf("could not get track list, err: %s", err)
	}

	var carNames, trackNames, trackLayouts []string

	for _, car := range cars {
		carNames = append(carNames, car.Name)
	}

	// @TODO eventually this will be loaded from somewhere
	currentRaceConfig := &ConfigIniDefault.Server.CurrentRaceConfig

	for _, track := range tracks {
		trackNames = append(trackNames, track.Name)

		for _, layout := range track.Layouts {
			trackLayouts = append(trackLayouts, fmt.Sprintf("%s:%s", track.Name, layout))
		}
	}

	form := NewForm(currentRaceConfig, map[string][]string{
		"CarOpts":         carNames,
		"TrackOpts":       trackNames,
		"TrackLayoutOpts": trackLayouts,
	}, "")

	if r.Method == http.MethodPost {
		err := form.Submit(r)

		if err != nil {
			logrus.Errorf("couldn't submit form, err: %s", err)
		}

		// save the config
		err = ConfigIniDefault.Write()

		if err != nil {
			logrus.Errorf("couldn't save config, err: %s", err)
		}
	}

	for i, layout := range trackLayouts {
		if layout == fmt.Sprintf("%s:%s", currentRaceConfig.Track, currentRaceConfig.TrackLayout) {
			// mark the current track layout so the javascript can correctly set it up.
			trackLayouts[i] += ":current"
			break
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, "current_race_options.html", map[string]interface{}{
		"form": form,
	})
}

func quickRaceHandler(w http.ResponseWriter, r *http.Request) {
	quickRaceData, err := raceManager.QuickRaceForm()

	if err != nil {
		logrus.Errorf("couldn't build quick race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "quick_race.html", quickRaceData)
}

func quickRaceSubmitHandler(w http.ResponseWriter, r *http.Request) {
	err := raceManager.SetupQuickRace(r)

	if err != nil {
		logrus.Errorf("couldn't apply quick race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func serverLogsHandler(w http.ResponseWriter, r *http.Request) {
	ViewRenderer.MustLoadTemplate(w, r, "server_logs.html", nil)
}

type logData struct {
	ServerLog, ManagerLog string
}

func apiServerLogHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(logData{
		ServerLog:  AssettoProcess.Logs(),
		ManagerLog: logOutput.String(),
	})
}
