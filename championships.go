package servermanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var championshipManager *ChampionshipManager

// ChampionshipClassColors are sequentially selected to indicate different classes within a Championship
var ChampionshipClassColors = []string{
	"#9ec6f5",
	"#91d8af",
	"#dba8ed",
	"#e3a488",
	"#e3819a",
	"#908ba1",
	"#a2b5b9",
	"#a681b4",
	"#c1929d",
	"#999ecf",
}

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
	BestLap:              0,
	PolePosition:         0,
	SecondRaceMultiplier: 1,
}

// ChampionshipPoints represent the potential points for positions as well as other awards in a Championship.
type ChampionshipPoints struct {
	Places       []int
	BestLap      int
	PolePosition int

	SecondRaceMultiplier float64
}

// NewChampionship creates a Championship with a given name, creating a UUID for the championship as well.
func NewChampionship(name string) *Championship {
	return &Championship{
		ID:      uuid.New(),
		Name:    name,
		Created: time.Now(),
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

	// OpenEntrants indicates that entrant names do not need to be specified in the EntryList.
	// As Entrants join a championship, the available Entrant slots will be filled by the information
	// provided by a join message. The EntryList for each class will still need creating, but
	// can omit names/GUIDs/teams as necessary. These can then be edited after the fact.
	OpenEntrants bool

	Classes []*ChampionshipClass
	Events  []*ChampionshipEvent
}

func (c *Championship) GetPlayerSummary(guid string) string {
	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			if entrant.GUID == guid {
				standings := class.Standings(c.Events)
				teamstandings := class.TeamStandings(c.Events)

				var driverPos, teamPos int
				var driverPoints, teamPoints float64

				for pos, standing := range standings {
					if standing.Entrant.GUID == guid {
						driverPos = pos + 1
						driverPoints = standing.Points
					}
				}

				var driverAhead string
				var driverAheadPoints float64

				if driverPos > 2 {
					driverAhead = standings[driverPos-2].Entrant.Name
					driverAheadPoints = standings[driverPos-2].Points
				}

				for pos, standing := range teamstandings {
					if standing.Team == entrant.Team {
						teamPos = pos + 1
						teamPoints = standing.Points
					}
				}

				var teamAhead string
				var teamAheadPoints float64

				if teamPos > 2 {
					teamAhead = teamstandings[teamPos-2].Team
					teamAheadPoints = teamstandings[teamPos-2].Points
				}

				out := fmt.Sprintf("You are currently %d%s with %.2f points. ", driverPos, ordinal(int64(driverPos)), driverPoints)

				if driverAhead != "" {
					out += fmt.Sprintf("The driver ahead of you is %s with %.2f points. ", driverAhead, driverAheadPoints)
				}

				if entrant.Team != "" {
					out += fmt.Sprintf("Your team '%s' has %.2f points. ", entrant.Team, teamPoints)

					if teamAhead != "" {
						out += fmt.Sprintf("The team ahead is '%s' with %.2f points. ", teamAhead, teamAheadPoints)
					}
				}

				return out
			}
		}
	}

	return ""
}

// IsMultiClass is true if the Championship has more than one Class
func (c *Championship) IsMultiClass() bool {
	return len(c.Classes) > 1
}

var ErrClassNotFound = errors.New("servermanager: championship class not found")

func (c *Championship) FindClassForCarModel(model string) (*ChampionshipClass, error) {
	for _, class := range c.Classes {
		for _, car := range class.ValidCarIDs() {
			if car == model {
				return class, nil
			}
		}
	}

	return nil, ErrClassNotFound
}

// NewChampionshipClass creates a championship class with the default points
func NewChampionshipClass(name string) *ChampionshipClass {
	return &ChampionshipClass{
		Name:     name,
		Points:   DefaultChampionshipPoints,
		Entrants: make(EntryList),
	}
}

// ChampionshipClass contains a Name, Entrants (including Cars, Skins) and Points for those Entrants
type ChampionshipClass struct {
	Name string

	Entrants EntryList
	Points   ChampionshipPoints
}

