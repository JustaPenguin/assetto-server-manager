package servermanager

import (
<<<<<<< HEAD
	"errors"
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/go-chi/chi"
	"net/http"
)

var servers = make(map[int]*MultiServer)

type MultiServer struct {
	process             ServerProcess
	raceManager         *RaceManager
	championshipManager *ChampionshipManager
	liveMapHub          *liveMapHub

	raceLooper    *RaceLooper
	raceScheduler *RaceScheduler
}

func NewMultiServer(store Store) (*MultiServer, error) {
	proc := NewAssettoServerProcess()
	raceManager := NewRaceManager(store, proc)
	championshipManager := NewChampionshipManager(raceManager)

	raceLooper := NewRaceLooper(proc, raceManager)
	raceScheduler := NewRaceScheduler(championshipManager)

	if err := raceScheduler.InitialiseScheduledCustomRaces(); err != nil {
		return nil, err
	}

	if err := raceScheduler.InitialiseScheduledChampionshipEvents(); err != nil {
		return nil, err
	}

	mapHub := newLiveMapHub(proc)

	// @TODO destroying a server should stop mapHub + loops + schedules
	go mapHub.run()
	go raceLooper.LoopRaces()

	ms := &MultiServer{
		process:             proc,
		raceManager:         raceManager,
		championshipManager: championshipManager,
		liveMapHub:          mapHub,
		raceLooper:          raceLooper,
		raceScheduler:       raceScheduler,
	}

	servers[len(servers)] = ms

	proc.SetUDPCallback(func(message udp.Message) {
		panicCapture(func() {
			if config != nil && config.LiveMap.IsEnabled() {
				go ms.LiveMapCallback(message)
			}

			championshipManager.ChampionshipEventCallback(message)
			ms.LiveTimingCallback(message)
			ms.raceLooper.LoopCallback(message)
		})
	})

	return ms, nil
}

var ErrServerNotFound = errors.New("servermanager: server not found")

func GetProcess(serverNum int) (ServerProcess, error) {
	if s, ok := servers[serverNum]; ok {
		return s.process, nil
	}

	return nil, ErrServerNotFound
}

func AnyServer() *MultiServer {
	for _, s := range servers {
		return s
	}

	return nil
}

func MultiServerSelectMiddleware(router chi.Router, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if false {
			//sess := getSession(r)
		} else {
			previousURL := *r.URL

			// otherwise just rewrite the URL to server 0
			r.URL.Path = "/server-0" + r.URL.Path

			if router.Match(chi.NewRouteContext(), r.Method, r.URL.Path) {
				next.ServeHTTP(w, r)
			} else {
				*r.URL = previousURL
				next.ServeHTTP(w, r)
			}
		}
	})
}
=======
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

type MultiServerManager struct {
	store               Store
	carManager          *CarManager
	notificationManager *NotificationManager
	baseHandler         *BaseHandler
	accountHandler      *AccountHandler
}

func NewMultiServerManager(store Store, carManager *CarManager, notificationManager *NotificationManager, baseHandler *BaseHandler, accountHandler *AccountHandler) *MultiServerManager {
	return &MultiServerManager{
		store:               store,
		carManager:          carManager,
		notificationManager: notificationManager,
		baseHandler:         baseHandler,
		accountHandler:      accountHandler,
	}
}

type Server struct {
	ID           uuid.UUID
	Created      time.Time
	Updated      time.Time
	Deleted      time.Time
	ServerConfig GlobalServerConfig

	Process ServerProcess `json:"-"`

	// Managers
	RaceManager           *RaceManager           `json:"-"`
	ChampionshipManager   *ChampionshipManager   `json:"-"`
	RaceWeekendManager    *RaceWeekendManager    `json:"-"`
	RaceControl           *RaceControl           `json:"-"`
	ContentManagerWrapper *ContentManagerWrapper `json:"-"`

	// Handlers
	QuickRaceHandler            *QuickRaceHandler            `json:"-"`
	CustomRaceHandler           *CustomRaceHandler           `json:"-"`
	ChampionshipsHandler        *ChampionshipsHandler        `json:"-"`
	RaceWeekendHandler          *RaceWeekendHandler          `json:"-"`
	RaceControlHandler          *RaceControlHandler          `json:"-"`
	AccountHandler              *AccountHandler              `json:"-"`
	ServerAdministrationHandler *ServerAdministrationHandler `json:"-"`
	PenaltiesHandler            *PenaltiesHandler            `json:"-"`
}

