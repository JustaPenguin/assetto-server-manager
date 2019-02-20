package servermanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var championshipManager *ChampionshipManager

// DefaultChampionshipPoints is the Formula 1 points system.
var DefaultChampionshipPoints = ChampionshipPoints{
	Places: []int{
		25,
		18,
		15,
		12,
		10,
		8,
		6,
		4,
		2,
		1,
	},
	BestLap:      0,
	PolePosition: 0,
}

// ChampionshipPoints represent the potential points for positions as well as other awards in a Championship.
type ChampionshipPoints struct {
	Places       []int
	BestLap      int
	PolePosition int
}

// NewChampionship creates a Championship with a given name, creating a UUID for the championship as well.
func NewChampionship(name string) *Championship {
	return &Championship{
		ID:      uuid.New(),
		Name:    name,
		Created: time.Now(),
		Points:  DefaultChampionshipPoints,
	}
}

// A Championship is a collection of ChampionshipEvents for a group of Entrants. Each Entrant in a Championship
// is awarded Points for their position in a ChampionshipEvent.
type Championship struct {
	ID      uuid.UUID
	Name    string
	Created time.Time
	Updated time.Time
	Deleted time.Time

	Entrants EntryList

	Events []*ChampionshipEvent
	Points ChampionshipPoints
}

func (c *Championship) EventByID(id string) (*ChampionshipEvent, error) {
	for _, e := range c.Events {
		if e.ID.String() == id {
			return e, nil
		}
	}

	return nil, ErrInvalidChampionshipEvent
}

// ValidCarIDs returns a set of all of the valid cars in the Championship - that is, the smallest possible list
// of Cars driven by the Entrants.
func (c *Championship) ValidCarIDs() []string {
	cars := make(map[string]bool)

	for _, e := range c.Entrants {
		cars[e.Model] = true
	}

	var out []string

	for car := range cars {
		out = append(out, car)
	}

	return out
}

// Progress of the Championship as a percentage
func (c *Championship) Progress() float64 {
	numRaces := float64(len(c.Events))

	if numRaces == 0 {
		return 0
	}

	numCompletedRaces := float64(0)

	for _, race := range c.Events {
		if race.Completed() {
			numCompletedRaces++
		}
	}

	return (numCompletedRaces / numRaces) * 100
}

// ChampionshipStanding is the current number of Points an Entrant in the Championship has.
type ChampionshipStanding struct {
	Entrant Entrant
	Points  int
}

// PointForPos uses the Championship's Points to determine what number should be awarded to a given position
func (c *Championship) PointForPos(i int) int {
	if i >= len(c.Points.Places) {
		return 0
	}

	return c.Points.Places[i]
}

// Standings returns the current Driver Standings for the Championship.
func (c *Championship) Standings() []*ChampionshipStanding {
	var out []*ChampionshipStanding

	entrants := make(map[string]*ChampionshipStanding)

	if len(c.Events) > 0 {
		for _, entrant := range c.Entrants {
			if entrants[entrant.GUID] == nil {
				entrants[entrant.GUID] = &ChampionshipStanding{
					Entrant: entrant,
				}
			}
		}
	}

	for _, event := range c.Events {
		qualifying, qualifyingOK := event.Sessions[SessionTypeQualifying]

		if qualifyingOK && qualifying.Results != nil {
			for pos, driver := range qualifying.Results.Result {
				if pos != 0 {
					continue
				}

				if _, ok := entrants[driver.DriverGUID]; ok {
					// if an entrant is removed from a championship this can panic, hence the ok check
					entrants[driver.DriverGUID].Points += c.Points.PolePosition
				}
			}
		}

		race, raceOK := event.Sessions[SessionTypeRace]

		if raceOK && race.Results != nil {
			fastestLap := race.Results.FastestLap()

			for pos, driver := range race.Results.Result {
				if driver.TotalTime <= 0 {
					continue
				}

				if _, ok := entrants[driver.DriverGUID]; ok {
					// if an entrant is removed from a championship this can panic, hence the ok check
					entrants[driver.DriverGUID].Points += c.PointForPos(pos)

					if fastestLap.DriverGUID == driver.DriverGUID {
						entrants[driver.DriverGUID].Points += c.Points.BestLap
					}
				}
			}
		}
	}

	for _, entrant := range entrants {
		out = append(out, entrant)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Entrant.Name < out[j].Entrant.Name
	})

	sort.Slice(out, func(i, j int) bool {
		return out[i].Points > out[j].Points
	})

	return out
}

// TeamStanding is the current number of Points a Team has.
type TeamStanding struct {
	Team   string
	Points int
}

