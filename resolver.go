package servermanager

import (
	"net/http"
)

type Resolver struct {
	store           Store
	templateLoader  TemplateLoader
	reloadTemplates bool

	multiServerManager    *MultiServerManager
	carManager            *CarManager
	accountManager        *AccountManager
	discordManager        *DiscordManager
	notificationManager   *NotificationManager
	scheduledRacesManager *ScheduledRacesManager

	viewRenderer *Renderer

	// handlers
	baseHandler           *BaseHandler
	accountHandler        *AccountHandler
	auditLogHandler       *AuditLogHandler
	carsHandler           *CarsHandler
	tracksHandler         *TracksHandler
	weatherHandler        *WeatherHandler
	penaltiesHandler      *PenaltiesHandler
	resultsHandler        *ResultsHandler
	scheduledRacesHandler *ScheduledRacesHandler
	contentUploadHandler  *ContentUploadHandler
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

func (r *Resolver) initViewRenderer() error {
	if r.viewRenderer != nil {
		return nil
	}

	viewRenderer, err := NewRenderer(r.templateLoader, r.store, nil, r.reloadTemplates)

	if err != nil {
		return err
	}

	r.viewRenderer = viewRenderer

	return nil
}

func (r *Resolver) ResolveStore() Store {
	return r.store
}

func (r *Resolver) resolveMultiServerManager() *MultiServerManager {
	if r.multiServerManager != nil {
		return r.multiServerManager
	}

	r.multiServerManager = NewMultiServerManager(
		r.ResolveStore(),
		r.resolveCarManager(),
		r.resolveNotificationManager(),
		r.resolveBaseHandler(),
		r.resolveAccountHandler(),
	)

	return r.multiServerManager
}

func (r *Resolver) resolveBaseHandler() *BaseHandler {
	if r.baseHandler != nil {
		return r.baseHandler
	}

	r.baseHandler = NewBaseHandler(r.viewRenderer)

	return r.baseHandler
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

func (r *Resolver) resolveResultsHandler() *ResultsHandler {
	if r.resultsHandler != nil {
		return r.resultsHandler
	}

	r.resultsHandler = NewResultsHandler(r.resolveBaseHandler(), r.ResolveStore())

	return r.resultsHandler
}

func (r *Resolver) resolveScheduledRacesManager() *ScheduledRacesManager {
	if r.scheduledRacesManager != nil {
		return r.scheduledRacesManager
	}

	r.scheduledRacesManager = NewScheduledRacesManager(r.ResolveStore())

	return r.scheduledRacesManager
}

func (r *Resolver) resolveScheduledRacesHandler() *ScheduledRacesHandler {
	if r.scheduledRacesHandler != nil {
		return r.scheduledRacesHandler
	}

	r.scheduledRacesHandler = NewScheduledRacesHandler(r.resolveBaseHandler(), r.resolveScheduledRacesManager())

	return r.scheduledRacesHandler
}

func (r *Resolver) resolveContentUploadHandler() *ContentUploadHandler {
	if r.contentUploadHandler != nil {
		return r.contentUploadHandler
	}

	r.contentUploadHandler = NewContentUploadHandler(r.resolveBaseHandler(), r.resolveCarManager())

	return r.contentUploadHandler
}

func (r *Resolver) resolveDiscordManager() *DiscordManager {
	if r.discordManager != nil {
		return r.discordManager
	}

	// if manager errors, it will log the error and return discordManager flagged as disabled, so no need to handle err
	r.discordManager, _ = NewDiscordManager(r.store, r.resolveScheduledRacesManager())

	return r.discordManager
}

func (r *Resolver) resolveNotificationManager() *NotificationManager {
	if r.notificationManager != nil {
		return r.notificationManager
	}

	r.notificationManager = NewNotificationManager(r.resolveDiscordManager(), r.resolveCarManager(), r.store)

	return r.notificationManager
}

func (r *Resolver) ResolveRouter(fs http.FileSystem) http.Handler {
	return Router(
		fs,
		r.resolveMultiServerManager(),
		r.resolveAccountHandler(),
		r.resolveAuditLogHandler(),
		r.resolveCarsHandler(),
		r.resolveTracksHandler(),
		r.resolveWeatherHandler(),
		r.resolveResultsHandler(),
		r.resolveContentUploadHandler(),
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
