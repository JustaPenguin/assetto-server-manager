package servermanager

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type ServerAdministrationHandler struct {
	*BaseHandler

	raceManager         *RaceManager
	championshipManager *ChampionshipManager
	process             ServerProcess
}

func NewServerAdministrationHandler(
	baseHandler *BaseHandler,
	raceManager *RaceManager,
	championshipManager *ChampionshipManager,
	process ServerProcess,
) *ServerAdministrationHandler {
	return &ServerAdministrationHandler{
		BaseHandler:         baseHandler,
		raceManager:         raceManager,
		championshipManager: championshipManager,
		process:             process,
	}
}

// homeHandler serves content to /
func (sah *ServerAdministrationHandler) home(w http.ResponseWriter, r *http.Request) {
	currentRace, entryList := sah.raceManager.CurrentRace()

	var customRace *CustomRace

	if currentRace != nil {
		customRace = &CustomRace{EntryList: entryList, RaceConfig: currentRace.CurrentRaceConfig}
	}

	sah.viewRenderer.MustLoadTemplate(w, r, "home.html", map[string]interface{}{
		"RaceDetails": customRace,
	})
}

const MOTDFilename = "motd.txt"

func (sah *ServerAdministrationHandler) motd(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		text := r.FormValue("motd")

		err := ioutil.WriteFile(filepath.Join(ServerInstallPath, MOTDFilename), []byte(text), 0644)

		if err != nil {
			logrus.WithError(err).Error("couldn't save message of the day")
			AddErrorFlash(w, r, "Failed to save message of the day changes")
		} else {
			AddFlash(w, r, "Server message of the day successfully changed!")
		}
	}

	b, err := ioutil.ReadFile(filepath.Join(ServerInstallPath, MOTDFilename))

	if err != nil && !os.IsNotExist(err) {
		logrus.WithError(err).Error("couldn't find motd.txt")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sah.viewRenderer.MustLoadTemplate(w, r, "server/motd.html", map[string]interface{}{
		"text": string(b),
	})
}

func (sah *ServerAdministrationHandler) options(w http.ResponseWriter, r *http.Request) {
	serverOpts, err := sah.raceManager.LoadServerOptions()

	if err != nil {
		logrus.Errorf("couldn't load server options, err: %s", err)
	}

	form := NewForm(serverOpts, nil, "", AccountFromRequest(r).Name == "admin")

	if r.Method == http.MethodPost {
		err := form.Submit(r)

		if err != nil {
			logrus.Errorf("couldn't submit form, err: %s", err)
		}

		UseShortenedDriverNames = serverOpts.UseShortenedDriverNames == 1

		// save the config
		err = sah.raceManager.SaveServerOptions(serverOpts)

		if err != nil {
			logrus.Errorf("couldn't save config, err: %s", err)
			AddErrorFlash(w, r, "Failed to save server options")
		} else {
			AddFlash(w, r, "Server options successfully saved!")
		}
	}

	sah.viewRenderer.MustLoadTemplate(w, r, "server/options.html", map[string]interface{}{
		"form": form,
	})
}

func (sah *ServerAdministrationHandler) blacklist(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// save to blacklist.txt
		text := r.FormValue("blacklist")

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
	sah.viewRenderer.MustLoadTemplate(w, r, "server/blacklist.html", map[string]interface{}{
		"text": string(b),
	})
}

func (sah *ServerAdministrationHandler) autoFillEntrantList(w http.ResponseWriter, r *http.Request) {
	entrants, err := sah.raceManager.ListAutoFillEntrants()

	if err != nil {
		logrus.WithError(err).Error("could not list entrants")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sah.viewRenderer.MustLoadTemplate(w, r, "server/autofill-entrants.html", map[string]interface{}{
		"Entrants": entrants,
	})
}

func (sah *ServerAdministrationHandler) autoFillEntrantDelete(w http.ResponseWriter, r *http.Request) {
	err := sah.raceManager.raceStore.DeleteEntrant(chi.URLParam(r, "entrantID"))

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

// serverProcessHandler modifies the server process.
func (sah *ServerAdministrationHandler) serverProcess(w http.ResponseWriter, r *http.Request) {
	var err error
	var txt string

	eventType := sah.process.EventType()

	switch chi.URLParam(r, "action") {
	case "stop":
		if eventType == EventTypeChampionship {
			err = sah.championshipManager.StopActiveEvent()
		} else {
			err = sah.process.Stop()
		}
		txt = "stopped"
	case "restart":
		if eventType == EventTypeChampionship {
			err = sah.championshipManager.RestartActiveEvent()
		} else {
			err = sah.process.Restart()
		}
		txt = "restarted"
	}

	noun := "Server"

	if eventType == EventTypeChampionship {
		noun = "Championship"
	}

	if err != nil {
		logrus.Errorf("could not change "+noun+" status, err: %s", err)
		AddErrorFlash(w, r, "Unable to change "+noun+" status")
	} else {
		AddFlash(w, r, noun+" successfully "+txt)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
