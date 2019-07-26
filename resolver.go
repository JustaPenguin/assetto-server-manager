package servermanager

import (
	"net/http"

	"github.com/cj123/assetto-server-manager/pkg/udp"
)

type Resolver struct {
	store           Store
	templateLoader  TemplateLoader
	reloadTemplates bool

	raceManager         *RaceManager
	carManager          *CarManager
	championshipManager *ChampionshipManager
	accountManager      *AccountManager

	viewRenderer          *Renderer
	serverProcess         ServerProcess
	raceControl           *RaceControl
	raceControlHub        *RaceControlHub
	contentManagerWrapper *ContentManagerWrapper

	// handlers
	baseHandler                 *BaseHandler
	quickRaceHandler            *QuickRaceHandler
	customRaceHandler           *CustomRaceHandler
	championshipsHandler        *ChampionshipsHandler
	accountHandler              *AccountHandler
	auditLogHandler             *AuditLogHandler
	carsHandler                 *CarsHandler
	tracksHandler               *TracksHandler
	weatherHandler              *WeatherHandler
	penaltiesHandler            *PenaltiesHandler
	resultsHandler              *ResultsHandler
	scheduledRacesHandler       *ScheduledRacesHandler
	contentUploadHandler        *ContentUploadHandler
	raceControlHandler          *RaceControlHandler
	serverAdministrationHandler *ServerAdministrationHandler
}

func NewResolver(templateLoader TemplateLoader, reloadTemplates bool, store Store) (*Resolver, error) {
	r := &Resolver{
		templateLoader:  templateLoader,
		reloadTemplates: reloadTemplates,
		store:           store,
	}

	err := r.initViewRenderer()

	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Resolver) UDPCallback(message udp.Message) {
	r.resolveRaceControl().UDPCallback(message)
	r.resolveChampionshipManager().ChampionshipEventCallback(message)
	r.resolveRaceManager().LoopCallback(message)
	r.resolveContentManagerWrapper().UDPCallback(message)
}

func (r *Resolver) initViewRenderer() error {
	if r.viewRenderer != nil {
		return nil
	}

	viewRenderer, err := NewRenderer(r.templateLoader, r.store, r.resolveServerProcess(), r.reloadTemplates)

	if err != nil {
		return err
	}

	r.viewRenderer = viewRenderer

	return nil
}

func (r *Resolver) ResolveStore() Store {
	return r.store
}

func (r *Resolver) resolveServerProcess() ServerProcess {
	if r.serverProcess != nil {
		return r.serverProcess
	}

	r.serverProcess = NewAssettoServerProcess(r.UDPCallback, r.resolveContentManagerWrapper())

	return r.serverProcess
}

func (r *Resolver) resolveContentManagerWrapper() *ContentManagerWrapper {
	if r.contentManagerWrapper != nil {
		return r.contentManagerWrapper
	}

	r.contentManagerWrapper = NewContentManagerWrapper(r.ResolveStore(), r.resolveCarManager())

	return r.contentManagerWrapper
}

func (r *Resolver) resolveRaceManager() *RaceManager {
	if r.raceManager != nil {
		return r.raceManager
	}

	r.raceManager = NewRaceManager(
		r.store,
		r.resolveServerProcess(),
		r.resolveCarManager(),
	)

	return r.raceManager
}

func (r *Resolver) resolveBaseHandler() *BaseHandler {
	if r.baseHandler != nil {
		return r.baseHandler
	}

	r.baseHandler = NewBaseHandler(r.viewRenderer)

	return r.baseHandler
}

func (r *Resolver) resolveCustomRaceHandler() *CustomRaceHandler {
	if r.customRaceHandler != nil {
		return r.customRaceHandler
	}

	r.customRaceHandler = NewCustomRaceHandler(r.resolveBaseHandler(), r.resolveRaceManager())

	return r.customRaceHandler
}

func (r *Resolver) resolveAccountManager() *AccountManager {
	if r.accountManager != nil {
		return r.accountManager
	}

	r.accountManager = NewAccountManager(r.store)

	return r.accountManager
}

func (r *Resolver) resolveAccountHandler() *AccountHandler {
	if r.accountHandler != nil {
		return r.accountHandler
	}

	r.accountHandler = NewAccountHandler(r.resolveBaseHandler(), r.store, r.resolveAccountManager())

	return r.accountHandler
}

func (r *Resolver) resolveQuickRaceHandler() *QuickRaceHandler {
	if r.quickRaceHandler != nil {
		return r.quickRaceHandler
	}

	r.quickRaceHandler = NewQuickRaceHandler(r.resolveBaseHandler(), r.resolveRaceManager())

	return r.quickRaceHandler
}

func (r *Resolver) resolveAuditLogHandler() *AuditLogHandler {
	if r.auditLogHandler != nil {
		return r.auditLogHandler
	}

	r.auditLogHandler = NewAuditLogHandler(r.resolveBaseHandler(), r.store)

	return r.auditLogHandler
}

func (r *Resolver) resolveCarManager() *CarManager {
	if r.carManager != nil {
		return r.carManager
	}

	r.carManager = NewCarManager()

	return r.carManager
}

