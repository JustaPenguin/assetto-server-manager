package servermanager

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

var (
	ViewRenderer  *Renderer
	store         sessions.Store
	logOutput     = newLogBuffer(MaxLogSizeBytes)
	pluginsOutput = newLogBuffer(MaxLogSizeBytes)
)

func init() {
	if os.Getenv("DEBUG") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.SetOutput(io.MultiWriter(os.Stdout, logOutput))
}

func Router(fs http.FileSystem) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/logout", logoutHandler)

	// readers
	r.Group(func(r chi.Router) {
		r.Use(ReadAccessMiddleware)

		// pages
		r.Get("/", homeHandler)

		// content
		r.Get("/cars", carsHandler)
		r.Get("/tracks", tracksHandler)
		r.Get("/weather", weatherHandler)
		r.Get("/setups", carSetupsHandler)

		// results
		r.Get("/results", resultsHandler)
		r.Get("/results/{fileName}", resultHandler)

		r.Get("/logs", serverLogsHandler)
		r.Get("/api/logs", apiServerLogHandler)

		// championships
		r.Get("/championships", listChampionshipsHandler)
		r.Get("/championship/{championshipID}", viewChampionshipHandler)
		r.Get("/championship/{championshipID}/export", exportChampionshipHandler)

		// live timings
		r.Get("/live-timing", liveTimingHandler)
		r.Get("/live-timing/get", liveTimingGetHandler)
		r.Get("/api/live-map", LiveMapHandler)

		FileServer(r, "/content", http.Dir(filepath.Join(ServerInstallPath, "content")))
		FileServer(r, "/results/download", http.Dir(filepath.Join(ServerInstallPath, "results")))
	})

	// writers
	r.Group(func(r chi.Router) {
		r.Use(WriteAccessMiddleware)

		// content
		r.Get("/track/delete/{name}", trackDeleteHandler)
		r.Get("/car/delete/{name}", carDeleteHandler)
		r.Get("/weather/delete/{key}", weatherDeleteHandler)

		// races
		r.Get("/quick", quickRaceHandler)
		r.Post("/quick/submit", quickRaceSubmitHandler)
		r.Get("/custom", customRaceListHandler)
		r.Get("/custom/new", customRaceNewOrEditHandler)
		r.Get("/custom/load/{uuid}", customRaceLoadHandler)
		r.Post("/custom/schedule/{uuid}", customRaceScheduleHandler)
		r.Get("/custom/schedule/{uuid}/remove", customRaceScheduleRemoveHandler)
		r.Get("/custom/edit/{uuid}", customRaceNewOrEditHandler)
		r.Get("/custom/delete/{uuid}", customRaceDeleteHandler)
		r.Get("/custom/star/{uuid}", customRaceStarHandler)
		r.Get("/custom/loop/{uuid}", customRaceLoopHandler)
		r.Post("/custom/new/submit", customRaceSubmitHandler)

		// server management
		r.Get("/process/{action}", serverProcessHandler)

		// championships
		r.Get("/championships/new", newOrEditChampionshipHandler)
		r.Post("/championships/new/submit", submitNewChampionshipHandler)
		r.Get("/championship/{championshipID}/edit", newOrEditChampionshipHandler)
		r.Get("/championship/{championshipID}/delete", deleteChampionshipHandler)
		r.Get("/championship/{championshipID}/event", championshipEventConfigurationHandler)
		r.Post("/championship/{championshipID}/event/submit", championshipSubmitEventConfigurationHandler)
		r.Get("/championship/{championshipID}/event/{eventID}/start", championshipStartEventHandler)
		r.Post("/championship/{championshipID}/event/{eventID}/schedule", championshipScheduleEventHandler)
		r.Get("/championship/{championshipID}/event/{eventID}/schedule/remove", championshipScheduleEventRemoveHandler)
		r.Get("/championship/{championshipID}/event/{eventID}/edit", championshipEventConfigurationHandler)
		r.Get("/championship/{championshipID}/event/{eventID}/delete", championshipDeleteEventHandler)
		r.Get("/championship/{championshipID}/event/{eventID}/practice", championshipStartPracticeEventHandler)
		r.Get("/championship/{championshipID}/event/{eventID}/cancel", championshipCancelEventHandler)
		r.Get("/championship/{championshipID}/event/{eventID}/restart", championshipRestartEventHandler)

		// penalties
		r.Post("/penalties/{sessionFile}/{driverGUID}", penaltyHandler)

		// endpoints
		r.Post("/api/track/upload", apiTrackUploadHandler)
		r.Post("/api/car/upload", apiCarUploadHandler)
		r.Post("/api/weather/upload", apiWeatherUploadHandler)
	})

	// admins
	r.Group(func(r chi.Router) {
		r.Use(AdminAccessMiddleware)

		r.HandleFunc("/server-options", serverOptionsHandler)
	})

	FileServer(r, "/static", fs)

	return r
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, fs.ServeHTTP)
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

	ViewRenderer.MustLoadTemplate(w, r, "server/options.html", map[string]interface{}{
		"form": form,
	})
}

type ContentFile struct {
	Name     string `json:"name"`
	FileType string `json:"type"`
	FilePath string `json:"filepath"`
	Data     string `json:"dataBase64"`
	Size     int    `json:"size"`
}

var base64HeaderRegex = regexp.MustCompile("^(data:.+;base64,)")

// Stores Files encoded into r.Body
func uploadHandler(w http.ResponseWriter, r *http.Request, contentType string) {
	var files []ContentFile

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
func addFiles(files []ContentFile, contentType string) error {
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
		parts := strings.Split(file.FilePath, "/")

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
				err := addTyresFromDataACD(file.FilePath, fileDecoded)

				if err != nil {
					logrus.Errorf("Could not create tyres for new car (%s), err: %s", file.FilePath, err)
				}
			} else if name == "tyres.ini" {
				// it seems some cars don't pack their data into an ACD file, it's just in a folder called 'data'
				// so we can just grab tyres.ini from there.
				err := addTyresFromTyresIni(file.FilePath, fileDecoded)

				if err != nil {
					logrus.Errorf("Could not create tyres for new car (%s), err: %s", file.FilePath, err)
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
	ViewRenderer.MustLoadTemplate(w, r, "server/logs.html", nil)
}

type logData struct {
	ServerLog, ManagerLog, PluginsLog string
}

func apiServerLogHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(logData{
		ServerLog:  AssettoProcess.Logs(),
		ManagerLog: logOutput.String(),
		PluginsLog: pluginsOutput.String(),
	})
}

func liveTimingHandler(w http.ResponseWriter, r *http.Request) {
	currentRace, entryList := raceManager.CurrentRace()

	var customRace *CustomRace

	if currentRace != nil {
		customRace = &CustomRace{EntryList: entryList, RaceConfig: currentRace.CurrentRaceConfig}
	}

	ViewRenderer.MustLoadTemplate(w, r, "live-timing.html", map[string]interface{}{
		"RaceDetails": customRace,
	})
}

func liveTimingGetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(liveInfo)
}
