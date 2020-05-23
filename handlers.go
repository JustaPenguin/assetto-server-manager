package servermanager

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cj123/sessions"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

var (
	ViewRenderer  *Renderer
	store         sessions.Store
	logOutput     = newLogBuffer(MaxLogSizeBytes)
	pluginsOutput = newLogBuffer(MaxLogSizeBytes)

	logMultiWriter io.Writer

	Debug = os.Getenv("DEBUG") == "true"
)

func init() {
	if !Debug {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logFile, err := os.OpenFile("server-manager.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)

	if err == nil {
		logMultiWriter = io.MultiWriter(os.Stdout, logOutput, logFile)
	} else {
		logrus.WithError(err).Errorf("Could not create server manager log file")
		logMultiWriter = io.MultiWriter(os.Stdout, logOutput)
	}

	logrus.SetOutput(logMultiWriter)
}

func Router(fs http.FileSystem) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(panicHandler)

	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/logout", logoutHandler)
	r.Handle("/metrics", prometheusMonitoringHandler())
	r.Mount("/debug/", middleware.Profiler())

	// readers
	r.Group(func(r chi.Router) {
		r.Use(ReadAccessMiddleware)

		// content
		r.Get("/cars", carsHandler)
		r.Get("/tracks", tracksHandler)
		r.Get("/weather", weatherHandler)

		r.Get("/events.ics", allScheduledRacesICalHandler)

		// results
		r.Get("/results", resultsHandler)
		r.Get("/results/{fileName}", resultHandler)

		// account management
		r.HandleFunc("/accounts/new-password", newPasswordHandler)

		FileServer(r, "/content", http.Dir(filepath.Join(ServerInstallPath, "content")))
		FileServer(r, "/results/download", http.Dir(filepath.Join(ServerInstallPath, "results")))
		FileServer(r, "/setups/download", http.Dir(filepath.Join(ServerInstallPath, "setups")))
	})

	// writers
	r.Group(func(r chi.Router) {
		r.Use(WriteAccessMiddleware)

		// content
		r.Post("/setups/upload", carSetupsUploadHandler)

		// endpoints
		r.Post("/api/track/upload", apiTrackUploadHandler)
		r.Post("/api/car/upload", apiCarUploadHandler)
		r.Post("/api/weather/upload", apiWeatherUploadHandler)
	})

	// deleters
	r.Group(func(r chi.Router) {
		r.Use(DeleteAccessMiddleware)

		r.Get("/track/delete/{name}", trackDeleteHandler)
		r.Get("/car/delete/{name}", carDeleteHandler)
		r.Get("/weather/delete/{key}", weatherDeleteHandler)
		r.Get("/setups/delete/{car}/{track}/{setup}", carSetupDeleteHandler)
	})

	for i, server := range servers {
		server := server

		r.Route(fmt.Sprintf("/server-%d", i), func(r chi.Router) {

			// readers
			r.Group(func(r chi.Router) {
				r.Use(ReadAccessMiddleware)

				// pages
				r.Get("/", server.homeHandler)

				r.Get("/logs", server.serverLogsHandler)
				r.Get("/api/logs", server.apiServerLogHandler)

				// championships
				r.Get("/championships", server.listChampionshipsHandler)
				r.Get("/championship/{championshipID}", server.viewChampionshipHandler)
				r.Get("/championship/{championshipID}/export", server.exportChampionshipHandler)
				r.HandleFunc("/championship/{championshipID}/export-results", server.exportChampionshipResultsHandler)
				r.Get("/championship/{championshipID}/ics", server.championshipICalHandler)
				r.Get("/championship/{championshipID}/sign-up", server.championshipSignUpFormHandler)
				r.Post("/championship/{championshipID}/sign-up", server.championshipSignUpFormHandler)

				// live timings
				r.Get("/live-timing", server.liveTimingHandler)
				r.Get("/live-timing/get", server.liveTimingGetHandler)
				r.Get("/api/live-map", server.liveMapHandler)
			})

			// writers
			r.Group(func(r chi.Router) {
				r.Use(WriteAccessMiddleware)

				// races
				r.Get("/quick", server.quickRaceHandler)
				r.Post("/quick/submit", server.quickRaceSubmitHandler)
				r.Get("/custom", server.customRaceListHandler)
				r.Get("/custom/new", server.customRaceNewOrEditHandler)
				r.Get("/custom/load/{uuid}", server.customRaceLoadHandler)
				r.Post("/custom/schedule/{uuid}", server.customRaceScheduleHandler)
				r.Get("/custom/schedule/{uuid}/remove", server.customRaceScheduleRemoveHandler)
				r.Get("/custom/edit/{uuid}", server.customRaceNewOrEditHandler)
				r.Get("/custom/star/{uuid}", server.customRaceStarHandler)
				r.Get("/custom/loop/{uuid}", server.customRaceLoopHandler)
				r.Post("/custom/new/submit", server.customRaceSubmitHandler)

				// server management
				r.Get("/process/{action}", server.serverProcessHandler)

				// championships
				r.Get("/championships/new", server.newOrEditChampionshipHandler)
				r.Post("/championships/new/submit", server.submitNewChampionshipHandler)
				r.Get("/championship/{championshipID}/edit", server.newOrEditChampionshipHandler)
				r.Get("/championship/{championshipID}/event", server.championshipEventConfigurationHandler)
				r.Post("/championship/{championshipID}/event/submit", server.championshipSubmitEventConfigurationHandler)
				r.Get("/championship/{championshipID}/event/{eventID}/start", server.championshipStartEventHandler)
				r.Post("/championship/{championshipID}/event/{eventID}/schedule", server.championshipScheduleEventHandler)
				r.Get("/championship/{championshipID}/event/{eventID}/schedule/remove", server.championshipScheduleEventRemoveHandler)
				r.Get("/championship/{championshipID}/event/{eventID}/edit", server.championshipEventConfigurationHandler)
				r.Get("/championship/{championshipID}/event/{eventID}/practice", server.championshipStartPracticeEventHandler)
				r.Get("/championship/{championshipID}/event/{eventID}/cancel", server.championshipCancelEventHandler)
				r.Get("/championship/{championshipID}/event/{eventID}/restart", server.championshipRestartEventHandler)
				r.Post("/championship/{championshipID}/driver-penalty/{classID}/{driverGUID}", server.championshipDriverPenaltyHandler)
				r.Post("/championship/{championshipID}/team-penalty/{classID}/{team}", server.championshipTeamPenaltyHandler)
				r.Get("/championship/{championshipID}/entrants", server.championshipSignedUpEntrantsHandler)
				r.Get("/championship/{championshipID}/entrants.csv", server.championshipSignedUpEntrantsCSVHandler)
				r.Get("/championship/{championshipID}/entrant/{entrantGUID}", server.championshipModifyEntrantStatusHandler)

				r.Get("/championship/import", server.importChampionshipHandler)
				r.Post("/championship/import", server.importChampionshipHandler)
				r.Get("/championship/{championshipID}/event/{eventID}/import", server.championshipEventImportHandler)
				r.Post("/championship/{championshipID}/event/{eventID}/import", server.championshipEventImportHandler)

				// penalties
				r.Post("/penalties/{sessionFile}/{driverGUID}", server.penaltyHandler)

				// live timings
				r.Post("/live-timing/save-frames", server.liveFrameSaveHandler)
			})

			// deleters
			r.Group(func(r chi.Router) {
				r.Use(DeleteAccessMiddleware)

				r.Get("/championship/{championshipID}/event/{eventID}/delete", server.championshipDeleteEventHandler)
				r.Get("/championship/{championshipID}/delete", server.deleteChampionshipHandler)
				r.Get("/custom/delete/{uuid}", server.customRaceDeleteHandler)
			})

			// admins
			r.Group(func(r chi.Router) {
				r.Use(AdminAccessMiddleware)

				r.HandleFunc("/server-options", server.serverOptionsHandler)
			})
		})
	}

	// admins
	r.Group(func(r chi.Router) {
		r.Use(AdminAccessMiddleware)

		r.HandleFunc("/blacklist", serverBlacklistHandler)
		r.HandleFunc("/motd", serverMOTDHandler) // @TODO make per server
		r.HandleFunc("/accounts/new", createOrEditAccountHandler)
		r.HandleFunc("/accounts/edit/{id}", createOrEditAccountHandler)
		r.HandleFunc("/accounts/delete/{id}", deleteAccountHandler)
		r.HandleFunc("/accounts/reset-password/{id}", resetPasswordHandler)
		r.HandleFunc("/accounts/toggle-open", toggleServerOpenStatusHandler)
		r.HandleFunc("/accounts", manageAccountsHandler)
	})

	FileServer(r, "/static", fs)

	return MultiServerSelectMiddleware(r, prometheusMonitoringWrapper(r))
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, fs.ServeHTTP)
}

