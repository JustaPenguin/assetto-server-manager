package servermanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"4d63.com/tz"
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

type raceWeekendListTemplateVars struct {
	BaseTemplateVars

	RaceWeekends []*RaceWeekend
}

func (rwh *RaceWeekendHandler) list(w http.ResponseWriter, r *http.Request) {
	raceWeekends, err := rwh.raceWeekendManager.ListRaceWeekends()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't list weekends")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/index.html", &raceWeekendListTemplateVars{
		RaceWeekends: raceWeekends,
	})
}

type raceWeekendViewTemplateVars struct {
	BaseTemplateVars

	RaceWeekend *RaceWeekend
	Account     *Account
}

func (rwh *RaceWeekendHandler) view(w http.ResponseWriter, r *http.Request) {
	raceWeekend, err := rwh.raceWeekendManager.LoadRaceWeekend(chi.URLParam(r, "raceWeekendID"))

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load race weekend")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/view.html", &raceWeekendViewTemplateVars{
		BaseTemplateVars: BaseTemplateVars{
			WideContainer: true,
		},
		RaceWeekend: raceWeekend,
		Account:     AccountFromRequest(r),
	})
}

func (rwh *RaceWeekendHandler) createOrEdit(w http.ResponseWriter, r *http.Request) {
	raceWeekendOpts, err := rwh.raceWeekendManager.BuildRaceWeekendTemplateOpts(r)

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load race weekend")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/new.html", raceWeekendOpts)
}

