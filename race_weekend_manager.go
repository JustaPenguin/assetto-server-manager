package servermanager

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type RaceWeekendManager struct {
	raceManager *RaceManager
	store       Store
	process     ServerProcess

	activeRaceWeekend *ActiveRaceWeekend
	mutex             sync.Mutex
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

func (rwm *RaceWeekendManager) BuildRaceWeekendTemplateOpts(r *http.Request) (*RaceTemplateVars, error) {
	opts, err := rwm.raceManager.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	var raceWeekend *RaceWeekend
	var isEditing bool

	if existingID := chi.URLParam(r, "raceWeekendID"); existingID != "" {
		isEditing = true
		currentRaceWeekend, err := rwm.store.LoadRaceWeekend(existingID)

		if err != nil {
			return nil, err
		}

		raceWeekend = currentRaceWeekend
	} else {
		isEditing = false
		raceWeekend = NewRaceWeekend()
	}

	var championshipID string

	if !isEditing {
		championshipID = r.URL.Query().Get("championshipID")
	} else {
		championshipID = raceWeekend.ChampionshipID.String()
	}

	if championshipID != "" {
		championship, err := rwm.store.LoadChampionship(championshipID)

		if err != nil {
			return nil, err
		}

		raceWeekend.ChampionshipID = championship.ID

		opts.Championship = championship
	}

	opts.RaceWeekend = raceWeekend
	opts.IsEditing = isEditing

	return opts, nil
}

func (rwm *RaceWeekendManager) SaveRaceWeekend(r *http.Request) (raceWeekend *RaceWeekend, edited bool, err error) {
	if err := r.ParseForm(); err != nil {
		return nil, false, err
	}

	if raceWeekendID := r.FormValue("Editing"); raceWeekendID != "" {
		raceWeekend, err = rwm.LoadRaceWeekend(raceWeekendID)

		if err != nil {
			return nil, edited, err
		}

		edited = true
	} else {
		raceWeekend = NewRaceWeekend()
	}

	raceWeekend.Name = r.FormValue("RaceWeekendName")

	if championshipID := r.FormValue("ChampionshipID"); championshipID != "" {
		champ, err := rwm.store.LoadChampionship(championshipID)

		if err != nil {
			return nil, edited, err
		}

		raceWeekend.ChampionshipID = champ.ID

		if !edited {
			// add a championship event for this race weekend
			event := NewChampionshipEvent()
			event.RaceWeekendID = raceWeekend.ID

			champ.Events = append(champ.Events, event)

			if err := rwm.store.UpsertChampionship(champ); err != nil {
				return nil, edited, err
			}
		}
	} else {
		entryList, err := rwm.raceManager.BuildEntryList(r, 0, len(r.Form["EntryList.Name"]))

		if err != nil {
			return nil, edited, err
		}

		raceWeekend.EntryList = entryList
	}

	return raceWeekend, edited, rwm.store.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) BuildRaceWeekendSessionOpts(r *http.Request) (*RaceTemplateVars, error) {
	opts, err := rwm.raceManager.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	// here we customise the opts to tell the template that this is a race weekend session.
	raceWeekend, err := rwm.LoadRaceWeekend(chi.URLParam(r, "raceWeekendID"))

	if err != nil {
		return nil, err
	}

	var raceWeekendSession *RaceWeekendSession

	if editSessionID := chi.URLParam(r, "sessionID"); editSessionID != "" {
		// editing a race weekend session
		raceWeekendSession, err = raceWeekend.FindSessionByID(editSessionID)

		if err != nil {
			return nil, err
		}

		opts.Current = raceWeekendSession.RaceConfig
		opts.IsEditing = true
		opts.EditingID = editSessionID
		entryList, err := raceWeekendSession.GetRaceWeekendEntryList(raceWeekend, nil, "")

		if err == ErrRaceWeekendSessionDependencyIncomplete {
			opts.CurrentEntrants = raceWeekend.EntryList

		} else if err != nil {
			return nil, err
		} else {
			opts.CurrentEntrants = entryList.AsEntryList()
		}
	} else {
		// creating a new race weekend session
		opts.IsEditing = false
		opts.CurrentEntrants = raceWeekend.EntryList

		// override Current race config if there is a previous race weekend race configured
		if len(raceWeekend.Sessions) > 0 {
			opts.Current = raceWeekend.Sessions[len(raceWeekend.Sessions)-1].RaceConfig

			opts.RaceWeekendHasAtLeastOneSession = true
		} else {
			current := ConfigIniDefault().CurrentRaceConfig
			delete(current.Sessions, SessionTypeBooking)
			delete(current.Sessions, SessionTypeQualifying)
			delete(current.Sessions, SessionTypeRace)

			opts.Current = current
			opts.RaceWeekendHasAtLeastOneSession = false
		}

		raceWeekendSession = NewRaceWeekendSession()
	}

	if raceWeekend.HasLinkedChampionship() {
		championship, err := rwm.store.LoadChampionship(raceWeekend.ChampionshipID.String())

		if err != nil {
			return nil, err
		}

		raceWeekend.EntryList = championship.AllEntrants()

		for _, class := range championship.Classes {
			raceWeekendSession.Points[class.ID] = &class.Points
		}
	}

	opts.RaceWeekendSession = raceWeekendSession
	opts.IsRaceWeekend = true
	opts.RaceWeekend = raceWeekend

	opts.AvailableSessions = AvailableSessionsNoBooking

	err = rwm.raceManager.applyCurrentRaceSetupToOptions(opts, opts.Current)

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

		raceWeekend.AddSession(session, nil)
	}

	// empty out previous parent IDs
	session.ParentIDs = []uuid.UUID{}

	// assign parents
	for _, parentID := range r.Form["ParentSessions"] {
		if parentID == "no_parent" {
			break
		}

		id, err := uuid.Parse(parentID)

		if err != nil {
			return nil, nil, false, err
		}

		session.ParentIDs = append(session.ParentIDs, id)
	}

	return raceWeekend, session, edited, rwm.store.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) applyConfigAndStart(config CurrentRaceConfig, entryList EntryList, raceWeekend *ActiveRaceWeekend) error {
	err := rwm.raceManager.applyConfigAndStart(config, entryList, false, raceWeekend)

	if err != nil {
		return err
	}

	rwm.activeRaceWeekend = raceWeekend

	return nil
}

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
		return ErrRaceWeekendSessionDependencyIncomplete
	}

	session.StartedTime = time.Now()

	if err := rwm.store.UpsertRaceWeekend(raceWeekend); err != nil {
		return err
	}

	raceWeekendEntryList, err := session.GetRaceWeekendEntryList(raceWeekend, nil, "")

	if err != nil {
		return err
	}

	entryList := raceWeekendEntryList.AsEntryList()

	session.RaceConfig.MaxClients = len(entryList)

	session.RaceConfig.Cars = strings.Join(entryList.CarIDs(), ";")

	session.RaceConfig.LockedEntryList = 1
	session.RaceConfig.PickupModeEnabled = 0

	// all race weekend sessions must be open so players can join
	for _, acSession := range session.RaceConfig.Sessions {
		acSession.IsOpen = 1
	}

	return rwm.applyConfigAndStart(session.RaceConfig, entryList, &ActiveRaceWeekend{
		Name:                raceWeekend.Name,
		RaceWeekendID:       raceWeekend.ID,
		SessionID:           session.ID,
		OverridePassword:    false, // @TODO
		ReplacementPassword: "",    // @TODO
		Description:         "",    // @TODO?
	})
}

