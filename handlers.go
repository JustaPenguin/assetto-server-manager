package servermanager

import (
	"encoding/base64"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/sessions"
	"bytes"
	"encoding/json"
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
	r.HandleFunc("/quick", quickRaceHandler)
	r.Methods(http.MethodPost).Path("/quick/submit").HandlerFunc(quickRaceSubmitHandler)
	r.HandleFunc("/logs", serverLogsHandler)
	r.HandleFunc("/process/{action}", serverProcessHandler)

	// endpoints
	r.HandleFunc("/api/logs", apiServerLogHandler)
	r.HandleFunc("/api/track/upload", apiTrackUploadHandler)

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

	ViewRenderer.MustLoadTemplate(w, r, "server_options.html", map[string]interface{}{
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

type trackFile struct {
	Name string `json:"name"`
	FileType string `json:"type"`
	FilePath string `json:"webkitRelativePath"`
	Data string `json:"dataBase64"`
	Size int `json:"size"`
}

var base64HeaderRegex = regexp.MustCompile("^(data:.+;base64,)")

func apiTrackUploadHandler(w http.ResponseWriter, r *http.Request) {
	var trackFiles []trackFile

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&trackFiles)

	if err != nil {
		// will this call onFail in manager.js?
		logrus.Fatalf("could not decode track json, err: %s", err)
	}

	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	existingTracks, err := ioutil.ReadDir(tracksPath)

	for _, existingTrack := range existingTracks {
		if strings.Contains(trackFiles[0].FilePath, existingTrack.Name()) {
			// @TODO track already exists - replace?
			// @TODO this check needs to be more stronk, actually check data content or something
			// @TODO otherwise names with similar bits will clash
		}
	}

	for _, file := range trackFiles {
		fileDecoded, err := base64.StdEncoding.DecodeString(base64HeaderRegex.ReplaceAllString(file.Data, ""))

		if err != nil {
			logrus.Fatalf("could not decode track file data, err: %s", err)
		}

		path := filepath.Join(tracksPath, file.FilePath)

		err = os.MkdirAll(filepath.Dir(path), 0755)

		if err != nil {
			logrus.Fatalf("could not create track file directory, err: %s", err)
		}

		err = ioutil.WriteFile(path, fileDecoded, 0644)

		if err != nil {
			logrus.Fatalf("could not decode write file, err: %s", err)
		}
	}
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
