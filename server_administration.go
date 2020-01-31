package servermanager

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/mitchellh/go-wordwrap"
	"github.com/sirupsen/logrus"
)

type ServerAdministrationHandler struct {
	*BaseHandler

	store               Store
	raceManager         *RaceManager
	championshipManager *ChampionshipManager
	raceWeekendManager  *RaceWeekendManager
	process             ServerProcess
	acsrClient          *ACSRClient
}

func NewServerAdministrationHandler(
	baseHandler *BaseHandler,
	store Store,
	raceManager *RaceManager,
	championshipManager *ChampionshipManager,
	raceWeekendManager *RaceWeekendManager,
	process ServerProcess,
	acsrClient *ACSRClient,
) *ServerAdministrationHandler {
	return &ServerAdministrationHandler{
		BaseHandler:         baseHandler,
		store:               store,
		raceManager:         raceManager,
		championshipManager: championshipManager,
		raceWeekendManager:  raceWeekendManager,
		process:             process,
		acsrClient:          acsrClient,
	}
}

type homeTemplateVars struct {
	BaseTemplateVars

	RaceDetails     *CustomRace
	PerformanceMode bool
}

// homeHandler serves content to /
func (sah *ServerAdministrationHandler) home(w http.ResponseWriter, r *http.Request) {
	currentRace, entryList := sah.raceManager.CurrentRace()

	var customRace *CustomRace

	if currentRace != nil {
		customRace = &CustomRace{EntryList: entryList, RaceConfig: currentRace.CurrentRaceConfig}
	}

	sah.viewRenderer.MustLoadTemplate(w, r, "home.html", &homeTemplateVars{
		RaceDetails:     customRace,
		PerformanceMode: config.Server.PerformanceMode,
	})
}

func (sah *ServerAdministrationHandler) premium(w http.ResponseWriter, r *http.Request) {
	sah.viewRenderer.MustLoadTemplate(w, r, "premium.html", nil)
}

const MOTDFilename = "motd.txt"

type motdTemplateVars struct {
	BaseTemplateVars

	MOTDText string
	Opts     *GlobalServerConfig
}

func (sah *ServerAdministrationHandler) motd(w http.ResponseWriter, r *http.Request) {
	opts, err := sah.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Error("couldn't load server options")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodPost {
		wrapped := wordwrap.WrapString(r.FormValue("motd"), 140)

		success := true

		err := ioutil.WriteFile(filepath.Join(ServerInstallPath, MOTDFilename), []byte(wrapped), 0644)

		if err != nil {
			logrus.WithError(err).Error("couldn't save message of the day")
			AddErrorFlash(w, r, "Failed to save message changes")
			success = false
		}

		opts.ServerJoinMessage = r.FormValue("serverJoinMessage")
		opts.ContentManagerWelcomeMessage = r.FormValue("contentManagerWelcomeMessage")

		if err := sah.store.UpsertServerOptions(opts); err != nil {
			logrus.WithError(err).Error("couldn't save messages")
			AddErrorFlash(w, r, "Failed to save message changes")
			success = false
		}

		if success {
			AddFlash(w, r, "Messages successfully saved!")
		}
	}

	b, err := ioutil.ReadFile(filepath.Join(ServerInstallPath, MOTDFilename))

	if err != nil && !os.IsNotExist(err) {
		logrus.WithError(err).Error("couldn't find motd.txt")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sah.viewRenderer.MustLoadTemplate(w, r, "server/motd.html", &motdTemplateVars{
		MOTDText: string(b),
		Opts:     opts,
	})
}

type serverOptionsTemplateVars struct {
	BaseTemplateVars

	Form *Form
}

func (sah *ServerAdministrationHandler) options(w http.ResponseWriter, r *http.Request) {
	serverOpts, err := sah.raceManager.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
	}

	form := NewForm(serverOpts, nil, "", AccountFromRequest(r).Name == "admin")

	if r.Method == http.MethodPost {
		err := form.Submit(r)

		if err != nil {
			logrus.WithError(err).Errorf("couldn't submit form")
		}

		UseShortenedDriverNames = serverOpts.UseShortenedDriverNames == 1
		UseFallBackSorting = serverOpts.FallBackResultsSorting == 1

		// save the config
		err = sah.raceManager.SaveServerOptions(serverOpts)

		if err != nil {
			logrus.WithError(err).Errorf("couldn't save config")
			AddErrorFlash(w, r, "Failed to save server options")
		} else {
			AddFlash(w, r, "Server options successfully saved!")
		}

		// update ACSR options to the client
		sah.acsrClient.AccountID = serverOpts.ACSRAccountID
		sah.acsrClient.APIKey = serverOpts.ACSRAPIKey
		sah.acsrClient.Enabled = serverOpts.EnableACSR
	}

	sah.viewRenderer.MustLoadTemplate(w, r, "server/options.html", &serverOptionsTemplateVars{
		Form: form,
	})
}

type serverBlacklistTemplateVars struct {
	BaseTemplateVars

	Text string
}

