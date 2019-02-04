package servermanager

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/sessions"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
	r.HandleFunc("/car/delete/{name}", carDeleteHandler)
	r.HandleFunc("/server-options", globalServerOptionsHandler)
	r.HandleFunc("/quick", quickRaceHandler)
	r.Methods(http.MethodPost).Path("/quick/submit").HandlerFunc(quickRaceSubmitHandler)
	r.HandleFunc("/logs", serverLogsHandler)
	r.HandleFunc("/process/{action}", serverProcessHandler)

	// endpoints
	r.HandleFunc("/api/logs", apiServerLogHandler)
	r.HandleFunc("/api/track/upload", apiTrackUploadHandler)
	r.HandleFunc("/api/car/upload", apiCarUploadHandler)

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

func apiCarUploadHandler(w http.ResponseWriter, r *http.Request) {
	uploadHandler(w, r, "Car")
}

func apiTrackUploadHandler(w http.ResponseWriter, r *http.Request) {
	uploadHandler(w, r, "Track")
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

type contentFile struct {
	Name     string `json:"name"`
	FileType string `json:"type"`
	FilePath string `json:"webkitRelativePath"`
	Data     string `json:"dataBase64"`
	Size     int    `json:"size"`
}

var base64HeaderRegex = regexp.MustCompile("^(data:.+;base64,)")

// Stores Files encoded into r.Body
func uploadHandler(w http.ResponseWriter, r *http.Request, contentType string) {
	var Files []contentFile

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&Files)

	if err != nil {
		logrus.Errorf("could not decode "+strings.ToLower(contentType)+" json, err: %s", err)
		return
	}

	err = addFiles(Files, contentType)

	if err != nil {
		err = AddFlashQuick(w, r, contentType+"(s) could not be added")

		if err != nil {
			logrus.Errorf("could not add flash, err: %s", err)
		}

		return
	}

	err = AddFlashQuick(w, r, contentType+"(s) added successfully!")

	if err != nil {
		logrus.Errorf("could not add flash, err: %s", err)
	}
}

// Stores files in the correct location
func addFiles(Files []contentFile, contentType string) error {
	var tracksPath string

	if contentType == "Track" {
		tracksPath = filepath.Join(ServerInstallPath, "content", "tracks")
	} else if contentType == "Car" {
		tracksPath = filepath.Join(ServerInstallPath, "content", "cars")
	}

	for _, file := range Files {
		fileDecoded, err := base64.StdEncoding.DecodeString(base64HeaderRegex.ReplaceAllString(file.Data, ""))

		if err != nil {
			logrus.Errorf("could not decode "+contentType+" file data, err: %s", err)
			return err
		}

		// If user uploaded a "tracks" or "cars" folder containing multiple tracks
		parts := strings.Split(file.FilePath, string(os.PathSeparator))

		if parts[0] == "tracks" || parts[0] == "cars" {
			parts = parts[1:]
			file.FilePath = ""

			for _, part := range parts {
				file.FilePath = filepath.Join(file.FilePath, part)
			}
		}

		path := filepath.Join(tracksPath, file.FilePath)

		// Makes any directories in the path that don't exist (there can be multiple)
		err = os.MkdirAll(filepath.Dir(path), 0755)

		if err != nil {
			logrus.Errorf("could not create "+contentType+" file directory, err: %s", err)
			return err
		}

		err = ioutil.WriteFile(path, fileDecoded, 0644)

		if err != nil {
			logrus.Errorf("could not write file, err: %s", err)
			return err
		}
	}

	return nil
}

func trackDeleteHandler(w http.ResponseWriter, r *http.Request) {
	trackName := mux.Vars(r)["name"]
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	existingTracks, err := ioutil.ReadDir(tracksPath)

	if err != nil {
		logrus.Errorf("could not get track list, err: %s", err)

		err = AddFlashQuick(w, r, "couldn't get track list")

		if err != nil {
			logrus.Fatalf("could not add flash, err: %s", err)
		}

		http.Redirect(w, r, r.Referer(), http.StatusFound)

		return
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

	var message string

	if found {
		// confirm deletion
		message = "Track successfully deleted!"
	} else {
		// inform track wasn't found
		message = "Sorry, track could not be deleted."
	}

	err = AddFlashQuick(w, r, message)

	if err != nil {
		logrus.Fatalf("could not add flash, err: %s", err)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func carDeleteHandler(w http.ResponseWriter, r *http.Request) {
	carName := mux.Vars(r)["name"]
	carsPath := filepath.Join(ServerInstallPath, "content", "cars")

	existingCars, err := ioutil.ReadDir(carsPath)

	if err != nil {
		logrus.Errorf("could not get car list, err: %s", err)

		err = AddFlashQuick(w, r, "couldn't get track list")

		if err != nil {
			logrus.Fatalf("could not add flash, err: %s", err)
		}

		http.Redirect(w, r, r.Referer(), http.StatusFound)

		return
	}

	var found bool

	for _, car := range existingCars {
		if car.Name() == carName {
			// Delete car
			found = true

			err := os.RemoveAll(filepath.Join(carsPath, carName))

			if err != nil {
				found = false
			}

			break
		}
	}

	var message string

	if found {
		// confirm deletion
		message = "Car successfully deleted!"
	} else {
		// inform car wasn't found
		message = "Sorry, car could not be deleted. Are you sure it was installed?"
	}

	err = AddFlashQuick(w, r, message)

	if err != nil {
		logrus.Fatalf("could not add flash, err: %s", err)
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

// Helper function to get message session and add a flash
func AddFlashQuick(w http.ResponseWriter, r *http.Request, message string) error {
	session, err := store.Get(r, "messages")

	if err != nil {
		return err
	}

	session.AddFlash(message)

	err = session.Save(r, w)

	if err != nil {
		logrus.Fatalf("could not save session, err: %s", err)
	}

	return nil
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
