package servermanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"
	"github.com/JustaPenguin/assetto-server-manager/pkg/when"

	"github.com/cj123/ini"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
)

type RaceWeekendManager struct {
	raceManager         *RaceManager
	championshipManager *ChampionshipManager
	notificationManager NotificationDispatcher
	store               Store
	process             ServerProcess
	acsrClient          *ACSRClient

	activeRaceWeekend *ActiveRaceWeekend
	mutex             sync.Mutex

	scheduledSessionTimers         map[string]*when.Timer
	scheduledSessionReminderTimers map[string]*when.Timer
}

func NewRaceWeekendManager(
	raceManager *RaceManager,
	championshipManager *ChampionshipManager,
	store Store,
	process ServerProcess,
	notificationManager NotificationDispatcher,
	acsrClient *ACSRClient,
) *RaceWeekendManager {
	return &RaceWeekendManager{
		raceManager:         raceManager,
		championshipManager: championshipManager,
		notificationManager: notificationManager,
		store:               store,
		process:             process,
		acsrClient:          acsrClient,

		scheduledSessionTimers:         make(map[string]*when.Timer),
		scheduledSessionReminderTimers: make(map[string]*when.Timer),
	}
}

func (rwm *RaceWeekendManager) ListRaceWeekends() ([]*RaceWeekend, error) {
	return rwm.store.ListRaceWeekends()
}

func (rwm *RaceWeekendManager) LoadRaceWeekend(id string) (*RaceWeekend, error) {
	raceWeekend, err := rwm.store.LoadRaceWeekend(id)

	if err != nil {
		return nil, err
	}

	if raceWeekend.HasLinkedChampionship() {
		raceWeekend.Championship, err = rwm.store.LoadChampionship(raceWeekend.ChampionshipID.String())

		if err != nil {
			return nil, err
		}

		// make sure that session points only exist for classes that exist.
		for _, session := range raceWeekend.Sessions {
			for championshipClassID := range session.Points {
				if _, err := raceWeekend.Championship.ClassByID(championshipClassID.String()); err != nil {
					delete(session.Points, championshipClassID)
				}
			}
		}
	}

	return raceWeekend, nil
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

	if championshipID != uuid.Nil.String() && championshipID != "" {
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

		// persist race weekend entrants in the autofill entry list
		if err := rwm.raceManager.SaveEntrantsForAutoFill(entryList); err != nil {
			return raceWeekend, edited, err
		}
	}

	return raceWeekend, edited, rwm.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) UpsertRaceWeekend(raceWeekend *RaceWeekend) error {
	err := rwm.store.UpsertRaceWeekend(raceWeekend)

	if err != nil {
		return err
	}

	if raceWeekend.HasLinkedChampionship() {
		championship, err := rwm.championshipManager.LoadChampionship(raceWeekend.ChampionshipID.String())

		if err != nil {
			return err
		}

		if championship.ACSR {
			rwm.acsrClient.SendChampionship(*championship)
		}
	}

	return nil
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
			opts.CurrentEntrants = raceWeekend.GetEntryList()
		} else if err != nil {
			return nil, err
		} else {
			opts.CurrentEntrants = entryList.AsEntryList()
		}
	} else {
		// creating a new race weekend session
		opts.IsEditing = false
		opts.CurrentEntrants = raceWeekend.GetEntryList()

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
		opts.Championship = raceWeekend.Championship

		if !opts.IsEditing {
			for _, class := range opts.Championship.Classes {
				raceWeekendSession.Points[class.ID] = &class.Points
			}
		}
	}

	opts.RaceWeekendSession = raceWeekendSession
	opts.IsRaceWeekend = true
	opts.RaceWeekend = raceWeekend

	opts.AvailableSessions = AvailableSessionsNoBooking
	opts.ShowOverridePasswordCard = !raceWeekend.HasLinkedChampionship()
	opts.OverridePassword = raceWeekendSession.OverridePassword
	opts.ReplacementPassword = raceWeekendSession.ReplacementPassword

	err = rwm.raceManager.applyCurrentRaceSetupToOptions(opts, opts.Current)

	if err != nil {
		return nil, err
	}

	return opts, nil
}

