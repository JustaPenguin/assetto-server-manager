package servermanager

import (
	"io/ioutil"
	"path/filepath"

	"github.com/gorilla/sessions"
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
	store        = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	logOutput    = new(bytes.Buffer)
)

func init() {
	logrus.SetOutput(io.MultiWriter(os.Stdout, logOutput))
}

func Router() *mux.Router {
	r := mux.NewRouter()

	// pages
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/cars", carsHandler)
	r.HandleFunc("/tracks", tracksHandler)
	r.HandleFunc("/track/delete/{name}", trackDeleteHandler)
	r.HandleFunc("/server-options", globalServerOptionsHandler)
	r.HandleFunc("/race-options", raceOptionsHandler)
	r.HandleFunc("/logs", serverLogsHandler)

	// endpoints
	r.HandleFunc("/api/logs", apiServerLogHandler)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))

	return r
}

// homeHandler serves content to /
func homeHandler(w http.ResponseWriter, r *http.Request) {
	ViewRenderer.MustLoadTemplate(w, r, "home.html", nil)
}

func globalServerOptionsHandler(w http.ResponseWriter, r *http.Request) {
	form := NewForm(&ConfigIniDefault.Server.GlobalServerConfig, nil)

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
	})

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

func carsHandler(w http.ResponseWriter, r *http.Request) {
	cars, err := ListCars()

	if err != nil {
		logrus.Fatalf("could not get car list, err: %s", err)
	}

	ViewRenderer.MustLoadTemplate(w, r, "cars.html", map[string]interface{}{
		"cars": cars,
	})
}

func tracksHandler(w http.ResponseWriter, r *http.Request) {
	tracks, err := ListTracks()

	if err != nil {
		logrus.Fatalf("could not get track list, err: %s", err)
	}

	ViewRenderer.MustLoadTemplate(w, r, "tracks.html", map[string]interface{}{
		"tracks": tracks,
	})
}

func trackDeleteHandler(w http.ResponseWriter, r *http.Request) {
	trackName := mux.Vars(r)["name"]
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	existingTracks, err := ioutil.ReadDir(tracksPath)

	if err != nil {
		logrus.Fatalf("could not get track list, err: %s", err)
	}

	var found bool

	for _, track := range existingTracks {
		if track.Name() == trackName {
			// Delete track
			found = true

			err := os.RemoveAll(filepath.Join(tracksPath, trackName))

			if err != nil {
				found = false
			}

			break
		}
	}

	session, err := getSession(r)

	if err != nil {
		logrus.Fatalf("could not get session, err: %s", err)
	}

	if found {
		// send flash, confirm deletion
		session.AddFlash("Track successfully deleted!")
	} else {
		// send flash, inform track wasn't found
		session.AddFlash("Sorry, track could not be deleted.")
	}

	err = session.Save(r, w)

	if err != nil {
		logrus.Fatalf("could not save session, err: %s", err)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func getSession(r *http.Request) (*sessions.Session, error) {
	session, err := store.Get(r, "messages")

	if err != nil {
		return nil, err
	}

	return session, nil
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
