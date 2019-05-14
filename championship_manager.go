package servermanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/haisum/recaptcha"
	"github.com/heindl/caldav-go/icalendar"
	"github.com/heindl/caldav-go/icalendar/components"
	"github.com/mitchellh/go-wordwrap"
	"github.com/sirupsen/logrus"
)

type ChampionshipManager struct {
	*RaceManager

	activeChampionship *ActiveChampionship
	mutex              sync.Mutex
}

type ActiveChampionship struct {
	Name                    string
	ChampionshipID, EventID uuid.UUID
	SessionType             SessionType
	OverridePassword        bool
	ReplacementPassword     string

	loadedEntrants map[udp.CarID]udp.SessionCarInfo

	NumLapsCompleted   int
	NumRaceStartEvents int
}

func (a *ActiveChampionship) IsChampionship() bool {
	return true
}

func (a *ActiveChampionship) OverrideServerPassword() bool {
	return a.OverridePassword
}

func (a *ActiveChampionship) ReplacementServerPassword() string {
	return a.ReplacementPassword
}

func (a *ActiveChampionship) EventName() string {
	return a.Name
}

func NewChampionshipManager(rm *RaceManager) *ChampionshipManager {
	return &ChampionshipManager{
		RaceManager: rm,
	}
}

func (cm *ChampionshipManager) applyConfigAndStart(config ServerConfig, entryList EntryList, championship *ActiveChampionship) error {
	err := cm.RaceManager.applyConfigAndStart(config, entryList, false, championship)

	if err != nil {
		return err
	}

	cm.activeChampionship = championship

	return nil
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
	champs, err := cm.raceStore.ListChampionships()

	if err != nil {
		return nil, err
	}

	sort.Slice(champs, func(i, j int) bool {
		return champs[i].Updated.After(champs[j].Updated)
	})

	return champs, nil
}

