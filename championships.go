package servermanager

import (
	"errors"
	"fmt"
	"html/template"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"

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

	CollisionWithDriver: 0,
	CollisionWithEnv:    0,
	CutTrack:            0,
}

// ChampionshipPoints represent the potential points for positions as well as other awards in a Championship.
type ChampionshipPoints struct {
	Places       []int
	BestLap      int
	PolePosition int

	CollisionWithDriver int
	CollisionWithEnv    int
	CutTrack            int

	SecondRaceMultiplier float64
}

// PointForPos uses the Championship's Points to determine what number should be awarded to a given position
func (pts *ChampionshipPoints) ForPos(i int) float64 {
	if i >= len(pts.Places) {
		return 0
	}

	return float64(pts.Places[i])
}

// NewChampionship creates a Championship with a given name, creating a UUID for the championship as well.
func NewChampionship(name string) *Championship {
	return &Championship{
		ID:                  uuid.New(),
		Name:                name,
		Created:             time.Now(),
		OpenEntrants:        false,
		PersistOpenEntrants: true,
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

	// acsr integration - sends the championship to acsr on save and event complete
	ACSR bool

	// Raw html can be attached to championships, used to share tracks/cars etc.
	Info template.HTML

	// URL to a specific OG Image for the championship
	OGImage string

	// OpenEntrants indicates that entrant names do not need to be specified in the EntryList.
	// As Entrants join a championship, the available Entrant slots will be filled by the information
	// provided by a join message. The EntryList for each class will still need creating, but
	// can omit names/GUIDs/teams as necessary. These can then be edited after the fact.
	OpenEntrants bool

	// PersistOpenEntrants (used with OpenEntrants) indicates that drivers who join the Championship
	// should be added to the Championship EntryList. This is ON by default.
	PersistOpenEntrants bool

	// SignUpForm gives anyone on the web access to a Championship Sign Up Form so that they can
	// mark themselves for participation in this Championship.
	SignUpForm ChampionshipSignUpForm

	Classes []*ChampionshipClass
	Events  []*ChampionshipEvent
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

func (c *Championship) FindLastResultForDriver(guid string) (out *SessionResult, teamName string) {
	events := make([]*ChampionshipEvent, 0)

	for _, event := range c.Events {
		if event.Completed() {
			events = append(events, event)
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].CompletedTime.Before(events[j].CompletedTime)
	})

	for _, event := range events {
		startedTime := time.Time{}

		for _, session := range event.Sessions {
			if session.StartedTime.After(startedTime) && session.Results != nil {
				startedTime = session.StartedTime

				for _, result := range session.Results.Result {
					if result.DriverGUID == guid {
						out = result
						break
					}
				}

				teamName = session.Results.GetTeamName(guid)
			}
		}
	}

	return out, teamName
}

func (c *Championship) GetPlayerSummary(guid string) string {
	if c.Progress() == 0 {
		if len(c.Events) <= 1 {
			return ""
		}

		return "This is the first event of the Championship!"
	}

	result, teamName := c.FindLastResultForDriver(guid)

	if result == nil {
		return ""
	}

	class, err := c.ClassByID(result.ClassID.String())

	if err != nil {
		logrus.WithError(err).Warnf("Could not find class by id: %s", result.ClassID.String())
		return ""
	}

	standings := class.Standings(c.Events)
	teamStandings := class.TeamStandings(c.Events)

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

	for pos, standing := range teamStandings {
		if standing.Team == teamName {
			teamPos = pos + 1
			teamPoints = standing.Points
		}
	}

	classText := ""

	if class.Name != "" {
		classText = fmt.Sprintf("in the class '%s' ", class.Name)
	}

	out := fmt.Sprintf("You are currently %d%s %swith %.2f points. ", driverPos, ordinal(int64(driverPos)), classText, driverPoints)

	if driverAhead != "" {
		out += fmt.Sprintf("The driver ahead of you is %s with %.2f points. ", driverAhead, driverAheadPoints)
	}

	if teamName != "" {
		var teamAhead string
		var teamAheadPoints float64

		if teamPos >= 2 {
			teamAhead = teamStandings[teamPos-2].Team
			teamAheadPoints = teamStandings[teamPos-2].Points
		}

		out += fmt.Sprintf("Your team '%s' has %.2f points. ", teamName, teamPoints)

		if teamAhead != "" {
			out += fmt.Sprintf("The team ahead is '%s' with %.2f points. ", teamAhead, teamAheadPoints)
		}
	}

	return out
}

func (c *Championship) GetURL() string {
	if config.HTTP.BaseURL != "" {
		return config.HTTP.BaseURL + "/championship/" + c.ID.String()
	}

	return ""
}

// IsMultiClass is true if the Championship has more than one Class
func (c *Championship) IsMultiClass() bool {
	return len(c.Classes) > 1
}

func (c *Championship) SignUpAvailable() bool {
	numTotalEntrants := 0
	numFilledSlots := 0

	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			if entrant.GUID != "" {
				numFilledSlots++
			}

			numTotalEntrants++
		}
	}

	return c.SignUpForm.Enabled && c.Progress() < 100.0 && numFilledSlots < numTotalEntrants
}

