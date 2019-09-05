package servermanager

import (
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type RaceWeekendHandler struct {
	*BaseHandler

	raceWeekendManager *RaceWeekendManager
}

func NewRaceWeekendHandler(baseHandler *BaseHandler, raceWeekendManager *RaceWeekendManager) *RaceWeekendHandler {
	return &RaceWeekendHandler{
		BaseHandler:        baseHandler,
		raceWeekendManager: raceWeekendManager,
	}
}

func (rwh *RaceWeekendHandler) list(w http.ResponseWriter, r *http.Request) {
	raceWeekends, err := rwh.raceWeekendManager.ListRaceWeekends()

	if err != nil {
		logrus.Errorf("couldn't list weekends, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/index.html", map[string]interface{}{
		"RaceWeekends": raceWeekends,
	})
}

func (rwh *RaceWeekendHandler) view(w http.ResponseWriter, r *http.Request) {
	raceWeekend, err := rwh.raceWeekendManager.LoadRaceWeekend(chi.URLParam(r, "raceWeekendID"))

	if err != nil {
		logrus.Errorf("couldn't load race weekend, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/view.html", map[string]interface{}{
		"RaceWeekend":   raceWeekend,
		"WideContainer": true,
		"Account":       AccountFromRequest(r),
	})
}

func (rwh *RaceWeekendHandler) createOrEdit(w http.ResponseWriter, r *http.Request) {
	raceWeekendOpts, err := rwh.raceWeekendManager.BuildRaceWeekendTemplateOpts(r)

	if err != nil {
		panic(err)
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/new.html", raceWeekendOpts)
}

func (rwh *RaceWeekendHandler) submit(w http.ResponseWriter, r *http.Request) {
	raceWeekend, edited, err := rwh.raceWeekendManager.SaveRaceWeekend(r)

	if err != nil {
		logrus.Errorf("couldn't create race weekend, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if edited {
		AddFlash(w, r, "Race Weekend successfully edited!")
		http.Redirect(w, r, "/race-weekend/"+raceWeekend.ID.String(), http.StatusFound)
	} else {
		AddFlash(w, r, "We've created the Race Weekend. Now you need to add some sessions!")
		http.Redirect(w, r, "/race-weekend/"+raceWeekend.ID.String()+"/session", http.StatusFound)
	}
}

func (rwh *RaceWeekendHandler) sessionConfiguration(w http.ResponseWriter, r *http.Request) {
	raceWeekendSessionOpts, err := rwh.raceWeekendManager.BuildRaceWeekendSessionOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build race weekend session, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "custom-race/new.html", raceWeekendSessionOpts)
}

func (rwh *RaceWeekendHandler) submitSessionConfiguration(w http.ResponseWriter, r *http.Request) {
	raceWeekend, session, edited, err := rwh.raceWeekendManager.SaveRaceWeekendSession(r)

	if err != nil {
		logrus.Errorf("couldn't build race weekend session, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if edited {
		AddFlash(w, r,
			fmt.Sprintf(
				"Race Weekend session at %s was successfully edited!",
				prettifyName(session.RaceConfig.Track, false),
			),
		)
	} else {
		AddFlash(w, r,
			fmt.Sprintf(
				"Race Weekend session at %s was successfully added!",
				prettifyName(session.RaceConfig.Track, false),
			),
		)
	}

	if r.FormValue("action") == "saveRaceWeekend" {
		// end the race creation flow
		http.Redirect(w, r, "/race-weekend/"+raceWeekend.ID.String(), http.StatusFound)
		return
	} else {
		// add another session
		http.Redirect(w, r, "/race-weekend/"+raceWeekend.ID.String()+"/session", http.StatusFound)
	}
}

func (rwh *RaceWeekendHandler) startSession(w http.ResponseWriter, r *http.Request) {
	err := rwh.raceWeekendManager.StartSession(chi.URLParam(r, "raceWeekendID"), chi.URLParam(r, "sessionID"))

	if err != nil {
		logrus.Errorf("Could not start Race Weekend session, err: %s", err)

		AddErrorFlash(w, r, "Couldn't start the Session")
	} else {
		AddFlash(w, r, "Session started successfully!")
		time.Sleep(time.Second * 1)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (rwh *RaceWeekendHandler) restartSession(w http.ResponseWriter, r *http.Request) {
	err := rwh.raceWeekendManager.RestartSession(chi.URLParam(r, "raceWeekendID"), chi.URLParam(r, "sessionID"))

	if err != nil {
		logrus.Errorf("Could not restart Race Weekend session, err: %s", err)

		AddErrorFlash(w, r, "Couldn't restart the Session")
	} else {
		AddFlash(w, r, "Session restarted successfully!")
		time.Sleep(time.Second * 1)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (rwh *RaceWeekendHandler) cancelSession(w http.ResponseWriter, r *http.Request) {
	err := rwh.raceWeekendManager.CancelSession(chi.URLParam(r, "raceWeekendID"), chi.URLParam(r, "sessionID"))

	if err != nil {
		logrus.Errorf("Could not cancel Race Weekend session, err: %s", err)

		AddErrorFlash(w, r, "Couldn't cancel the Session")
	} else {
		AddFlash(w, r, "Session canceled successfully!")
		time.Sleep(time.Second * 1)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (rwh *RaceWeekendHandler) deleteSession(w http.ResponseWriter, r *http.Request) {
	err := rwh.raceWeekendManager.DeleteSession(chi.URLParam(r, "raceWeekendID"), chi.URLParam(r, "sessionID"))

	if err != nil {
		logrus.Errorf("Could not delete Race Weekend session, err: %s", err)

		AddErrorFlash(w, r, "Couldn't delete the Session")
	} else {
		AddFlash(w, r, "Session deleted successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (rwh *RaceWeekendHandler) importSessionResults(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	sessionID := chi.URLParam(r, "sessionID")

	if r.Method == http.MethodPost {
		err := rwh.raceWeekendManager.ImportSession(raceWeekendID, sessionID, r)

		if err != nil {
			logrus.Errorf("Could not import race weekend session, error: %s", err)
			AddErrorFlash(w, r, "Could not import race weekend session files")
		} else {
			AddFlash(w, r, "Successfully imported session files!")
			http.Redirect(w, r, "/race-weekend/"+raceWeekendID, http.StatusFound)
			return
		}
	}

	session, results, err := rwh.raceWeekendManager.ListAvailableResultsFilesForSession(raceWeekendID, sessionID)

	if err != nil {
		logrus.Errorf("Couldn't load session files, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/import-session.html", map[string]interface{}{
		"Results":       results,
		"RaceWeekendID": raceWeekendID,
		"Session":       session,
	})
}

func (rwh *RaceWeekendHandler) manageFilters(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	parentSessionID := r.URL.Query().Get("parentSessionID")
	childSessionID := r.URL.Query().Get("childSessionID")

	raceWeekend, parentSession, childSession, err := rwh.raceWeekendManager.FindConnectedSessions(raceWeekendID, parentSessionID, childSessionID)

	if err != nil {
		panic(err) // @TODO
	}

	filter, err := raceWeekend.GetFilterOrUseDefault(parentSessionID, childSessionID)

	if err != nil {
		panic(err) // @TODO
	}

	rwh.viewRenderer.MustLoadPartial(w, r, "race-weekend/popups/manage-filters.html", map[string]interface{}{
		"RaceWeekend":   raceWeekend,
		"ParentSession": parentSession,
		"ChildSession":  childSession,
		"Filter":        filter,
	})
}

func (rwh *RaceWeekendHandler) gridPreview(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	parentSessionID := r.URL.Query().Get("parentSessionID")
	childSessionID := r.URL.Query().Get("childSessionID")

	var filter *EntrantPositionFilter

	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		panic(err) // @TODO
	}

	previewResponse, err := rwh.raceWeekendManager.PreviewGrid(raceWeekendID, parentSessionID, childSessionID, filter)

	if err != nil {
		panic(err) // @TODO
	}

	w.Header().Add("Content-Type", "application/json")

	spew.Dump(previewResponse) // @TODO

	_ = json.NewEncoder(w).Encode(previewResponse)
}

func (rwh *RaceWeekendHandler) updateGrid(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	parentSessionID := r.URL.Query().Get("parentSessionID")
	childSessionID := r.URL.Query().Get("childSessionID")

	var filter *EntrantPositionFilter

	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		panic(err) // @TODO
	}

	if err := rwh.raceWeekendManager.UpdateGrid(raceWeekendID, parentSessionID, childSessionID, filter); err != nil {
		panic(err) // @TODO
	}

	w.WriteHeader(http.StatusCreated)
}
