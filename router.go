package servermanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cj123/sessions"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-http-utils/etag"
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
	multiServerManager *MultiServerManager,
	accountHandler *AccountHandler,
	auditLogHandler *AuditLogHandler,
	carsHandler *CarsHandler,
	tracksHandler *TracksHandler,
	weatherHandler *WeatherHandler,
	resultsHandler *ResultsHandler,
	contentUploadHandler *ContentUploadHandler,
	scheduledRacesHandler *ScheduledRacesHandler,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.DefaultCompress)
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

		// content
		r.Get("/cars", carsHandler.list)
		r.Get("/cars/search.json", carsHandler.searchJSON)
		r.Get("/car/{car_id}", carsHandler.view)
		r.Get("/tracks", tracksHandler.list)
		r.Get("/weather", weatherHandler.list)

		r.Get("/events.ics", scheduledRacesHandler.allScheduledRacesICalHandler)

		// results
		r.Get("/results", resultsHandler.list)
		r.Get("/results/{fileName}", resultsHandler.view)
		r.HandleFunc("/results/{fileName}/collisions", resultsHandler.renderCollisions)
		r.HandleFunc("/results/download/{fileName}", resultsHandler.file)

		// calendar
		r.Get("/calendar", scheduledRacesHandler.calendar)
		r.Get("/calendar.json", scheduledRacesHandler.calendarJSON)

		// account management
		r.HandleFunc("/accounts/new-password", accountHandler.newPassword)
		r.HandleFunc("/accounts/update", accountHandler.update)
		r.Get("/accounts/update/steam", accountHandler.redirectToSteamLogin(func(r *http.Request) string {
			return "/accounts/update"
		}))
		r.HandleFunc("/accounts/dismiss-changelog", accountHandler.dismissChangelog)

		FileServer(r, "/content", http.Dir(filepath.Join(ServerInstallPath, "content")), true)
		FileServer(r, "/setups/download", http.Dir(filepath.Join(ServerInstallPath, "setups")), true)
	})

	// writers
	r.Group(func(r chi.Router) {
		r.Use(accountHandler.WriteAccessMiddleware)
		if config.Server.AuditLogging {
			r.Use(auditLogHandler.Middleware)
		}

		// content
		r.Post("/setups/upload", carSetupsUploadHandler)
		r.HandleFunc("/car/{name}/tags", carsHandler.tags)
		r.Post("/car/{name}/metadata", carsHandler.saveMetadata)
		r.Post("/car/{name}/skin", carsHandler.uploadSkin)

		// results
		r.Post("/results/{fileName}/edit", resultsHandler.edit)

		// endpoints
		r.Post("/api/track/upload", contentUploadHandler.upload(ContentTypeTrack))
		r.Post("/api/car/upload", contentUploadHandler.upload(ContentTypeCar))
		r.Post("/api/weather/upload", contentUploadHandler.upload(ContentTypeWeather))

	})

	// deleters
	r.Group(func(r chi.Router) {
		r.Use(accountHandler.DeleteAccessMiddleware)
		if config.Server.AuditLogging {
			r.Use(auditLogHandler.Middleware)
		}

		r.Get("/track/delete/{name}", tracksHandler.delete)
		r.Get("/car/{name}/delete", carsHandler.delete)
		r.Post("/car/{name}/skin/delete", carsHandler.deleteSkin)
		r.Get("/weather/delete/{key}", weatherHandler.delete)
		r.Get("/setups/delete/{car}/{track}/{setup}", carSetupDeleteHandler)

	})

	// admins
	r.Group(func(r chi.Router) {
		r.Use(accountHandler.AdminAccessMiddleware)
		if config.Server.AuditLogging {
			r.Use(auditLogHandler.Middleware)
		}

		r.HandleFunc("/audit-logs", auditLogHandler.viewLogs)
		r.HandleFunc("/accounts/new", accountHandler.createOrEditAccount)
		r.HandleFunc("/accounts/edit/{id}", accountHandler.createOrEditAccount)
		r.HandleFunc("/accounts/delete/{id}", accountHandler.deleteAccount)
		r.HandleFunc("/accounts/reset-password/{id}", accountHandler.resetPassword)
		r.HandleFunc("/accounts/toggle-open", accountHandler.toggleServerOpenStatus)
		r.HandleFunc("/accounts", accountHandler.manageAccounts)
		r.HandleFunc("/search-index", carsHandler.rebuildSearchIndex)

	})

	FileServer(r, "/static", fs, false)

	// routes which are not found are passed on to multiserver handling.
	r.NotFound(multiServerManager.ServerHandler)

	return prometheusMonitoringWrapper(r)
}

func FileServer(r chi.Router, path string, root http.FileSystem, useRevalidation bool) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, AssetCacheHeaders(fs.ServeHTTP, useRevalidation))
}

const maxAge30Days = 2592000

func AssetCacheHeaders(next http.HandlerFunc, useRevalidation bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if useRevalidation {
			w.Header().Add("Cache-Control", fmt.Sprintf("public, must-revalidate"))
			etag.Handler(next, false).ServeHTTP(w, r)
		} else {
			w.Header().Add("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge30Days))

			next(w, r)
		}
	}
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
