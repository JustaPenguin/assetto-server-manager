package servermanager

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cj123/sessions"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

var (
	logOutput     = newLogBuffer(MaxLogSizeBytes)
	pluginsOutput = newLogBuffer(MaxLogSizeBytes)

	logMultiWriter io.Writer

	Debug = os.Getenv("DEBUG") == "true"
)

func InitLogging() {
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

func Router(
	fs http.FileSystem,
	quickRaceHandler *QuickRaceHandler,
	customRaceHandler *CustomRaceHandler,
	championshipsHandler *ChampionshipsHandler,
	accountHandler *AccountHandler,
	auditLogHandler *AuditLogHandler,
	carsHandler *CarsHandler,
	tracksHandler *TracksHandler,
	weatherHandler *WeatherHandler,
	penaltiesHandler *PenaltiesHandler,
	resultsHandler *ResultsHandler,
	serverAdministrationHandler *ServerAdministrationHandler,
	raceControlHandler *RaceControlHandler,
	raceScheduler *ScheduledRacesHandler,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(panicHandler)

	r.HandleFunc("/login", accountHandler.login)
	r.HandleFunc("/logout", accountHandler.logout)
	r.Handle("/metrics", prometheusMonitoringHandler())

	if Debug {
		r.Mount("/debug/", middleware.Profiler())
	}

	// readers
	r.Group(func(r chi.Router) {
		r.Use(accountHandler.ReadAccessMiddleware)

		// pages
		r.Get("/", serverAdministrationHandler.home)

		// content
		r.Get("/cars", carsHandler.list)
		r.Get("/car/{car_id}", carsHandler.view)
		r.Get("/tracks", tracksHandler.list)
		r.Get("/weather", weatherHandler.list)

		r.Get("/events.ics", raceScheduler.allScheduledRacesICalHandler)

		// results
		r.Get("/results", resultsHandler.list)
		r.Get("/results/{fileName}", resultsHandler.view)
		r.HandleFunc("/results/download/{fileName}", resultsHandler.file)

		// championships
		r.Get("/championships", championshipsHandler.list)
		r.Get("/championship/{championshipID}", championshipsHandler.view)
		r.Get("/championship/{championshipID}/export", championshipsHandler.export)
		r.HandleFunc("/championship/{championshipID}/export-results", championshipsHandler.exportResults)
		r.Get("/championship/{championshipID}/ics", championshipsHandler.icalFeed)
		r.Get("/championship/{championshipID}/sign-up", championshipsHandler.signUpForm)
		r.Post("/championship/{championshipID}/sign-up", championshipsHandler.signUpForm)

		// race control
		r.Get("/live-timing", raceControlHandler.liveTiming)
		r.Get("/api/race-control", raceControlHandler.websocket)

		// account management
		r.HandleFunc("/accounts/new-password", accountHandler.newPassword)

		FileServer(r, "/content", http.Dir(filepath.Join(ServerInstallPath, "content")))
		FileServer(r, "/setups/download", http.Dir(filepath.Join(ServerInstallPath, "setups")))
	})

	// writers
	r.Group(func(r chi.Router) {
		r.Use(accountHandler.WriteAccessMiddleware)
		if config.Server.AuditLogging {
			r.Use(auditLogHandler.Middleware)
		}

		// content
		r.Post("/setups/upload", carSetupsUploadHandler)
		r.HandleFunc("/car/{car_id}/tags", carsHandler.tags)

		// races
		r.Get("/quick", quickRaceHandler.create)
		r.Post("/quick/submit", quickRaceHandler.submit)
		r.Get("/custom", customRaceHandler.list)
		r.Get("/custom/new", customRaceHandler.createOrEdit)
		r.Get("/custom/load/{uuid}", customRaceHandler.start)
		r.Post("/custom/schedule/{uuid}", customRaceHandler.schedule)
		r.Get("/custom/schedule/{uuid}/remove", customRaceHandler.removeSchedule)
		r.Get("/custom/edit/{uuid}", customRaceHandler.createOrEdit)
		r.Get("/custom/star/{uuid}", customRaceHandler.star)
		r.Get("/custom/loop/{uuid}", customRaceHandler.loop)
		r.Post("/custom/new/submit", customRaceHandler.submit)

		// server management
		r.Get("/process/{action}", serverAdministrationHandler.serverProcess)
		r.Get("/logs", serverAdministrationHandler.logs)
		r.Get("/api/logs", serverAdministrationHandler.logsAPI)

		// championships
		r.Get("/championships/new", championshipsHandler.createOrEdit)
		r.Post("/championships/new/submit", championshipsHandler.submit)
		r.Get("/championship/{championshipID}/edit", championshipsHandler.createOrEdit)
		r.Get("/championship/{championshipID}/event", championshipsHandler.eventConfiguration)
		r.Post("/championship/{championshipID}/event/submit", championshipsHandler.submitEventConfiguration)
		r.Get("/championship/{championshipID}/event/{eventID}/start", championshipsHandler.startEvent)
		r.Post("/championship/{championshipID}/event/{eventID}/schedule", championshipsHandler.scheduleEvent)
		r.Get("/championship/{championshipID}/event/{eventID}/schedule/remove", championshipsHandler.scheduleEventRemove)
		r.Get("/championship/{championshipID}/event/{eventID}/edit", championshipsHandler.eventConfiguration)
		r.Get("/championship/{championshipID}/event/{eventID}/practice", championshipsHandler.startPracticeEvent)
		r.Get("/championship/{championshipID}/event/{eventID}/cancel", championshipsHandler.cancelEvent)
		r.Get("/championship/{championshipID}/event/{eventID}/restart", championshipsHandler.restartEvent)
		r.Post("/championship/{championshipID}/driver-penalty/{classID}/{driverGUID}", championshipsHandler.driverPenalty)
		r.Post("/championship/{championshipID}/team-penalty/{classID}/{team}", championshipsHandler.teamPenalty)
		r.Get("/championship/{championshipID}/entrants", championshipsHandler.signedUpEntrants)
		r.Get("/championship/{championshipID}/entrants.csv", championshipsHandler.signedUpEntrantsCSV)
		r.Get("/championship/{championshipID}/entrant/{entrantGUID}", championshipsHandler.modifyEntrantStatus)

		r.Get("/championship/import", championshipsHandler.importChampionship)
		r.Post("/championship/import", championshipsHandler.importChampionship)
		r.Get("/championship/{championshipID}/event/{eventID}/import", championshipsHandler.eventImport)
		r.Post("/championship/{championshipID}/event/{eventID}/import", championshipsHandler.eventImport)

		// penalties
		r.Post("/penalties/{sessionFile}/{driverGUID}", penaltiesHandler.managePenalty)

		// live timings
		r.Post("/live-timing/save-frames", raceControlHandler.saveIFrames)

		// endpoints
		r.Post("/api/track/upload", uploadHandler("Track"))
		r.Post("/api/car/upload", uploadHandler("Car"))
		r.Post("/api/weather/upload", uploadHandler("Weather"))
	})

	// deleters
	r.Group(func(r chi.Router) {
		r.Use(accountHandler.DeleteAccessMiddleware)
		if config.Server.AuditLogging {
			r.Use(auditLogHandler.Middleware)
		}

		r.Get("/championship/{championshipID}/event/{eventID}/delete", championshipsHandler.deleteEvent)
		r.Get("/championship/{championshipID}/delete", championshipsHandler.delete)
		r.Get("/custom/delete/{uuid}", customRaceHandler.delete)

		r.Get("/track/delete/{name}", tracksHandler.delete)
		r.Get("/car/delete/{name}", carsHandler.delete)
		r.Get("/weather/delete/{key}", weatherHandler.delete)
		r.Get("/setups/delete/{car}/{track}/{setup}", carSetupDeleteHandler)

		r.Get("/autofill-entrants", serverAdministrationHandler.autoFillEntrantList)
		r.Get("/autofill-entrants/delete/{entrantID}", serverAdministrationHandler.autoFillEntrantDelete)
	})

	// admins
	r.Group(func(r chi.Router) {
		r.Use(accountHandler.AdminAccessMiddleware)
		if config.Server.AuditLogging {
			r.Use(auditLogHandler.Middleware)
		}

		r.HandleFunc("/server-options", serverAdministrationHandler.options)
		r.HandleFunc("/blacklist", serverAdministrationHandler.blacklist)
		r.HandleFunc("/motd", serverAdministrationHandler.motd)
		r.HandleFunc("/audit-logs", auditLogHandler.viewLogs)
		r.HandleFunc("/accounts/new", accountHandler.createOrEditAccount)
		r.HandleFunc("/accounts/edit/{id}", accountHandler.createOrEditAccount)
		r.HandleFunc("/accounts/delete/{id}", accountHandler.deleteAccount)
		r.HandleFunc("/accounts/reset-password/{id}", accountHandler.resetPassword)
		r.HandleFunc("/accounts/toggle-open", accountHandler.toggleServerOpenStatus)
		r.HandleFunc("/accounts", accountHandler.manageAccounts)
	})

	FileServer(r, "/static", fs)

	return prometheusMonitoringWrapper(r)
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

var sessionsStore sessions.Store

func getSession(r *http.Request) *sessions.Session {
	session, _ := sessionsStore.Get(r, "messages")

	return session
}

func getErrSession(r *http.Request) *sessions.Session {
	session, _ := sessionsStore.Get(r, "errors")

	return session
}

// Helper function to get message session and add a flash
func AddFlash(w http.ResponseWriter, r *http.Request, message string) {
	session := getSession(r)

	session.AddFlash(message)

	// gorilla sessions is dumb and errors weirdly
	_ = session.Save(r, w)
}

func AddErrorFlash(w http.ResponseWriter, r *http.Request, message string) {
	session := getErrSession(r)

	session.AddFlash(message)

	// gorilla sessions is dumb and errors weirdly
	_ = session.Save(r, w)
}