func (rwm *RaceWeekendManager) SaveRaceWeekendSession(r *http.Request) (raceWeekend *RaceWeekend, session *RaceWeekendSession, edited bool, err error) {
	if err := r.ParseForm(); err != nil {
		return nil, nil, edited, err
	}

	raceWeekend, err = rwm.LoadRaceWeekend(chi.URLParam(r, "raceWeekendID"))

	if err != nil {
		return nil, nil, edited, err
	}

	raceConfig, err := rwm.raceManager.BuildCustomRaceFromForm(r)

	if err != nil {
		return nil, nil, edited, err
	}

	raceConfig.Cars = strings.Join(raceWeekend.GetEntryList().CarIDs(), ";")

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
			return nil, nil, edited, err
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
			// empty out any existing ones
			session.ParentIDs = []uuid.UUID{}
			break
		}

		id, err := uuid.Parse(parentID)

		if err != nil {
			return nil, nil, edited, err
		}

		session.ParentIDs = append(session.ParentIDs, id)
	}

	if len(session.ParentIDs) == 0 {
		session.ParentIDs = append(session.ParentIDs, raceWeekend.ID)
	}

	session.OverridePassword = r.FormValue("OverridePassword") == "1"
	session.ReplacementPassword = r.FormValue("ReplacementPassword")

	if raceWeekend.HasLinkedChampionship() {
		// points
		previousNumPoints := 0

		for i := 0; i < len(r.Form["ClassID"]); i++ {
			classID, err := uuid.Parse(r.Form["ClassID"][i])

			if err != nil {
				return nil, nil, edited, err
			}

			numPointsForClass := formValueAsInt(r.Form["NumPointsForClass"][i])

			pts := &ChampionshipPoints{}

			pts.Places = make([]int, 0)

			for i := previousNumPoints; i < previousNumPoints+numPointsForClass; i++ {
				pts.Places = append(pts.Places, formValueAsInt(r.Form["Points.Place"][i]))
			}

			pts.BestLap = formValueAsInt(r.Form["Points.BestLap"][i])
			pts.CollisionWithDriver = formValueAsInt(r.Form["Points.CollisionWithDriver"][i])
			pts.CollisionWithEnv = formValueAsInt(r.Form["Points.CollisionWithEnv"][i])
			pts.CutTrack = formValueAsInt(r.Form["Points.CutTrack"][i])

			previousNumPoints += numPointsForClass
			session.Points[classID] = pts
		}
	}

	return raceWeekend, session, edited, rwm.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) applyConfigAndStart(raceWeekend *ActiveRaceWeekend) error {
	rwm.activeRaceWeekend = raceWeekend

	err := rwm.raceManager.applyConfigAndStart(raceWeekend)

	if err != nil {
		return err
	}

	return nil
}

func (rwm *RaceWeekendManager) StartPracticeSession(raceWeekendID string, raceWeekendSessionID string) error {
	return rwm.StartSession(raceWeekendID, raceWeekendSessionID, true)
}

