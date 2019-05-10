package servermanager

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"sync"
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
	ID                  uuid.UUID
	Name                string
	Created             time.Time
	Updated             time.Time
	Deleted             time.Time
	OverridePassword    bool
	ReplacementPassword string

	// Raw html can be attached to championships, used to share tracks/cars etc.
	Info template.HTML
	// Deprecated, replaced with Info above.
	Links map[string]string

	// OpenEntrants indicates that entrant names do not need to be specified in the EntryList.
	// As Entrants join a championship, the available Entrant slots will be filled by the information
	// provided by a join message. The EntryList for each class will still need creating, but
	// can omit names/GUIDs/teams as necessary. These can then be edited after the fact.
	OpenEntrants bool

	// SignUpForm gives anyone on the web access to a Championship Sign Up Form so that they can
	// mark themselves for participation in this Championship.
	SignUpForm ChampionshipSignUpForm

	Classes []*ChampionshipClass
	Events  []*ChampionshipEvent

	entryListMutex sync.Mutex
}

type ChampionshipSignUpForm struct {
	Enabled          bool
	AskForEmail      bool
	AskForTeam       bool
	HideCarChoice    bool
	ExtraFields      []string
	RequiresApproval bool

	Responses []*ChampionshipSignUpResponse
}

func (c ChampionshipSignUpForm) EmailList(group string) string {
	var filteredStatus ChampionshipEntrantStatus

	switch group {
	case "accepted":
		filteredStatus = ChampionshipEntrantAccepted
	case "rejected":
		filteredStatus = ChampionshipEntrantRejected
	case "pending":
		filteredStatus = ChampionshipEntrantPending
	case "all":
		filteredStatus = ChampionshipEntrantAll
	default:
		panic("unknown entrant status: " + group)
	}

	var filteredEmails []string

	for _, entrant := range c.Responses {
		if entrant.Status == filteredStatus || filteredStatus == ChampionshipEntrantAll && entrant.Email != "" {
			filteredEmails = append(filteredEmails, entrant.Email)
		}
	}

	return strings.Join(filteredEmails, ",")
}

type ChampionshipEntrantStatus string

const (
	ChampionshipEntrantAll      = "All"
	ChampionshipEntrantAccepted = "Accepted"
	ChampionshipEntrantRejected = "Rejected"
	ChampionshipEntrantPending  = "Pending Approval"
)

type ChampionshipSignUpResponse struct {
	Created time.Time

	Name      string
	GUID      string
	Team      string
	Email     string
	Car       string
	Skin      string
	Questions map[string]string

	Status ChampionshipEntrantStatus
}

func (csr ChampionshipSignUpResponse) GetName() string {
	return csr.Name
}

func (csr ChampionshipSignUpResponse) GetTeam() string {
	return csr.Team
}

func (csr ChampionshipSignUpResponse) GetCar() string {
	return csr.Car
}

func (csr ChampionshipSignUpResponse) GetSkin() string {
	return csr.Skin
}

func (csr ChampionshipSignUpResponse) GetGUID() string {
	return csr.GUID
}

