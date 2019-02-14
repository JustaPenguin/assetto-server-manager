package servermanager

import (
	"errors"
	"net/http"
	"strings"

	"github.com/etcd-io/bbolt"
	"github.com/gorilla/mux"
)

type ChampionshipManager struct {
	*RaceManager
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
		event = &ChampionshipEvent{
			RaceSetup: *raceConfig,
		}

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

	defaults := ConfigIniDefault

	raceSetup := championship.Events[eventID].RaceSetup

	for sessionName, sess := range raceSetup.Sessions {
		if sessionName != SessionTypePractice {
			// remove non-practice sessions
			delete(raceSetup.Sessions, sessionName)
			continue
		}

		sess.Time = 120 // 2hrs
	}

	raceSetup.LoopMode = 1

	defaults.CurrentRaceConfig = raceSetup

	return cm.applyConfigAndStart(defaults, championship.Entrants)
}
