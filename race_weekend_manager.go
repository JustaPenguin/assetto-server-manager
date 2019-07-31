package servermanager

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

type RaceWeekendManager struct {
	raceManager *RaceManager
	store       Store
	process     ServerProcess
}

func NewRaceWeekendManager(raceManager *RaceManager, store Store, process ServerProcess) *RaceWeekendManager {
	return &RaceWeekendManager{
		raceManager: raceManager,
		store:       store,
		process:     process,
	}
}

func (rwm *RaceWeekendManager) ListRaceWeekends() ([]*RaceWeekend, error) {
	return rwm.store.ListRaceWeekends()
}

func (rwm *RaceWeekendManager) LoadRaceWeekend(id string) (*RaceWeekend, error) {
	return rwm.store.LoadRaceWeekend(id)
}

func (rwm *RaceWeekendManager) BuildRaceWeekendTemplateOpts(r *http.Request) (map[string]interface{}, error) {
	opts, err := rwm.raceManager.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	if existingID := chi.URLParam(r, "raceWeekendID"); existingID != "" {
		opts["IsEditing"] = true
		currentRaceWeekend, err := rwm.store.LoadRaceWeekend(existingID)

		if err != nil {
			return nil, err
		}

		opts["Current"] = currentRaceWeekend
	} else {
		opts["IsEditing"] = false
		opts["Current"] = NewRaceWeekend()
	}

	return opts, nil
}

func (rwm *RaceWeekendManager) SaveRaceWeekend(r *http.Request) (raceWeekend *RaceWeekend, edited bool, err error) {
	if err := r.ParseForm(); err != nil {
		return nil, false, err
	}

	entryList, err := rwm.raceManager.BuildEntryList(r, 0, len(r.Form["EntryList.Name"]))

	if err != nil {
		return nil, edited, err
	}

	if raceWeekendID := r.FormValue("Editing"); raceWeekendID != "" {
		raceWeekend, err = rwm.LoadRaceWeekend(raceWeekendID)

		if err != nil {
			return nil, edited, err
		}
	} else {
		raceWeekend = NewRaceWeekend()
	}

	raceWeekend.Name = r.FormValue("RaceWeekendName")
	raceWeekend.EntryList = entryList

	return raceWeekend, edited, rwm.store.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) BuildRaceWeekendSessionOpts(r *http.Request) (map[string]interface{}, error) {
	opts, err := rwm.raceManager.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	// here we customise the opts to tell the template that this is a race weekend session.
	raceWeekend, err := rwm.LoadRaceWeekend(chi.URLParam(r, "raceWeekendID"))

	if err != nil {
		return nil, err
	}

	opts["IsRaceWeekend"] = true
	opts["RaceWeekend"] = raceWeekend

	if editSessionID := chi.URLParam(r, "sessionID"); editSessionID != "" {
		// editing a race weekend session
		session, err := raceWeekend.FindSessionByID(editSessionID)

		if err != nil {
			return nil, err
		}

		opts["Current"] = session.RaceConfig
		opts["IsEditing"] = true
		opts["EditingID"] = editSessionID
		opts["CurrentEntrants"], err = session.GetEntryList(raceWeekend)

		if err != nil {
			return nil, err
		}
	} else {
		// creating a new championship event
		opts["IsEditing"] = false
		opts["CurrentEntrants"] = raceWeekend.EntryList

		// override Current race config if there is a previous championship race configured
		if len(raceWeekend.Sessions) > 0 {
			opts["Current"] = raceWeekend.Sessions[len(raceWeekend.Sessions)-1].RaceConfig
			spew.Dump(raceWeekend.Sessions[len(raceWeekend.Sessions)-1].RaceConfig.Sessions)

			opts["RaceWeekendHasAtLeastOneSession"] = true
		} else {
			current := ConfigIniDefault().CurrentRaceConfig
			delete(current.Sessions, SessionTypeBooking)
			delete(current.Sessions, SessionTypeQualifying)
			delete(current.Sessions, SessionTypeRace)

			opts["Current"] = current
			opts["RaceWeekendHasAtLeastOneSession"] = false
		}
	}

	opts["AvailableSessions"] = AvailableSessionsNoBooking

	err = rwm.raceManager.applyCurrentRaceSetupToOptions(opts, opts["Current"].(CurrentRaceConfig))

	if err != nil {
		return nil, err
	}

	return opts, nil
}

func (rwm *RaceWeekendManager) SaveRaceWeekendSession(r *http.Request) (raceWeekend *RaceWeekend, session *RaceWeekendSession, edited bool, err error) {
	if err := r.ParseForm(); err != nil {
		return nil, nil, false, err
	}

	raceWeekend, err = rwm.LoadRaceWeekend(chi.URLParam(r, "raceWeekendID"))

	if err != nil {
		return nil, nil, false, err
	}

	raceConfig, err := rwm.raceManager.BuildCustomRaceFromForm(r)

	if err != nil {
		return nil, nil, false, err
	}

	raceConfig.Cars = strings.Join(raceWeekend.EntryList.CarIDs(), ";")

	// remove all but the active session from the setup.
	activeSession := r.FormValue("SessionType")

	for session := range raceConfig.Sessions {
		if session.String() != activeSession {
			delete(raceConfig.Sessions, session)
		}
	}

	if sessionID := r.FormValue("Editing"); sessionID != "" {
		edited = true

		session, err = raceWeekend.FindSessionByID(sessionID)

		if err != nil {
			return nil, nil, true, err
		}

		// we're editing an existing session
		session.RaceConfig = *raceConfig
	} else {
		// creating a new event
		session = NewRaceWeekendSession()
		session.RaceConfig = *raceConfig

		raceWeekend.AddSession(session, nil) // @TODO how do we link sessions?
	}

	return raceWeekend, session, edited, rwm.store.UpsertRaceWeekend(raceWeekend)
}

var ErrIncompleteSessionDependencies = errors.New("servermanager: incomplete session dependencies")

func (rwm *RaceWeekendManager) StartSession(raceWeekendID string, raceWeekendSessionID string) error {
	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return err
	}

	session, err := raceWeekend.FindSessionByID(raceWeekendSessionID)

	if err != nil {
		return err
	}

	if !raceWeekend.SessionCanBeRun(session) {
		return ErrIncompleteSessionDependencies
	}

	session.StartedTime = time.Now()

	if err := rwm.store.UpsertRaceWeekend(raceWeekend); err != nil {
		return err
	}

	entryList, err := session.GetEntryList(raceWeekend)

	if err != nil {
		return err
	}

	// @TODO replace normalEvent with something better here
	return rwm.raceManager.applyConfigAndStart(session.RaceConfig, entryList, false, normalEvent{})
}