func (cm *ChampionshipManager) BuildChampionshipOpts(r *http.Request) (championship *Championship, opts map[string]interface{}, err error) {
	raceOpts, err := cm.BuildRaceOpts(r)

	if err != nil {
		return nil, nil, err
	}

	raceOpts["DefaultPoints"] = DefaultChampionshipPoints

	championshipID := chi.URLParam(r, "championshipID")

	isEditingChampionship := championshipID != ""
	raceOpts["IsEditing"] = isEditingChampionship

	if isEditingChampionship {
		championship, err = cm.LoadChampionship(championshipID)

		if err != nil {
			return nil, nil, err
		}
	} else {
		championship = NewChampionship("")
	}

	raceOpts["Current"] = championship

	return championship, raceOpts, nil
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

		championship.Classes = []*ChampionshipClass{}
	} else {
		// new championship
		championship = NewChampionship("")
	}

	championship.Name = r.FormValue("ChampionshipName")
	championship.OpenEntrants = r.FormValue("ChampionshipOpenEntrants") == "on" || r.FormValue("ChampionshipOpenEntrants") == "1"
	championship.SignUpForm.Enabled = r.FormValue("Championship.SignUpForm.Enabled") == "on" || r.FormValue("Championship.SignUpForm.Enabled") == "1"
	championship.SignUpForm.AskForEmail = r.FormValue("Championship.SignUpForm.AskForEmail") == "on" || r.FormValue("Championship.SignUpForm.AskForEmail") == "1"
	championship.SignUpForm.AskForTeam = r.FormValue("Championship.SignUpForm.AskForTeam") == "on" || r.FormValue("Championship.SignUpForm.AskForTeam") == "1"
	championship.SignUpForm.HideCarChoice = !(r.FormValue("Championship.SignUpForm.HideCarChoice") == "on" || r.FormValue("Championship.SignUpForm.HideCarChoice") == "1")
	championship.SignUpForm.RequiresApproval = r.FormValue("Championship.SignUpForm.RequiresApproval") == "on" || r.FormValue("Championship.SignUpForm.RequiresApproval") == "1"

	championship.SignUpForm.ExtraFields = []string{}

	for _, question := range r.Form["Championship.SignUpForm.ExtraFields"] {
		if question == "" {
			continue
		}

		championship.SignUpForm.ExtraFields = append(championship.SignUpForm.ExtraFields, question)
	}

	championship.Info = template.HTML(r.FormValue("ChampionshipInfo"))
	championship.OverridePassword = r.FormValue("OverridePassword") == "1"
	championship.ReplacementPassword = r.FormValue("ReplacementPassword")

	previousNumEntrants := 0
	previousNumPoints := 0

	for i := 0; i < len(r.Form["ClassName"]); i++ {
		class := NewChampionshipClass(r.Form["ClassName"][i])

		if classID := r.Form["ClassID"][i]; classID != "" && classID != uuid.Nil.String() {
			class.ID = uuid.MustParse(classID)
		}

		numEntrantsForClass := formValueAsInt(r.Form["EntryList.NumEntrants"][i])
		numPointsForClass := formValueAsInt(r.Form["NumPoints"][i])

		class.Entrants, err = cm.BuildEntryList(r, previousNumEntrants, numEntrantsForClass)

		if err != nil {
			return nil, edited, err
		}

		class.Points.Places = make([]int, 0)

		for i := previousNumPoints; i < previousNumPoints+numPointsForClass; i++ {
			class.Points.Places = append(class.Points.Places, formValueAsInt(r.Form["Points.Place"][i]))
		}

		class.Points.PolePosition = formValueAsInt(r.Form["Points.PolePosition"][i])
		class.Points.BestLap = formValueAsInt(r.Form["Points.BestLap"][i])
		class.Points.SecondRaceMultiplier = formValueAsFloat(r.Form["Points.SecondRaceMultiplier"][i])

		previousNumEntrants += numEntrantsForClass
		previousNumPoints += numPointsForClass
		championship.AddClass(class)
	}

	// persist any entrants so that they can be autofilled
	if err := cm.SaveEntrantsForAutoFill(championship.AllEntrants()); err != nil {
		return nil, edited, err
	}

	// look to see if any entrants have their team points set to transfer, move them across to the team they are in now
	for _, class := range championship.Classes {
		for _, entrant := range class.Entrants {
			if !entrant.TransferTeamPoints {
				continue
			}

			logrus.Infof("Renaming team for entrant: %s (%s)", entrant.Name, entrant.GUID)

			for _, event := range championship.Events {
				for _, session := range event.Sessions {
					if session.Results == nil {
						continue
					}

					class.AttachEntrantToResult(entrant, session.Results)
				}
			}
		}
	}

	// look at each entrant to see if their properties should overwrite all event properties set up in the
	// event entrylist. this is useful for globally changing skins, restrictor values etc.
	for _, class := range championship.Classes {
		for _, entrant := range class.Entrants {
			if !entrant.OverwriteAllEvents {
				continue
			}

			for _, event := range championship.Events {
				eventEntrant := event.EntryList.FindEntrantByInternalUUID(entrant.InternalUUID)

				logrus.Infof("Overwriting properties for entrant: %s (%s)", entrant.Name, entrant.GUID)

				eventEntrant.OverwriteProperties(entrant)
			}
		}
	}

	return championship, edited, cm.UpsertChampionship(championship)
}