// ValidCarIDs returns a set of all cars chosen within the given class
func (c *ChampionshipClass) ValidCarIDs() []string {
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

// EventByID finds a ChampionshipEvent by its ID string.
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

	for _, class := range c.Classes {
		for _, e := range class.Entrants {
			cars[e.Model] = true
		}
	}

	var out []string

	for car := range cars {
		out = append(out, car)
	}

	return out
}

// NumEntrants is the number of entrants across all Classes in a Championship.
func (c *Championship) NumEntrants() int {
	entrants := 0

	for _, class := range c.Classes {
		entrants += len(class.Entrants)
	}

	return entrants
}

// AllEntrants returns the list of all entrants in the championship (across ALL classes)
func (c *Championship) AllEntrants() EntryList {
	e := make(EntryList)

	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			e.Add(entrant)
		}
	}

	return e
}

// AddClass to the championship
func (c *Championship) AddClass(class *ChampionshipClass) {
	c.Classes = append(c.Classes, class)
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
	Entrant *Entrant
	Points  float64
}

// PointForPos uses the Championship's Points to determine what number should be awarded to a given position
func (c *ChampionshipClass) PointForPos(i int) float64 {
	if i >= len(c.Points.Places) {
		return 0
	}

	return float64(c.Points.Places[i])
}

func (c *ChampionshipClass) CarInClass(car string) bool {
	for _, id := range c.ValidCarIDs() {
		if id == car {
			return true
		}
	}

	return false
}