func (rwm *RaceWeekendManager) StartSession(raceWeekendID string, raceWeekendSessionID string, isPracticeSession bool) error {
	if !Premium() {
		return errors.New("servermanager: premium required")
	}

	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return err
	}

	session, err := raceWeekend.FindSessionByID(raceWeekendSessionID)

	if err != nil {
		return err
	}

	if !isPracticeSession {
		if !raceWeekend.SessionCanBeRun(session) {
			return ErrRaceWeekendSessionDependencyIncomplete
		}

		session.StartedTime = time.Now()

		if err := rwm.UpsertRaceWeekend(raceWeekend); err != nil {
			return err
		}
	}

	raceWeekendEntryList, err := session.GetRaceWeekendEntryList(raceWeekend, nil, "")

	if err != nil {
		return err
	}

	entryList := raceWeekendEntryList.AsEntryList()

	if isPracticeSession && !raceWeekend.SessionCanBeRun(session) {
		// practice sessions run with the whole race weekend entry list if they are not yet available
		entryList = raceWeekend.GetEntryList()
	}

	for k, entrant := range entryList {
		if entrant.IsPlaceHolder {
			// placeholder entrants should not be added to our final entry list
			delete(entryList, k)
			continue
		}

		// look through the user configured entry list and apply any of the options that they set to this entrant.
		for _, raceWeekendEntrant := range raceWeekend.GetEntryList() {
			if raceWeekendEntrant.GUID == entrant.GUID {
				entrant.Model = raceWeekendEntrant.Model
				entrant.Ballast = raceWeekendEntrant.Ballast
				entrant.Restrictor = raceWeekendEntrant.Restrictor
				if entrant.FixedSetup == "" {
					entrant.FixedSetup = raceWeekendEntrant.FixedSetup
				}
				entrant.Skin = raceWeekendEntrant.Skin
				break
			}
		}
	}

	session.RaceConfig.MaxClients = len(entryList)
	session.RaceConfig.Cars = strings.Join(entryList.CarIDs(), ";")
	session.RaceConfig.LockedEntryList = 1
	session.RaceConfig.PickupModeEnabled = 0

	// all race weekend sessions must be open so players can join
	for _, acSession := range session.RaceConfig.Sessions {
		acSession.IsOpen = 1
	}

	overridePassword := session.OverridePassword
	replacementPassword := session.ReplacementPassword

	if raceWeekend.HasLinkedChampionship() {
		overridePassword = raceWeekend.Championship.OverridePassword
		replacementPassword = raceWeekend.Championship.ReplacementPassword
	}

	raceWeekendRaceEvent := &ActiveRaceWeekend{
		Name:                raceWeekend.Name,
		RaceWeekendID:       raceWeekend.ID,
		SessionID:           session.ID,
		OverridePassword:    overridePassword,
		ReplacementPassword: replacementPassword,
		Description:         fmt.Sprintf("This is a session in the '%s' Race Weekend.", raceWeekend.Name),
		RaceConfig:          session.RaceConfig,
		EntryList:           entryList,
	}

	if isPracticeSession {
		delete(session.RaceConfig.Sessions, SessionTypePractice)
		delete(session.RaceConfig.Sessions, SessionTypeQualifying)
		delete(session.RaceConfig.Sessions, SessionTypeRace)

		session.RaceConfig.Sessions[SessionTypePractice] = &SessionConfig{
			Name:   "Practice",
			Time:   120,
			IsOpen: 1,
		}

		session.RaceConfig.LoopMode = 1

		raceWeekendRaceEvent.IsPracticeSession = true
		raceWeekendRaceEvent.RaceConfig = session.RaceConfig
		raceWeekendRaceEvent.EntryList = entryList

		return rwm.raceManager.applyConfigAndStart(raceWeekendRaceEvent)
	}

	return rwm.applyConfigAndStart(raceWeekendRaceEvent)
}

func (rwm *RaceWeekendManager) UDPCallback(message udp.Message) {
	rwm.mutex.Lock()
	defer rwm.mutex.Unlock()

	if !rwm.RaceWeekendSessionIsRunning() {
		return
	}

	if m, ok := message.(udp.EndSession); ok {
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

		raceWeekend.EnhanceResults(results)

		err = saveResults(filename, results)

		if err != nil {
			logrus.WithError(err).Errorf("Could not update session results %s", filename)
			return
		}

		session.Results = results

		if err := rwm.UpsertRaceWeekend(raceWeekend); err != nil {
			logrus.WithError(err).Errorf("Could not persist race weekend: %s", raceWeekend.ID.String())
			return
		}

		if err := rwm.process.Stop(); err != nil {
			logrus.WithError(err).Error("Could not stop assetto server process")
		}

		if err := rwm.ClearLockedTyreSetups(raceWeekend, session); err != nil {
			logrus.WithError(err).Error("Could not clear previous locked tyres")
		}

		// first, look at siblings of this session and see if they were due to be started
		for _, parent := range session.ParentIDs {
			siblings := raceWeekend.FindChildren(parent.String())

			for _, sibling := range siblings {
				if !sibling.Completed() && sibling.StartWhenParentHasFinished {
					err := rwm.StartSession(raceWeekend.ID.String(), sibling.ID.String(), false)

					if err != nil {
						logrus.WithError(err).Error("Could not start child session")
					}

					return
				}
			}
		}

		// now we can look and see if any child sessions of this session should be started when it finishes
		children := raceWeekend.FindChildren(session.ID.String())

		for _, child := range children {
			if !child.Completed() && child.StartWhenParentHasFinished {
				err := rwm.StartSession(raceWeekend.ID.String(), child.ID.String(), false)

				if err != nil {
					logrus.WithError(err).Error("Could not start child session")
				}

				return
			}
		}
	}
}