var (
	ErrInvalidChampionshipEvent = errors.New("servermanager: invalid championship event")
	ErrInvalidChampionshipClass = errors.New("servermanager: invalid championship class")
)

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
		opts["CurrentEntrants"] = event.CombineEntryLists(championship)
	} else {
		// creating a new championship event
		opts["IsEditing"] = false
		opts["CurrentEntrants"] = championship.AllEntrants()

		// override Current race config if there is a previous championship race configured
		if len(championship.Events) > 0 {
			opts["Current"] = championship.Events[len(championship.Events)-1].RaceSetup
			opts["ChampionshipHasAtLeastOnceRace"] = true
		} else {
			opts["Current"] = ConfigIniDefault.CurrentRaceConfig
			opts["ChampionshipHasAtLeastOnceRace"] = false
		}
	}

	if !championship.OpenEntrants {
		opts["AvailableSessions"] = AvailableSessionsClosedChampionship
	}

	err = cm.applyCurrentRaceSetupToOptions(opts, opts["Current"].(CurrentRaceConfig))

	if err != nil {
		return nil, err
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

	entryList, err := cm.BuildEntryList(r, 0, len(r.Form["EntryList.Name"]))

	if err != nil {
		return nil, nil, false, err
	}

	if eventID := r.FormValue("Editing"); eventID != "" {
		edited = true

		event, err = championship.EventByID(eventID)

		if err != nil {
			return nil, nil, true, err
		}

		// we're editing an existing event
		event.RaceSetup = *raceConfig
		event.EntryList = entryList
	} else {
		// creating a new event
		event = NewChampionshipEvent()
		event.RaceSetup = *raceConfig
		event.EntryList = entryList

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

	raceSetup.Cars = strings.Join(championship.ValidCarIDs(), ";")
	raceSetup.LoopMode = 1
	raceSetup.MaxClients = championship.NumEntrants()

	config.CurrentRaceConfig = raceSetup

	return cm.RaceManager.applyConfigAndStart(config, event.CombineEntryLists(championship), false, normalEvent{
		OverridePassword:    championship.OverridePassword,
		ReplacementPassword: championship.ReplacementPassword,
	})
}

func (cm *ChampionshipManager) StartEvent(championshipID string, eventID string) error {
	championship, event, err := cm.GetChampionshipAndEvent(championshipID, eventID)

	if err != nil {
		return err
	}

	if championship.OpenEntrants {
		event.RaceSetup.PickupModeEnabled = 1
	} else {
		event.RaceSetup.PickupModeEnabled = 0
	}

	event.RaceSetup.Cars = strings.Join(championship.ValidCarIDs(), ";")
	event.RaceSetup.MaxClients = championship.NumEntrants()

	config := ConfigIniDefault
	config.CurrentRaceConfig = event.RaceSetup

	logrus.Infof("Starting Championship Event: %s at %s (%s) with %d entrants", event.RaceSetup.Cars, event.RaceSetup.Track, event.RaceSetup.TrackLayout, event.RaceSetup.MaxClients)

	entryList := event.CombineEntryLists(championship)

	if championship.SignUpForm.Enabled && !championship.OpenEntrants {
		filteredEntryList := make(EntryList)

		// a sign up championship (which is not open) should remove all empty entrants before the event starts
		// here we are building a new filtered entry list so grid positions are not 'missing'
		for _, entrant := range entryList {
			if entrant.GUID != "" {
				filteredEntryList.Add(entrant)
			}
		}

		entryList = filteredEntryList

		// sign up championships also have pickup mode disabled
		config.CurrentRaceConfig.PickupModeEnabled = 0
	} else {
		if championship.OpenEntrants {
			config.CurrentRaceConfig.PickupModeEnabled = 1
		} else {
			config.CurrentRaceConfig.PickupModeEnabled = 0
		}
	}

	// track that this is the current event
	return cm.applyConfigAndStart(config, entryList, &ActiveChampionship{
		ChampionshipID:      championship.ID,
		EventID:             event.ID,
		Name:                championship.Name,
		OverridePassword:    championship.OverridePassword,
		ReplacementPassword: championship.ReplacementPassword,
	})
}

func (cm *ChampionshipManager) GetChampionshipAndEvent(championshipID string, eventID string) (*Championship, *ChampionshipEvent, error) {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return nil, nil, err
	}

	event, err := championship.EventByID(eventID)

	if err != nil {
		return nil, nil, err
	}

	return championship, event, nil
}

