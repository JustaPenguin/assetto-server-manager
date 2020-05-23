package servermanager

import (
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