func (rwm *RaceWeekendManager) UDPCallback(message udp.Message) {
	rwm.mutex.Lock()
	defer rwm.mutex.Unlock()

	if !rwm.process.Event().IsRaceWeekend() || rwm.activeRaceWeekend == nil {
		return
	}

	switch m := message.(type) {
	case udp.EndSession:
		filename := filepath.Base(string(m))
		logrus.Infof("Race Weekend: End session found, result file: %s", filename)

		results, err := LoadResult(filename)

		if err != nil {
			logrus.WithError(err).Errorf("Could not read session results for race weekend: %s, session: %s", rwm.activeRaceWeekend.RaceWeekendID.String(), rwm.activeRaceWeekend.SessionID.String())
			return
		}

		raceWeekend, err := rwm.LoadRaceWeekend(rwm.activeRaceWeekend.RaceWeekendID.String())

		if err != nil {
			logrus.WithError(err).Errorf("Could not load active race weekend")
			return
		}

		session, err := raceWeekend.FindSessionByID(rwm.activeRaceWeekend.SessionID.String())

		if err != nil {
			logrus.WithError(err).Errorf("Could not load active race weekend session")
			return
		}

		session.CompletedTime = time.Now()
		session.Results = results

		if err := rwm.store.UpsertRaceWeekend(raceWeekend); err != nil {
			logrus.WithError(err).Errorf("Could not persist race weekend: %s", raceWeekend.ID.String())
			return
		}

		if err := rwm.process.Stop(); err != nil {
			logrus.WithError(err).Error("Could not stop assetto server process")
		}
	}
}