func (msm *MultiServerManager) NewServer(serverConfig GlobalServerConfig) (*Server, error) {
	server := &Server{}
	server.ID = uuid.New()
	server.Created = time.Now()
	server.ServerConfig = serverConfig

	server.ContentManagerWrapper = NewContentManagerWrapper(msm.store, msm.carManager)
	server.Process = NewAssettoServerProcess(server.UDPCallback, server.ContentManagerWrapper)
	server.RaceManager = NewRaceManager(msm.store, server.Process, msm.carManager, msm.notificationManager)
	server.ChampionshipManager = NewChampionshipManager(server.RaceManager)
	server.RaceWeekendManager = NewRaceWeekendManager(server.RaceManager, server.ChampionshipManager, msm.store, server.Process, msm.notificationManager)

	raceControlHub := newRaceControlHub()

	server.RaceControl = NewRaceControl(raceControlHub, filesystemTrackData{}, server.Process)

	server.AccountHandler = msm.accountHandler
	server.QuickRaceHandler = NewQuickRaceHandler(msm.baseHandler, server.RaceManager)
	server.CustomRaceHandler = NewCustomRaceHandler(msm.baseHandler, server.RaceManager)
	server.ChampionshipsHandler = NewChampionshipsHandler(msm.baseHandler, server.ChampionshipManager)
	server.RaceWeekendHandler = NewRaceWeekendHandler(msm.baseHandler, server.RaceWeekendManager)
	server.RaceControlHandler = NewRaceControlHandler(msm.baseHandler, msm.store, server.RaceManager, server.RaceControl, raceControlHub, server.Process)
	server.ServerAdministrationHandler = NewServerAdministrationHandler(msm.baseHandler, msm.store, server.RaceManager, server.ChampionshipManager, server.RaceWeekendManager, server.Process)
	server.PenaltiesHandler = NewPenaltiesHandler(msm.baseHandler, server.ChampionshipManager, server.RaceWeekendManager)

	if err := msm.store.UpsertServer(server); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Server) UDPCallback(message udp.Message) {
	if !config.Server.PerformanceMode {
		s.RaceControl.UDPCallback(message)
	}
	s.ChampionshipManager.ChampionshipEventCallback(message)
	s.RaceWeekendManager.UDPCallback(message)
	s.RaceManager.LoopCallback(message)
	s.ContentManagerWrapper.UDPCallback(message)
}