// homeHandler serves content to /
func (ms *MultiServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	currentRace, entryList := ms.raceManager.CurrentRace()

	var customRace *CustomRace

	if currentRace != nil {
		customRace = &CustomRace{EntryList: entryList, RaceConfig: currentRace.CurrentRaceConfig}
	}

	ViewRenderer.MustLoadTemplate(w, r, "home.html", map[string]interface{}{
		"RaceDetails": customRace,
	})
}

func (ms *MultiServer) serverOptionsHandler(w http.ResponseWriter, r *http.Request) {
	serverOpts, err := ms.raceManager.LoadServerOptions()

	if err != nil {
		logrus.Errorf("couldn't load server options, err: %s", err)
	}

	form := NewForm(serverOpts, nil, "", AccountFromRequest(r).Name == "admin")

	if r.Method == http.MethodPost {
		err := form.Submit(r)

		if err != nil {
			logrus.Errorf("couldn't submit form, err: %s", err)
		}

		// save the config
		err = ms.raceManager.SaveServerOptions(serverOpts)

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

func serverBlacklistHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// save to blacklist.txt
		text := r.FormValue("blacklist")

		err := ioutil.WriteFile(filepath.Join(ServerInstallPath, "blacklist.txt"), []byte(text), 0644)

		if err != nil {
			logrus.WithError(err).Error("couldn't save blacklist")
			AddErrFlashQuick(w, r, "Failed to save Server blacklist changes")
		} else {
			AddFlashQuick(w, r, "Server blacklist successfully changed!")
		}
	}

	// load blacklist.txt
	b, err := ioutil.ReadFile(filepath.Join(ServerInstallPath, "blacklist.txt")) // just pass the file name
	if err != nil {
		logrus.WithError(err).Error("couldn't find blacklist.txt")
	}

	// render blacklist edit page
	ViewRenderer.MustLoadTemplate(w, r, "server/blacklist.html", map[string]interface{}{
		"text": string(b),
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
		var fileDecoded []byte

		if file.Size > 0 {
			// zero-size files will still be created, just with no content. (some data files exist but are empty)
			var err error
			fileDecoded, err = base64.StdEncoding.DecodeString(base64HeaderRegex.ReplaceAllString(file.Data, ""))

			if err != nil {
				logrus.Errorf("could not decode "+contentType+" file data, err: %s", err)
				return err
			}
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
		err := os.MkdirAll(filepath.Dir(path), 0755)

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

func (ms *MultiServer) serverLogsHandler(w http.ResponseWriter, r *http.Request) {
	ViewRenderer.MustLoadTemplate(w, r, "server/logs.html", nil)
}

type logData struct {
	ServerLog, ManagerLog, PluginsLog string
}

func (ms *MultiServer) apiServerLogHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(logData{
		ServerLog:  ms.process.Logs(),
		ManagerLog: logOutput.String(),
		PluginsLog: pluginsOutput.String(),
	})
}

func (ms *MultiServer) liveTimingHandler(w http.ResponseWriter, r *http.Request) {
	currentRace, entryList := ms.raceManager.CurrentRace()

	var customRace *CustomRace

	if currentRace != nil {
		customRace = &CustomRace{EntryList: entryList, RaceConfig: currentRace.CurrentRaceConfig}
	}

	frameLinks, err := ms.raceManager.GetLiveFrames()

	if err != nil {
		logrus.Errorf("could not get frame links, err: %s", err)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "live-timing.html", map[string]interface{}{
		"RaceDetails": customRace,
		"FrameLinks":  frameLinks,
	})
}

func (ms *MultiServer) liveTimingGetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(liveInfo)
}

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func (ms *MultiServer) liveFrameSaveHandler(w http.ResponseWriter, r *http.Request) {
	// Save the frame links from the form
	err := r.ParseForm()

	if err != nil {
		logrus.Errorf("could not load parse form, err: %s", err)
		return
	}

	err = ms.raceManager.UpsertLiveFrames(deleteEmpty(r.Form["frame-link"]))

	if err != nil {
		logrus.Errorf("could not save frame links, err: %s", err)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
