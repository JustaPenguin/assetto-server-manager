package servermanager

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	contentUploadHandler *ContentUploadHandler,
	serverAdministrationHandler *ServerAdministrationHandler,
	raceControlHandler *RaceControlHandler,
	scheduledRacesHandler *ScheduledRacesHandler,
	raceWeekendHandler *RaceWeekendHandler,
	strackerHandler *StrackerHandler,
	healthCheck *HealthCheck,
	kissMyRankHandler *KissMyRankHandler,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.DefaultCompress)
	r.Use(panicHandler)

	r.HandleFunc("/login", accountHandler.login)
	r.HandleFunc("/logout", accountHandler.logout)
	r.HandleFunc("/robots.txt", serverAdministrationHandler.robots)
	r.Handle("/metrics", prometheusMonitoringHandler())
	r.Get("/healthcheck.json", healthCheck.ServeHTTP)

	if Debug {
		r.Mount("/debug/", middleware.Profiler())
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// if a user comes from an stracker page and hits a 404, the likelihood is that they found a link that does
		// not have an stracker prefix added to it. Catch it, and forward them back to an URL that has the prefix.
		u, err := url.Parse(r.Referer())

		if err == nil && strings.HasPrefix(u.Path, "/stracker/") {
			// try to redirect back to /stracker/<url request>
			r.URL.Path = "/stracker" + r.URL.Path

			http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
			return
		}

		http.NotFound(w, r)
	})

	// readers
	r.Group(func(r chi.Router) {
		r.Use(accountHandler.ReadAccessMiddleware)

		// pages
		r.Get("/", serverAdministrationHandler.home)
		r.Get("/changelog", serverAdministrationHandler.changelog)
		r.Get("/premium", serverAdministrationHandler.premium)

		r.Mount("/stracker/", http.HandlerFunc(strackerHandler.proxy))

		// content
		r.Get("/cars", carsHandler.list)
		r.Get("/cars/search.json", carsHandler.searchJSON)
		r.Get("/car/{car_id}", carsHandler.view)
		r.Get("/tracks", tracksHandler.list)
		r.Get("/track/{track_id}", tracksHandler.view)
		r.Get("/weather", weatherHandler.list)

		r.Get("/events.ics", scheduledRacesHandler.allScheduledRacesICalHandler)

		// results
		r.Get("/results", resultsHandler.list)
		r.Get("/results/{fileName}", resultsHandler.view)
		r.HandleFunc("/results/{fileName}/collisions", resultsHandler.renderCollisions)
		r.HandleFunc("/results/download/{fileName}", resultsHandler.file)

		// championships
		r.Get("/championships", championshipsHandler.list)
		r.Get("/championship/{championshipID}", championshipsHandler.view)
		r.Get("/championship/{championshipID}/export", championshipsHandler.export)
		r.HandleFunc("/championship/{championshipID}/export-results", championshipsHandler.exportResults)
		r.Get("/championship/{championshipID}/ics", championshipsHandler.icalFeed)
		r.Get("/championship/{championshipID}/sign-up", championshipsHandler.signUpForm)
		r.Post("/championship/{championshipID}/sign-up", championshipsHandler.signUpForm)
		r.Get("/championship/{championshipID}/sign-up/steam", championshipsHandler.redirectToSteamLogin(func(r *http.Request) string {
			return fmt.Sprintf("/championship/%s/sign-up", chi.URLParam(r, "championshipID"))
		}))

		// race control
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				fn := func(w http.ResponseWriter, req *http.Request) {
					if config.Server.PerformanceMode {
						http.NotFound(w, req)
					} else {
						next.ServeHTTP(w, req)
					}
				}

				return http.HandlerFunc(fn)
			})

			r.Get("/live-timing", raceControlHandler.liveTiming)
			r.Get("/api/race-control", raceControlHandler.websocket)
		})

		// calendar
		r.Get("/calendar", scheduledRacesHandler.calendar)
		r.Get("/calendar.json", scheduledRacesHandler.calendarJSON)

		// account management
		r.HandleFunc("/accounts/new-password", accountHandler.newPassword)
		r.HandleFunc("/accounts/update", accountHandler.update)
		r.Get("/accounts/update/steam", championshipsHandler.redirectToSteamLogin(func(r *http.Request) string {
			return "/accounts/update"
		}))
		r.HandleFunc("/accounts/dismiss-changelog", accountHandler.dismissChangelog)

		FileServer(r, "/content", http.Dir(filepath.Join(ServerInstallPath, "content")), true)
		FileServer(r, "/setups/download", http.Dir(filepath.Join(ServerInstallPath, "setups")), true)

		// race weekends
		r.Get("/race-weekends", raceWeekendHandler.list)
		r.Get("/race-weekend/{raceWeekendID}", raceWeekendHandler.view)
		r.Get("/race-weekend/{raceWeekendID}/filters", raceWeekendHandler.manageFilters)
		r.Get("/race-weekend/{raceWeekendID}/entrylist", raceWeekendHandler.manageEntryList)
		r.Post("/race-weekend/{raceWeekendID}/grid-preview", raceWeekendHandler.gridPreview)
		r.Get("/race-weekend/{raceWeekendID}/entrylist-preview", raceWeekendHandler.entryListPreview)
		r.Get("/race-weekend/{raceWeekendID}/export", raceWeekendHandler.export)
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
		r.Post("/track/{name}/metadata", tracksHandler.saveMetadata)
		r.Post("/results/upload", resultsHandler.uploadHandler)

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
		r.Get("/api/log-download/{logFile}", serverAdministrationHandler.logsDownload)

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
		r.Get("/championship/{championshipID}/event/{eventID}/duplicate", championshipsHandler.duplicateEvent)

		r.Post("/championship/{championshipID}/driver-penalty/{classID}/{driverGUID}", championshipsHandler.driverPenalty)
		r.Post("/championship/{championshipID}/team-penalty/{classID}/{team}", championshipsHandler.teamPenalty)
		r.Get("/championship/{championshipID}/entrants", championshipsHandler.signedUpEntrants)
		r.Get("/championship/{championshipID}/entrants.csv", championshipsHandler.signedUpEntrantsCSV)
		r.Get("/championship/{championshipID}/entrant/{entrantGUID}", championshipsHandler.modifyEntrantStatus)
		r.Post("/championship/{championshipID}/reorder-events", championshipsHandler.reorderEvents)

		r.Get("/championship/import", championshipsHandler.importChampionship)
		r.Post("/championship/import", championshipsHandler.importChampionship)
		r.Get("/championship/{championshipID}/event/{eventID}/import", championshipsHandler.eventImport)
		r.Post("/championship/{championshipID}/event/{eventID}/import", championshipsHandler.eventImport)

		r.Get("/championship/{championshipID}/custom/list", championshipsHandler.listCustomRacesForImport)
		r.Get("/championship/{championshipID}/custom/{eventID}/import", championshipsHandler.customRaceImport)
		r.Get("/championship/{championshipID}/race-weekend/list", championshipsHandler.listRaceWeekendsForImport)
		r.Get("/championship/{championshipID}/race-weekend/{weekendID}/import", championshipsHandler.raceWeekendImport)

		// penalties
		r.Post("/penalties/{sessionFile}/{driverGUID}", penaltiesHandler.managePenalty)

		// results
		r.Post("/results/{fileName}/edit", resultsHandler.edit)

		// live timings
		r.Post("/live-timing/save-frames", raceControlHandler.saveIFrames)

		// endpoints
		r.Post("/api/track/upload", contentUploadHandler.upload(ContentTypeTrack))
		r.Post("/api/car/upload", contentUploadHandler.upload(ContentTypeCar))
		r.Post("/api/weather/upload", contentUploadHandler.upload(ContentTypeWeather))

		// race weekend
		r.Get("/race-weekends/new", raceWeekendHandler.createOrEdit)
		r.Post("/race-weekends/new/submit", raceWeekendHandler.submit)
		r.Get("/race-weekend/{raceWeekendID}/delete", raceWeekendHandler.delete)
		r.Get("/race-weekend/{raceWeekendID}/edit", raceWeekendHandler.createOrEdit)
		r.Get("/race-weekend/{raceWeekendID}/session", raceWeekendHandler.sessionConfiguration)
		r.Post("/race-weekend/{raceWeekendID}/session/submit", raceWeekendHandler.submitSessionConfiguration)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/edit", raceWeekendHandler.sessionConfiguration)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/start", raceWeekendHandler.startSession)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/practice", raceWeekendHandler.startPracticeSession)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/restart", raceWeekendHandler.restartSession)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/cancel", raceWeekendHandler.cancelSession)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/import", raceWeekendHandler.importSessionResults)
		r.Post("/race-weekend/{raceWeekendID}/session/{sessionID}/import", raceWeekendHandler.importSessionResults)
		r.Post("/race-weekend/{raceWeekendID}/update-grid", raceWeekendHandler.updateGrid)
		r.Get("/race-weekend/{raceWeekendID}/update-entrylist", raceWeekendHandler.updateEntryList)
		r.Get("/race-weekend/import", raceWeekendHandler.importRaceWeekend)
		r.Post("/race-weekend/import", raceWeekendHandler.importRaceWeekend)
		r.Post("/race-weekend/{raceWeekendID}/session/{sessionID}/schedule", raceWeekendHandler.scheduleSession)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/schedule/remove", raceWeekendHandler.removeSessionSchedule)
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

		r.Get("/track/{name}/delete", tracksHandler.delete)
		r.Get("/car/{name}/delete", carsHandler.delete)
		r.Post("/car/{name}/skin/delete", carsHandler.deleteSkin)
		r.Get("/weather/delete/{key}", weatherHandler.delete)
		r.Get("/setups/delete/{car}/{track}/{setup}", carSetupDeleteHandler)

		r.Get("/autofill-entrants", serverAdministrationHandler.autoFillEntrantList)
		r.Get("/autofill-entrants/delete/{entrantID}", serverAdministrationHandler.autoFillEntrantDelete)

		// race weekend
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/delete", raceWeekendHandler.deleteSession)
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
		r.HandleFunc("/search-index", carsHandler.rebuildSearchIndex)

		r.HandleFunc("/restart-session", raceControlHandler.restartSession)
		r.HandleFunc("/next-session", raceControlHandler.nextSession)
		r.HandleFunc("/broadcast-chat", raceControlHandler.broadcastChat)
		r.HandleFunc("/admin-command", raceControlHandler.adminCommand)
		r.HandleFunc("/kick-user", raceControlHandler.kickUser)
		r.HandleFunc("/send-chat", raceControlHandler.sendChat)

		r.HandleFunc("/stracker/options", strackerHandler.options)
		r.HandleFunc("/kissmyrank/options", kissMyRankHandler.options)
	})

	FileServer(r, "/static", fs, false)

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

	r.Get(path, AssetCacheHeaders(fs, useRevalidation))
}

const maxAge30Days = 2592000

func AssetCacheHeaders(next http.Handler, useRevalidation bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if useRevalidation {
			w.Header().Add("Cache-Control", fmt.Sprintf("public, must-revalidate"))
			etag.Handler(next, false).ServeHTTP(w, r)
		} else {
			w.Header().Add("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge30Days))

			next.ServeHTTP(w, r)
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