func (s *Server) Router() chi.Router {
	// @TODO audit logging
	r := chi.NewRouter()

	// readers
	r.Group(func(r chi.Router) {
		r.Use(s.AccountHandler.ReadAccessMiddleware)

		// pages
		r.Get("/", s.ServerAdministrationHandler.home)
		r.Get("/changelog", s.ServerAdministrationHandler.changelog)

		// championships
		r.Get("/championships", s.ChampionshipsHandler.list)
		r.Get("/championship/{championshipID}", s.ChampionshipsHandler.view)
		r.Get("/championship/{championshipID}/export", s.ChampionshipsHandler.export)
		r.HandleFunc("/championship/{championshipID}/export-results", s.ChampionshipsHandler.exportResults)
		r.Get("/championship/{championshipID}/ics", s.ChampionshipsHandler.icalFeed)
		r.Get("/championship/{championshipID}/sign-up", s.ChampionshipsHandler.signUpForm)
		r.Post("/championship/{championshipID}/sign-up", s.ChampionshipsHandler.signUpForm)
		r.Get("/championship/{championshipID}/sign-up/steam", s.ChampionshipsHandler.redirectToSteamLogin(func(r *http.Request) string {
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

			r.Get("/live-timing", s.RaceControlHandler.liveTiming)
			r.Get("/api/race-control", s.RaceControlHandler.websocket)
		})

		// race weekends
		r.Get("/race-weekends", s.RaceWeekendHandler.list)
		r.Get("/race-weekend/{raceWeekendID}", s.RaceWeekendHandler.view)
		r.Get("/race-weekend/{raceWeekendID}/filters", s.RaceWeekendHandler.manageFilters)
		r.Get("/race-weekend/{raceWeekendID}/entrylist", s.RaceWeekendHandler.manageEntryList)
		r.Post("/race-weekend/{raceWeekendID}/grid-preview", s.RaceWeekendHandler.gridPreview)
		r.Get("/race-weekend/{raceWeekendID}/entrylist-preview", s.RaceWeekendHandler.entryListPreview)
		r.Get("/race-weekend/{raceWeekendID}/export", s.RaceWeekendHandler.export)
	})

	// writers
	r.Group(func(r chi.Router) {
		r.Use(s.AccountHandler.WriteAccessMiddleware)

		// races
		r.Get("/quick", s.QuickRaceHandler.create)
		r.Post("/quick/submit", s.QuickRaceHandler.submit)
		r.Get("/custom", s.CustomRaceHandler.list)
		r.Get("/custom/new", s.CustomRaceHandler.createOrEdit)
		r.Get("/custom/star/{uuid}", s.CustomRaceHandler.star)
		r.Get("/custom/loop/{uuid}", s.CustomRaceHandler.loop)
		r.Get("/custom/load/{uuid}", s.CustomRaceHandler.start)
		r.Post("/custom/schedule/{uuid}", s.CustomRaceHandler.schedule)
		r.Get("/custom/schedule/{uuid}/remove", s.CustomRaceHandler.removeSchedule)
		r.Get("/custom/edit/{uuid}", s.CustomRaceHandler.createOrEdit)
		r.Post("/custom/new/submit", s.CustomRaceHandler.submit)

		// championships
		r.Get("/championships/new", s.ChampionshipsHandler.createOrEdit)
		r.Post("/championships/new/submit", s.ChampionshipsHandler.submit)
		r.Get("/championship/{championshipID}/edit", s.ChampionshipsHandler.createOrEdit)
		r.Get("/championship/{championshipID}/event", s.ChampionshipsHandler.eventConfiguration)
		r.Post("/championship/{championshipID}/event/submit", s.ChampionshipsHandler.submitEventConfiguration)

		r.Get("/championship/{championshipID}/event/{eventID}/edit", s.ChampionshipsHandler.eventConfiguration)
		r.Post("/championship/{championshipID}/driver-penalty/{classID}/{driverGUID}", s.ChampionshipsHandler.driverPenalty)
		r.Post("/championship/{championshipID}/team-penalty/{classID}/{team}", s.ChampionshipsHandler.teamPenalty)
		r.Get("/championship/{championshipID}/entrants", s.ChampionshipsHandler.signedUpEntrants)
		r.Get("/championship/{championshipID}/entrants.csv", s.ChampionshipsHandler.signedUpEntrantsCSV)
		r.Get("/championship/{championshipID}/entrant/{entrantGUID}", s.ChampionshipsHandler.modifyEntrantStatus)
		r.Post("/championship/{championshipID}/reorder-events", s.ChampionshipsHandler.reorderEvents)

		r.Get("/championship/import", s.ChampionshipsHandler.importChampionship)
		r.Post("/championship/import", s.ChampionshipsHandler.importChampionship)
		r.Get("/championship/{championshipID}/event/{eventID}/import", s.ChampionshipsHandler.eventImport)
		r.Post("/championship/{championshipID}/event/{eventID}/import", s.ChampionshipsHandler.eventImport)
		r.Get("/championship/{championshipID}/event/{eventID}/start", s.ChampionshipsHandler.startEvent)
		r.Post("/championship/{championshipID}/event/{eventID}/schedule", s.ChampionshipsHandler.scheduleEvent)
		r.Get("/championship/{championshipID}/event/{eventID}/schedule/remove", s.ChampionshipsHandler.scheduleEventRemove)
		r.Get("/championship/{championshipID}/event/{eventID}/practice", s.ChampionshipsHandler.startPracticeEvent)
		r.Get("/championship/{championshipID}/event/{eventID}/cancel", s.ChampionshipsHandler.cancelEvent)
		r.Get("/championship/{championshipID}/event/{eventID}/restart", s.ChampionshipsHandler.restartEvent)

		// server management
		r.Get("/process/{action}", s.ServerAdministrationHandler.serverProcess)
		r.Get("/logs", s.ServerAdministrationHandler.logs)
		r.Get("/api/logs", s.ServerAdministrationHandler.logsAPI)

		// penalties
		r.Post("/penalties/{sessionFile}/{driverGUID}", s.PenaltiesHandler.managePenalty)

		// race weekends
		r.Get("/race-weekends/new", s.RaceWeekendHandler.createOrEdit)
		r.Post("/race-weekends/new/submit", s.RaceWeekendHandler.submit)
		r.Get("/race-weekend/{raceWeekendID}/delete", s.RaceWeekendHandler.delete)
		r.Get("/race-weekend/{raceWeekendID}/edit", s.RaceWeekendHandler.createOrEdit)
		r.Get("/race-weekend/{raceWeekendID}/session", s.RaceWeekendHandler.sessionConfiguration)
		r.Post("/race-weekend/{raceWeekendID}/session/submit", s.RaceWeekendHandler.submitSessionConfiguration)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/edit", s.RaceWeekendHandler.sessionConfiguration)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/import", s.RaceWeekendHandler.importSessionResults)
		r.Post("/race-weekend/{raceWeekendID}/session/{sessionID}/import", s.RaceWeekendHandler.importSessionResults)
		r.Post("/race-weekend/{raceWeekendID}/update-grid", s.RaceWeekendHandler.updateGrid)
		r.Get("/race-weekend/{raceWeekendID}/update-entrylist", s.RaceWeekendHandler.updateEntryList)
		r.Get("/race-weekend/import", s.RaceWeekendHandler.importRaceWeekend)
		r.Post("/race-weekend/import", s.RaceWeekendHandler.importRaceWeekend)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/start", s.RaceWeekendHandler.startSession)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/practice", s.RaceWeekendHandler.startPracticeSession)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/restart", s.RaceWeekendHandler.restartSession)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/cancel", s.RaceWeekendHandler.cancelSession)
		r.Post("/race-weekend/{raceWeekendID}/session/{sessionID}/schedule", s.RaceWeekendHandler.scheduleSession)
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/schedule/remove", s.RaceWeekendHandler.removeSessionSchedule)

		// live timings
		r.Post("/live-timing/save-frames", s.RaceControlHandler.saveIFrames)
	})

	// deleters
	r.Group(func(r chi.Router) {
		r.Use(s.AccountHandler.DeleteAccessMiddleware)

		r.Get("/championship/{championshipID}/event/{eventID}/delete", s.ChampionshipsHandler.deleteEvent)
		r.Get("/championship/{championshipID}/delete", s.ChampionshipsHandler.delete)
		r.Get("/custom/delete/{uuid}", s.CustomRaceHandler.delete)

		r.Get("/autofill-entrants", s.ServerAdministrationHandler.autoFillEntrantList)
		r.Get("/autofill-entrants/delete/{entrantID}", s.ServerAdministrationHandler.autoFillEntrantDelete)

		// race weekend
		r.Get("/race-weekend/{raceWeekendID}/session/{sessionID}/delete", s.RaceWeekendHandler.deleteSession)
	})

	// admins
	r.Group(func(r chi.Router) {
		r.Use(s.AccountHandler.AdminAccessMiddleware)

		r.HandleFunc("/restart-session", s.RaceControlHandler.restartSession)
		r.HandleFunc("/next-session", s.RaceControlHandler.nextSession)
		r.HandleFunc("/broadcast-chat", s.RaceControlHandler.broadcastChat)
		r.HandleFunc("/admin-command", s.RaceControlHandler.adminCommand)
		r.HandleFunc("/kick-user", s.RaceControlHandler.kickUser)

		r.HandleFunc("/server-options", s.ServerAdministrationHandler.options)
		r.HandleFunc("/blacklist", s.ServerAdministrationHandler.blacklist)
		r.HandleFunc("/motd", s.ServerAdministrationHandler.motd)
	})

	return r
}

const sessionServerIDKey = "ServerID"

func (msm *MultiServerManager) ServerHandler(w http.ResponseWriter, r *http.Request) {
	servers, err := msm.store.ListServers()

	if err != nil {
		logrus.WithError(err).Error("Could not list servers")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sess := getSession(r)

	serverID, ok := sess.Values[sessionServerIDKey]

	if ok {
		for _, server := range servers {
			if server.ID.String() == serverID.(string) {
				server.Router().ServeHTTP(w, r)
				return
			}
		}
	}

	for _, server := range servers {
		server.Router().ServeHTTP(w, r)
		return
	}
}

func (msm *MultiServerManager) ChooseServerHandler(w http.ResponseWriter, r *http.Request) {
	sess := getSession(r)
	sess.Values[sessionServerIDKey] = r.URL.Query().Get("serverID")
	_ = sess.Save(r, w)

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
>>>>>>> multiserver3
