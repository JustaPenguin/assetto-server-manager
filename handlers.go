package servermanager

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

var (
	ViewRenderer *Renderer
	store        = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	logOutput    = new(bytes.Buffer)
)

func init() {
	if os.Getenv("DEBUG") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.SetOutput(io.MultiWriter(os.Stdout, logOutput))
}

func Router() *mux.Router {
	r := mux.NewRouter()

	// pages
	r.HandleFunc("/", homeHandler)

	// content
	r.HandleFunc("/cars", carsHandler)
	r.HandleFunc("/tracks", tracksHandler)
	r.HandleFunc("/weather", weatherHandler)
	r.HandleFunc("/track/delete/{name}", trackDeleteHandler)
	r.HandleFunc("/car/delete/{name}", carDeleteHandler)
	r.HandleFunc("/weather/delete/{key}", weatherDeleteHandler)

	// results
	r.HandleFunc("/results", resultsHandler)
	r.HandleFunc("/results/{fileName}", resultHandler)
	r.HandleFunc("/server-options", serverOptionsHandler)

	// races
	r.HandleFunc("/quick", quickRaceHandler)
	r.Methods(http.MethodPost).Path("/quick/submit").HandlerFunc(quickRaceSubmitHandler)
	r.HandleFunc("/custom", customRaceListHandler)
	r.HandleFunc("/custom/new", customRaceNewHandler)
	r.HandleFunc("/custom/load/{uuid}", customRaceLoadHandler)
	r.HandleFunc("/custom/delete/{uuid}", customRaceDeleteHandler)
	r.HandleFunc("/custom/star/{uuid}", customRaceStarHandler)
	r.Methods(http.MethodPost).Path("/custom/new/submit").HandlerFunc(customRaceSubmitHandler)

	// server management
	r.HandleFunc("/logs", serverLogsHandler)
	r.HandleFunc("/process/{action}", serverProcessHandler)

	// championships
	r.HandleFunc("/championships", listChampionshipsHandler)
	r.HandleFunc("/championships/new", newChampionshipHandler)
	r.HandleFunc("/championships/new/submit", submitNewChampionshipHandler)
	r.HandleFunc("/championship/{championshipID}/race", championshipRaceConfigurationHandler)

	// endpoints
	r.HandleFunc("/api/logs", apiServerLogHandler)
	r.HandleFunc("/api/track/upload", apiTrackUploadHandler)
	r.HandleFunc("/api/car/upload", apiCarUploadHandler)
	r.HandleFunc("/api/weather/upload", apiWeatherUploadHandler)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	r.PathPrefix("/content/").Handler(http.StripPrefix("/content", http.FileServer(http.Dir(filepath.Join(ServerInstallPath, "content")))))
	r.PathPrefix("/results/download").Handler(http.StripPrefix("/results/download", http.FileServer(http.Dir(filepath.Join(ServerInstallPath, "results")))))

	return r
}

// homeHandler serves content to /
func homeHandler(w http.ResponseWriter, r *http.Request) {
	currentRace, entryList := raceManager.CurrentRace()

	var customRace *CustomRace

	if currentRace != nil {
		customRace = &CustomRace{EntryList: entryList, RaceConfig: currentRace.CurrentRaceConfig}
	}

	ViewRenderer.MustLoadTemplate(w, r, "home.html", map[string]interface{}{
		"RaceDetails": customRace,
	})
}

// serverProcessHandler modifies the server process.
func serverProcessHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var txt string

	switch mux.Vars(r)["action"] {
	case "start":
		err = AssettoProcess.Start()
		txt = "started"
	case "stop":
		err = AssettoProcess.Stop()
		txt = "stopped"
	case "restart":
		err = AssettoProcess.Restart()
		txt = "restarted"
	}

	if err != nil {
		logrus.Errorf("could not change server process status, err: %s", err)
		AddErrFlashQuick(w, r, "Unable to change server status")
	} else {
		AddFlashQuick(w, r, "Server successfully "+txt)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func serverOptionsHandler(w http.ResponseWriter, r *http.Request) {
	serverOpts, err := raceManager.LoadServerOptions()

	if err != nil {
		logrus.Errorf("couldn't load server options, err: %s", err)
	}

	form := NewForm(serverOpts, nil, "")

	if r.Method == http.MethodPost {
		err := form.Submit(r)

		if err != nil {
			logrus.Errorf("couldn't submit form, err: %s", err)
		}

		// save the config
		err = raceManager.SaveServerOptions(serverOpts)

		if err != nil {
			logrus.Errorf("couldn't save config, err: %s", err)
			AddErrFlashQuick(w, r, "Failed to save server options")
		} else {
			AddFlashQuick(w, r, "Server options successfully saved!")
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, "server_options.html", map[string]interface{}{
		"form": form,
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
	var files []contentFile

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&files)

	if err != nil {
		logrus.Errorf("could not decode "+strings.ToLower(contentType)+" json, err: %s", err)
		return
	}

	err = addFiles(files, contentType)

	if err != nil {
		AddErrFlashQuick(w, r, contentType+"(s) could not be added")

		return
	}

	AddFlashQuick(w, r, contentType+"(s) added successfully!")
}

// Stores files in the correct location
func addFiles(files []contentFile, contentType string) error {
	var contentPath string

	switch contentType {
	case "Track":
		contentPath = filepath.Join(ServerInstallPath, "content", "tracks")
	case "Car":
		contentPath = filepath.Join(ServerInstallPath, "content", "cars")
	case "Weather":
		contentPath = filepath.Join(ServerInstallPath, "content", "weather")
	}

	for _, file := range files {
		fileDecoded, err := base64.StdEncoding.DecodeString(base64HeaderRegex.ReplaceAllString(file.Data, ""))

		if err != nil {
			logrus.Errorf("could not decode "+contentType+" file data, err: %s", err)
			return err
		}

		// If user uploaded a "tracks" or "cars" folder containing multiple tracks
		parts := strings.Split(file.FilePath, string(os.PathSeparator))

		if parts[0] == "tracks" || parts[0] == "cars" || parts[0] == "weather" {
			parts = parts[1:]
			file.FilePath = ""

			for _, part := range parts {
				file.FilePath = filepath.Join(file.FilePath, part)
			}
		}

		path := filepath.Join(contentPath, file.FilePath)

		// Makes any directories in the path that don't exist (there can be multiple)
		err = os.MkdirAll(filepath.Dir(path), 0755)

		if err != nil {
			logrus.Errorf("could not create "+contentType+" file directory, err: %s", err)
			return err
		}

		if contentType == "Car" {
			if _, name := filepath.Split(file.FilePath); name == "data.acd" {
				err := addTyresForNewCar(file.FilePath, fileDecoded)

				if err != nil {
					logrus.Errorf("Could not create tyres for new car, err: %s", err)
				}
			}
		}

		err = ioutil.WriteFile(path, fileDecoded, 0644)

		if err != nil {
			logrus.Errorf("could not write file, err: %s", err)
			return err
		}
	}

	return nil
}

func getSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, "messages")

	return session
}

func getErrSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, "errors")

	return session
}

// Helper function to get message session and add a flash
func AddFlashQuick(w http.ResponseWriter, r *http.Request, message string) {
	session := getSession(r)

	session.AddFlash(message)

	// gorilla sessions is dumb and errors weirdly
	_ = session.Save(r, w)
}

func AddErrFlashQuick(w http.ResponseWriter, r *http.Request, message string) {
	session := getErrSession(r)

	session.AddFlash(message)

	// gorilla sessions is dumb and errors weirdly
	_ = session.Save(r, w)
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