func (rwm *RaceWeekendManager) RestartSession(raceWeekendID string, raceWeekendSessionID string) error {
	err := rwm.CancelSession(raceWeekendID, raceWeekendSessionID)

	if err != nil {
		return err
	}

	return rwm.StartSession(raceWeekendID, raceWeekendSessionID)
}

func (rwm *RaceWeekendManager) CancelSession(raceWeekendID string, raceWeekendSessionID string) error {
	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return err
	}

	session, err := raceWeekend.FindSessionByID(raceWeekendSessionID)

	if err != nil {
		return err
	}

	session.StartedTime = time.Time{}
	session.CompletedTime = time.Time{}
	session.Results = nil

	if err := rwm.process.Stop(); err != nil {
		return err
	}

	return rwm.store.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) DeleteSession(raceWeekendID string, raceWeekendSessionID string) error {
	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return err
	}

	raceWeekend.DelSession(raceWeekendSessionID)

	return rwm.store.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) DeleteRaceWeekend(id string) error {
	return rwm.store.DeleteRaceWeekend(id)
}

func (rwm *RaceWeekendManager) StopActiveSession() error {
	if !rwm.process.Event().IsRaceWeekend() || rwm.activeRaceWeekend == nil {
		return ErrNoActiveEvent
	}

	return rwm.CancelSession(rwm.activeRaceWeekend.RaceWeekendID.String(), rwm.activeRaceWeekend.SessionID.String())
}

func (rwm *RaceWeekendManager) ImportSession(raceWeekendID string, raceWeekendSessionID string, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return err
	}

	session, err := raceWeekend.FindSessionByID(raceWeekendSessionID)

	if err != nil {
		return err
	}

	session.Results, err = LoadResult(r.FormValue("ResultFile") + ".json")

	if err != nil {
		return err
	}

	session.CompletedTime = session.Results.Date

	return rwm.store.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) ListAvailableResultsFilesForSession(raceWeekendID string, raceWeekendSessionID string) (*RaceWeekendSession, []SessionResults, error) {
	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return nil, nil, err
	}

	session, err := raceWeekend.FindSessionByID(raceWeekendSessionID)

	if err != nil {
		return nil, nil, err
	}

	results, err := ListAllResults()

	if err != nil {
		return nil, nil, err
	}

	var filteredResults []SessionResults

	for _, result := range results {
		if result.TrackName == session.RaceConfig.Track && result.TrackConfig == session.RaceConfig.TrackLayout {
			filteredResults = append(filteredResults, result)
		}
	}

	return session, filteredResults, nil
}

func (rwm *RaceWeekendManager) FindSession(raceWeekendID, sessionID string) (*RaceWeekend, *RaceWeekendSession, error) {
	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return nil, nil, err
	}

	session, err := raceWeekend.FindSessionByID(sessionID)

	if err != nil {
		return nil, nil, err
	}

	return raceWeekend, session, nil
}

