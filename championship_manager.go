package servermanager

import (
	"errors"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ChampionshipManager struct {
	*RaceManager

	activeChampionship *ActiveChampionship
	mutex              sync.Mutex
}

type ActiveChampionship struct {
	ChampionshipID, EventID uuid.UUID
	SessionType             SessionType

	NumLapsCompleted int
}

func NewChampionshipManager(rm *RaceManager) *ChampionshipManager {
	return &ChampionshipManager{
		RaceManager: rm,
	}
}

func (cm *ChampionshipManager) LoadChampionship(id string) (*Championship, error) {
	return cm.raceStore.LoadChampionship(id)
}

func (cm *ChampionshipManager) UpsertChampionship(c *Championship) error {
	return cm.raceStore.UpsertChampionship(c)
}

func (cm *ChampionshipManager) DeleteChampionship(id string) error {
	return cm.raceStore.DeleteChampionship(id)
}

func (cm *ChampionshipManager) ListChampionships() ([]*Championship, error) {
	return cm.raceStore.ListChampionships()
}

func (cm *ChampionshipManager) BuildChampionshipOpts(r *http.Request) (map[string]interface{}, error) {
	raceOpts, err := cm.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	raceOpts["DefaultPoints"] = DefaultChampionshipPoints

	championshipID := chi.URLParam(r, "championshipID")

	isEditingChampionship := championshipID != ""
	raceOpts["IsEditing"] = isEditingChampionship

	if isEditingChampionship {
		current, err := cm.LoadChampionship(championshipID)

		if err != nil {
			return nil, err
		}

		raceOpts["Current"] = current
	} else {
		raceOpts["Current"] = NewChampionship("")
	}

	return raceOpts, nil
}

func (cm *ChampionshipManager) HandleCreateChampionship(r *http.Request) (championship *Championship, edited bool, err error) {
	if err := r.ParseForm(); err != nil {
		return nil, false, err
	}

	if championshipID := r.FormValue("Editing"); championshipID != "" {
		// championship is being edited. find the current version
		edited = true

		var err error

		championship, err = cm.LoadChampionship(championshipID)

		if err != nil {
			return nil, edited, err
		}
	} else {
		// new championship
		championship = NewChampionship("")
	}

	championship.Name = r.FormValue("ChampionshipName")

	previousNumEntrants := 0

	for i := 0; i < len(r.Form["ClassName"]); i++ {
		class := NewChampionshipClass(r.Form["ClassName"][i])
		class.Color = ChampionshipClassColors[i%len(ChampionshipClassColors)]

		numEntrantsForClass := formValueAsInt(r.Form["EntryList.NumEntrants"][i])

		class.Entrants, err = cm.BuildEntryList(r, previousNumEntrants, numEntrantsForClass)

		if err != nil {
			return nil, edited, err
		}

		class.Points.Places = make([]int, 0)

		for i := previousNumEntrants; i < previousNumEntrants+numEntrantsForClass; i++ {
			class.Points.Places = append(class.Points.Places, formValueAsInt(r.Form["Points.Place"][i]))
		}

		class.Points.PolePosition = formValueAsInt(r.Form["Points.PolePosition"][i])
		class.Points.BestLap = formValueAsInt(r.Form["Points.BestLap"][i])

		previousNumEntrants += numEntrantsForClass
		championship.AddClass(class)
	}

	return championship, edited, cm.UpsertChampionship(championship)
}

var ErrInvalidChampionshipEvent = errors.New("servermanager: invalid championship event")

func (cm *ChampionshipManager) BuildChampionshipEventOpts(r *http.Request) (map[string]interface{}, error) {
	opts, err := cm.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	// here we customise the opts to tell the template that this is a championship race.
	championship, err := cm.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		return nil, err
	}

	opts["IsChampionship"] = true
	opts["Championship"] = championship

	if editEventID := chi.URLParam(r, "eventID"); editEventID != "" {
		// editing a championship event
		event, err := championship.EventByID(editEventID)

		if err != nil {
			return nil, err
		}

		opts["Current"] = event.RaceSetup
		opts["IsEditing"] = true
		opts["EditingID"] = editEventID
	} else {
		// creating a new championship event
		opts["IsEditing"] = false

		// override Current race config if there is a previous championship race configured
		if len(championship.Events) > 0 {
			opts["Current"] = championship.Events[len(championship.Events)-1].RaceSetup
			opts["ChampionshipHasAtLeastOnceRace"] = true
		} else {
			opts["Current"] = ConfigIniDefault.CurrentRaceConfig
			opts["ChampionshipHasAtLeastOnceRace"] = false
		}
	}

	return opts, nil
}

