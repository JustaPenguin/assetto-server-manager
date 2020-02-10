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

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"
	"github.com/JustaPenguin/assetto-server-manager/pkg/when"

	"github.com/cj123/caldav-go/icalendar"
	"github.com/cj123/caldav-go/icalendar/components"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/haisum/recaptcha"
	"github.com/mitchellh/go-wordwrap"
	"github.com/sirupsen/logrus"
)

type ChampionshipManager struct {
	*RaceManager
	acsrClient *ACSRClient

	activeChampionship *ActiveChampionship
	mutex              sync.Mutex

	championshipEventStartTimers    map[string]*when.Timer
	championshipEventReminderTimers map[string]*when.Timer
}

func NewChampionshipManager(raceManager *RaceManager, acsrClient *ACSRClient) *ChampionshipManager {
	return &ChampionshipManager{
		RaceManager: raceManager,
		acsrClient:  acsrClient,
	}
}

func (cm *ChampionshipManager) applyConfigAndStart(championship *ActiveChampionship) error {
	cm.activeChampionship = championship
	err := cm.RaceManager.applyConfigAndStart(championship)

	if err != nil {
		return err
	}

	return nil
}

func (cm *ChampionshipManager) LoadChampionship(id string) (*Championship, error) {
	championship, err := cm.store.LoadChampionship(id)

	if err != nil {
		return nil, err
	}

	for _, event := range championship.Events {
		if event.IsRaceWeekend() {
			event.RaceWeekend, err = cm.store.LoadRaceWeekend(event.RaceWeekendID.String())

			if err != nil {
				return nil, err
			}
		}
	}

	return championship, nil
}

func (cm *ChampionshipManager) UpsertChampionship(c *Championship) error {
	err := cm.store.UpsertChampionship(c)

	if err != nil {
		return err
	}

	if c.ACSR {
		cm.acsrClient.SendChampionship(*c)
	}

	return nil
}

func (cm *ChampionshipManager) DeleteChampionship(id string) error {
	championship, err := cm.store.LoadChampionship(id)

	if err != nil {
		return err
	}

	if championship.ACSR {
		championship.ACSR = false

		cm.acsrClient.SendChampionship(*championship)
	}

	return cm.store.DeleteChampionship(id)
}

func (cm *ChampionshipManager) ListChampionships() ([]*Championship, error) {
	championships, err := cm.store.ListChampionships()

	if err != nil {
		return nil, err
	}

	sort.Slice(championships, func(i, j int) bool {
		return championships[i].Updated.After(championships[j].Updated)
	})

	for _, championship := range championships {
		for _, event := range championship.Events {
			if event.IsRaceWeekend() {
				event.RaceWeekend, err = cm.store.LoadRaceWeekend(event.RaceWeekendID.String())

				if err == ErrRaceWeekendNotFound {
					continue
				} else if err != nil {
					return nil, err
				}
			}
		}
	}

	return championships, nil
}

func (cm *ChampionshipManager) LoadACSRRatings(championship *Championship) (map[string]*ACSRDriverRating, error) {
	if !championship.ACSR || !Premium() {
		return nil, nil
	}

	guidMap := make(map[string]bool)

	for _, class := range championship.Classes {
		for _, standing := range class.Standings(championship.Events) {
			guidMap[standing.Car.Driver.GUID] = true
		}
	}

	for _, entrant := range championship.AllEntrants() {
		guidMap[entrant.GUID] = true
	}

	var guids []string

	for guid := range guidMap {
		guids = append(guids, guid)
	}

	return cm.acsrClient.GetRating(guids...)
}

type ChampionshipTemplateVars struct {
	*RaceTemplateVars

	DefaultPoints ChampionshipPoints
	DefaultClass  *ChampionshipClass
	ACSREnabled   bool
}