func (rwm *RaceWeekendManager) FindConnectedSessions(raceWeekendID, parentSessionID, childSessionID string) (*RaceWeekend, *RaceWeekendSession, *RaceWeekendSession, error) {
	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return nil, nil, nil, err
	}

	parentSession, err := raceWeekend.FindSessionByID(parentSessionID)

	if err != nil {
		return nil, nil, nil, err
	}

	childSession, err := raceWeekend.FindSessionByID(childSessionID)

	if err != nil {
		return nil, nil, nil, err
	}

	return raceWeekend, parentSession, childSession, nil
}

type RaceWeekendGridPreview struct {
	Results map[int]string
	Grid    map[int]string
}

func NewRaceWeekendGridPreview() *RaceWeekendGridPreview {
	return &RaceWeekendGridPreview{
		Results: make(map[int]string),
		Grid:    make(map[int]string),
	}
}

func (rwm *RaceWeekendManager) PreviewGrid(raceWeekendID, parentSessionID, childSessionID string, filter *RaceWeekendSessionToSessionFilter) (*RaceWeekendGridPreview, error) {
	raceWeekend, parentSession, childSession, err := rwm.FindConnectedSessions(raceWeekendID, parentSessionID, childSessionID)

	if err != nil {
		return nil, err
	}

	preview := NewRaceWeekendGridPreview()

	for i, result := range parentSession.FinishingGrid(raceWeekend) {
		preview.Results[i+1] = result.Car.GetName()
	}

	entryList, err := childSession.GetRaceWeekendEntryList(raceWeekend, filter, parentSessionID)

	if err != nil {
		return nil, err
	}

	for _, entrant := range entryList.Sorted() {
		sess, err := raceWeekend.FindSessionByID(entrant.SessionID.String())

		if err != nil {
			continue
		}

		preview.Grid[entrant.PitBox+1] = fmt.Sprintf("%s (%s)", entrant.Car.GetName(), sess.Name())
	}

	return preview, nil
}

func (rwm *RaceWeekendManager) UpdateGrid(raceWeekendID, parentSessionID, childSessionID string, filter *RaceWeekendSessionToSessionFilter) error {
	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return err
	}

	raceWeekend.AddFilter(parentSessionID, childSessionID, filter)

	return rwm.store.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) PreviewSessionEntryList(raceWeekendID, sessionID, sortType string, reverseGrid int) (*RaceWeekendGridPreview, error) {
	raceWeekend, session, err := rwm.FindSession(raceWeekendID, sessionID)

	if err != nil {
		return nil, err
	}

	session.SortType = sortType
	session.NumEntrantsToReverse = reverseGrid

	entryList, err := session.GetRaceWeekendEntryList(raceWeekend, nil, "")

	if err != nil {
		return nil, err
	}

	preview := NewRaceWeekendGridPreview()

	for i, entrant := range entryList.Sorted() {
		entrantSession, err := raceWeekend.FindSessionByID(entrant.SessionID.String())

		if err != nil {
			continue
		}

		sessionText := entrantSession.Name()

		if entrantSession.Completed() {
			foundEntrant := false

			for i, result := range entrantSession.Results.Result {
				if result.DriverGUID == entrant.Car.Driver.GUID {
					sessionText += fmt.Sprintf(" - %d%s", i+1, ordinal(int64(i+1)))
					foundEntrant = true
					break
				}
			}

			if !foundEntrant {
				sessionText += " - No Time"
			}
		}

		preview.Grid[i+1] = fmt.Sprintf("%s (%s)", entrant.Car.GetName(), sessionText)
	}

	return preview, nil
}

func (rwm *RaceWeekendManager) UpdateSessionSorting(raceWeekendID, sessionID string, sortType string, numEntrantsToReverse int) error {
	raceWeekend, session, err := rwm.FindSession(raceWeekendID, sessionID)

	if err != nil {
		return err
	}

	session.SortType = sortType
	session.NumEntrantsToReverse = numEntrantsToReverse

	return rwm.store.UpsertRaceWeekend(raceWeekend)
}
