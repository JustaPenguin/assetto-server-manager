package servermanager

import (
	"errors"
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/etcd-io/bbolt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
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
	championships, err := cm.raceStore.ListChampionships()

	if err == bbolt.ErrBucketNotFound {
		return nil, nil
	}

	return championships, err
}

func (cm *ChampionshipManager) BuildChampionshipOpts(r *http.Request) (map[string]interface{}, error) {
	raceOpts, err := cm.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	raceOpts["DefaultPoints"] = DefaultChampionshipPoints

	return raceOpts, nil
}

func (cm *ChampionshipManager) HandleCreateChampionship(r *http.Request) (*Championship, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	championship := NewChampionship(r.FormValue("ChampionshipName"))

	var err error

	championship.Entrants, err = cm.BuildEntryList(r)

	if err != nil {
		return nil, err
	}

	for i := 0; i < len(r.Form["Points.Place"]); i++ {
		championship.Points.Places = append(championship.Points.Places, formValueAsInt(r.Form["Points.Place"][i]))
	}

	championship.Points.PolePosition = formValueAsInt(r.FormValue("Points.PolePosition"))
	championship.Points.BestLap = formValueAsInt(r.FormValue("Points.BestLap"))

	return championship, cm.UpsertChampionship(championship)
}

var ErrInvalidChampionshipEvent = errors.New("servermanager: invalid championship event")

func (cm *ChampionshipManager) BuildChampionshipEventOpts(r *http.Request) (map[string]interface{}, error) {
	opts, err := cm.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	vars := mux.Vars(r)

	// here we customise the opts to tell the template that this is a championship race.
	championship, err := cm.LoadChampionship(vars["championshipID"])

	if err != nil {
		return nil, err
	}

	opts["IsChampionship"] = true
	opts["Championship"] = championship

	if editEventID, ok := vars["eventID"]; ok {
		// editing a championship event
		toEdit := formValueAsInt(editEventID)

		if toEdit > len(championship.Events) || toEdit < 0 {
			return nil, ErrInvalidChampionshipEvent
		}

		opts["Current"] = championship.Events[toEdit].RaceSetup
		opts["IsEditing"] = true
		opts["EditingID"] = toEdit
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

	championship, err = cm.LoadChampionship(mux.Vars(r)["championshipID"])

	if err != nil {
		return nil, nil, false, err
	}

	raceConfig, err := cm.BuildCustomRaceFromForm(r)

	if err != nil {
		return nil, nil, false, err
	}

	raceConfig.Cars = strings.Join(championship.ValidCarIDs(), ";")

	if eventIDStr := r.FormValue("Editing"); eventIDStr != "" {
		eventID := formValueAsInt(eventIDStr)

		if eventID > len(championship.Events) || eventID < 0 {
			return nil, nil, true, ErrInvalidChampionshipEvent
		}

		// we're editing an existing event
		championship.Events[eventID].RaceSetup = *raceConfig
		event = championship.Events[eventID]
		edited = true
	} else {
		// creating a new event
		event = NewChampionshipEvent()
		event.RaceSetup = *raceConfig

		championship.Events = append(championship.Events, event)
	}

	return championship, event, edited, cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) DeleteEvent(championshipID string, eventID int) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	if eventID > len(championship.Events) || eventID < 0 {
		return ErrInvalidChampionshipEvent
	}

	championship.Events = append(championship.Events[:eventID], championship.Events[eventID+1:]...)

	return cm.UpsertChampionship(championship)
}

// Start a 2hr long Practice Event based off the existing championship event with eventID
func (cm *ChampionshipManager) StartPracticeEvent(championshipID string, eventID int) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	if eventID > len(championship.Events) || eventID < 0 {
		return ErrInvalidChampionshipEvent
	}

	config := ConfigIniDefault

	raceSetup := championship.Events[eventID].RaceSetup

	raceSetup.Sessions = make(map[SessionType]*SessionConfig)
	raceSetup.Sessions[SessionTypePractice] = &SessionConfig{
		Name:   "Practice",
		Time:   120,
		IsOpen: 1,
	}

	raceSetup.LoopMode = 1
	raceSetup.MaxClients = len(championship.Entrants)

	config.CurrentRaceConfig = raceSetup

	return cm.applyConfigAndStart(config, championship.Entrants)
}

