package servermanager

import (
	"errors"
	"fmt"
	"github.com/teambition/rrule-go"
	"html/template"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

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

func (c *Championship) HasTeamNames() bool {
	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			if entrant.Team != "" {
				return true
			}
		}
	}

	return false
}

func (c *Championship) HasScheduledEvents() bool {
	for _, event := range c.Events {
		if !event.Scheduled.IsZero() {
			return true
		}
	}

	return false
}

func (c *Championship) ClearEntrant(entrantGUID string) {
	for _, class := range c.Classes {
		for _, classEntrant := range class.Entrants {
			if entrantGUID == classEntrant.GUID {
				// remove the entrant from the championship.
				classEntrant.Name = ""
				classEntrant.GUID = ""
				classEntrant.Team = ""
				return
			}
		}
	}
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

func (c *Championship) AddEntrantFromSession(potentialEntrant PotentialChampionshipEntrant) (foundFreeEntrantSlot bool, entrantClass *ChampionshipClass, err error) {
	c.entryListMutex.Lock()
	defer c.entryListMutex.Unlock()

	classForCar, err := c.FindClassForCarModel(potentialEntrant.GetCar())

	if err != nil {
		logrus.Errorf("Could not find class for car: %s in championship", potentialEntrant.GetCar())

		return false, nil, err
	}

	var oldEntrant *Entrant

	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			if entrant.GUID == potentialEntrant.GetGUID() {
				// we found the entrant, but there's a possibility that they changed cars
				// keep the old entrant so we can swap its properties with the one that is being written to
				oldEntrant = entrant

				entrant.GUID = ""
				entrant.Name = ""
			}
		}
	}

	// now look for empty Entrants in the Entrylist
	for carNum, entrant := range classForCar.Entrants {
		if entrant.Name == "" && entrant.GUID == "" && entrant.Model == potentialEntrant.GetCar() {
			if oldEntrant != nil {
				// swap the old entrant properties
				oldEntrant.SwapProperties(entrant)
			}

			entrant.Name = potentialEntrant.GetName()
			entrant.GUID = potentialEntrant.GetGUID()
			entrant.Model = potentialEntrant.GetCar()
			entrant.Skin = potentialEntrant.GetSkin()

			// #386: don't replace a team with no team.
			if potentialEntrant.GetTeam() != "" {
				entrant.Team = potentialEntrant.GetTeam()
			}

			logrus.Infof("Championship entrant: %s (%s) has been assigned to %s in %s", entrant.Name, entrant.GUID, carNum, classForCar.Name)

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

func (c *ChampionshipClass) StandingsForEvent(event *ChampionshipEvent) []*ChampionshipStanding {
	return c.Standings([]*ChampionshipEvent{event})
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

func (cr *ChampionshipEvent) HasSignUpForm() bool {
	return cr.championship.SignUpForm.Enabled
}

func (cr *ChampionshipEvent) ReadOnlyEntryList() EntryList {
	return cr.CombineEntryLists(cr.championship)
}

func (cr *ChampionshipEvent) SetRecurrenceRule(input string) error {
	return nil
}

func (cr *ChampionshipEvent) GetRecurrenceRule() (*rrule.RRule, error) {
	return nil, nil
}

func (cr *ChampionshipEvent) HasRecurrenceRule() bool {
	return false
}

func (cr *ChampionshipEvent) ClearRecurrenceRule() {
	return
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

type ActiveChampionship struct {
	Name                    string
	ChampionshipID, EventID uuid.UUID
	SessionType             SessionType
	OverridePassword        bool
	ReplacementPassword     string
	Description             string
	IsPracticeSession       bool

	loadedEntrants map[udp.CarID]udp.SessionCarInfo

	NumLapsCompleted   int
	NumRaceStartEvents int
}

func (a *ActiveChampionship) GetURL() string {
	if config.HTTP.BaseURL != "" {
		return config.HTTP.BaseURL + "/championship/" + a.ChampionshipID.String()
	} else {
		return ""
	}
}

func (a *ActiveChampionship) IsChampionship() bool {
	return !a.IsPracticeSession
}

func (a *ActiveChampionship) OverrideServerPassword() bool {
	return a.OverridePassword
}

func (a *ActiveChampionship) ReplacementServerPassword() string {
	return a.ReplacementPassword
}

func (a *ActiveChampionship) EventName() string {
	if a.IsPracticeSession {
		return a.Name + " - Looping Practice"
	}

	return a.Name
}

func (a *ActiveChampionship) EventDescription() string {
	return a.Description
}