func (r *Resolver) resolveCarsHandler() *CarsHandler {
	if r.carsHandler != nil {
		return r.carsHandler
	}

	r.carsHandler = NewCarsHandler(r.resolveBaseHandler(), r.resolveCarManager())

	return r.carsHandler
}

func (r *Resolver) resolveChampionshipManager() *ChampionshipManager {
	if r.championshipManager != nil {
		return r.championshipManager
	}

	r.championshipManager = NewChampionshipManager(
		r.resolveRaceManager(),
	)

	return r.championshipManager
}

func (r *Resolver) resolveChampionshipsHandler() *ChampionshipsHandler {
	if r.championshipsHandler != nil {
		return r.championshipsHandler
	}

	r.championshipsHandler = NewChampionshipsHandler(r.resolveBaseHandler(), r.resolveChampionshipManager())

	return r.championshipsHandler
}

func (r *Resolver) resolveTracksHandler() *TracksHandler {
	if r.tracksHandler != nil {
		return r.tracksHandler
	}

	r.tracksHandler = NewTracksHandler(r.resolveBaseHandler())

	return r.tracksHandler
}

func (r *Resolver) resolveWeatherHandler() *WeatherHandler {
	if r.weatherHandler != nil {
		return r.weatherHandler
	}

	r.weatherHandler = NewWeatherHandler(r.resolveBaseHandler())

	return r.weatherHandler
}

func (r *Resolver) resolvePenaltiesHandler() *PenaltiesHandler {
	if r.penaltiesHandler != nil {
		return r.penaltiesHandler
	}

	r.penaltiesHandler = NewPenaltiesHandler(r.resolveBaseHandler(), r.resolveChampionshipManager())

	return r.penaltiesHandler
}

func (r *Resolver) resolveResultsHandler() *ResultsHandler {
	if r.resultsHandler != nil {
		return r.resultsHandler
	}

	r.resultsHandler = NewResultsHandler(r.resolveBaseHandler())

	return r.resultsHandler
}

func (r *Resolver) resolveScheduledRacesHandler() *ScheduledRacesHandler {
	if r.scheduledRacesHandler != nil {
		return r.scheduledRacesHandler
	}

	r.scheduledRacesHandler = NewScheduledRacesHandler(r.resolveBaseHandler(), r.store, r.resolveRaceManager(), r.resolveChampionshipManager())

	return r.scheduledRacesHandler
}

func (r *Resolver) resolveServerAdministrationHandler() *ServerAdministrationHandler {
	if r.serverAdministrationHandler != nil {
		return r.serverAdministrationHandler
	}

	r.serverAdministrationHandler = NewServerAdministrationHandler(
		r.resolveBaseHandler(),
		r.resolveRaceManager(),
		r.resolveChampionshipManager(),
		r.resolveServerProcess(),
	)

	return r.serverAdministrationHandler
}

func (r *Resolver) resolveContentUploadHandler() *ContentUploadHandler {
	if r.contentUploadHandler != nil {
		return r.contentUploadHandler
	}

	r.contentUploadHandler = NewContentUploadHandler(r.resolveBaseHandler(), r.resolveCarManager())

	return r.contentUploadHandler
}

func (r *Resolver) resolveRaceControlHub() *RaceControlHub {
	if r.raceControlHub != nil {
		return r.raceControlHub
	}

	r.raceControlHub = newRaceControlHub()
	go r.raceControlHub.run()

	return r.raceControlHub
}

func (r *Resolver) resolveRaceControl() *RaceControl {
	if r.raceControl != nil {
		return r.raceControl
	}

	r.raceControl = NewRaceControl(r.resolveRaceControlHub(), filesystemTrackData{}, r.resolveServerProcess())

	return r.raceControl
}

func (r *Resolver) resolveRaceControlHandler() *RaceControlHandler {
	if r.raceControlHandler != nil {
		return r.raceControlHandler
	}

	r.raceControlHandler = NewRaceControlHandler(
		r.resolveBaseHandler(),
		r.ResolveStore(),
		r.resolveRaceManager(),
		r.resolveRaceControl(),
		r.resolveRaceControlHub(),
		r.resolveServerProcess(),
	)

	return r.raceControlHandler
}

func (r *Resolver) ResolveRouter(fs http.FileSystem) http.Handler {
	return Router(
		fs,
		r.resolveQuickRaceHandler(),
		r.resolveCustomRaceHandler(),
		r.resolveChampionshipsHandler(),
		r.resolveAccountHandler(),
		r.resolveAuditLogHandler(),
		r.resolveCarsHandler(),
		r.resolveTracksHandler(),
		r.resolveWeatherHandler(),
		r.resolvePenaltiesHandler(),
		r.resolveResultsHandler(),
		r.resolveContentUploadHandler(),
		r.resolveServerAdministrationHandler(),
		r.resolveRaceControlHandler(),
		r.resolveScheduledRacesHandler(),
	)
}

type BaseHandler struct {
	viewRenderer *Renderer
}

func NewBaseHandler(viewRenderer *Renderer) *BaseHandler {
	return &BaseHandler{
		viewRenderer: viewRenderer,
	}
}