func (rwm *RaceWeekendManager) ClearLockedTyreSetups(raceWeekend *RaceWeekend, session *RaceWeekendSession) error {
	matches, err := zglob.Glob(filepath.Join(ServerInstallPath, "setups", "**", lockedTyreSetupFolder, "race_weekend_session_*.ini"))

	if err != nil {
		return err
	}

	for _, match := range matches {
		i, err := ini.Load(match)

		if err != nil {
			return err
		}

		section, err := i.GetSection("RACE_WEEKEND")

		if err != nil {
			return err
		}

		if raceWeekendID, err := section.GetKey("ID"); err == nil && raceWeekendID.String() == raceWeekend.ID.String() {
			if sessionID, err := section.GetKey("SESSION_ID"); err == nil && sessionID.String() == session.ID.String() {
				// this file was from the session that just finished. delete it
				err := os.Remove(match)

				if err != nil {
					return err
				}
			} else if err != nil {
				logrus.WithError(err).Warn("Could not read SessionID from setup file")
				continue
			}
		} else if err != nil {
			logrus.WithError(err).Warn("Could not read RaceWeekendID from setup file")
			continue
		}
	}

	return nil
}

func (rwm *RaceWeekendManager) RestartSession(raceWeekendID string, raceWeekendSessionID string) error {
	err := rwm.CancelSession(raceWeekendID, raceWeekendSessionID)

	if err != nil {
		return err
	}

	return rwm.StartSession(raceWeekendID, raceWeekendSessionID, false)
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

	return rwm.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) DeleteSession(raceWeekendID string, raceWeekendSessionID string) error {
	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return err
	}

	raceWeekend.DelSession(raceWeekendSessionID)

	return rwm.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) DeleteRaceWeekend(id string) error {
	return rwm.store.DeleteRaceWeekend(id)
}

var ErrNoActiveRaceWeekendSession = errors.New("servermanager: no active race weekend session")

func (rwm *RaceWeekendManager) RaceWeekendSessionIsRunning() bool {
	return rwm.process.Event().IsRaceWeekend() && !rwm.process.Event().IsPractice() && rwm.activeRaceWeekend != nil
}

func (rwm *RaceWeekendManager) StopActiveSession() error {
	if !rwm.RaceWeekendSessionIsRunning() {
		return ErrNoActiveRaceWeekendSession
	}

	return rwm.CancelSession(rwm.activeRaceWeekend.RaceWeekendID.String(), rwm.activeRaceWeekend.SessionID.String())
}

func (rwm *RaceWeekendManager) RestartActiveSession() error {
	if !rwm.RaceWeekendSessionIsRunning() {
		return ErrNoActiveRaceWeekendSession
	}

	return rwm.RestartSession(rwm.activeRaceWeekend.RaceWeekendID.String(), rwm.activeRaceWeekend.SessionID.String())
}