func (sah *ServerAdministrationHandler) blacklist(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// save to blacklist.txt
		var text string

		if r.FormValue("type") == "single" {
			// we're adding a single GUID, load the existing blacklist list then append
			b, err := ioutil.ReadFile(filepath.Join(ServerInstallPath, "blacklist.txt"))
			if err != nil {
				logrus.WithError(err).Error("couldn't find blacklist.txt")
			}

			text = string(b) + r.FormValue("blacklist")
		} else {
			text = r.FormValue("blacklist")
		}

		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}

		err := ioutil.WriteFile(filepath.Join(ServerInstallPath, "blacklist.txt"), []byte(text), 0644)

		if err != nil {
			logrus.WithError(err).Error("couldn't save blacklist")
			AddErrorFlash(w, r, "Failed to save Server blacklist changes")
		} else {
			AddFlash(w, r, "Server blacklist successfully changed!")
		}
	}

	// load blacklist.txt
	b, err := ioutil.ReadFile(filepath.Join(ServerInstallPath, "blacklist.txt")) // just pass the file name
	if err != nil {
		logrus.WithError(err).Error("couldn't find blacklist.txt")
	}

	// render blacklist edit page
	sah.viewRenderer.MustLoadTemplate(w, r, "server/blacklist.html", &serverBlacklistTemplateVars{
		Text: string(b),
	})
}

type autoFillEntrantListTemplateVars struct {
	BaseTemplateVars

	Entrants []*Entrant
}

func (sah *ServerAdministrationHandler) autoFillEntrantList(w http.ResponseWriter, r *http.Request) {
	entrants, err := sah.raceManager.ListAutoFillEntrants()

	if err != nil {
		logrus.WithError(err).Error("could not list entrants")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sah.viewRenderer.MustLoadTemplate(w, r, "server/autofill-entrants.html", &autoFillEntrantListTemplateVars{
		Entrants: entrants,
	})
}

func (sah *ServerAdministrationHandler) autoFillEntrantDelete(w http.ResponseWriter, r *http.Request) {
	err := sah.raceManager.store.DeleteEntrant(chi.URLParam(r, "entrantID"))

	if err != nil {
		logrus.WithError(err).Error("could not delete entrant")
		AddErrorFlash(w, r, "Could not delete entrant")
	} else {
		AddFlash(w, r, "Successfully deleted entrant")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (sah *ServerAdministrationHandler) logs(w http.ResponseWriter, r *http.Request) {
	sah.viewRenderer.MustLoadTemplate(w, r, "server/logs.html", nil)
}

type logData struct {
	ServerLog, ManagerLog, PluginsLog string
}

func (sah *ServerAdministrationHandler) logsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(logData{
		ServerLog:  sah.process.Logs(),
		ManagerLog: logOutput.String(),
		PluginsLog: pluginsOutput.String(),
	})
}

// downloading logfiles
func (sah *ServerAdministrationHandler) logsDownload(w http.ResponseWriter, r *http.Request) {
	logFile := chi.URLParam(r, "logFile")
	var outputString string

	if logFile == "server" {
		outputString = sah.process.Logs()
	} else if logFile == "manager" {
		outputString = logOutput.String()
	} else if logFile == "plugins" {
		outputString = pluginsOutput.String()
	} else {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	// tell the browser this is a file download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename= \""+logFile+"_"+time.Now().Format(time.RFC3339)+".log\"")

	_, err := w.Write([]byte(outputString))

	if err != nil {
		logrus.WithError(err).Error("failed to return log " + logFile + " as file via http")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// serverProcessHandler modifies the server process.
func (sah *ServerAdministrationHandler) serverProcess(w http.ResponseWriter, r *http.Request) {
	var err error
	var txt string

	event := sah.process.Event()

	switch chi.URLParam(r, "action") {
	case "stop":
		if event.IsChampionship() && !event.IsPractice() {
			err = sah.championshipManager.StopActiveEvent()
		} else if event.IsRaceWeekend() && !event.IsPractice() {
			err = sah.raceWeekendManager.StopActiveSession()
		} else {
			err = sah.process.Stop()
		}
		txt = "stopped"
	case "restart":
		if event.IsChampionship() && !event.IsPractice() {
			err = sah.championshipManager.RestartActiveEvent()
		} else if event.IsRaceWeekend() && !event.IsPractice() {
			err = sah.raceWeekendManager.RestartActiveSession()
		} else {
			err = sah.process.Restart()
		}
		txt = "restarted"
	}

	noun := "Server"

	if event.IsChampionship() {
		noun = "Championship"
	} else if event.IsRaceWeekend() {
		noun = "Race Weekend"
	}

	if event.IsPractice() {
		noun += " Practice"
	}

	if err != nil {
		logrus.WithError(err).Errorf("could not change " + noun + " status")
		AddErrorFlash(w, r, "Unable to change "+noun+" status")
	} else {
		AddFlash(w, r, noun+" successfully "+txt)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

type changelogTemplateVars struct {
	BaseTemplateVars

	Changelog template.HTML
}

func (sah *ServerAdministrationHandler) changelog(w http.ResponseWriter, r *http.Request) {
	sah.viewRenderer.MustLoadTemplate(w, r, "changelog.html", &changelogTemplateVars{
		Changelog: Changelog,
	})
}

func (sah *ServerAdministrationHandler) robots(w http.ResponseWriter, r *http.Request) {
	// do we want to let robots on the internet know things about us?!?
	serverOpts, err := sah.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var response string

	w.Header().Set("Content-Type", "text/plain")

	if serverOpts.PreventWebCrawlers == 1 {
		response = "User-agent: *\nDisallow: /"
	} else {
		response = "User-agent: *\nDisallow:"
	}

	_, err = w.Write([]byte(response))

	if err != nil {
		logrus.WithError(err).Errorf("couldn't write response text")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