func (cm *ChampionshipManager) SaveChampionshipEvent(r *http.Request) (championship *Championship, event *ChampionshipEvent, edited bool, err error) {
	if err := r.ParseForm(); err != nil {
		return nil, nil, false, err
	}

	championship, err = cm.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		return nil, nil, false, err
	}

	raceConfig, err := cm.BuildCustomRaceFromForm(r)

	if err != nil {
		return nil, nil, false, err
	}

	raceConfig.Cars = strings.Join(championship.ValidCarIDs(), ";")

	if eventID := r.FormValue("Editing"); eventID != "" {
		edited = true

		event, err = championship.EventByID(eventID)

		if err != nil {
			return nil, nil, true, err
		}

		// we're editing an existing event
		event.RaceSetup = *raceConfig
	} else {
		// creating a new event
		event = NewChampionshipEvent()
		event.RaceSetup = *raceConfig

		championship.Events = append(championship.Events, event)
	}

	return championship, event, edited, cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) DeleteEvent(championshipID string, eventID string) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	toDelete := -1

	for i, event := range championship.Events {
		if event.ID.String() == eventID {
			toDelete = i
			break
		}
	}

	if toDelete < 0 {
		return ErrInvalidChampionshipEvent
	}

	championship.Events = append(championship.Events[:toDelete], championship.Events[toDelete+1:]...)

	return cm.UpsertChampionship(championship)
}

// Start a 2hr long Practice Event based off the existing championship event with eventID
func (cm *ChampionshipManager) StartPracticeEvent(championshipID string, eventID string) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	event, err := championship.EventByID(eventID)

	if err != nil {
		return err
	}

	config := ConfigIniDefault

	raceSetup := event.RaceSetup

	raceSetup.Sessions = make(map[SessionType]SessionConfig)
	raceSetup.Sessions[SessionTypePractice] = SessionConfig{
		Name:   "Practice",
		Time:   120,
		IsOpen: 1,
	}

	raceSetup.LoopMode = 1
	raceSetup.MaxClients = championship.NumEntrants()

	config.CurrentRaceConfig = raceSetup

	return cm.applyConfigAndStart(config, championship.AllEntrants(), false)
}

func (cm *ChampionshipManager) StartEvent(championshipID string, eventID string) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	event, err := championship.EventByID(eventID)

	if err != nil {
		return err
	}

	// championship events always have locked entry lists
	event.RaceSetup.LockedEntryList = 1
	event.RaceSetup.MaxClients = championship.NumEntrants()

	config := ConfigIniDefault
	config.CurrentRaceConfig = event.RaceSetup

	// track that this is the current event
	cm.activeChampionship = &ActiveChampionship{
		ChampionshipID: championship.ID,
		EventID:        event.ID,
	}

	return cm.applyConfigAndStart(config, championship.AllEntrants(), false)
}

func (cm *ChampionshipManager) ChampionshipEventCallback(message udp.Message) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.activeChampionship == nil {
		logrus.Debugf("No active championship set up, not performing championship callbacks")
		return
	}

	championship, err := cm.LoadChampionship(cm.activeChampionship.ChampionshipID.String())

	if err != nil {
		logrus.Errorf("Couldn't load championship with ID: %s, err: %s", cm.activeChampionship.ChampionshipID.String(), err)
		return
	}

	currentEventIndex := -1

	for index, event := range championship.Events {
		if event.ID == cm.activeChampionship.EventID {
			currentEventIndex = index
			break
		}
	}

	if currentEventIndex < 0 {
		logrus.Errorf("Couldn't load championship event with given id")
		return
	}

	cm.handleSessionChanges(message, championship, currentEventIndex)
}