func (rwh *RaceWeekendHandler) delete(w http.ResponseWriter, r *http.Request) {
	err := rwh.raceWeekendManager.DeleteRaceWeekend(chi.URLParam(r, "raceWeekendID"))

	if err != nil {
		logrus.WithError(err).Errorf("couldn't delete race weekend")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Race Weekend successfully deleted!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (rwh *RaceWeekendHandler) submit(w http.ResponseWriter, r *http.Request) {
	raceWeekend, edited, err := rwh.raceWeekendManager.SaveRaceWeekend(r)

	if err != nil {
		logrus.WithError(err).Errorf("couldn't create race weekend")
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
		logrus.WithError(err).Errorf("couldn't build race weekend session")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "custom-race/new.html", raceWeekendSessionOpts)
}

func (rwh *RaceWeekendHandler) submitSessionConfiguration(w http.ResponseWriter, r *http.Request) {
	raceWeekend, session, edited, err := rwh.raceWeekendManager.SaveRaceWeekendSession(r)

	if err != nil {
		logrus.WithError(err).Errorf("couldn't build race weekend session")
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
	}

	// add another session
	http.Redirect(w, r, "/race-weekend/"+raceWeekend.ID.String()+"/session", http.StatusFound)
}

func (rwh *RaceWeekendHandler) startSession(w http.ResponseWriter, r *http.Request) {
	err := rwh.raceWeekendManager.StartSession(chi.URLParam(r, "raceWeekendID"), chi.URLParam(r, "sessionID"), false)

	if err != nil {
		logrus.WithError(err).Errorf("Could not start Race Weekend session")

		AddErrorFlash(w, r, "Couldn't start the Session")
	} else {
		AddFlash(w, r, "Session started successfully!")
		time.Sleep(time.Second * 1)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (rwh *RaceWeekendHandler) startPracticeSession(w http.ResponseWriter, r *http.Request) {
	err := rwh.raceWeekendManager.StartPracticeSession(chi.URLParam(r, "raceWeekendID"), chi.URLParam(r, "sessionID"))

	if err != nil {
		logrus.WithError(err).Errorf("Could not start Race Weekend practice session")

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
		logrus.WithError(err).Errorf("Could not restart Race Weekend session")

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
		logrus.WithError(err).Errorf("Could not cancel Race Weekend session")

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
		logrus.WithError(err).Errorf("Could not delete Race Weekend session")

		AddErrorFlash(w, r, "Couldn't delete the Session")
	} else {
		AddFlash(w, r, "Session deleted successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

type raceWeekendSessionImportTemplateVars struct {
	BaseTemplateVars

	Results       []SessionResults
	RaceWeekendID string
	Session       *RaceWeekendSession
}

func (rwh *RaceWeekendHandler) importSessionResults(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	sessionID := chi.URLParam(r, "sessionID")

	if r.Method == http.MethodPost {
		err := rwh.raceWeekendManager.ImportSession(raceWeekendID, sessionID, r)

		if err != nil {
			logrus.WithError(err).Errorf("Could not import race weekend session")
			AddErrorFlash(w, r, "Could not import race weekend session files")
		} else {
			AddFlash(w, r, "Successfully imported session files!")
			http.Redirect(w, r, "/race-weekend/"+raceWeekendID, http.StatusFound)
			return
		}
	}

	session, results, err := rwh.raceWeekendManager.ListAvailableResultsFilesForSession(raceWeekendID, sessionID)

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't load session files")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/import-session.html", &raceWeekendSessionImportTemplateVars{
		Results:       results,
		RaceWeekendID: raceWeekendID,
		Session:       session,
	})
}

type raceWeekendFilterTemplateVars struct {
	BaseTemplateVars

	RaceWeekend                 *RaceWeekend
	ParentSession, ChildSession *RaceWeekendSession
	ResultsAvailableForSorting  []SessionResults
	Filter                      *RaceWeekendSessionToSessionFilter
	AvailableSorters            []RaceWeekendEntryListSorterDescription
}

func (rwh *RaceWeekendHandler) manageFilters(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	parentSessionID := r.URL.Query().Get("parentSessionID")
	childSessionID := r.URL.Query().Get("childSessionID")

	raceWeekend, parentSession, childSession, err := rwh.raceWeekendManager.FindConnectedSessions(raceWeekendID, parentSessionID, childSessionID)

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't load connected sessions")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	filter, err := raceWeekend.GetFilterOrUseDefault(parentSessionID, childSessionID)

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't load session filters")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sessionResults, err := rwh.raceWeekendManager.ListAvailableResultsFilesForSorting(raceWeekend, childSession)

	if err != nil {
		logrus.WithError(err).Error("Couldn't list results files for sorting")
	}

	rwh.viewRenderer.MustLoadPartial(w, r, "race-weekend/popups/manage-filters.html", &raceWeekendFilterTemplateVars{
		RaceWeekend:                raceWeekend,
		ParentSession:              parentSession,
		ChildSession:               childSession,
		ResultsAvailableForSorting: sessionResults,
		Filter:                     filter,
		AvailableSorters:           RaceWeekendEntryListSorters,
	})
}

func (rwh *RaceWeekendHandler) gridPreview(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	parentSessionID := r.URL.Query().Get("parentSessionID")
	childSessionID := r.URL.Query().Get("childSessionID")

	var filter *RaceWeekendSessionToSessionFilter

	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	filter.IsPreview = true

	previewResponse, err := rwh.raceWeekendManager.PreviewGrid(raceWeekendID, parentSessionID, childSessionID, filter)

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't preview session grid")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(previewResponse)
}

func (rwh *RaceWeekendHandler) updateGrid(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	parentSessionID := r.URL.Query().Get("parentSessionID")
	childSessionID := r.URL.Query().Get("childSessionID")

	var filter *RaceWeekendSessionToSessionFilter

	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := rwh.raceWeekendManager.UpdateGrid(raceWeekendID, parentSessionID, childSessionID, filter); err != nil {
		logrus.WithError(err).Errorf("Couldn't update session grid")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

type raceWeekendManageEntryListTemplateVars struct {
	BaseTemplateVars

	RaceWeekend      *RaceWeekend
	Session          *RaceWeekendSession
	AvailableSorters []RaceWeekendEntryListSorterDescription
}

func (rwh *RaceWeekendHandler) manageEntryList(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	sessionID := r.URL.Query().Get("sessionID")

	raceWeekend, session, err := rwh.raceWeekendManager.FindSession(raceWeekendID, sessionID)

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't load manage entry list popup")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadPartial(w, r, "race-weekend/popups/manage-entrylist.html", &raceWeekendManageEntryListTemplateVars{
		RaceWeekend:      raceWeekend,
		Session:          session,
		AvailableSorters: RaceWeekendEntryListSorters,
	})
}

func (rwh *RaceWeekendHandler) entryListPreview(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	sessionID := r.URL.Query().Get("sessionID")
	sortType := r.URL.Query().Get("sortType")
	reverseNumber := formValueAsInt(r.URL.Query().Get("reverseGrid"))

	previewResponse, err := rwh.raceWeekendManager.PreviewSessionEntryList(raceWeekendID, sessionID, sortType, reverseNumber)

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't preview session grid")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(previewResponse)
}

func (rwh *RaceWeekendHandler) updateEntryList(w http.ResponseWriter, r *http.Request) {
	raceWeekendID := chi.URLParam(r, "raceWeekendID")
	parentSessionID := r.URL.Query().Get("sessionID")
	sortType := r.URL.Query().Get("sortType")
	reverseNumber := formValueAsInt(r.URL.Query().Get("reverseGrid"))

	if err := rwh.raceWeekendManager.UpdateSessionSorting(raceWeekendID, parentSessionID, sortType, reverseNumber); err != nil {
		logrus.WithError(err).Errorf("Couldn't update session entrylist sorting")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (rwh *RaceWeekendHandler) export(w http.ResponseWriter, r *http.Request) {
	raceWeekend, err := rwh.raceWeekendManager.LoadRaceWeekend(chi.URLParam(r, "raceWeekendID"))

	if err != nil {
		logrus.WithError(err).Errorf("couldn't export race weeeknd")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !AccountFromRequest(r).HasGroupPrivilege(GroupWrite) {
		// if you don't have write access or above you can't see the replacement password
		for _, session := range raceWeekend.Sessions {
			session.ReplacementPassword = ""
		}
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.json"`, raceWeekend.Name))

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(raceWeekend)
}

func (rwh *RaceWeekendHandler) importRaceWeekend(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		raceWeekendID, err := rwh.raceWeekendManager.ImportRaceWeekend(r.FormValue("import"))

		if err != nil {
			logrus.WithError(err).Error("could not import race weekend")
			AddErrorFlash(w, r, "Sorry, we couldn't import that Race Weekend! Check your JSON formatting.")
		} else {
			AddFlash(w, r, "Race Weekend successfully imported!")
			http.Redirect(w, r, "/race-weekend/"+raceWeekendID, http.StatusFound)
		}
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/import-raceweekend.html", nil)
}

func (rwh *RaceWeekendHandler) scheduleSession(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logrus.WithError(err).Errorf("couldn't parse schedule race form")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	championshipID := chi.URLParam(r, "raceWeekendID")
	championshipEventID := chi.URLParam(r, "sessionID")
	dateString := r.FormValue("session-schedule-date")
	timeString := r.FormValue("session-schedule-time")
	timezone := r.FormValue("session-schedule-timezone")
	startWhenParentFinished := formValueAsInt(r.FormValue("session-start-after-parent")) == 1

	var message string
	var date time.Time

	if !startWhenParentFinished {
		var location *time.Location

		location, err := tz.LoadLocation(timezone)

		if err != nil {
			logrus.WithError(err).Errorf("could not find location: %s", location)
			location = time.Local
		}

		// Parse time in correct time zone
		date, err = time.ParseInLocation("2006-01-02-15:04", dateString+"-"+timeString, location)

		if err != nil {
			logrus.WithError(err).Errorf("couldn't parse schedule race weekend session date")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		message = fmt.Sprintf("We have scheduled the Race Weekend Session to begin at %s", date.Format(time.RFC1123))

	} else {
		message = fmt.Sprintf("We have scheduled the Race Weekend Session to begin after the parent session(s) complete.")
	}

	err := rwh.raceWeekendManager.ScheduleSession(championshipID, championshipEventID, date, startWhenParentFinished)

	if err != nil {
		logrus.WithError(err).Errorf("couldn't schedule race weekend session")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, message)
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (rwh *RaceWeekendHandler) removeSessionSchedule(w http.ResponseWriter, r *http.Request) {
	err := rwh.raceWeekendManager.DeScheduleSession(chi.URLParam(r, "raceWeekendID"), chi.URLParam(r, "sessionID"))

	if err != nil {
		logrus.WithError(err).Errorf("couldn't de-schedule race weekend session")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
