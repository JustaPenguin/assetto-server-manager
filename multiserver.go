package servermanager

import (
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

type MultiServerManager struct {
	store               Store
	carManager          *CarManager
	notificationManager *NotificationManager
	baseHandler         *BaseHandler
	accountHandler      *AccountHandler

	Servers []*Server
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
	ID uuid.UUID

	ServerConfig GlobalServerConfig

	Process ServerProcess

	// Managers
	RaceManager           *RaceManager
	ChampionshipManager   *ChampionshipManager
	RaceWeekendManager    *RaceWeekendManager
	RaceControl           *RaceControl
	ContentManagerWrapper *ContentManagerWrapper

	// Handlers
	QuickRaceHandler            *QuickRaceHandler
	CustomRaceHandler           *CustomRaceHandler
	ChampionshipsHandler        *ChampionshipsHandler
	RaceWeekendHandler          *RaceWeekendHandler
	RaceControlHandler          *RaceControlHandler
	AccountHandler              *AccountHandler
	ServerAdministrationHandler *ServerAdministrationHandler
	PenaltiesHandler            *PenaltiesHandler
}

func (msm *MultiServerManager) NewServer(serverConfig GlobalServerConfig) *Server {
	contentManagerWrapper := NewContentManagerWrapper(msm.store, msm.carManager)
	process := NewAssettoServerProcess(nil, contentManagerWrapper)
	raceManager := NewRaceManager(msm.store, process, msm.carManager, msm.notificationManager)
	championshipManager := NewChampionshipManager(raceManager)
	raceWeekendManager := NewRaceWeekendManager(raceManager, championshipManager, msm.store, process, msm.notificationManager)
	raceControlHub := newRaceControlHub()
	raceControl := NewRaceControl(raceControlHub, filesystemTrackData{}, process)

	quickRaceHandler := NewQuickRaceHandler(msm.baseHandler, raceManager)
	customRaceHandler := NewCustomRaceHandler(msm.baseHandler, raceManager)
	championshipHandler := NewChampionshipsHandler(msm.baseHandler, championshipManager)
	raceWeekendHandler := NewRaceWeekendHandler(msm.baseHandler, raceWeekendManager)
	raceControlHandler := NewRaceControlHandler(msm.baseHandler, msm.store, raceManager, raceControl, raceControlHub, process)
	serverAdministrationHandler := NewServerAdministrationHandler(msm.baseHandler, msm.store, raceManager, championshipManager, raceWeekendManager, process)
	penaltiesHandler := NewPenaltiesHandler(msm.baseHandler, championshipManager, raceWeekendManager)

	return &Server{
		ID:                          uuid.New(),
		ServerConfig:                serverConfig,
		Process:                     process,
		RaceManager:                 raceManager,
		ChampionshipManager:         championshipManager,
		RaceWeekendManager:          raceWeekendManager,
		RaceControl:                 raceControl,
		ContentManagerWrapper:       contentManagerWrapper,
		QuickRaceHandler:            quickRaceHandler,
		CustomRaceHandler:           customRaceHandler,
		ChampionshipsHandler:        championshipHandler,
		RaceWeekendHandler:          raceWeekendHandler,
		RaceControlHandler:          raceControlHandler,
		AccountHandler:              msm.accountHandler,
		ServerAdministrationHandler: serverAdministrationHandler,
		PenaltiesHandler:            penaltiesHandler,
	}
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
	})

	// writers
	r.Group(func(r chi.Router) {
		r.Use(s.AccountHandler.WriteAccessMiddleware)

		// races
		r.Get("/quick", s.QuickRaceHandler.create)
		r.Post("/quick/submit", s.QuickRaceHandler.submit)
		r.Get("/custom/load/{uuid}", s.CustomRaceHandler.start)
		r.Post("/custom/schedule/{uuid}", s.CustomRaceHandler.schedule)
		r.Get("/custom/schedule/{uuid}/remove", s.CustomRaceHandler.removeSchedule)
		r.Get("/custom/edit/{uuid}", s.CustomRaceHandler.createOrEdit)
		r.Post("/custom/new/submit", s.CustomRaceHandler.submit)

		// championships
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
	})

	// deleters
	r.Group(func(r chi.Router) {
		r.Use(s.AccountHandler.DeleteAccessMiddleware)

	})

	// admins
	r.Group(func(r chi.Router) {
		r.Use(s.AccountHandler.AdminAccessMiddleware)

		r.HandleFunc("/restart-session", s.RaceControlHandler.restartSession)
		r.HandleFunc("/next-session", s.RaceControlHandler.nextSession)
		r.HandleFunc("/broadcast-chat", s.RaceControlHandler.broadcastChat)
		r.HandleFunc("/admin-command", s.RaceControlHandler.adminCommand)
		r.HandleFunc("/kick-user", s.RaceControlHandler.kickUser)
	})

	return r
}

const sessionServerIDKey = "ServerID"

func (msm *MultiServerManager) ServerChoiceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := getSession(r)

		serverID, ok := sess.Values[sessionServerIDKey]

		if ok {
			for _, server := range msm.Servers {
				if server.ID.String() == serverID.(string) {
					server.Router().ServeHTTP(w, r)
					return
				}
			}
		}

		for _, server := range msm.Servers {
			server.Router().ServeHTTP(w, r)
			return
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	})
}

func (msm *MultiServerManager) ChooseServerHandler(w http.ResponseWriter, r *http.Request) {
	sess := getSession(r)
	sess.Values[sessionServerIDKey] = r.URL.Query().Get("serverID")
	_ = sess.Save(r, w)

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