// TeamStandings returns the current position of Teams in the Championship.
func (c *Championship) TeamStandings() []*TeamStanding {
	teams := make(map[string]int)

	for _, driver := range c.Standings() {
		if _, ok := teams[driver.Entrant.Team]; !ok {
			teams[driver.Entrant.Team] = driver.Points
		} else {
			teams[driver.Entrant.Team] += driver.Points
		}
	}

	var out []*TeamStanding

	for name, pts := range teams {
		out = append(out, &TeamStanding{
			Team:   name,
			Points: pts,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Team < out[j].Team
	})

	sort.Slice(out, func(i, j int) bool {
		return out[i].Points > out[j].Points
	})

	return out
}

// NewChampionshipEvent creates a ChampionshipEvent with an ID
func NewChampionshipEvent() *ChampionshipEvent {
	return &ChampionshipEvent{
		ID: uuid.New(),
	}
}

// A ChampionshipEvent is a given RaceSetup with Sessions.
type ChampionshipEvent struct {
	ID uuid.UUID

	RaceSetup CurrentRaceConfig
	Sessions  map[SessionType]*ChampionshipSession

	StartedTime   time.Time
	CompletedTime time.Time
}

func (cr *ChampionshipEvent) InProgress() bool {
	return !cr.StartedTime.IsZero() && cr.CompletedTime.IsZero()
}

// Completed ChampionshipEvents have a non-zero CompletedTime
func (cr *ChampionshipEvent) Completed() bool {
	return !cr.CompletedTime.IsZero()
}

type ChampionshipSession struct {
	StartedTime   time.Time
	CompletedTime time.Time

	Results *SessionResults
}

func (ce *ChampionshipSession) InProgress() bool {
	return !ce.StartedTime.IsZero() && ce.CompletedTime.IsZero()
}

// Completed ChampionshipSessions have a non-zero CompletedTime
func (ce *ChampionshipSession) Completed() bool {
	return !ce.CompletedTime.IsZero()
}

// listChampionshipsHandler lists all available Championships known to Server Manager
func listChampionshipsHandler(w http.ResponseWriter, r *http.Request) {
	championships, err := championshipManager.ListChampionships()

	if err != nil {
		logrus.Errorf("couldn't list championships, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "championships/index.html", map[string]interface{}{
		"championships": championships,
	})
}

// newOrEditChampionshipHandler builds a Championship form for the user to create a Championship.
func newOrEditChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	opts, err := championshipManager.BuildChampionshipOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build championship form, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "championships/new.html", opts)
}

// submitNewChampionshipHandler creates a given Championship and redirects the user to begin
// the flow of adding events to the new Championship
func submitNewChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	championship, edited, err := championshipManager.HandleCreateChampionship(r)

	if err != nil {
		logrus.Errorf("couldn't create championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if edited {
		AddFlashQuick(w, r, "Championship successfully edited!")
		http.Redirect(w, r, "/championship/"+championship.ID.String(), http.StatusFound)
	} else {
		AddFlashQuick(w, r, "We've created the Championship. Now you need to add some Events!")
		http.Redirect(w, r, "/championship/"+championship.ID.String()+"/event", http.StatusFound)
	}
}

// viewChampionshipHandler shows details of a given Championship
func viewChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.Errorf("couldn't load championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	eventInProgress := false

	for _, event := range championship.Events {
		if event.InProgress() {
			eventInProgress = true
			break
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, "championships/view.html", map[string]interface{}{
		"Championship":    championship,
		"EventInProgress": eventInProgress,
	})
}

// exportChampionshipHandler returns all known data about a Championship in JSON format.
func exportChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.Errorf("couldn't export championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(championship)
}

// deleteChampionshipHandler soft deletes a Championship.
func deleteChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.DeleteChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.Errorf("couldn't delete championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlashQuick(w, r, "Championship deleted!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// championshipEventConfigurationHandler builds a Custom Race form with slight modifications
// to allow a user to configure a ChampionshipEvent.
func championshipEventConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	championshipRaceOpts, err := championshipManager.BuildChampionshipEventOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build championship race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "custom-race/new.html", championshipRaceOpts)
}

// championshipSubmitEventConfigurationHandler takes an Event Configuration from a form and
// builds an event optionally, this is used for editing ChampionshipEvents.
func championshipSubmitEventConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	championship, event, edited, err := championshipManager.SaveChampionshipEvent(r)

	if err != nil {
		logrus.Errorf("couldn't build championship race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if edited {
		AddFlashQuick(w, r,
			fmt.Sprintf(
				"Championship race at %s was successfully edited!",
				prettifyName(event.RaceSetup.Track, false),
			),
		)
	} else {
		AddFlashQuick(w, r,
			fmt.Sprintf(
				"Championship race at %s was successfully added!",
				prettifyName(event.RaceSetup.Track, false),
			),
		)
	}

	if r.FormValue("action") == "saveChampionship" {
		// end the race creation flow
		http.Redirect(w, r, "/championship/"+championship.ID.String(), http.StatusFound)
		return
	} else {
		// add another event
		http.Redirect(w, r, "/championship/"+championship.ID.String()+"/event", http.StatusFound)
	}
}

func championshipStartEventHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.StartEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not start championship event, err: %s", err)

		AddErrFlashQuick(w, r, "Couldn't start the Event")
	} else {
		AddFlashQuick(w, r, "Event started successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func championshipDeleteEventHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.DeleteEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not delete championship event, err: %s", err)

		AddErrFlashQuick(w, r, "Couldn't delete the Event")
	} else {
		AddFlashQuick(w, r, "Event deleted successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func championshipStartPracticeEventHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.StartPracticeEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not start practice championship event, err: %s", err)

		AddErrFlashQuick(w, r, "Couldn't start the Practice Event")
	} else {
		AddFlashQuick(w, r, "Practice Event started successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func championshipCancelEventHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.CancelEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not cancel championship event, err: %s", err)

		AddErrFlashQuick(w, r, "Couldn't cancel the Championship Event")
	} else {
		AddFlashQuick(w, r, "Event cancelled successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func championshipRestartEventHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.RestartEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not restart championship event, err: %s", err)

		AddErrFlashQuick(w, r, "Couldn't restart the Championship Event")
	} else {
		AddFlashQuick(w, r, "Event restarted successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