func (cm *ChampionshipManager) handleSessionChanges(message udp.Message, championship *Championship, currentEventIndex int) {
	if championship.Events[currentEventIndex].Completed() {
		logrus.Infof("Event is complete. Ignoring messages")
		return
	}

	saveChampionship := true

	defer func() {
		if !saveChampionship {
			return
		}

		err := cm.UpsertChampionship(championship)

		if err != nil {
			logrus.Errorf("Could not save session results to championship %s, err: %s", cm.activeChampionship.ChampionshipID.String(), err)
			return
		}
	}()

	switch a := message.(type) {
	case udp.SessionInfo:
		if a.Event() == udp.EventNewSession {
			if championship.Events[currentEventIndex].StartedTime.IsZero() {
				championship.Events[currentEventIndex].StartedTime = time.Now()
			}

			if championship.Events[currentEventIndex].Sessions == nil {
				championship.Events[currentEventIndex].Sessions = make(map[SessionType]*ChampionshipSession)
			}

			// new session created
			logrus.Infof("New Session: %s at %s (%s) - %d laps | %d minutes", a.Name, a.Track, a.TrackConfig, a.Laps, a.Time)
			sessionType, err := cm.findSessionWithName(championship.Events[currentEventIndex], a.Name)

			if err != nil {
				logrus.Errorf("Unexpected session: %s. Cannot track championship progress for this session", a.Name)
				return
			}

			currentSession, ok := championship.Events[currentEventIndex].Sessions[sessionType]

			if !ok {
				currentSession = &ChampionshipSession{
					StartedTime: time.Now(),
				}

				championship.Events[currentEventIndex].Sessions[sessionType] = currentSession
			}

			previousSessionType := cm.activeChampionship.SessionType
			previousSession, hasPreviousSession := championship.Events[currentEventIndex].Sessions[previousSessionType]
			previousSessionNumLaps := cm.activeChampionship.NumLapsCompleted

			defer func() {
				if cm.activeChampionship == nil {
					return
				}

				cm.activeChampionship.NumLapsCompleted = 0
				cm.activeChampionship.SessionType = sessionType
			}()

			if hasPreviousSession && previousSessionType != sessionType && previousSessionNumLaps > 0 && !previousSession.StartedTime.IsZero() && previousSession.CompletedTime.IsZero() {
				resultsFile, err := cm.findLastWrittenSessionFile()

				if err == nil {
					logrus.Infof("Assetto Server didn't give us a results file for the session: %s, but we found it at %s", cm.activeChampionship.SessionType.String(), resultsFile)
					cm.handleSessionChanges(udp.EndSession(resultsFile), championship, currentEventIndex)
					return
				}
			}
		}
	case udp.LapCompleted:
		cm.activeChampionship.NumLapsCompleted++

	case udp.EndSession:
		filename := filepath.Base(string(a))
		logrus.Infof("End Session, file outputted at: %s", filename)

		results, err := LoadResult(filename)

		if err != nil {
			logrus.Errorf("Could not read session results for %s, err: %s", cm.activeChampionship.SessionType.String(), err)
			return
		}

		// Update the old results json file with championship ID, required for applying penalties properly
		results.ChampionshipID = cm.activeChampionship.ChampionshipID.String()
		err = saveResults(filename, results)

		if err != nil {
			logrus.Errorf("Could not update session results for %s, err: %s", cm.activeChampionship.SessionType.String(), err)
			return
		}

		currentSession, ok := championship.Events[currentEventIndex].Sessions[cm.activeChampionship.SessionType]

		if ok {
			currentSession.CompletedTime = time.Now()
			currentSession.Results = results
		} else {
			logrus.Errorf("Received and EndSession with no matching NewSession")
			return
		}

		if cm.activeChampionship.SessionType == SessionTypeRace {
			logrus.Infof("End of Race Session detected. Marking championship event %s complete", cm.activeChampionship.EventID.String())
			championship.Events[currentEventIndex].CompletedTime = time.Now()

			// clear out all current session stuff
			cm.activeChampionship = nil

			// stop the server
			err := AssettoProcess.Stop()

			if err != nil {
				logrus.Errorf("Could not stop Assetto Process, err: %s", err)
			}
		}
	default:
		saveChampionship = false
		return
	}

}

var ErrSessionNotFound = errors.New("servermanager: session not found")

func (cm *ChampionshipManager) findSessionWithName(event *ChampionshipEvent, name string) (SessionType, error) {
	for t, sess := range event.RaceSetup.Sessions {
		if sess.Name == name {
			return t, nil
		}
	}

	return "", ErrSessionNotFound
}

func (cm *ChampionshipManager) findLastWrittenSessionFile() (string, error) {
	resultsPath := filepath.Join(ServerInstallPath, "results")
	resultFiles, err := ioutil.ReadDir(resultsPath)

	if err != nil {
		return "", err
	}

	sort.Slice(resultFiles, func(i, j int) bool {
		return resultFiles[i].ModTime().After(resultFiles[j].ModTime())
	})

	if len(resultFiles) == 0 {
		return "", errors.New("servermanager: results files not found")
	}

	return resultFiles[0].Name(), nil
}

func (cm *ChampionshipManager) CancelEvent(championshipID string, eventID string) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	event, err := championship.EventByID(eventID)

	if err != nil {
		return err
	}

	event.StartedTime = time.Time{}
	event.CompletedTime = time.Time{}

	event.Sessions = make(map[SessionType]*ChampionshipSession)

	if err := AssettoProcess.Stop(); err != nil {
		return err
	}

	return cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) RestartEvent(championshipID string, eventID string) error {
	err := cm.CancelEvent(championshipID, eventID)

	if err != nil {
		return err
	}

	return cm.StartEvent(championshipID, eventID)
}