func (c *Championship) HasTeamNames() bool {
	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			if entrant.Team != "" {
				return true
			}
		}
	}
	for _, event := range c.Events {
		for _, session := range event.Sessions {
			if session.Completed() && session.Results != nil {
				for _, car := range session.Results.Cars {
					if car.Driver.Team != "" {
						return true
					}
				}
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

	if model == AnyCarModel {
		// randomly assign a class based from whatever classes we have with AnyCarModel in their entrylist.
		classes := make([]*ChampionshipClass, len(c.Classes))

		copy(classes, c.Classes)

		rand.Shuffle(len(classes), func(i, j int) {
			classes[i], classes[j] = classes[j], classes[i]
		})

		for _, class := range classes {
			for _, entrant := range class.Entrants {
				if entrant.Model == AnyCarModel {
					return class, nil
				}
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

	Entrants      EntryList
	Points        ChampionshipPoints
	AvailableCars []string

	DriverPenalties, TeamPenalties map[string]int
}

// ValidCarIDs returns a set of all cars chosen within the given class
func (c *ChampionshipClass) ValidCarIDs() []string {
	if len(c.AvailableCars) == 0 {
		cars := make(map[string]bool)

		for _, e := range c.Entrants {
			cars[e.Model] = true
		}

		var availableCars []string

		for car := range cars {
			availableCars = append(availableCars, car)
		}

		return availableCars
	}

	return c.AvailableCars
}

func (c *Championship) ImportEvent(eventToImport interface{}) (*ChampionshipEvent, error) {
	var newEvent *ChampionshipEvent

	switch event := eventToImport.(type) {
	case *CustomRace:
		newEvent = &ChampionshipEvent{
			ID:        uuid.New(),
			RaceSetup: event.RaceConfig,
			EntryList: c.AllEntrants(),
		}
	case *RaceWeekend:
		// the filter between Entry List and any events uses the race weekend ID in the map
		// as we are updating this ID we need to update it in the map too or filters between
		// the Entry List and any event will fail!
		oldID := event.ID.String()
		event.ID = uuid.New()

		if event.Filters != nil {
			oldFilter := event.Filters[oldID]
			delete(event.Filters, oldID)
			event.Filters[event.ID.String()] = oldFilter
		}

		newEvent = &ChampionshipEvent{
			ID:            uuid.New(),
			EntryList:     c.AllEntrants(),
			RaceWeekendID: event.ID,
			RaceWeekend:   event,
		}

		event.Championship = c
		event.ChampionshipID = c.ID

		// reset session progress
		for _, session := range event.Sessions {
			session.CompletedTime = time.Time{}
			session.StartedTime = time.Time{}
			session.Results = nil
			session.ScheduledTime = time.Time{}

			// if a session has the Entry List as a parent we need to replace the old
			// Race Weekend ID with the new one
			for i, parentID := range session.ParentIDs {
				if parentID.String() == oldID {
					session.ParentIDs[i] = event.ID
				}
			}

			for _, class := range c.Classes {
				if session.SessionType() == SessionTypeRace {
					session.Points[class.ID] = &class.Points
				} else {
					var points []int

					for range class.Points.Places {
						points = append(points, 0)
					}

					session.Points[class.ID] = &ChampionshipPoints{
						Places:               points,
						BestLap:              0,
						PolePosition:         0,
						CollisionWithDriver:  0,
						CollisionWithEnv:     0,
						CutTrack:             0,
						SecondRaceMultiplier: 0,
					}
				}
			}
		}

	case *ChampionshipEvent:
		newEvent = &ChampionshipEvent{
			ID:        uuid.New(),
			RaceSetup: event.RaceSetup,
			EntryList: event.EntryList,
		}
	default:
		return nil, errors.New("servermanager: unknown event type")
	}

	c.Events = append(c.Events, newEvent)

	return newEvent, nil
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
		for _, model := range class.ValidCarIDs() {
			cars[model] = true
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

	if c.SignUpForm.Enabled && !c.OpenEntrants {
		// closed sign up form championships report only the taken slots as their num entrants
		for _, class := range c.Classes {
			for _, entrant := range class.Entrants {
				if entrant.GUID != "" && !entrant.IsPlaceHolder {
					entrants++
				}
			}
		}
	} else {
		for _, class := range c.Classes {
			entrants += len(class.Entrants)
		}
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

func (c *Championship) EntrantAttendance(guid string) int {
	i := 0

	for _, event := range ExtractRaceWeekendSessionsIntoIndividualEvents(c.Events) {
		if event.Completed() {
			for _, class := range c.Classes {
				standings := class.StandingsForEvent(event)

				for _, standing := range standings {
					if standing.Car.GetGUID() == guid {
						i++
					}
				}
			}
		}
	}

	return i
}

func (c *Championship) NumCompletedEvents() int {
	i := 0

	for _, event := range ExtractRaceWeekendSessionsIntoIndividualEvents(c.Events) {
		if event.Completed() {
			i++
		}
	}

	return i
}

// AddClass to the championship
func (c *Championship) AddClass(class *ChampionshipClass) {
	c.Classes = append(c.Classes, class)
}

// Progress of the Championship as a percentage
func (c *Championship) Progress() float64 {
	numEvents := float64(len(c.Events))

	for _, event := range c.Events {
		if event.IsRaceWeekend() && event.RaceWeekend != nil {
			numEvents += float64(len(event.RaceWeekend.Sessions)) - 1
		}
	}

	if numEvents == 0 {
		return 0
	}

	numCompletedEvents := float64(0)

	for _, event := range c.Events {

		if event.Completed() {
			numCompletedEvents++
		}

		if event.IsRaceWeekend() && event.RaceWeekend != nil {
			for _, session := range event.RaceWeekend.Sessions {
				if session.Completed() {
					numCompletedEvents++
				}
			}
		}
	}

	return (numCompletedEvents / numEvents) * 100
}

func (c *Championship) NumPendingSignUps() int {
	num := 0

	for _, response := range c.SignUpForm.Responses {
		if response.Status == ChampionshipEntrantPending {
			num++
		}
	}

	return num
}

type PotentialChampionshipEntrant interface {
	GetName() string
	GetTeam() string
	GetCar() string
	GetSkin() string
	GetGUID() string
}

var ErrEntryListFull = errors.New("servermanager: entry list is full")

var entryListMutex = sync.Mutex{}

func (c *Championship) AddEntrantInFirstFreeSlot(potentialEntrant PotentialChampionshipEntrant) (foundFreeEntrantSlot bool, entrant *Entrant, entrantClass *ChampionshipClass, err error) {
	entryListMutex.Lock()
	defer entryListMutex.Unlock()

	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			if entrant.GUID == potentialEntrant.GetGUID() {
				// update the entrant details
				entrant.Name = potentialEntrant.GetName()
				entrant.Team = potentialEntrant.GetTeam()

				return true, entrant, class, nil
			}
		}
	}

	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			if entrant.Name == "" && entrant.GUID == "" {
				// take the slot
				entrant.Name = potentialEntrant.GetName()
				entrant.GUID = potentialEntrant.GetGUID()
				entrant.Team = potentialEntrant.GetTeam()

				return true, entrant, class, nil
			}
		}
	}

	return false, nil, nil, ErrEntryListFull
}

func (c *Championship) AddEntrantFromSession(potentialEntrant PotentialChampionshipEntrant) (foundFreeEntrantSlot bool, entrant *Entrant, entrantClass *ChampionshipClass, err error) {
	entryListMutex.Lock()
	defer entryListMutex.Unlock()

	classForCar, err := c.FindClassForCarModel(potentialEntrant.GetCar())

	if err != nil {
		logrus.Errorf("Could not find class for car: %s in championship", potentialEntrant.GetCar())

		return false, nil, nil, err
	}

	var oldEntrant *Entrant
	var oldEntrantClass *ChampionshipClass

	for _, class := range c.Classes {
		for _, entrant := range class.Entrants {
			if entrant.GUID == potentialEntrant.GetGUID() {
				// we found the entrant, but there's a possibility that they changed cars
				// keep the old entrant so we can swap its properties with the one that is being written to
				oldEntrant = entrant
				oldEntrantClass = class

				entrant.GUID = ""
				entrant.Name = ""
			}
		}
	}

	// now look for empty Entrants in the Entrylist with a matching car
	for carNum, entrant := range classForCar.Entrants {
		if entrant.Name == "" && entrant.GUID == "" && entrant.Model == potentialEntrant.GetCar() {
			if oldEntrant != nil {
				// swap the old entrant properties
				oldEntrant.SwapProperties(entrant, oldEntrantClass == classForCar)
			}

			classForCar.AssignToFreeEntrantSlot(entrant, potentialEntrant)
			logrus.Infof("Championship entrant: %s (%s) has been assigned to %s in %s (matching car)", entrant.Name, entrant.GUID, carNum, c.Name)

			return true, entrant, classForCar, nil
		}
	}

	// now look for empty Entrants in the Entrylist with an 'any free car' slot
	for carNum, entrant := range classForCar.Entrants {
		if entrant.Name == "" && entrant.GUID == "" && entrant.Model == AnyCarModel {
			if oldEntrant != nil {
				// swap the old entrant properties
				oldEntrant.SwapProperties(entrant, oldEntrantClass == classForCar)
			}

			classForCar.AssignToFreeEntrantSlot(entrant, potentialEntrant)
			logrus.Infof("Championship entrant: %s (%s) has been assigned an to %s in %s (any car slot)", entrant.Name, entrant.GUID, carNum, c.Name)

			return true, entrant, classForCar, nil
		}
	}

	return false, nil, classForCar, nil
}

// EnhanceResults takes a set of SessionResults and attaches Championship information to them.
func (c *Championship) EnhanceResults(results *SessionResults) {
	if results == nil {
		return
	}

	results.ChampionshipID = c.ID.String()

	c.AttachClassIDToResults(results)

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

func (c *ChampionshipClass) DriverInClass(result *SessionResult) bool {
	return result.ClassID == c.ID
}

func (c *ChampionshipClass) AssignToFreeEntrantSlot(entrant *Entrant, potentialEntrant PotentialChampionshipEntrant) {
	entrant.Name = potentialEntrant.GetName()
	entrant.GUID = potentialEntrant.GetGUID()
	entrant.Model = potentialEntrant.GetCar()
	entrant.Skin = potentialEntrant.GetSkin()

	// #386: don't replace a team with no team.
	if potentialEntrant.GetTeam() != "" {
		entrant.Team = potentialEntrant.GetTeam()
	}
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

func (c *Championship) AttachClassIDToResults(results *SessionResults) {
	for _, lap := range results.Laps {
		class, err := c.FindClassForCarModel(lap.CarModel)

		if err != nil {
			logrus.Warnf("Couldn't find class for car model: %s (entrant: %s/%s)", lap.CarModel, lap.DriverName, lap.DriverGUID)
			continue
		}

		lap.ClassID = class.ID
	}

	for _, result := range results.Result {
		class, err := c.FindClassForCarModel(result.CarModel)

		if err != nil {
			logrus.Warnf("Couldn't find class for car model: %s (entrant: %s/%s)", result.CarModel, result.DriverName, result.DriverGUID)
			continue
		}

		result.ClassID = class.ID
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
	}

	return 0
}

func (c *ChampionshipClass) PenaltyForTeam(name string) int {
	if teamPenalty, ok := c.TeamPenalties[name]; ok {
		return teamPenalty
	}

	return 0
}

type PointsReason int

const (
	PointsEventFinish PointsReason = iota
	PointsPolePosition
	PointsFastestLap
	PointsCollisionWithCar
	PointsCollisionWithEnvironment
	PointsCutTrack
)

func (c *ChampionshipClass) standings(events []*ChampionshipEvent, givePoints func(event *ChampionshipEvent, driverGUID string, points float64, reason PointsReason)) {
	eventsReverseCompletedOrder := make([]*ChampionshipEvent, len(events))

	copy(eventsReverseCompletedOrder, events)

	sort.Slice(eventsReverseCompletedOrder, func(i, j int) bool {
		return eventsReverseCompletedOrder[i].CompletedTime.After(eventsReverseCompletedOrder[j].CompletedTime)
	})

	for _, event := range eventsReverseCompletedOrder {
		for sessionType, session := range event.Sessions {
			if !session.Completed() || session.Results == nil {
				continue
			}

			points := c.Points
			pointsMultiplier := 1.0

			if session.IsRaceWeekend() {
				// race weekend sessions are valid points, as specified by the session itself.
				classPoints, ok := session.RaceWeekendSession.Points[c.ID]

				if !ok {
					logrus.Warnf("Could not find points for Race Weekend Session class: %s", c.ID)
				} else {
					points = *classPoints
				}
			} else {
				switch sessionType {
				case SessionTypeQualifying:
					// non race weekend qualifying results get pole position points
					for pos, driver := range c.ResultsForClass(session.Results.Result) {
						if pos != 0 {
							continue
						}

						givePoints(event, driver.DriverGUID, float64(points.PolePosition)*pointsMultiplier, PointsPolePosition)
					}

					continue

				case SessionTypeSecondRace:
					pointsMultiplier = points.SecondRaceMultiplier
				case SessionTypeBooking, SessionTypePractice:
					continue

				default:
					// race sessions fall through
				}
			}

			fastestLap := session.Results.FastestLapInClass(c.ID)

			for pos, driver := range c.ResultsForClass(session.Results.Result) {
				if driver.TotalTime <= 0 || driver.Disqualified {
					continue
				}

				givePoints(event, driver.DriverGUID, points.ForPos(pos)*pointsMultiplier, PointsEventFinish)

				if fastestLap.DriverGUID == driver.DriverGUID {
					givePoints(event, driver.DriverGUID, float64(points.BestLap)*pointsMultiplier, PointsFastestLap)
				}

				if sessionType == SessionTypeRace || sessionType == SessionTypeSecondRace {
					givePoints(event, driver.DriverGUID, float64(points.CollisionWithDriver*session.Results.GetCrashesOfType(driver.DriverGUID, "COLLISION_WITH_CAR"))*pointsMultiplier*-1, PointsCollisionWithCar)
					givePoints(event, driver.DriverGUID, float64(points.CollisionWithEnv*session.Results.GetCrashesOfType(driver.DriverGUID, "COLLISION_WITH_ENV"))*pointsMultiplier*-1, PointsCollisionWithEnvironment)
					givePoints(event, driver.DriverGUID, float64(points.CutTrack*session.Results.GetCuts(driver.DriverGUID, driver.CarModel))*pointsMultiplier*-1, PointsCutTrack)
				}
			}
		}
	}
}

var championshipStandingSessionOrder = []SessionType{
	SessionTypeSecondRace,
	SessionTypeRace,
	SessionTypeQualifying,
	SessionTypePractice,
	SessionTypeBooking,
}

// Standings returns the current Driver Standings for the Championship.
func (c *ChampionshipClass) Standings(inEvents []*ChampionshipEvent) []*ChampionshipStanding {
	var out []*ChampionshipStanding

	// make a copy of events so we do not persist race weekend sessions
	events := ExtractRaceWeekendSessionsIntoIndividualEvents(inEvents)

	for _, event := range events {
		for _, session := range event.Sessions {
			if session.Results == nil {
				continue
			}

			// sort session result cars by the last car they completed a lap in (reversed).
			// this means that below when we're looking for the car that matches a driver, we find
			// the most recent car that they drove.

			// sorting occurs outside the standings call below for performance reasons.
			sort.Slice(session.Results.Cars, func(i, j int) bool {
				carI := session.Results.Cars[i]
				carJ := session.Results.Cars[j]

				carILastLap := 0
				carJLastLap := 0

				for _, lap := range session.Results.Laps {
					if lap.CarID == carI.CarID && lap.Timestamp > carILastLap {
						carILastLap = lap.Timestamp
					}

					if lap.CarID == carJ.CarID && lap.Timestamp > carJLastLap {
						carJLastLap = lap.Timestamp
					}
				}

				return carILastLap > carJLastLap
			})
		}
	}

	standings := make(map[string]*ChampionshipStanding)

	c.standings(events, func(event *ChampionshipEvent, driverGUID string, points float64, reason PointsReason) {
		var car *SessionCar

		for _, sessionType := range championshipStandingSessionOrder {
			session, ok := event.Sessions[sessionType]

			if !ok || session.Results == nil {
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

		if reason == PointsEventFinish {
			// only increment team finishes for a 'finish' reason
			standings[driverGUID].AddEventForTeam(car.Driver.Team)
		}
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

// extractRaceWeekendSessionsIntoIndividualEvents looks for race weekend events, and makes each indiivdual session of that
// race weekend a Championship Event, to aide with points tallying
func ExtractRaceWeekendSessionsIntoIndividualEvents(inEvents []*ChampionshipEvent) []*ChampionshipEvent {
	events := make([]*ChampionshipEvent, 0)

	for _, event := range inEvents {
		if !event.IsRaceWeekend() {
			events = append(events, event)
		} else if event.RaceWeekend != nil {
			for _, session := range event.RaceWeekend.Sessions {
				e := NewChampionshipEvent()

				e.ID = session.ID
				e.RaceSetup = session.RaceConfig
				e.CompletedTime = session.CompletedTime
				e.StartedTime = session.StartedTime
				e.Scheduled = session.ScheduledTime

				e.Sessions[session.SessionType()] = &ChampionshipSession{
					StartedTime:        session.StartedTime,
					CompletedTime:      session.CompletedTime,
					Results:            session.Results,
					RaceWeekendSession: session,
				}

				events = append(events, e)
			}
		}
	}

	return events
}

// TeamStanding is the current number of Points a Team has.
type TeamStanding struct {
	Team   string
	Points float64
}

// TeamStandings returns the current position of Teams in the Championship.
func (c *ChampionshipClass) TeamStandings(inEvents []*ChampionshipEvent) []*TeamStanding {
	teams := make(map[string]float64)

	// make a copy of events so we do not persist race weekend sessions
	events := ExtractRaceWeekendSessionsIntoIndividualEvents(inEvents)

	c.standings(events, func(event *ChampionshipEvent, driverGUID string, points float64, reason PointsReason) {
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
		ID:       uuid.New(),
		Sessions: make(map[SessionType]*ChampionshipSession),
	}
}

// copied an existing ChampionshipEvent but assigns a new ID
func DuplicateChampionshipEvent(event *ChampionshipEvent) *ChampionshipEvent {
	newEvent := *event

	newEvent.ID = uuid.New()
	newEvent.CompletedTime = time.Time{}

	return &newEvent
}

// A ChampionshipEvent is a given RaceSetup with Sessions.
type ChampionshipEvent struct {
	ScheduledEventBase

	ID uuid.UUID

	RaceSetup CurrentRaceConfig
	EntryList EntryList

	Sessions map[SessionType]*ChampionshipSession `json:",omitempty"`

	// RaceWeekendID is the ID of the linked RaceWeekend for this ChampionshipEvent
	RaceWeekendID uuid.UUID
	// If RaceWeekendID is non-nil, RaceWeekend will be populated on loading the Championship.
	RaceWeekend *RaceWeekend

	StartedTime   time.Time
	CompletedTime time.Time

	championship *Championship
}

func (cr *ChampionshipEvent) IsRaceWeekend() bool {
	return cr.RaceWeekendID != uuid.Nil
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

func (cr *ChampionshipEvent) GetID() uuid.UUID {
	return cr.ID
}

func (cr *ChampionshipEvent) GetRaceSetup() CurrentRaceConfig {
	return cr.RaceSetup
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
	}

	return c.ValidCarIDs()
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
	switch {
	case cr.RaceSetup.HasMultipleRaces():
		// there must be two races as reversed grid positions are on
		return SessionTypeSecondRace
	case cr.RaceSetup.HasSession(SessionTypeRace):
		return SessionTypeRace
	case cr.RaceSetup.HasSession(SessionTypeQualifying):
		return SessionTypeQualifying
	case cr.RaceSetup.HasSession(SessionTypePractice):
		return SessionTypePractice
	default:
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

	RaceWeekendSession *RaceWeekendSession
}

func (ce *ChampionshipSession) IsRaceWeekend() bool {
	return ce.RaceWeekendSession != nil
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
	RaceConfig              CurrentRaceConfig
	EntryList               EntryList

	loadedEntrants map[udp.CarID]udp.SessionCarInfo

	NumLapsCompleted   int `json:"-"`
	NumRaceStartEvents int `json:"-"`
}

func (a *ActiveChampionship) GetRaceConfig() CurrentRaceConfig {
	return a.RaceConfig
}

func (a *ActiveChampionship) GetEntryList() EntryList {
	return a.EntryList
}

func (a *ActiveChampionship) IsLooping() bool {
	return false
}

func (a *ActiveChampionship) IsPractice() bool {
	return a.IsPracticeSession
}

func (a *ActiveChampionship) GetURL() string {
	if config.HTTP.BaseURL != "" {
		return config.HTTP.BaseURL + "/championship/" + a.ChampionshipID.String()
	}

	return ""
}

func (a *ActiveChampionship) IsChampionship() bool {
	return true
}

func (a *ActiveChampionship) IsRaceWeekend() bool {
	return false
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

func (a *ActiveChampionship) GetForceStopTime() time.Duration {
	return 0
}

func (a *ActiveChampionship) GetForceStopWithDrivers() bool {
	return false
}

func ChampionshipClassColor(i int) string {
	return ChampionshipClassColors[i%len(ChampionshipClassColors)]
}