func (c *ChampionshipClass) ResultsForClass(results []*SessionResult) (filtered []*SessionResult) {
	for _, result := range results {
		if c.CarInClass(result.CarModel) && result.TotalTime > 0 {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// Standings returns the current Driver Standings for the Championship.
func (c *ChampionshipClass) Standings(events []*ChampionshipEvent) []*ChampionshipStanding {
	var out []*ChampionshipStanding

	entrants := make(map[string]*ChampionshipStanding)

	if len(events) > 0 {
		for _, entrant := range c.Entrants {
			if entrants[entrant.GUID] == nil {
				entrants[entrant.GUID] = &ChampionshipStanding{
					Entrant: entrant,
				}
			}
		}
	}

	for _, event := range events {
		qualifying, qualifyingOK := event.Sessions[SessionTypeQualifying]

		if qualifyingOK && qualifying.Results != nil {
			for pos, driver := range c.ResultsForClass(qualifying.Results.Result) {
				if pos != 0 {
					continue
				}

				if _, ok := entrants[driver.DriverGUID]; ok {
					// if an entrant is removed from a championship this can panic, hence the ok check
					entrants[driver.DriverGUID].Points += float64(c.Points.PolePosition)
				}
			}
		}

		race, raceOK := event.Sessions[SessionTypeRace]

		if raceOK && race.Results != nil {
			fastestLap := race.Results.FastestLap()

			for pos, driver := range c.ResultsForClass(race.Results.Result) {
				if driver.TotalTime <= 0 {
					continue
				}

				if _, ok := entrants[driver.DriverGUID]; ok {
					// if an entrant is removed from a championship this can panic, hence the ok check
					entrants[driver.DriverGUID].Points += c.PointForPos(pos)

					if fastestLap.DriverGUID == driver.DriverGUID {
						entrants[driver.DriverGUID].Points += float64(c.Points.BestLap)
					}
				}
			}
		}

		race2, race2OK := event.Sessions[SessionTypeSecondRace]

		if race2OK && race2.Results != nil {
			fastestLap := race2.Results.FastestLap()

			for pos, driver := range c.ResultsForClass(race2.Results.Result) {
				if driver.TotalTime <= 0 {
					continue
				}

				if _, ok := entrants[driver.DriverGUID]; ok {
					// if an entrant is removed from a championship this can panic, hence the ok check
					entrants[driver.DriverGUID].Points += c.PointForPos(pos) * c.Points.SecondRaceMultiplier

					if fastestLap.DriverGUID == driver.DriverGUID {
						entrants[driver.DriverGUID].Points += float64(c.Points.BestLap) * c.Points.SecondRaceMultiplier
					}
				}
			}
		}
	}

	for _, entrant := range entrants {
		if entrant.Entrant.Name == "" {
			continue
		}

		out = append(out, entrant)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Points == out[j].Points {
			return out[i].Entrant.Name < out[j].Entrant.Name
		}

		return out[i].Points > out[j].Points
	})

	return out
}

// TeamStanding is the current number of Points a Team has.
type TeamStanding struct {
	Team   string
	Points float64
}

// TeamStandings returns the current position of Teams in the Championship.
func (c *ChampionshipClass) TeamStandings(events []*ChampionshipEvent) []*TeamStanding {
	teams := make(map[string]float64)

	for _, driver := range c.Standings(events) {
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
	Scheduled     time.Time
}

// LastSession returns the last configured session in the championship, in the following order:
// Race, Qualifying, Practice, Booking
func (cr *ChampionshipEvent) LastSession() SessionType {
	if cr.RaceSetup.HasMultipleRaces() {
		// there must be two races as reversed grid positions are on
		return SessionTypeSecondRace
	} else if cr.RaceSetup.HasSession(SessionTypeRace) {
		return SessionTypeRace
	} else if cr.RaceSetup.HasSession(SessionTypeQualifying) {
		return SessionTypeQualifying
	} else if cr.RaceSetup.HasSession(SessionTypePractice) {
		return SessionTypePractice
	} else {
		return SessionTypeBooking
	}
}

// InProgress indicates whether a ChampionshipEvent has been started but not stopped
func (cr *ChampionshipEvent) InProgress() bool {
	return !cr.StartedTime.IsZero() && cr.CompletedTime.IsZero()
}

// Completed ChampionshipEvents have a non-zero CompletedTime
func (cr *ChampionshipEvent) Completed() bool {
	return !cr.CompletedTime.IsZero()
}

// A ChampionshipSession contains information found from the live portion of the Championship tool
type ChampionshipSession struct {
	StartedTime   time.Time
	CompletedTime time.Time

	Results *SessionResults
}

// InProgress indicates whether a ChampionshipSession has been started but not stopped
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

// championshipStartEventHandler begins a championship event given by its ID
func championshipStartEventHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.StartEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not start championship event, err: %s", err)

		AddErrFlashQuick(w, r, "Couldn't start the Event")
	} else {
		AddFlashQuick(w, r, "Event started successfully!")
		time.Sleep(time.Second * 1)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func championshipScheduleEventHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logrus.Errorf("couldn't parse schedule race form, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	championshipID := chi.URLParam(r, "championshipID")
	championshipEventID := chi.URLParam(r, "eventID")
	dateString := r.FormValue("event-schedule-date")
	timeString := r.FormValue("event-schedule-time")

	dateTimeString := dateString + "-" + timeString

	date, err := time.Parse("2006-01-02-15:04", dateTimeString)

	if err != nil {
		logrus.Errorf("couldn't parse schedule championship event date, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = championshipManager.ScheduleEvent(championshipID, championshipEventID, date, r.FormValue("action"))

	if err != nil {
		logrus.Errorf("couldn't schedule championship event, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func championshipScheduleEventRemoveHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.ScheduleEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"),
		time.Time{}, "remove")

	if err != nil {
		logrus.Errorf("couldn't schedule championship event, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// championshipDeleteEventHandler soft deletes a championship event
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

// championshipStartPracticeEventHandler starts a Practice session for a given event
func championshipStartPracticeEventHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.StartPracticeEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not start practice championship event, err: %s", err)

		AddErrFlashQuick(w, r, "Couldn't start the Practice Event")
	} else {
		AddFlashQuick(w, r, "Practice Event started successfully!")
		time.Sleep(time.Second * 1)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// championshipCancelEventHandler stops a running championship event and clears any saved results
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

// championshipCancelEventHandler stops a running championship event and clears any saved results
// then starts the event again.
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