func (cm *ChampionshipManager) BuildChampionshipOpts(r *http.Request) (championship *Championship, opts *ChampionshipTemplateVars, err error) {
	raceOpts, err := cm.BuildRaceOpts(r)

	if err != nil {
		return nil, nil, err
	}

	defaultClass := NewChampionshipClass("")
	defaultClass.ID = uuid.Nil

	opts = &ChampionshipTemplateVars{
		RaceTemplateVars: raceOpts,
		DefaultPoints:    DefaultChampionshipPoints,
		DefaultClass:     defaultClass,
	}

	championshipID := chi.URLParam(r, "championshipID")

	isEditingChampionship := championshipID != ""
	opts.IsEditing = isEditingChampionship

	if isEditingChampionship {
		championship, err = cm.LoadChampionship(championshipID)

		if err != nil {
			return nil, nil, err
		}
	} else {
		championship = NewChampionship("")
	}

	opts.Championship = championship
	opts.ACSREnabled = cm.acsrClient.Enabled

	return championship, opts, nil
}

func (cm *ChampionshipManager) HandleCreateChampionship(r *http.Request) (championship *Championship, edited bool, err error) {
	if err := r.ParseForm(); err != nil {
		return nil, false, err
	}

	previousClasses := make(map[uuid.UUID]ChampionshipClass)

	if championshipID := r.FormValue("Editing"); championshipID != "" {
		// championship is being edited. find the current version
		edited = true

		var err error

		championship, err = cm.LoadChampionship(championshipID)

		if err != nil {
			return nil, edited, err
		}

		for _, class := range championship.Classes {
			previousClasses[class.ID] = *class
		}

		championship.Classes = []*ChampionshipClass{}
	} else {
		// new championship
		championship = NewChampionship("")
	}

	championship.Name = r.FormValue("ChampionshipName")
	championship.OpenEntrants = r.FormValue("ChampionshipOpenEntrants") == "on" || r.FormValue("ChampionshipOpenEntrants") == "1"
	championship.PersistOpenEntrants = r.FormValue("ChampionshipPersistOpenEntrants") == "on" || r.FormValue("ChampionshipPersistOpenEntrants") == "1"
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
	championship.OverridePassword = r.FormValue("OverridePassword") == "on" || r.FormValue("OverridePassword") == "1"

	if Premium() {
		championship.OGImage = r.FormValue("ChampionshipOGImage")
	}

	newACSR := r.FormValue("ACSR") == "on" || r.FormValue("ACSR") == "1"

	if championship.ACSR && !newACSR {
		championship.ACSR = newACSR

		cm.acsrClient.SendChampionship(*championship)
	} else {
		championship.ACSR = newACSR
	}

	if championship.ACSR {
		championship.OverridePassword = true
		championship.ReplacementPassword = ""
		championship.OpenEntrants = false
		championship.SignUpForm.Enabled = true
	} else {
		championship.ReplacementPassword = r.FormValue("ReplacementPassword")
	}

	previousNumEntrants := 0
	previousNumPoints := 0
	previousNumCars := 0

	for i := 0; i < len(r.Form["ClassName"]); i++ {
		class := NewChampionshipClass(r.Form["ClassName"][i])

		if classID := r.Form["ClassID"][i]; classID != "" && classID != uuid.Nil.String() {
			class.ID = uuid.MustParse(classID)
		}

		numEntrantsForClass := formValueAsInt(r.Form["EntryList.NumEntrants"][i])
		numPointsForClass := formValueAsInt(r.Form["NumPoints"][i])
		numCarsForClass := formValueAsInt(r.Form["NumCars"][i])

		class.AvailableCars = r.Form["Cars"][previousNumCars : previousNumCars+numCarsForClass]

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
		class.Points.CollisionWithDriver = formValueAsInt(r.Form["Points.CollisionWithDriver"][i])
		class.Points.CollisionWithEnv = formValueAsInt(r.Form["Points.CollisionWithEnv"][i])
		class.Points.CutTrack = formValueAsInt(r.Form["Points.CutTrack"][i])

		if previousClass, ok := previousClasses[class.ID]; ok {
			// look for previous penalties and apply them back across
			class.DriverPenalties = previousClass.DriverPenalties
			class.TeamPenalties = previousClass.TeamPenalties
		}

		championship.AddClass(class)

		// ensure that linked race weekends (if any) have points set up for this class
		for _, event := range championship.Events {
			if !event.IsRaceWeekend() {
				continue
			}

			updatedRaceWeekend := false

			for _, session := range event.RaceWeekend.Sessions {
				if _, ok := session.Points[class.ID]; ok {
					continue
				}

				if session.SessionType() == SessionTypeRace {
					session.Points[class.ID] = &class.Points
				} else {
					session.Points[class.ID] = &ChampionshipPoints{Places: make([]int, len(class.Points.Places))}
				}

				updatedRaceWeekend = true
			}

			if updatedRaceWeekend {
				if err := cm.store.UpsertRaceWeekend(event.RaceWeekend); err != nil {
					return nil, edited, err
				}
			}
		}

		previousNumEntrants += numEntrantsForClass
		previousNumPoints += numPointsForClass
		previousNumCars += numCarsForClass
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

func (cm *ChampionshipManager) BuildChampionshipEventOpts(r *http.Request) (*RaceTemplateVars, error) {
	opts, err := cm.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	// here we customise the opts to tell the template that this is a championship race.
	championship, err := cm.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		return nil, err
	}

	opts.IsChampionship = true
	opts.Championship = championship

	if editEventID := chi.URLParam(r, "eventID"); editEventID != "" {
		// editing a championship event
		event, err := championship.EventByID(editEventID)

		if err != nil {
			return nil, err
		}

		opts.Current = event.RaceSetup
		opts.IsEditing = true
		opts.EditingID = editEventID
		opts.CurrentEntrants = event.CombineEntryLists(championship)
	} else {
		// creating a new championship event
		opts.IsEditing = false
		opts.CurrentEntrants = championship.AllEntrants()

		// override Current race config if there is a previous championship race configured
		if len(championship.Events) > 0 {
			foundEvent := false

			// championship events should only inherit from non-race weekend events
			for i := len(championship.Events) - 1; i >= 0; i-- {
				event := championship.Events[i]

				if !event.IsRaceWeekend() {
					opts.Current = event.RaceSetup
					foundEvent = true
					break
				}
			}

			if !foundEvent {
				defaultConfig := ConfigIniDefault()
				opts.Current = defaultConfig.CurrentRaceConfig
			}

			opts.ChampionshipHasAtLeastOnceRace = true
		} else {
			defaultConfig := ConfigIniDefault()

			opts.Current = defaultConfig.CurrentRaceConfig
			opts.ChampionshipHasAtLeastOnceRace = false
		}
	}

	if !championship.OpenEntrants {
		opts.AvailableSessions = AvailableSessionsNoBooking
	}

	opts.ShowOverridePasswordCard = false

	err = cm.applyCurrentRaceSetupToOptions(opts, opts.Current)

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

			if event.IsRaceWeekend() {
				if err := cm.store.DeleteRaceWeekend(event.RaceWeekendID.String()); err != nil {
					return err
				}
			}

			if !event.Scheduled.IsZero() {
				if err := cm.ScheduleEvent(championshipID, eventID, time.Time{}, "", ""); err != nil {
					return err
				}
			}

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
	return cm.StartEvent(championshipID, eventID, true)
}

func (cm *ChampionshipManager) StartEvent(championshipID string, eventID string, isPreChampionshipPracticeEvent bool) error {
	championship, event, err := cm.GetChampionshipAndEvent(championshipID, eventID)

	if err != nil {
		return err
	}

	event.RaceSetup.Cars = strings.Join(championship.ValidCarIDs(), ";")

	if config.Lua.Enabled && Premium() {
		err = championshipEventStartPlugin(event, championship)

		if err != nil {
			logrus.WithError(err).Error("championship event start plugin script failed")
		}
	}

	entryList := event.CombineEntryLists(championship)

	if championship.SignUpForm.Enabled && !championship.OpenEntrants && !isPreChampionshipPracticeEvent {
		filteredEntryList := make(EntryList)

		// a sign up championship (which is not open) should remove all empty entrants before the event starts
		// here we are building a new filtered entry list so grid positions are not 'missing'
		for _, entrant := range entryList {
			if entrant.GUID != "" {
				filteredEntryList.AddInPitBox(entrant, entrant.PitBox)
			}
		}

		entryList = filteredEntryList

		event.RaceSetup.PickupModeEnabled = 1
		event.RaceSetup.LockedEntryList = 1
	} else {
		if championship.OpenEntrants {
			event.RaceSetup.PickupModeEnabled = 1
			event.RaceSetup.LockedEntryList = 0
		} else {
			event.RaceSetup.PickupModeEnabled = 1
			event.RaceSetup.LockedEntryList = 1
		}
	}

	event.RaceSetup.LoopMode = 1

	if event.RaceSetup.HasSession(SessionTypeBooking) {
		logrus.Infof("Championship event has a booking session. Disabling PickupMode, clearing EntryList")
		// championship events with booking do not have an entry list. pick up mode is disabled.
		event.RaceSetup.PickupModeEnabled = 0
		entryList = nil
	} else {
		event.RaceSetup.MaxClients = len(entryList)
	}

	if !isPreChampionshipPracticeEvent {
		logrus.Infof("Starting Championship Event: %s at %s (%s) with %d entrants", event.RaceSetup.Cars, event.RaceSetup.Track, event.RaceSetup.TrackLayout, event.RaceSetup.MaxClients)

		// track that this is the current event
		return cm.applyConfigAndStart(&ActiveChampionship{
			ChampionshipID:      championship.ID,
			EventID:             event.ID,
			Name:                championship.Name,
			OverridePassword:    championship.OverridePassword,
			ReplacementPassword: championship.ReplacementPassword,
			Description:         string(championship.Info),
			RaceConfig:          event.RaceSetup,
			EntryList:           entryList,
		})
	}

	// delete all sessions other than booking (if there is a booking session)
	delete(event.RaceSetup.Sessions, SessionTypePractice)
	delete(event.RaceSetup.Sessions, SessionTypeQualifying)
	delete(event.RaceSetup.Sessions, SessionTypeRace)

	event.RaceSetup.Sessions[SessionTypePractice] = &SessionConfig{
		Name:   "Practice",
		Time:   120,
		IsOpen: 1,
	}

	if !event.RaceSetup.HasSession(SessionTypeBooking) {
		// #271: override pickup mode to ON for practice sessions
		event.RaceSetup.PickupModeEnabled = 1
	}

	return cm.RaceManager.applyConfigAndStart(&ActiveChampionship{
		ChampionshipID:      championship.ID,
		EventID:             event.ID,
		Name:                championship.Name,
		OverridePassword:    championship.OverridePassword,
		ReplacementPassword: championship.ReplacementPassword,
		Description:         string(championship.Info),
		IsPracticeSession:   true,
		RaceConfig:          event.RaceSetup,
		EntryList:           entryList,
	})
}

func championshipEventStartPlugin(event *ChampionshipEvent, championship *Championship) error {
	var standings [][]*ChampionshipStanding

	for _, class := range championship.Classes {
		standings = append(standings, class.Standings(championship.Events))
	}

	p := &LuaPlugin{}

	newEvent, newChampionship := NewChampionshipEvent(), NewChampionship(championship.Name)

	p.Inputs(event, championship, standings).Outputs(newEvent, newChampionship)
	err := p.Call("./plugins/championship.lua", "onChampionshipEventStart")

	if err != nil {
		return err
	}

	*event, *championship = *newEvent, *newChampionship

	return nil
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

var ErrScheduledTimeIsZero = errors.New("servermanager: can't schedule race for zero time")

func (cm *ChampionshipManager) ScheduleEvent(championshipID string, eventID string, date time.Time, action string, recurrence string) error {
	championship, event, err := cm.GetChampionshipAndEvent(championshipID, eventID)

	if err != nil {
		return err
	}

	event.Scheduled = date
	event.ScheduledServerID = serverID

	// if there is an existing schedule timer for this event stop it
	if timer := cm.championshipEventStartTimers[event.ID.String()]; timer != nil {
		timer.Stop()
	}

	if timer := cm.championshipEventReminderTimers[event.ID.String()]; timer != nil {
		timer.Stop()
	}

	if action == "add" {
		if date.IsZero() {
			return ErrScheduledTimeIsZero
		}

		// add a scheduled event on date
		if recurrence != "already-set" {
			if recurrence != "" {
				err := event.SetRecurrenceRule(recurrence)

				if err != nil {
					return err
				}

				// only set once when the event is first scheduled
				event.ScheduledInitial = date
			} else {
				event.ClearRecurrenceRule()
			}
		}

		if config.Lua.Enabled && Premium() {
			err = championshipEventSchedulePlugin(championship, event)

			if err != nil {
				logrus.WithError(err).Error("event schedule plugin script failed")
			}
		}

		cm.championshipEventStartTimers[event.ID.String()], err = when.When(date, func() {
			err := cm.StartScheduledEvent(championship, event)

			if err != nil {
				logrus.WithError(err).Errorf("couldn't start scheduled championship event")
			}
		})

		if err != nil {
			return err
		}

		if cm.notificationManager.HasNotificationReminders() {
			for _, timer := range cm.notificationManager.GetNotificationReminders() {
				thisTimer := timer

				cm.championshipEventReminderTimers[event.ID.String()], err = when.When(date.Add(time.Duration(0-timer)*time.Minute), func() {
					err := cm.notificationManager.SendChampionshipReminderMessage(championship, event, thisTimer)

					if err != nil {
						logrus.WithError(err).Errorf("couldn't send championship reminder message")
					}
				})

				if err != nil {
					logrus.WithError(err).Errorf("couldn't send championship reminder message")
				}
			}
		}
	} else {
		event.ClearRecurrenceRule()
	}

	return cm.UpsertChampionship(championship)
}

func championshipEventSchedulePlugin(championship *Championship, event *ChampionshipEvent) error {
	p := &LuaPlugin{}

	var standings [][]*ChampionshipStanding

	for _, class := range championship.Classes {
		standings = append(standings, class.Standings(championship.Events))
	}

	newEvent, newChampionship := NewChampionshipEvent(), NewChampionship(championship.Name)

	p.Inputs(event, championship, standings).Outputs(newEvent, newChampionship)
	err := p.Call("./plugins/championship.lua", "onChampionshipEventSchedule")

	if err != nil {
		return err
	}

	*event, *championship = *newEvent, *newChampionship

	return nil
}

func (cm *ChampionshipManager) StartScheduledEvent(championship *Championship, event *ChampionshipEvent) error {
	if event.HasRecurrenceRule() {
		// makes a copy of this event and schedules it based on the recurrence rule
		err := cm.ScheduleNextEventFromRecurrence(championship, event)

		if err != nil {
			return err
		}
	} else {
		// our copy of the championship is outdated, get the latest version
		var err error

		championship, err = cm.store.LoadChampionship(championship.ID.String())

		if err != nil {
			return err
		}

		event, err = championship.EventByID(event.ID.String())

		if err != nil {
			return err
		}

		event.Scheduled = time.Time{}

		err = cm.UpsertChampionship(championship)

		if err != nil {
			return err
		}
	}

	return cm.StartEvent(championship.ID.String(), event.ID.String(), false)
}

func (cm *ChampionshipManager) ScheduleNextEventFromRecurrence(championship *Championship, event *ChampionshipEvent) error {
	// make sure the championship is on the event
	event.championship = championship

	// duplicate the event with new ID and no schedule/completed time
	eventCopy := DuplicateChampionshipEvent(event)
	championship.Events = append(championship.Events, eventCopy)

	err := cm.UpsertChampionship(championship)

	if err != nil {
		return err
	}

	return cm.ScheduleEvent(championship.ID.String(), eventCopy.ID.String(), cm.FindNextEventRecurrence(event, event.Scheduled), "add", "already-set")
}

func (cm *ChampionshipManager) FindNextEventRecurrence(event *ChampionshipEvent, start time.Time) time.Time {
	rule, err := event.GetRecurrenceRule()

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't get recurrence rule for race: %s, %s", event.ID.String(), event.Recurrence)
		return time.Time{}
	}

	next := rule.After(start, false)

	if next.After(time.Now()) {
		return next
	}

	return cm.FindNextEventRecurrence(event, next)
}

func (cm *ChampionshipManager) ChampionshipEventCallback(message udp.Message) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if !cm.ChampionshipEventIsRunning() {
		return
	}

	championship, err := cm.LoadChampionship(cm.activeChampionship.ChampionshipID.String())

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't load championship with ID: %s", cm.activeChampionship.ChampionshipID.String())
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
	return string(s.DriverGUID)
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
			logrus.WithError(err).Errorf("Could not save session results to championship %s", cm.activeChampionship.ChampionshipID.String())
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

		if championship.OpenEntrants && championship.PersistOpenEntrants && a.Event() == udp.EventNewConnection {
			// a person joined, check to see if they need adding to the championship
			foundSlot, classForCar, err := cm.AddEntrantFromSessionData(championship, sessionEntrantWrapper(a), false, false)

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
				"This event is part of the %s%s! %s%s\n",
				championship.Name,
				championshipText,
				championship.GetPlayerSummary(string(entrant.DriverGUID)),
				visitServer,
			),
			60,
		), "\n")

		for _, msg := range wrapped {
			welcomeMessage, err := udp.NewSendChat(entrant.CarID, msg)

			if err == nil {
				err := cm.process.SendUDPMessage(welcomeMessage)

				if err != nil {
					logrus.WithError(err).Errorf("Unable to send welcome message to: %s", entrant.DriverName)
				}
			} else {
				logrus.WithError(err).Errorf("Unable to build welcome message to: %s", entrant.DriverName)
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
				if !cm.ChampionshipEventIsRunning() {
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
		} else {
			saveChampionship = false
		}
	case udp.LapCompleted:
		cm.activeChampionship.NumLapsCompleted++

	case udp.EndSession:
		filename := filepath.Base(string(a))
		logrus.Infof("End Session, file outputted at: %s", filename)

		results, err := LoadResult(filename)

		if err != nil {
			logrus.WithError(err).Errorf("Could not read session results for %s", cm.activeChampionship.SessionType.String())
			return
		}

		// Update the old results json file with more championship information, required for applying penalties properly
		championship.EnhanceResults(results)
		err = saveResults(filename, results)

		if err != nil {
			logrus.WithError(err).Errorf("Could not update session results for %s", cm.activeChampionship.SessionType.String())
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
			err := cm.process.Stop()

			if err != nil {
				logrus.WithError(err).Errorf("Could not stop Assetto Process")
			}
		}

		if championship.ACSR {
			go panicCapture(func() {
				cm.acsrClient.SendChampionship(*championship)
			})
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
			savedRace, hasSavedRace := event.Sessions[SessionTypeRace]

			if t == SessionTypeRace && event.RaceSetup.HasMultipleRaces() && hasSavedRace && savedRace.Completed() {
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

	if err := cm.process.Stop(); err != nil {
		return err
	}

	return cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) RestartEvent(championshipID string, eventID string) error {
	err := cm.CancelEvent(championshipID, eventID)

	if err != nil {
		return err
	}

	return cm.StartEvent(championshipID, eventID, false)
}

var ErrNoActiveChampionshipEvent = errors.New("servermanager: no active championship event")

func (cm *ChampionshipManager) ChampionshipEventIsRunning() bool {
	return cm.process.Event().IsChampionship() && !cm.process.Event().IsPractice() && cm.activeChampionship != nil
}

func (cm *ChampionshipManager) RestartActiveEvent() error {
	if !cm.ChampionshipEventIsRunning() {
		return ErrNoActiveChampionshipEvent
	}

	return cm.RestartEvent(cm.activeChampionship.ChampionshipID.String(), cm.activeChampionship.EventID.String())
}

func (cm *ChampionshipManager) StopActiveEvent() error {
	if !cm.ChampionshipEventIsRunning() {
		return ErrNoActiveChampionshipEvent
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

	for _, event := range championship.Events {
		if event.IsRaceWeekend() {
			err := cm.store.UpsertRaceWeekend(event.RaceWeekend)

			if err != nil {
				return "", err
			}
		}
	}

	return championship.ID.String(), cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) ImportEventSetup(championshipID string, eventID string) error {
	race, err := cm.store.FindCustomRaceByID(eventID)

	if err != nil {
		return err
	}

	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	_, err = championship.ImportEvent(race)

	if err != nil {
		return err
	}

	return cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) ImportRaceWeekendSetup(championshipID string, eventID string) error {
	weekend, err := cm.store.LoadRaceWeekend(eventID)

	if err != nil {
		return err
	}

	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	_, err = championship.ImportEvent(weekend)

	if err != nil {
		return err
	}

	err = cm.store.UpsertRaceWeekend(weekend)

	if err != nil {
		return err
	}

	return cm.UpsertChampionship(championship)
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

	sessionsOrdered := []SessionType{
		SessionTypePractice,
		SessionTypeQualifying,
		SessionTypeRace,
		SessionTypeSecondRace,
	}

	for _, sessionType := range sessionsOrdered {
		sessionFile, ok := sessions[sessionType]

		if !ok || sessionFile == "" {
			continue
		}

		results, err := LoadResult(sessionFile + ".json")

		if err != nil {
			return err
		}

		if championship.OpenEntrants && championship.PersistOpenEntrants {
			// if the championship is open, we might have entrants in this session file who have not
			// raced in this championship before. add them to the championship as they would be added
			// if they joined during a race.
			for _, car := range results.Cars {
				if car.GetGUID() == "" {
					continue
				}

				foundFreeSlot, _, err := cm.AddEntrantFromSessionData(championship, car, false, false)

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

func (cm *ChampionshipManager) AddEntrantFromSessionData(championship *Championship, potentialEntrant PotentialChampionshipEntrant, overwriteSkinForAllEvents bool, takeFirstFreeSlot bool) (foundFreeEntrantSlot bool, entrantClass *ChampionshipClass, err error) {
	var entrant *Entrant

	if takeFirstFreeSlot {
		foundFreeEntrantSlot, entrant, entrantClass, err = championship.AddEntrantInFirstFreeSlot(potentialEntrant)
	} else {
		foundFreeEntrantSlot, entrant, entrantClass, err = championship.AddEntrantFromSession(potentialEntrant)
	}

	if err != nil {
		return foundFreeEntrantSlot, entrantClass, err
	}

	if foundFreeEntrantSlot {
		if overwriteSkinForAllEvents {
			// the user's skin setup should be applied to all event settings
			for _, event := range championship.Events {
				eventEntrant := event.EntryList.FindEntrantByInternalUUID(entrant.InternalUUID)

				eventEntrant.Skin = potentialEntrant.GetSkin()
			}
		}

		newEntrant := NewEntrant()

		newEntrant.GUID = potentialEntrant.GetGUID()
		newEntrant.Name = potentialEntrant.GetName()
		newEntrant.Team = potentialEntrant.GetTeam()

		e := make(EntryList)

		e.Add(newEntrant)

		err := cm.SaveEntrantsForAutoFill(e)

		if err != nil {
			logrus.Errorf("Couldn't add entrant (GUID: %s, Name: %s) to autofill list", newEntrant.GUID, newEntrant.Name)
		}
	}

	return foundFreeEntrantSlot, entrantClass, nil
}

func (cm *ChampionshipManager) BuildICalFeed(championshipID string, w io.Writer) error {
	championship, err := cm.LoadChampionship(championshipID)

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

func (cm *ChampionshipManager) ReorderChampionshipEvents(championshipID string, championshipEventIDsInOrder []string) error {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return err
	}

	var orderedEvents []*ChampionshipEvent

	for _, championshipEventID := range championshipEventIDsInOrder {
		event, err := championship.EventByID(championshipEventID)

		if err != nil {
			return err
		}

		orderedEvents = append(orderedEvents, event)
	}

	championship.Events = orderedEvents

	return cm.UpsertChampionship(championship)
}

type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}

var steamGUIDRegex = regexp.MustCompile("^[0-9]{17}(;[0-9]{17})*$")

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
		if championship.SignUpForm.AskForEmail && entrant.Email == signUpResponse.Email && entrant.GUID != signUpResponse.GUID {
			return signUpResponse, false, ValidationError("Someone has already registered with this email address.")
		}
	}

	if !championship.SignUpForm.RequiresApproval {
		// check to see if there is room in the entrylist for the user in their specific car
		foundSlot, _, err = cm.AddEntrantFromSessionData(championship, signUpResponse, true, championship.SignUpForm.HideCarChoice)

		if err != nil {
			return signUpResponse, foundSlot, err
		}

		if foundSlot {
			signUpResponse.Status = ChampionshipEntrantAccepted
		} else {
			signUpResponse.Status = ChampionshipEntrantRejected
		}
	}

	updatingRegistration := false

	for index, response := range championship.SignUpForm.Responses {
		if response.GUID == signUpResponse.GUID {
			championship.SignUpForm.Responses[index] = signUpResponse
			updatingRegistration = true
			break
		}
	}

	if !updatingRegistration {
		championship.SignUpForm.Responses = append(championship.SignUpForm.Responses, signUpResponse)
	}

	return signUpResponse, foundSlot, cm.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) InitScheduledChampionships() error {
	cm.championshipEventStartTimers = make(map[string]*when.Timer)
	cm.championshipEventReminderTimers = make(map[string]*when.Timer)
	championships, err := cm.ListChampionships()

	if err != nil {
		return err
	}

	for _, championship := range championships {
		championship := championship

		for _, event := range championship.Events {
			event := event

			if event.ScheduledServerID != serverID {
				continue
			}

			if event.Scheduled.After(time.Now()) {
				// add a scheduled event on date
				cm.championshipEventStartTimers[event.ID.String()], err = when.When(event.Scheduled, func() {
					err := cm.StartScheduledEvent(championship, event)

					if err != nil {
						logrus.WithError(err).Errorf("couldn't start scheduled championship event")
					}
				})

				if err != nil {
					logrus.WithError(err).Errorf("Could not schedule event: %s", event.ID.String())
					continue
				}

				if cm.notificationManager.HasNotificationReminders() {
					for _, timer := range cm.notificationManager.GetNotificationReminders() {
						if event.Scheduled.Add(time.Duration(0-timer) * time.Minute).After(time.Now()) {
							thisTimer := timer

							cm.championshipEventReminderTimers[event.ID.String()], err = when.When(event.Scheduled.Add(time.Duration(0-timer)*time.Minute), func() {
								err := cm.notificationManager.SendChampionshipReminderMessage(championship, event, thisTimer)

								if err != nil {
									logrus.WithError(err).Errorf("Could not send championship reminder message for event: %s", event.ID.String())
								}
							})

							if err != nil {
								logrus.WithError(err).Errorf("Could not schedule event: %s", event.ID.String())
								continue
							}
						}
					}
				}

				return cm.UpsertChampionship(championship)
			}

			zeroTime := time.Time{}

			if event.Scheduled != zeroTime {
				logrus.Infof("Looks like the server was offline whilst a scheduled event was meant to start!"+
					" Start time: %s. The schedule has been cleared. Start the event manually if you wish to run it.", event.Scheduled.String())

				event.Scheduled = zeroTime

				return cm.UpsertChampionship(championship)
			}
		}
	}

	return nil
}

func (cm *ChampionshipManager) DuplicateEvent(championshipID, eventID string) (*ChampionshipEvent, error) {
	championship, err := cm.LoadChampionship(championshipID)

	if err != nil {
		return nil, err
	}

	event, err := championship.EventByID(eventID)

	if err != nil {
		return nil, err
	}

	var newEvent *ChampionshipEvent

	if !event.IsRaceWeekend() {
		newEvent, err = championship.ImportEvent(event)

		if err != nil {
			return nil, err
		}
	} else {
		duplicateRaceWeekend, err := event.RaceWeekend.Duplicate()

		if err != nil {
			return nil, err
		}

		newEvent, err = championship.ImportEvent(duplicateRaceWeekend)

		if err != nil {
			return nil, err
		}

		newEvent.RaceWeekend.Name = "Duplicate: " + newEvent.RaceWeekend.Name

		if err := cm.store.UpsertRaceWeekend(newEvent.RaceWeekend); err != nil {
			return nil, err
		}
	}

	if err := cm.UpsertChampionship(championship); err != nil {
		return nil, err
	}

	return newEvent, nil
}