func (cm *ChampionshipManager) StartEvent(championshipID string, eventID int) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	if eventID > len(championship.Events) || eventID < 0 {
		return ErrInvalidChampionshipEvent
	}

	event := championship.Events[eventID]

	// championship events always have locked entry lists
	event.RaceSetup.LockedEntryList = 1
	event.RaceSetup.MaxClients = len(championship.Entrants)

	config := ConfigIniDefault
	config.CurrentRaceConfig = event.RaceSetup

	// track that this is the current event
	cm.activeChampionship = &ActiveChampionship{
		ChampionshipID: championship.ID,
		EventID:        event.ID,
	}

	return cm.applyConfigAndStart(config, championship.Entrants)
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

	var currentEvent *ChampionshipEvent

	for _, event := range championship.Events {
		if event.ID == cm.activeChampionship.EventID {
			currentEvent = event
			break
		}
	}

	if currentEvent == nil {
		logrus.Errorf("Couldn't load championship event with given id")
		return
	}

	cm.handleSessionChanges(message, championship, currentEvent)
}

func (cm *ChampionshipManager) handleSessionChanges(message udp.Message, championship *Championship, currentEvent *ChampionshipEvent) {
	switch a := message.(type) {
	case udp.SessionInfo:
		if a.Event() == udp.EventNewSession {
			if currentEvent.StartedTime.IsZero() {
				currentEvent.StartedTime = time.Now()

				err := cm.UpsertChampionship(championship)

				if err != nil {
					logrus.Errorf("Could not save session results to championship %s, err: %s", cm.activeChampionship.ChampionshipID.String(), err)
					return
				}
			}

			// new session created
			logrus.Infof("New Session: %s at %s (%s) - %d laps | %d minutes", a.Name, a.Track, a.TrackConfig, a.Laps, a.Time)
			sessionType, err := cm.findSessionWithName(currentEvent, a.Name)

			if err != nil {
				logrus.Errorf("Unexpected session: %s. Cannot track championship progress for this session", a.Name)
				return
			}

			if cm.activeChampionship.SessionType != "" && cm.activeChampionship.SessionType != sessionType && !currentEvent.StartedTime.IsZero() && currentEvent.CompletedTime.IsZero() {
				resultsFile, err := cm.findLastWrittenSessionFile()

				if err == nil {
					logrus.Infof("Assetto Server didn't give us a results file, but we found it at %s", resultsFile)
					cm.handleSessionChanges(udp.EndSession(resultsFile), championship, currentEvent)
				}
			}

			cm.activeChampionship.SessionType = sessionType
		}
	case udp.EndSession:
		filename := string(a)
		logrus.Infof("End Session, file outputted at: %s", filename)

		results, err := LoadResult(filepath.Base(filename))

		if err != nil {
			logrus.Errorf("Could not read session results for %s, err: %s", cm.activeChampionship.SessionType.String(), err)
			return
		}

		if currentEvent.Results == nil {
			currentEvent.Results = make(map[SessionType]*SessionResults)
		}

		currentEvent.Results[cm.activeChampionship.SessionType] = results

		if cm.activeChampionship.SessionType == SessionTypeRace {
			logrus.Infof("End of Race Session detected. Marking championship event %s complete", cm.activeChampionship.EventID.String())
			currentEvent.CompletedTime = time.Now()

			// clear out all current session stuff
			cm.activeChampionship = nil

			// stop the server
			err := AssettoProcess.Stop()

			if err != nil {
				logrus.Errorf("Could not stop Assetto Process, err: %s", err)
			}
		}

		err = cm.UpsertChampionship(championship)

		if err != nil {
			logrus.Errorf("Could not save session results to championship %s, err: %s", cm.activeChampionship.ChampionshipID.String(), err)
			return
		}
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
		d1, _ := getResultDate(resultFiles[i].Name())
		d2, _ := getResultDate(resultFiles[j].Name())

		return d1.After(d2)
	})

	if len(resultFiles) == 0 {
		return "", errors.New("servermanager: results files not found")
	}

	return resultFiles[0].Name(), nil
}