func (cm *ChampionshipManager) ScheduleEvent(championshipID string, eventID string, date time.Time, action string) error {
	championship, event, err := cm.GetChampionshipAndEvent(championshipID, eventID)

	if err != nil {
		return err
	}

	event.Scheduled = date

	// if there is an existing schedule timer for this event stop it
	if timer := ChampionshipEventStartTimers[event.ID.String()]; timer != nil {
		timer.Stop()
	}

	if action == "add" {
		// add a scheduled event on date
		duration := time.Until(date)

		ChampionshipEventStartTimers[event.ID.String()] = time.AfterFunc(duration, func() {
			err := cm.StartEvent(championship.ID.String(), event.ID.String())

			if err != nil {
				logrus.Errorf("couldn't start scheduled race, err: %s", err)
			}
		})
	}

	return cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) ChampionshipEventCallback(message udp.Message) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.activeChampionship == nil {
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

type sessionEntrantWrapper udp.SessionCarInfo

func (s sessionEntrantWrapper) GetName() string {
	return s.DriverName
}

func (s sessionEntrantWrapper) GetCar() string {
	return s.CarModel
}

func (s sessionEntrantWrapper) GetSkin() string {
	return s.CarSkin
}

func (s sessionEntrantWrapper) GetGUID() string {
	return s.DriverGUID
}

func (s sessionEntrantWrapper) GetTeam() string {
	return ""
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

	case udp.SessionCarInfo:

		if a.Event() == udp.EventNewConnection {

			if cm.activeChampionship.loadedEntrants == nil {
				cm.activeChampionship.loadedEntrants = make(map[udp.CarID]udp.SessionCarInfo)
			}

			cm.activeChampionship.loadedEntrants[a.CarID] = a

		}

		if championship.OpenEntrants && a.Event() == udp.EventNewConnection {
			// a person joined, check to see if they need adding to the championship
			foundSlot, classForCar, err := championship.AddEntrantFromSessionData(sessionEntrantWrapper(a))

			if err != nil {
				saveChampionship = false
				logrus.WithError(err).WithField("entrant", a).Errorf("could not add entrant to open championship")

				return
			}

			saveChampionship = foundSlot

			if !foundSlot {
				logrus.Errorf("Could not find free entrant slot in class: %s for %s (%s)", classForCar.Name, a.DriverName, a.DriverGUID)
				return
			}
		}

	case udp.ClientLoaded:

		entrant, ok := cm.activeChampionship.loadedEntrants[udp.CarID(a)]

		if !ok {
			return
		}

		championshipText := " Championship"

		if strings.HasSuffix(strings.ToLower(championship.Name), "championship") {
			championshipText = ""
		}

		visitServer := ""

		if config != nil && config.HTTP.BaseURL != "" {
			visitServer = fmt.Sprintf(" You can check out the results of this championship in detail at %s.",
				config.HTTP.BaseURL+"/championship/"+championship.ID.String())
		}

		wrapped := strings.Split(wordwrap.WrapString(

			fmt.Sprintf(
				"Hi, %s! Welcome to the %s%s! %s%s Make this race count!\n",
				entrant.DriverName,
				championship.Name,
				championshipText,
				championship.GetPlayerSummary(entrant.DriverGUID),
				visitServer,
			),
			60,
		), "\n")

		for _, msg := range wrapped {
			welcomeMessage, err := udp.NewSendChat(entrant.CarID, msg)

			if err == nil {
				err := AssettoProcess.SendUDPMessage(welcomeMessage)

				if err != nil {
					logrus.Errorf("Unable to send welcome message to: %s, err: %s", entrant.DriverName, err)
				}
			} else {
				logrus.Errorf("Unable to build welcome message to: %s, err: %s", entrant.DriverName, err)
			}
		}
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

			if sessionType == SessionTypeRace {
				// keep track of the number of race end events so we can determine if we're on race 2
				// if the session has ReversedGridPositions != 0
				cm.activeChampionship.NumRaceStartEvents++
			}

			_, ok := championship.Events[currentEventIndex].Sessions[sessionType]

			if !ok {
				championship.Events[currentEventIndex].Sessions[sessionType] = &ChampionshipSession{
					StartedTime: time.Now(),
				}
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

		// Update the old results json file with more championship information, required for applying penalties properly
		championship.EnhanceResults(results)
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

		lastSession := championship.Events[currentEventIndex].LastSession()

		if cm.activeChampionship.SessionType == lastSession {
			logrus.Infof("End of %s Session detected. Marking championship event %s complete", lastSession.String(), cm.activeChampionship.EventID.String())
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

var (
	ErrSessionNotFound    = errors.New("servermanager: session not found")
	ErrResultFileNotFound = errors.New("servermanager: results files not found")
)

func (cm *ChampionshipManager) findSessionWithName(event *ChampionshipEvent, name string) (SessionType, error) {
	for t, sess := range event.RaceSetup.Sessions {
		if sess.Name == name {
			if t == SessionTypeRace && event.RaceSetup.ReversedGridRacePositions != 0 && cm.activeChampionship.NumRaceStartEvents == 1 {
				// this is a second race session
				return SessionTypeSecondRace, nil
			}

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
		return "", ErrResultFileNotFound
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

var ErrNoActiveEvent = errors.New("servermanager: no active championship event")

func (cm *ChampionshipManager) RestartActiveEvent() error {
	if cm.activeChampionship == nil {
		return ErrNoActiveEvent
	}

	return cm.RestartEvent(cm.activeChampionship.ChampionshipID.String(), cm.activeChampionship.EventID.String())
}

func (cm *ChampionshipManager) StopActiveEvent() error {
	if cm.activeChampionship == nil {
		return ErrNoActiveEvent
	}

	return cm.CancelEvent(cm.activeChampionship.ChampionshipID.String(), cm.activeChampionship.EventID.String())
}

func (cm *ChampionshipManager) ListAvailableResultsFilesForEvent(championshipID string, eventID string) (*ChampionshipEvent, []SessionResults, error) {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return nil, nil, err
	}

	event, err := championship.EventByID(eventID)

	if err != nil {
		return nil, nil, err
	}

	results, err := ListAllResults()

	if err != nil {
		return nil, nil, err
	}

	var filteredResults []SessionResults

	for _, result := range results {
		if result.TrackName == event.RaceSetup.Track && result.TrackConfig == event.RaceSetup.TrackLayout {
			filteredResults = append(filteredResults, result)
		}
	}

	return event, filteredResults, nil
}

func (cm *ChampionshipManager) ImportChampionship(jsonData string) (string, error) {
	var championship *Championship

	err := json.Unmarshal([]byte(jsonData), &championship)

	if err != nil {
		return "", err
	}

	// generate a new ID to avoid clashes
	championship.ID = uuid.New()

	return championship.ID.String(), cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) ImportEvent(championshipID string, eventID string, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	event, err := championship.EventByID(eventID)

	if err != nil {
		return err
	}

	event.Sessions = make(map[SessionType]*ChampionshipSession)

	sessions := map[SessionType]string{
		SessionTypePractice:   r.FormValue("PracticeResult"),
		SessionTypeQualifying: r.FormValue("QualifyingResult"),
		SessionTypeRace:       r.FormValue("RaceResult"),
		SessionTypeSecondRace: r.FormValue("SecondRaceResult"),
	}

	for sessionType, sessionFile := range sessions {
		if sessionFile == "" {
			continue
		}

		results, err := LoadResult(sessionFile + ".json")

		if err != nil {
			return err
		}

		if championship.OpenEntrants {
			// if the championship is open, we might have entrants in this session file who have not
			// raced in this championship before. add them to the championship as they would be added
			// if they joined during a race.
			for _, car := range results.Cars {
				if car.GetGUID() == "" {
					continue
				}

				foundFreeSlot, _, err := championship.AddEntrantFromSessionData(car)

				if err != nil {
					return err
				}

				if !foundFreeSlot {
					logrus.WithField("car", car).Warn("Could not add entrant to championship. No free slot found")
				}
			}
		}

		championship.EnhanceResults(results)

		if err := saveResults(sessionFile+".json", results); err != nil {
			return err
		}

		event.Sessions[sessionType] = &ChampionshipSession{
			StartedTime:   results.Date.Add(-time.Minute * 30),
			CompletedTime: results.Date,
			Results:       results,
		}

		event.CompletedTime = results.Date
	}

	return cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) BuildICalFeed(championshipID string, w io.Writer) error {
	championship, err := cm.raceStore.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	cal := components.NewCalendar()

	for _, event := range championship.Events {
		if event.Scheduled.IsZero() {
			continue
		}

		event.championship = championship

		icalEvent := BuildICalEvent(event)

		cal.Events = append(cal.Events, icalEvent)
	}

	str, err := icalendar.Marshal(cal)

	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, str)

	return err
}

type PenaltyAction string

const (
	SetPenalty    PenaltyAction = "set"
	RemovePenalty PenaltyAction = "remove"
)

func (cm *ChampionshipManager) ModifyDriverPenalty(championshipID, classID, driverGUID string, action PenaltyAction, penalty int) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	class, err := championship.ClassByID(classID)

	if err != nil {
		return err
	}

	if class.DriverPenalties == nil {
		class.DriverPenalties = make(map[string]int)
	}

	switch action {
	case SetPenalty:
		class.DriverPenalties[driverGUID] = penalty
	case RemovePenalty:
		delete(class.DriverPenalties, driverGUID)
	default:
		return fmt.Errorf("servermanager: invalid penalty action specified: '%s'", action)
	}

	return cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) ModifyTeamPenalty(championshipID, classID, team string, action PenaltyAction, penalty int) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	class, err := championship.ClassByID(classID)

	if err != nil {
		return err
	}

	if class.TeamPenalties == nil {
		class.TeamPenalties = make(map[string]int)
	}

	switch action {
	case SetPenalty:
		class.TeamPenalties[team] = penalty
	case RemovePenalty:
		delete(class.TeamPenalties, team)
	default:
		return fmt.Errorf("servermanager: invalid penalty action specified: '%s'", action)
	}

	return cm.UpsertChampionship(championship)
}

type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}

var steamGUIDRegex = regexp.MustCompile("^[0-9]{17}$")

func (cm *ChampionshipManager) HandleChampionshipSignUp(r *http.Request) (response *ChampionshipSignUpResponse, foundSlot bool, err error) {
	if err := r.ParseForm(); err != nil {
		return nil, false, err
	}

	championship, err := cm.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		return nil, false, err
	}

	signUpResponse := &ChampionshipSignUpResponse{
		Created: time.Now(),
		Name:    r.FormValue("Name"),
		GUID:    r.FormValue("GUID"),
		Team:    r.FormValue("Team"),
		Email:   r.FormValue("Email"),

		Car:  r.FormValue("Car"),
		Skin: r.FormValue("Skin"),

		Questions: make(map[string]string),
		Status:    ChampionshipEntrantPending,
	}

	for index, question := range championship.SignUpForm.ExtraFields {
		signUpResponse.Questions[question] = r.FormValue(fmt.Sprintf("Question.%d", index))
	}

	if config.Championships.RecaptchaConfig.SecretKey != "" {
		captcha := recaptcha.R{
			Secret: config.Championships.RecaptchaConfig.SecretKey,
		}

		if !captcha.Verify(*r) {
			return signUpResponse, false, ValidationError("Please complete the reCAPTCHA.")
		}
	}

	if !steamGUIDRegex.MatchString(signUpResponse.GUID) {
		return signUpResponse, false, ValidationError("Please enter a valid SteamID64.")
	}

	for _, entrant := range championship.SignUpForm.Responses {
		if entrant.GUID == signUpResponse.GUID {
			return signUpResponse, false, ValidationError("This GUID is already registered.")
		}

		if championship.SignUpForm.AskForEmail && entrant.Email == signUpResponse.Email {
			return signUpResponse, false, ValidationError("Someone has already registered with this email address.")
		}
	}

	if !championship.SignUpForm.RequiresApproval {
		// check to see if there is room in the entrylist for the user in their specific car
		foundSlot, _, err = championship.AddEntrantFromSessionData(signUpResponse)

		if err != nil {
			return signUpResponse, foundSlot, err
		}

		if foundSlot {
			signUpResponse.Status = ChampionshipEntrantAccepted
		} else {
			signUpResponse.Status = ChampionshipEntrantRejected
		}
	}

	championship.SignUpForm.Responses = append(championship.SignUpForm.Responses, signUpResponse)

	return signUpResponse, foundSlot, cm.UpsertChampionship(championship)
}