func (rwm *RaceWeekendManager) ImportSession(raceWeekendID string, raceWeekendSessionID string, r *http.Request) error {
	if !Premium() {
		return errors.New("servermanager: premium required")
	}

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

	filename := r.FormValue("ResultFile") + ".json"

	session.Results, err = LoadResult(filename)

	if err != nil {
		return err
	}

	raceWeekend.EnhanceResults(session.Results)

	err = saveResults(filename, session.Results)

	if err != nil {
		return err
	}

	session.CompletedTime = session.Results.Date

	return rwm.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) ListAvailableResultsFilesForSorting(raceWeekend *RaceWeekend, session *RaceWeekendSession) ([]SessionResults, error) {
	results, err := ListAllResults()

	if err != nil {
		return nil, err
	}

	var filteredResults []SessionResults

	for _, result := range results {

		found := false

	carCheck:
		for _, car := range result.Cars {
			for _, entryListCar := range raceWeekend.GetEntryList().CarIDs() {
				if car.Model == entryListCar {
					// result car found in entry list
					found = true
					break carCheck
				}
			}
		}

		if result.TrackName == session.RaceConfig.Track && result.TrackConfig == session.RaceConfig.TrackLayout && found {
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults, nil
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
	Results map[int]SessionPreviewEntrant
	Grid    map[int]SessionPreviewEntrant
	Classes map[string]string
}

type SessionPreviewEntrant struct {
	Name       string
	Session    string
	Class      string
	ClassColor string
}

func NewRaceWeekendGridPreview() *RaceWeekendGridPreview {
	return &RaceWeekendGridPreview{
		Results: make(map[int]SessionPreviewEntrant),
		Grid:    make(map[int]SessionPreviewEntrant),
		Classes: make(map[string]string),
	}
}

func (rwm *RaceWeekendManager) PreviewGrid(raceWeekendID, parentSessionID, childSessionID string, filter *RaceWeekendSessionToSessionFilter) (*RaceWeekendGridPreview, error) {
	raceWeekend, parentSession, childSession, err := rwm.FindConnectedSessions(raceWeekendID, parentSessionID, childSessionID)

	if err != nil {
		return nil, err
	}

	preview := NewRaceWeekendGridPreview()

	finishingGrid, err := parentSession.FinishingGrid(raceWeekend)

	if err != nil {
		return nil, err
	}

	for i, result := range finishingGrid {
		class := result.ChampionshipClass(raceWeekend)

		color, ok := preview.Classes[class.Name]

		if !ok {
			color = ChampionshipClassColor(len(preview.Classes))
			preview.Classes[class.Name] = color
		}

		preview.Results[i+1] = SessionPreviewEntrant{
			Name:       result.Car.GetName(),
			Session:    parentSession.Name(),
			Class:      class.Name,
			ClassColor: color,
		}
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

		class := entrant.ChampionshipClass(raceWeekend)

		color, ok := preview.Classes[class.Name]

		if !ok {
			color = ChampionshipClassColor(len(preview.Classes))
			preview.Classes[class.Name] = color
		}

		preview.Grid[entrant.PitBox+1] = SessionPreviewEntrant{
			Name:       fmt.Sprintf("%s (%s)", entrant.Car.GetName(), sess.Name()),
			Session:    sess.Name(),
			Class:      class.Name,
			ClassColor: color,
		}
	}

	return preview, nil
}

func (rwm *RaceWeekendManager) UpdateGrid(raceWeekendID, parentSessionID, childSessionID string, filter *RaceWeekendSessionToSessionFilter) error {
	raceWeekend, err := rwm.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return err
	}

	raceWeekend.AddFilter(parentSessionID, childSessionID, filter)

	return rwm.UpsertRaceWeekend(raceWeekend)
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

		entrantPositionText := "No Time"

		if entrantSession.Completed() {
			for i, result := range entrantSession.Results.Result {
				if result.DriverGUID == entrant.Car.Driver.GUID {
					entrantPositionText = fmt.Sprintf("%d%s", i+1, ordinal(int64(i+1)))
					break
				}
			}
		}

		class := entrant.ChampionshipClass(raceWeekend)

		color, ok := preview.Classes[class.Name]

		if !ok {
			color = ChampionshipClassColor(len(preview.Classes))
			preview.Classes[class.Name] = color
		}

		preview.Grid[i+1] = SessionPreviewEntrant{
			Name:       fmt.Sprintf("%s (%s - %s)", entrant.Car.GetName(), entrantSession.Name(), entrantPositionText),
			Session:    session.Name(),
			Class:      class.Name,
			ClassColor: color,
		}
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

	return rwm.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) ImportRaceWeekend(data string) (string, error) {
	var raceWeekend *RaceWeekend

	err := json.Unmarshal([]byte(data), &raceWeekend)

	if err != nil {
		return "", err
	}

	return raceWeekend.ID.String(), rwm.UpsertRaceWeekend(raceWeekend)
}

func (rwm *RaceWeekendManager) WatchForScheduledSessions() error {
	raceWeekends, err := rwm.ListRaceWeekends()

	if err != nil {
		return err
	}

	for _, raceWeekend := range raceWeekends {
		raceWeekend := raceWeekend

		for _, session := range raceWeekend.Sessions {
			session := session

			if session.ScheduledServerID != serverID {
				continue
			}

			if session.ScheduledTime.After(time.Now()) {
				err := rwm.setupScheduledSessionTimer(raceWeekend, session)

				if err != nil {
					return err
				}
			} else if !session.ScheduledTime.IsZero() {
				logrus.Infof("The %s Session in the %s Race Weekend was scheduled to run, but the server was offline. Please start the session manually.", session.Name(), raceWeekend.Name)
			}
		}
	}

	return nil
}

func (rwm *RaceWeekendManager) clearScheduledSessionTimer(session *RaceWeekendSession) {
	if timer := rwm.scheduledSessionTimers[session.ID.String()]; timer != nil {
		timer.Stop()
	}
}

func (rwm *RaceWeekendManager) setupScheduledSessionTimer(raceWeekend *RaceWeekend, session *RaceWeekendSession) error {
	rwm.clearScheduledSessionTimer(session)

	var err error

	rwm.scheduledSessionTimers[session.ID.String()], err = when.When(session.ScheduledTime, func() {
		err := rwm.StartSession(raceWeekend.ID.String(), session.ID.String(), false)

		if err != nil {
			logrus.WithError(err).Errorf("Could not start scheduled race weekend session")
		}

		raceWeekend, session, err := rwm.FindSession(raceWeekend.ID.String(), session.ID.String())

		if err != nil {
			logrus.WithError(err).Error("Could not clear scheduled time on started Race Weekend Session")
			return
		}

		session.ScheduledTime = time.Time{}

		if err := rwm.UpsertRaceWeekend(raceWeekend); err != nil {
			logrus.WithError(err).Error("Could not update race weekend with cleared scheduled time")
		}
	})

	if err != nil {
		return err
	}

	if rwm.notificationManager.HasNotificationReminders() {
		for _, timer := range rwm.notificationManager.GetNotificationReminders() {
			reminderTime := session.ScheduledTime.Add(time.Duration(-timer) * time.Minute)

			if reminderTime.After(time.Now()) {
				// add reminder
				thisTimer := timer

				rwm.scheduledSessionReminderTimers[session.ID.String()], err = when.When(reminderTime, func() {
					err := rwm.notificationManager.SendRaceWeekendReminderMessage(raceWeekend, session, thisTimer)

					if err != nil {
						logrus.WithError(err).Errorf("Could not send race weekend reminder message")
					}
				})

				if err != nil {
					logrus.WithError(err).Error("Could not set up race weekend reminder timer")
				}
			}
		}
	}

	return nil
}

func (rwm *RaceWeekendManager) ScheduleSession(raceWeekendID, sessionID string, date time.Time, startWhenParentFinishes bool) error {
	raceWeekend, session, err := rwm.FindSession(raceWeekendID, sessionID)

	if err != nil {
		return err
	}

	session.ScheduledTime = date
	session.StartWhenParentHasFinished = startWhenParentFinishes
	session.ScheduledServerID = serverID

	if config.Lua.Enabled && Premium() {
		err = raceWeekendEventSchedulePlugin(raceWeekend, session)

		if err != nil {
			logrus.WithError(err).Error("race weekend session schedule plugin script failed")
		}
	}

	if !session.ScheduledTime.IsZero() {
		err = rwm.setupScheduledSessionTimer(raceWeekend, session)

		if err != nil {
			return err
		}
	}

	return rwm.UpsertRaceWeekend(raceWeekend)
}

func raceWeekendEventSchedulePlugin(raceWeekend *RaceWeekend, raceWeekendSession *RaceWeekendSession) error {
	p := &LuaPlugin{}

	newRaceWeekendSession, newRaceWeekend := NewRaceWeekendSession(), NewRaceWeekend()

	p.Inputs(raceWeekendSession, raceWeekend).Outputs(newRaceWeekendSession, newRaceWeekend)
	err := p.Call("./plugins/events.lua", "onRaceWeekendEventSchedule")

	if err != nil {
		return err
	}

	*raceWeekendSession, *raceWeekend = *newRaceWeekendSession, *newRaceWeekend

	return nil
}

func (rwm *RaceWeekendManager) DeScheduleSession(raceWeekendID, sessionID string) error {
	raceWeekend, session, err := rwm.FindSession(raceWeekendID, sessionID)

	if err != nil {
		return err
	}

	session.ScheduledTime = time.Time{}
	session.StartWhenParentHasFinished = false

	rwm.clearScheduledSessionTimer(session)

	return rwm.UpsertRaceWeekend(raceWeekend)
}