func (c *Championship) GetPlayerSummary(guid string) string {
	if c.Progress() == 0 {
		return "This is the first event of the Championship!"
	}

	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			if entrant.GUID == guid {
				standings := class.Standings(c.Events)
				teamstandings := class.TeamStandings(c.Events)

				var driverPos, teamPos int
				var driverPoints, teamPoints float64

				for pos, standing := range standings {
					if standing.Car.Driver.GUID == guid {
						driverPos = pos + 1
						driverPoints = standing.Points
					}
				}

				var driverAhead string
				var driverAheadPoints float64

				if driverPos >= 2 {
					driverAhead = standings[driverPos-2].Car.Driver.Name
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

				if teamPos >= 2 {
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

func (c *Championship) HasScheduledEvents() bool {
	for _, event := range c.Events {
		if !event.Scheduled.IsZero() {
			return true
		}
	}

	return false
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
		ID:       uuid.New(),
		Name:     name,
		Points:   DefaultChampionshipPoints,
		Entrants: make(EntryList),
	}
}

// ChampionshipClass contains a Name, Entrants (including Cars, Skins) and Points for those Entrants
type ChampionshipClass struct {
	ID   uuid.UUID
	Name string

	Entrants EntryList
	Points   ChampionshipPoints

	DriverPenalties, TeamPenalties map[string]int
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

// ClassByID finds a ChampionshipClass by its ID string.
func (c *Championship) ClassByID(id string) (*ChampionshipClass, error) {
	for _, x := range c.Classes {
		if x.ID.String() == id {
			return x, nil
		}
	}

	return nil, ErrInvalidChampionshipClass
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

type PotentialChampionshipEntrant interface {
	GetName() string
	GetTeam() string
	GetCar() string
	GetSkin() string
	GetGUID() string
}

func (c *Championship) AddEntrantFromSessionData(potentialEntrant PotentialChampionshipEntrant) (foundFreeEntrantSlot bool, entrantClass *ChampionshipClass, err error) {
	foundFreeSlot, class, err := c.AddEntrantFromSession(potentialEntrant)

	if err != nil {
		return foundFreeSlot, class, err
	}

	if foundFreeSlot {
		newEntrant := NewEntrant()

		newEntrant.GUID = potentialEntrant.GetGUID()
		newEntrant.Name = potentialEntrant.GetName()
		newEntrant.Team = potentialEntrant.GetTeam()

		e := make(EntryList)

		e.Add(newEntrant)

		err := raceManager.SaveEntrantsForAutoFill(e)

		if err != nil {
			logrus.Errorf("Couldn't add entrant (GUID; %s, Name; %s) to autofill list", newEntrant.GUID, newEntrant.Name)
		}
	}

	return foundFreeSlot, class, nil
}

func (c *Championship) AddEntrantFromSession(potentialEntrant PotentialChampionshipEntrant) (foundFreeEntrantSlot bool, entrantClass *ChampionshipClass, err error) {
	c.entryListMutex.Lock()
	defer c.entryListMutex.Unlock()

	classForCar, err := c.FindClassForCarModel(potentialEntrant.GetCar())

	if err != nil {
		logrus.Errorf("Could not find class for car: %s in championship", potentialEntrant.GetCar())

		return false, nil, err
	}

classLoop:
	for _, class := range c.Classes {
		for entrantKey, entrant := range class.Entrants {
			if entrant.GUID == potentialEntrant.GetGUID() {
				if class == classForCar {
					// the person is already in the EntryList and this class, update their information
					logrus.Debugf("Entrant: %s (%s) already found in EntryList. updating their info...", potentialEntrant.GetName(), potentialEntrant.GetGUID())
					entrant.Model = potentialEntrant.GetCar()
					entrant.Skin = potentialEntrant.GetSkin()

					return true, class, nil
				} else {
					// the user needs removing from this class
					logrus.Infof("Entrant: %s (%s) found in EntryList, but changed classes (%s -> %s). removing from original class.", potentialEntrant.GetName(), potentialEntrant.GetGUID(), class.Name, classForCar.Name)
					delete(class.Entrants, entrantKey)
					break classLoop
				}
			}
		}
	}
	// now look for empty Entrants in the Entrylist
	for carNum, entrant := range classForCar.Entrants {
		if entrant.Name == "" && entrant.GUID == "" {
			entrant.Name = potentialEntrant.GetName()
			entrant.GUID = potentialEntrant.GetGUID()
			entrant.Model = potentialEntrant.GetCar()
			entrant.Skin = potentialEntrant.GetSkin()
			entrant.Team = potentialEntrant.GetTeam()

			logrus.Infof("New championship entrant: %s (%s) has been assigned to %s in %s", entrant.Name, entrant.GUID, carNum, classForCar.Name)

			return true, classForCar, nil
		}
	}

	return false, classForCar, nil
}

// EnhanceResults takes a set of SessionResults and attaches Championship information to them.
func (c *Championship) EnhanceResults(results *SessionResults) {
	if results == nil {
		return
	}

	results.ChampionshipID = c.ID.String()

	// update names / teams to the values we know to be correct due to championship setup
	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			class.AttachEntrantToResult(entrant, results)
		}
	}
}

func NewChampionshipStanding(car *SessionCar) *ChampionshipStanding {
	return &ChampionshipStanding{
		Car:   car,
		Teams: make(map[string]int),
	}
}

// ChampionshipStanding is the current number of Points an Entrant in the Championship has.
type ChampionshipStanding struct {
	Car *SessionCar

	// Teams is a map of Team Name to how many Events per team were completed.
	Teams  map[string]int
	Points float64
}

func (cs *ChampionshipStanding) AddEventForTeam(team string) {
	if _, ok := cs.Teams[team]; ok {
		cs.Teams[team]++
	} else {
		cs.Teams[team] = 1
	}
}

func (cs *ChampionshipStanding) TeamSummary() string {
	if len(cs.Teams) == 1 {
		for team := range cs.Teams {
			return team
		}
	} else {
		// more than one team
		var summary []string

		for team, races := range cs.Teams {
			summary = append(summary, fmt.Sprintf("%s (%d races)", team, races))
		}

		return strings.Join(summary, ", ")
	}

	return ""
}

// PointForPos uses the Championship's Points to determine what number should be awarded to a given position
func (c *ChampionshipClass) PointForPos(i int) float64 {
	if i >= len(c.Points.Places) {
		return 0
	}

	return float64(c.Points.Places[i])
}

func (c *ChampionshipClass) DriverInClass(result *SessionResult) bool {
	return result.ClassID == c.ID
}

func (c *ChampionshipClass) AttachEntrantToResult(entrant *Entrant, results *SessionResults) {
	for _, car := range results.Cars {
		if car.Driver.GUID == entrant.GUID {
			car.Driver.AssignEntrant(entrant, c.ID)
		}
	}

	for _, event := range results.Events {
		if event.Driver.GUID == entrant.GUID {
			event.Driver.AssignEntrant(entrant, c.ID)
		}

		if event.OtherDriver.GUID == entrant.GUID {
			event.OtherDriver.AssignEntrant(entrant, c.ID)
		}
	}

	for _, lap := range results.Laps {
		if lap.DriverGUID == entrant.GUID {
			lap.DriverName = entrant.Name
			lap.ClassID = c.ID
		}
	}

	for _, result := range results.Result {
		if result.DriverGUID == entrant.GUID {
			result.DriverName = entrant.Name
			result.ClassID = c.ID
		}
	}
}

func (c *ChampionshipClass) ResultsForClass(results []*SessionResult) (filtered []*SessionResult) {
	for _, result := range results {
		if c.DriverInClass(result) && result.TotalTime > 0 {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

func (c *ChampionshipClass) PenaltyForGUID(guid string) int {
	if penalty, ok := c.DriverPenalties[guid]; ok {
		return penalty
	} else {
		return 0
	}
}

func (c *ChampionshipClass) PenaltyForTeam(name string) int {
	if teamPenalty, ok := c.TeamPenalties[name]; ok {
		return teamPenalty
	} else {
		return 0
	}
}

func (c *ChampionshipClass) standings(events []*ChampionshipEvent, givePoints func(event *ChampionshipEvent, driverGUID string, points float64)) {
	eventsReverseCompletedOrder := make([]*ChampionshipEvent, len(events))

	copy(eventsReverseCompletedOrder, events)

	sort.Slice(eventsReverseCompletedOrder, func(i, j int) bool {
		return eventsReverseCompletedOrder[i].CompletedTime.After(eventsReverseCompletedOrder[j].CompletedTime)
	})

	for _, event := range eventsReverseCompletedOrder {
		qualifying, qualifyingOK := event.Sessions[SessionTypeQualifying]

		if qualifyingOK && qualifying.Results != nil {
			for pos, driver := range c.ResultsForClass(qualifying.Results.Result) {
				if pos != 0 {
					continue
				}

				givePoints(event, driver.DriverGUID, float64(c.Points.PolePosition))
			}
		}

		race, raceOK := event.Sessions[SessionTypeRace]

		if raceOK && race.Results != nil {
			fastestLap := race.Results.FastestLap()

			for pos, driver := range c.ResultsForClass(race.Results.Result) {
				if driver.TotalTime <= 0 || driver.Disqualified {
					continue
				}

				givePoints(event, driver.DriverGUID, c.PointForPos(pos))

				if fastestLap.DriverGUID == driver.DriverGUID {
					givePoints(event, driver.DriverGUID, float64(c.Points.BestLap))
				}
			}
		}

		race2, race2OK := event.Sessions[SessionTypeSecondRace]

		if race2OK && race2.Results != nil {
			fastestLap := race2.Results.FastestLap()

			for pos, driver := range c.ResultsForClass(race2.Results.Result) {
				if driver.TotalTime <= 0 || driver.Disqualified {
					continue
				}

				givePoints(event, driver.DriverGUID, c.PointForPos(pos)*c.Points.SecondRaceMultiplier)

				if fastestLap.DriverGUID == driver.DriverGUID {
					givePoints(event, driver.DriverGUID, float64(c.Points.BestLap)*c.Points.SecondRaceMultiplier)
				}
			}
		}
	}
}

// Standings returns the current Driver Standings for the Championship.
func (c *ChampionshipClass) Standings(events []*ChampionshipEvent) []*ChampionshipStanding {
	var out []*ChampionshipStanding

	standings := make(map[string]*ChampionshipStanding)

	c.standings(events, func(event *ChampionshipEvent, driverGUID string, points float64) {
		var car *SessionCar

		for _, session := range event.Sessions {
			if session.Results == nil {
				continue
			}

			for _, sessionCar := range session.Results.Cars {
				if sessionCar.Driver.GUID == driverGUID {
					car = sessionCar
					break
				}
			}
		}

		if car == nil {
			return
		}

		if _, ok := standings[driverGUID]; !ok {
			standings[driverGUID] = NewChampionshipStanding(car)
		}

		standings[driverGUID].Points += points
		standings[driverGUID].AddEventForTeam(car.Driver.Team)
	})

	for _, standing := range standings {
		if standing.Car.Driver.Name == "" {
			continue
		}

		standing.Points -= float64(c.PenaltyForGUID(standing.Car.Driver.GUID))

		out = append(out, standing)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Points == out[j].Points {
			return out[i].Car.Driver.Name < out[j].Car.Driver.Name
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

	c.standings(events, func(event *ChampionshipEvent, driverGUID string, points float64) {
		var team string

		// find the team the driver was in for this race.
		for _, session := range event.Sessions {
			if session.Results != nil {
				for _, car := range session.Results.Cars {
					if car.Driver.GUID == driverGUID {
						team = car.Driver.Team
						break
					}
				}
				break
			}
		}

		if _, ok := teams[team]; !ok {
			teams[team] = points
		} else {
			teams[team] += points
		}
	})

	var out []*TeamStanding

	for name, pts := range teams {
		out = append(out, &TeamStanding{
			Team:   name,
			Points: pts - float64(c.PenaltyForTeam(name)),
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
	EntryList EntryList

	Sessions map[SessionType]*ChampionshipSession

	StartedTime   time.Time
	CompletedTime time.Time
	Scheduled     time.Time

	championship *Championship
}

func (cr *ChampionshipEvent) GetSummary() string {
	return fmt.Sprintf("(%s)", cr.championship.Name)
}

func (cr *ChampionshipEvent) GetURL() string {
	return "/championship/" + cr.championship.ID.String()
}

func (cr *ChampionshipEvent) GetEntryList() EntryList {
	return cr.CombineEntryLists(cr.championship)
}

func (cr *ChampionshipEvent) GetID() uuid.UUID {
	return cr.ID
}

func (cr *ChampionshipEvent) GetRaceSetup() CurrentRaceConfig {
	return cr.RaceSetup
}

func (cr *ChampionshipEvent) GetScheduledTime() time.Time {
	return cr.Scheduled
}

func (cr *ChampionshipEvent) Cars(c *Championship) []string {
	if cr.Completed() {
		cars := make(map[string]bool)

		// look for cars in the session results
		for _, session := range cr.Sessions {
			if session.Results != nil {
				for _, car := range session.Results.Cars {
					cars[car.Model] = true
				}
			}
		}

		var out []string

		for car := range cars {
			out = append(out, car)
		}

		return out
	} else {
		return c.ValidCarIDs()
	}
}

func (cr *ChampionshipEvent) CombineEntryLists(championship *Championship) EntryList {
	entryList := championship.AllEntrants()

	if cr.EntryList == nil {
		// no specific entry list for this event, just use the default
		return entryList
	}

	for _, entrant := range entryList {
		for _, eventEntrant := range cr.EntryList {
			if entrant.InternalUUID != uuid.Nil && entrant.InternalUUID == eventEntrant.InternalUUID && entrant.Model == eventEntrant.Model {
				entrant.OverwriteProperties(eventEntrant)

				break
			}
		}
	}

	return entryList
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
	_, opts, err := championshipManager.BuildChampionshipOpts(r)

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

	// sign up responses are hidden for data protection reasons
	championship.SignUpForm.Responses = nil

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(championship)
}

// importChampionshipHandler reads Championship data from JSON.
func importChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		championshipID, err := championshipManager.ImportChampionship(r.FormValue("import"))

		if err != nil {
			logrus.Errorf("couldn't import championship, err: %s", err)
			AddErrFlashQuick(w, r, "Sorry, we couldn't import that championship! Check your JSON formatting.")
		} else {
			AddFlashQuick(w, r, "Championship successfully imported!")
			http.Redirect(w, r, "/championship/"+championshipID, http.StatusFound)
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, "championships/import-championship.html", nil)
}

type championshipResultsCollection struct {
	Name    string                `json:"name"`
	Results []championshipResults `json:"results"`
}

type championshipResults struct {
	Name string   `json:"name"`
	Log  []string `json:"log"`
}

// exportChampionshipResults returns championship result files in JSON format.
func exportChampionshipResultsHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.Errorf("couldn't export championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var results []championshipResults

	for _, event := range championship.Events {

		if !event.Completed() {
			continue
		}

		var sessionFiles []string

		for _, session := range event.Sessions {
			sessionFiles = append(sessionFiles, session.Results.GetURL())
		}

		results = append(results, championshipResults{
			Name: "Event at " + prettifyName(event.RaceSetup.Track, false) + ", completed on " + event.CompletedTime.Format("Monday, January 2, 2006 3:04 PM (MST)"),
			Log:  sessionFiles,
		})
	}

	champResultsCollection := championshipResultsCollection{
		Name:    championship.Name,
		Results: results,
	}

	w.Header().Add("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(champResultsCollection)
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

func championshipEventImportHandler(w http.ResponseWriter, r *http.Request) {
	championshipID := chi.URLParam(r, "championshipID")
	eventID := chi.URLParam(r, "eventID")

	if r.Method == http.MethodPost {
		err := championshipManager.ImportEvent(championshipID, eventID, r)

		if err != nil {
			logrus.Errorf("Could not import championship event, error: %s", err)
			AddErrFlashQuick(w, r, "Could not import session files")
		} else {
			AddFlashQuick(w, r, "Successfully imported session files!")
			http.Redirect(w, r, "/championship/"+championshipID, http.StatusFound)
			return
		}
	}

	event, results, err := championshipManager.ListAvailableResultsFilesForEvent(championshipID, eventID)

	if err != nil {
		logrus.Errorf("Couldn't load session files, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "championships/import-event.html", map[string]interface{}{
		"Results":        results,
		"ChampionshipID": championshipID,
		"Event":          event,
	})
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
	timezone := r.FormValue("event-schedule-timezone")

	location, err := time.LoadLocation(timezone)

	if err != nil {
		logrus.WithError(err).Errorf("could not find location: %s", location)
		location = time.Local
	}

	// Parse time in correct time zone
	date, err := time.ParseInLocation("2006-01-02-15:04", dateString+"-"+timeString, location)

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

	AddFlashQuick(w, r, fmt.Sprintf("We have scheduled the Championship Event to begin at %s", date.Format(time.RFC1123)))
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

func championshipICalHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Add("Content-Disposition", "inline; filename=championship.ics")

	err := championshipManager.BuildICalFeed(chi.URLParam(r, "championshipID"), w)

	if err != nil {
		logrus.WithError(err).Error("could not build scheduled races feed")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func championshipDriverPenaltyHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.ModifyDriverPenalty(
		chi.URLParam(r, "championshipID"),
		chi.URLParam(r, "classID"),
		chi.URLParam(r, "driverGUID"),
		PenaltyAction(r.FormValue("action")),
		formValueAsInt(r.FormValue("PointsPenalty")),
	)

	if err != nil {
		logrus.Errorf("Could not modify championship driver penalty, err: %s", err)

		AddErrFlashQuick(w, r, "Couldn't modify driver penalty")
	} else {
		AddFlashQuick(w, r, "Driver penalty successfully modified")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func championshipTeamPenaltyHandler(w http.ResponseWriter, r *http.Request) {
	err := championshipManager.ModifyTeamPenalty(
		chi.URLParam(r, "championshipID"),
		chi.URLParam(r, "classID"),
		chi.URLParam(r, "team"),
		PenaltyAction(r.FormValue("action")),
		formValueAsInt(r.FormValue("PointsPenalty")),
	)

	if err != nil {
		logrus.Errorf("Could not modify championship penalty, err: %s", err)

		AddErrFlashQuick(w, r, "Couldn't modify team penalty")
	} else {
		AddFlashQuick(w, r, "Team penalty successfully modified")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func championshipSignUpFormHandler(w http.ResponseWriter, r *http.Request) {
	championship, opts, err := championshipManager.BuildChampionshipOpts(r)

	if err != nil {
		logrus.WithError(err).Error("couldn't load championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !championship.SignUpForm.Enabled {
		http.NotFound(w, r)
		return
	}

	opts["FormData"] = &ChampionshipSignUpResponse{}

	if r.Method == http.MethodPost {
		signUpResponse, foundSlot, err := championshipManager.HandleChampionshipSignUp(r)

		if err != nil {
			switch err.(type) {
			case ValidationError:
				opts["FormData"] = signUpResponse
				opts["ValidationError"] = err.Error()
			default:
				logrus.WithError(err).Error("couldn't handle championship")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		} else {
			if championship.SignUpForm.RequiresApproval {
				AddFlashQuick(w, r, "Thanks for registering for the championship! Your registration is pending approval by an administrator.")
				http.Redirect(w, r, "/championship/"+championship.ID.String(), http.StatusFound)
				return
			} else {
				if foundSlot {
					AddFlashQuick(w, r, "Thanks for registering for the championship!")
					http.Redirect(w, r, "/championship/"+championship.ID.String(), http.StatusFound)
					return
				} else {
					opts["FormData"] = signUpResponse
					opts["ValidationError"] = fmt.Sprintf("There are no more available slots for the car: %s. Please pick a different car.", prettifyName(signUpResponse.GetCar(), true))
				}
			}
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, "championships/sign-up.html", opts)
}

func championshipSignedUpEntrantsHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.WithError(err).Error("couldn't load championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !championship.SignUpForm.Enabled {
		http.NotFound(w, r)
		return
	}

	sort.Slice(championship.SignUpForm.Responses, func(i, j int) bool {
		return championship.SignUpForm.Responses[i].Created.After(championship.SignUpForm.Responses[j].Created)
	})

	ViewRenderer.MustLoadTemplate(w, r, "championships/signed-up-entrants.html", map[string]interface{}{
		"Championship": championship,
	})
}

func championshipSignedUpEntrantsCSVHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.WithError(err).Error("couldn't load championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	headers := []string{
		"Created",
		"Name",
		"Team",
		"GUID",
		"Email",
		"Car",
		"Skin",
		"Status",
	}

	for _, question := range championship.SignUpForm.ExtraFields {
		headers = append(headers, question)
	}

	var out [][]string

	out = append(out, headers)

	for _, entrant := range championship.SignUpForm.Responses {
		data := []string{
			entrant.Created.String(),
			entrant.Name,
			entrant.Team,
			entrant.GUID,
			entrant.Email,
			entrant.Car,
			entrant.Skin,
			string(entrant.Status),
		}

		for _, question := range championship.SignUpForm.ExtraFields {
			if response, ok := entrant.Questions[question]; ok {
				data = append(data, response)
			} else {
				data = append(data, "")
			}
		}

		out = append(out, data)
	}

	w.Header().Add("Content-Type", "text/csv")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment;filename=Entrants_%s.csv", championship.Name))
	wr := csv.NewWriter(w)
	wr.UseCRLF = true
	_ = wr.WriteAll(out)
}

func championshipModifyEntrantStatusHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.WithError(err).Error("couldn't load championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !championship.SignUpForm.Enabled {
		http.NotFound(w, r)
		return
	}

	entrantGUID := chi.URLParam(r, "entrantGUID")

	for _, entrant := range championship.SignUpForm.Responses {
		if entrant.GUID != entrantGUID {
			continue
		}

		switch r.URL.Query().Get("action") {
		case "accept":
			if entrant.Status == ChampionshipEntrantAccepted {
				AddFlashQuick(w, r, "This entrant has already been accepted.")
				break
			}

			// add the entrant to the entrylist
			foundSlot, _, err := championship.AddEntrantFromSessionData(entrant)

			if err != nil {
				logrus.WithError(err).Error("couldn't add entrant to championship")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if foundSlot {
				entrant.Status = ChampionshipEntrantAccepted

				AddFlashQuick(w, r, "The entrant was successfully accepted!")
			} else {
				AddErrFlashQuick(w, r, "There are no more slots available for the given entrant and car. Please check the Championship configuration.")
			}
		case "reject":
			entrant.Status = ChampionshipEntrantRejected

		classLoop:
			for _, class := range championship.Classes {
				for _, classEntrant := range class.Entrants {
					if entrantGUID == classEntrant.GUID {
						// remove the entrant from the championship.
						classEntrant.Name = ""
						classEntrant.GUID = ""
						classEntrant.Team = ""
						break classLoop
					}
				}
			}
		default:
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	if err := championshipManager.UpsertChampionship(championship); err != nil {
		logrus.WithError(err).Error("couldn't save championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
