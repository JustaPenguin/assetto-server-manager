package servermanager

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/teambition/rrule-go"
)

func init() {
	gob.Register(&RaceWeekend{})
}

// RaceWeekends are a collection of sessions, where one session influences the EntryList of the next.
type RaceWeekend struct {
	ID      uuid.UUID
	Name    string
	Created time.Time
	Updated time.Time
	Deleted time.Time

	// Filters is a map of Parent ID -> Child ID -> Filter
	Filters map[string]map[string]*RaceWeekendSessionToSessionFilter

	// Deprecated: use GetEntryList() instead
	EntryList EntryList
	Sessions  []*RaceWeekendSession

	// ChampionshipID links a RaceWeekend to a Championship. It can be uuid.Nil
	ChampionshipID uuid.UUID
	// Championship is the Championship that is linked to the RaceWeekend.
	// If ChampionshipID is uuid.Nil, Championship will also be nil
	Championship *Championship `json:"-"`
}

// NewRaceWeekend creates a RaceWeekend
func NewRaceWeekend() *RaceWeekend {
	return &RaceWeekend{
		ID:      uuid.New(),
		Created: time.Now(),
	}
}

func (rw *RaceWeekend) Duplicate() (*RaceWeekend, error) {
	buf := new(bytes.Buffer)

	var newRaceWeekend RaceWeekend

	if err := gob.NewEncoder(buf).Encode(rw); err != nil {
		return nil, err
	}

	if err := gob.NewDecoder(buf).Decode(&newRaceWeekend); err != nil {
		return nil, err
	}

	return &newRaceWeekend, nil
}

func (rw *RaceWeekend) HasLinkedChampionship() bool {
	return rw.ChampionshipID != uuid.Nil
}

func (rw *RaceWeekend) InProgress() bool {
	for _, session := range rw.Sessions {
		if session.InProgress() {
			return true
		}
	}

	return false
}

func (rw *RaceWeekend) GetEntryList() EntryList {
	if rw.HasLinkedChampionship() {
		entryList := make(EntryList)

		count := 0

		// filter out drivers with no GUID (open championships etc...)
		for _, entrant := range rw.Championship.AllEntrants().AlphaSlice() {
			if entrant.GUID == "" || entrant.IsPlaceHolder {
				entrant.GUID = uuid.New().String()
				entrant.Name = fmt.Sprintf("Placeholder Entrant %d", count)
				entrant.IsPlaceHolder = true
			}

			entryList.AddInPitBox(entrant, count)
			count++
		}

		return entryList
	}

	return rw.EntryList
}

func (rw *RaceWeekend) Completed() bool {
	for _, session := range rw.Sessions {
		if !session.Completed() {
			return false
		}
	}

	return true
}

// AddFilter creates a link between parentID and childID with a filter that specifies how to take the EntrantResult of parent
// and modify them to make an EntryList for child
func (rw *RaceWeekend) AddFilter(parentID, childID string, filter *RaceWeekendSessionToSessionFilter) {
	if rw.Filters == nil {
		rw.Filters = make(map[string]map[string]*RaceWeekendSessionToSessionFilter)
	}

	if _, ok := rw.Filters[parentID]; !ok {
		rw.Filters[parentID] = make(map[string]*RaceWeekendSessionToSessionFilter)
	}

	rw.Filters[parentID][childID] = filter
}

// RemoveFilter removes the link between parent and child
func (rw *RaceWeekend) RemoveFilter(parentID, childID string) {
	delete(rw.Filters[parentID], childID)
}

var ErrRaceWeekendFilterNotFound = errors.New("servermanager: race weekend filter not found")

// GetFilter returns the Filter between parentID and childID, erroring if no Filter is found.
func (rw *RaceWeekend) GetFilter(parentID, childID string) (*RaceWeekendSessionToSessionFilter, error) {
	if parentFilters, ok := rw.Filters[parentID]; ok {
		if childFilter, ok := parentFilters[childID]; ok {
			return childFilter, nil
		}
	}

	return nil, ErrRaceWeekendFilterNotFound
}

// GetFilterOrUseDefault attempts to find a filter between parentID and childID. If a filter is not found, a 'default' filter
// is created. The default filter takes all entrants from the parent results and applies them directly to the child entrylist,
// in their finishing order.
func (rw *RaceWeekend) GetFilterOrUseDefault(parentID, childID string) (*RaceWeekendSessionToSessionFilter, error) {
	filter, err := rw.GetFilter(parentID, childID)

	if err == ErrRaceWeekendFilterNotFound {
		parentSession, err := rw.FindSessionByID(parentID)

		if err != nil {
			return nil, err
		}

		// filter not found, load defaults
		if parentSession.Completed() && parentSession.Results != nil && !parentSession.IsBase() {
			filter = &RaceWeekendSessionToSessionFilter{
				ResultStart:          1,
				ResultEnd:            len(parentSession.Results.Cars),
				NumEntrantsToReverse: 0,
				EntryListStart:       1,
			}
		} else {
			filter = &RaceWeekendSessionToSessionFilter{
				ResultStart:          1,
				ResultEnd:            len(rw.GetEntryList()),
				NumEntrantsToReverse: 0,
				EntryListStart:       1,
			}
		}
	} else if err != nil {
		return nil, err
	}

	return filter, nil
}

// AddSession adds a RaceWeekendSession to a RaceWeekend, with an optional parent session (can be nil).
func (rw *RaceWeekend) AddSession(s *RaceWeekendSession, parent *RaceWeekendSession) {
	if parent != nil {
		s.ParentIDs = append(s.ParentIDs, parent.ID)
	}

	rw.Sessions = append(rw.Sessions, s)
}

// DelSession removes a RaceWeekendSession from a RaceWeekend. This also removes any parent links from the
// removed session to any other sessions.
func (rw *RaceWeekend) DelSession(sessionID string) {
	toDelete := -1

	for sessionIndex, sess := range rw.Sessions {
		if sess.ID.String() == sessionID {
			toDelete = sessionIndex
		}
	}

	if toDelete < 0 {
		return
	}

	sessionToDelete := rw.Sessions[toDelete]

	// loop through all parents of the deleted session. if we can, put them as parents of the children of this session
	for _, parentID := range sessionToDelete.ParentIDs {
		parent, err := rw.FindSessionByID(parentID.String())

		if err != nil {
			logrus.WithError(err).Warnf("Could not find parentID: %s", parentID.String())
			continue
		}

		for _, child := range rw.FindChildren(sessionToDelete.ID.String()) {
			if !rw.HasParentRecursive(parent, child.ID.String()) {
				child.ParentIDs = append(child.ParentIDs, parentID)
			}
		}
	}

	rw.Sessions = append(rw.Sessions[:toDelete], rw.Sessions[toDelete+1:]...)

	for _, session := range rw.Sessions {
		session.RemoveParent(sessionID)
	}
}

func (rw *RaceWeekend) FindChildren(parentID string) []*RaceWeekendSession {
	var children []*RaceWeekendSession

	for _, session := range rw.Sessions {
		if session.HasParent(parentID) {
			children = append(children, session)
		}
	}

	return children
}

// SessionCanBeRun determines whether a RaceWeekendSession has all of its parent dependencies met to be allowed to run
// (i.e. all parent RaceWeekendSessions must be complete to allow it to run)
func (rw *RaceWeekend) SessionCanBeRun(s *RaceWeekendSession) bool {
	if s.IsBase() {
		return true
	}

	for _, parentID := range s.ParentIDs {
		parent, err := rw.FindSessionByID(parentID.String())

		if err == ErrRaceWeekendSessionNotFound {
			logrus.Warnf("Race weekend session for id: %s not found", parentID.String())
			continue
		} else if err != nil {
			logrus.WithError(err).Errorf("an unknown error occurred while checking session dependencies")
			return false
		}

		if !rw.SessionCanBeRun(parent) || !parent.Completed() {
			return false
		}
	}

	return true
}

// SortedSessions returns the RaceWeekendSessions in order by the number of parents the sessions have.
func (rw *RaceWeekend) SortedSessions() []*RaceWeekendSession {
	sessions := make([]*RaceWeekendSession, len(rw.Sessions))

	copy(sessions, rw.Sessions)

	sort.Slice(sessions, func(i, j int) bool {
		iChildren := rw.FindTotalNumParents(sessions[i])
		jChildren := rw.FindTotalNumParents(sessions[j])

		return iChildren < jChildren
	})

	return sessions
}

// Progress indicates how far (0 -> 1) a RaceWeekend has progressed.
func (rw *RaceWeekend) Progress() float64 {
	numSessions := float64(len(rw.Sessions))

	if numSessions == 0 {
		return 0
	}

	numCompletedSessions := float64(0)

	for _, session := range rw.Sessions {
		if session.Completed() {
			numCompletedSessions++
		}
	}

	return (numCompletedSessions / numSessions) * 100
}

// FindTotalNumParents recursively finds all parents for a given session, including their parents etc...
func (rw *RaceWeekend) FindTotalNumParents(session *RaceWeekendSession) int {
	if len(session.ParentIDs) == 0 {
		return 0
	}

	out := len(session.ParentIDs)

	for _, otherSessID := range session.ParentIDs {
		sess, err := rw.FindSessionByID(otherSessID.String())

		if err != nil {
			continue
		}

		out += rw.FindTotalNumParents(sess)
	}

	return out
}

// HasTeamNames indicates whether a RaceWeekend entrylist has team names in it
func (rw *RaceWeekend) HasTeamNames() bool {
	for _, entrant := range rw.GetEntryList() {
		if entrant.Team != "" {
			return true
		}
	}

	return false
}

// HasParentRecursive looks for otherSessionID in session's parents, grandparents, etc...
func (rw *RaceWeekend) HasParentRecursive(session *RaceWeekendSession, otherSessionID string) bool {
	if session.HasParent(otherSessionID) {
		return true
	} else if len(session.ParentIDs) == 0 {
		return false
	}

	for _, parentID := range session.ParentIDs {
		sess, err := rw.FindSessionByID(parentID.String())

		if err != nil {
			continue
		}

		if rw.HasParentRecursive(sess, otherSessionID) {
			return true
		}
	}

	return false
}

func (rw *RaceWeekend) TrackOverview() string {
	trackInfoMap := make(map[string]bool)

	for _, session := range rw.Sessions {
		trackInfo := trackInfo(session.RaceConfig.Track, session.RaceConfig.TrackLayout)

		var trackDescription string

		if trackInfo != nil {
			trackDescription = trackInfo.Name

			if trackInfo.Country != "" {
				trackDescription += " - " + trackInfo.Country
			}
		} else {
			trackDescription = prettifyName(session.RaceConfig.Track, false)

			if session.RaceConfig.TrackLayout != "" {
				trackDescription = " (" + prettifyName(session.RaceConfig.TrackLayout, true) + ")"
			}
		}

		trackInfoMap[trackDescription] = true
	}

	var trackInfo []string

	for info := range trackInfoMap {
		trackInfo = append(trackInfo, info)
	}

	return strings.Join(trackInfo, ", ")
}

var (
	ErrRaceWeekendNotFound        = errors.New("servermanager: race weekend not found")
	ErrRaceWeekendSessionNotFound = errors.New("servermanager: race weekend session not found")
)

// FindSessionByID finds a RaceWeekendSession by its unique identifier
func (rw *RaceWeekend) FindSessionByID(id string) (*RaceWeekendSession, error) {
	for _, sess := range rw.Sessions {
		if sess.ID.String() == id {
			return sess, nil
		}
	}

	if id == rw.ID.String() {
		// this is likely a request to find the parent session of a 'base' session. return a dummy session
		sess := NewRaceWeekendSession()
		sess.ID = rw.ID
		sess.isBase = true
		sess.CompletedTime = time.Now()
		sess.Results = &SessionResults{}

		return sess, nil
	}

	return nil, ErrRaceWeekendSessionNotFound
}

// EnhanceResults takes a set of SessionResults and attaches Championship information to them.
func (rw *RaceWeekend) EnhanceResults(results *SessionResults) {
	if results == nil {
		return
	}

	results.RaceWeekendID = rw.ID.String()

	if rw.HasLinkedChampionship() {
		// linked championships determine class IDs etc for drivers.
		rw.Championship.EnhanceResults(results)
	}
}

// A RaceWeekendSessionEntrant is someone who has entered at least one RaceWeekend event.
type RaceWeekendSessionEntrant struct {
	// SessionID is the last session the Entrant participated in
	SessionID uuid.UUID
	// EntrantResult is the result of the last session the Entrant participated in
	EntrantResult *SessionResult
	// Car is the car from the EntryList that matches the results of the last session the Entrant participated in
	Car *SessionCar
	// PitBox is used to determine the starting pitbox of the Entrant.
	PitBox int
	// SessionResults are the whole results for the session the entrant took part in
	SessionResults *SessionResults `json:"-"`

	IsPlaceholder bool `json:"-"`

	// OverrideSetupFile is a path to an overridden setup for a Race Weekend
	OverrideSetupFile string
}

// NewRaceWeekendSessionEntrant creates a RaceWeekendSessionEntrant
func NewRaceWeekendSessionEntrant(previousSessionID uuid.UUID, car *SessionCar, entrantResult *SessionResult, sessionResults *SessionResults) *RaceWeekendSessionEntrant {
	return &RaceWeekendSessionEntrant{
		SessionID:      previousSessionID,
		EntrantResult:  entrantResult,
		Car:            car,
		SessionResults: sessionResults,
	}
}

// GetEntrant returns the RaceWeekendSessionEntrant as an EntryList Entrant (used for building the final entry_list.ini)
func (se *RaceWeekendSessionEntrant) GetEntrant() *Entrant {
	e := NewEntrant()

	e.AssignFromResult(se.EntrantResult, se.Car)
	e.IsPlaceHolder = se.IsPlaceholder
	e.FixedSetup = se.OverrideSetupFile

	return e
}

func (se *RaceWeekendSessionEntrant) ChampionshipClass(raceWeekend *RaceWeekend) *ChampionshipClass {
	if !raceWeekend.HasLinkedChampionship() {
		return &ChampionshipClass{}
	}

	class, err := raceWeekend.Championship.FindClassForCarModel(se.Car.Model)

	if err != nil {
		logrus.WithError(err).Warnf("Could not find class for car: %s", se.Car.Model)
		return &ChampionshipClass{}
	}

	return class
}

// A RaceWeekendSession is a single session within a RaceWeekend. It can have parent sessions. It must have a RaceConfig.
// Once completed, a RaceWeekendSession will contain EntrantResult from that session.
type RaceWeekendSession struct {
	ID      uuid.UUID
	Created time.Time
	Updated time.Time
	Deleted time.Time

	ParentIDs            []uuid.UUID
	SortType             string
	NumEntrantsToReverse int

	RaceConfig          CurrentRaceConfig
	OverridePassword    bool
	ReplacementPassword string

	StartedTime                time.Time
	CompletedTime              time.Time
	ScheduledTime              time.Time
	ScheduledServerID          ServerID
	Results                    *SessionResults
	StartWhenParentHasFinished bool

	Points map[uuid.UUID]*ChampionshipPoints

	isBase bool

	// raceWeekend is here for use when satisfying the ScheduledEvent interface.
	raceWeekend *RaceWeekend
}

func (rws *RaceWeekendSession) GetID() uuid.UUID {
	return rws.ID
}

func (rws *RaceWeekendSession) GetRaceSetup() CurrentRaceConfig {
	return rws.RaceConfig
}

func (rws *RaceWeekendSession) GetScheduledTime() time.Time {
	return rws.ScheduledTime
}

func (rws *RaceWeekendSession) GetSummary() string {
	return "(Race Weekend)"
}

func (rws *RaceWeekendSession) GetURL() string {
	return "/race-weekend/" + rws.raceWeekend.ID.String()
}

func (rws *RaceWeekendSession) HasSignUpForm() bool {
	return false
}

func (rws *RaceWeekendSession) ReadOnlyEntryList() EntryList {
	rwe, err := rws.GetRaceWeekendEntryList(rws.raceWeekend, nil, "")

	if err != nil {
		logrus.WithError(err).Error("Could not get race weekend entry list")
		return make(EntryList)
	}

	return rwe.AsEntryList()
}

func (rws *RaceWeekendSession) HasRecurrenceRule() bool {
	return false
}

func (rws *RaceWeekendSession) GetRecurrenceRule() (*rrule.RRule, error) {
	return nil, nil
}

func (rws *RaceWeekendSession) SetRecurrenceRule(input string) error {
	return nil // no-op
}

func (rws *RaceWeekendSession) ClearRecurrenceRule() {
	// no-op
}

// NewRaceWeekendSession creates an empty RaceWeekendSession
func NewRaceWeekendSession() *RaceWeekendSession {
	return &RaceWeekendSession{
		ID:      uuid.New(),
		Created: time.Now(),
		Points:  make(map[uuid.UUID]*ChampionshipPoints),
	}
}

// Name of the RaceWeekendSession
func (rws *RaceWeekendSession) Name() string {
	if rws.isBase {
		return "Entry List"
	}

	return rws.SessionInfo().Name
}

// SessionInfo returns the information about the Assetto Corsa Session (i.e. practice, qualifying, race)
func (rws *RaceWeekendSession) SessionInfo() *SessionConfig {
	for _, sess := range rws.RaceConfig.Sessions {
		return sess
	}

	return &SessionConfig{}
}

// SessionType returns the type of the RaceWeekendSession (practice, qualifying, race)
func (rws *RaceWeekendSession) SessionType() SessionType {
	for sessType := range rws.RaceConfig.Sessions {
		return sessType
	}

	return ""
}

// FinishingGrid returns the finishing grid of the session, if complete. Otherwise, it returns the EntryList of that session
func (rws *RaceWeekendSession) FinishingGrid(raceWeekend *RaceWeekend) ([]*RaceWeekendSessionEntrant, error) {
	var out []*RaceWeekendSessionEntrant

	entryList, err := rws.GetRaceWeekendEntryList(raceWeekend, nil, "")

	if err != nil {
		return nil, err
	}

	if rws.Completed() {
		for _, result := range rws.Results.Result {
			if result.DriverGUID == "" || result.Disqualified {
				// filter out invalid results
				continue
			}

			car, err := rws.Results.FindCarByGUIDAndModel(result.DriverGUID, result.CarModel)

			if err != nil {
				continue
			}

			out = append(out, NewRaceWeekendSessionEntrant(rws.ID, car, result, rws.Results))
		}

		// look for entrants not in our session results who started the session.
		for _, entrant := range entryList {
			foundEntrant := false

			for _, driver := range out {
				if driver.Car.GetGUID() == entrant.Car.GetGUID() {
					foundEntrant = true
					break
				}
			}

			if !foundEntrant && entrant.Car.GetGUID() != "" && (rws.IsBase() || !entrant.IsPlaceholder) {
				if raceWeekend.HasLinkedChampionship() {
					// find the class ID for the car
					class, err := raceWeekend.Championship.FindClassForCarModel(entrant.Car.GetCar())

					if err != nil {
						logrus.WithError(err).Warnf("Could not find class for car model: %s for entrant %s", entrant.Car.GetCar(), entrant.Car.GetGUID())
					} else {
						entrant.EntrantResult.ClassID = class.ID
					}
				}

				e := NewRaceWeekendSessionEntrant(rws.ID, entrant.Car, entrant.EntrantResult, rws.Results)
				e.IsPlaceholder = entrant.IsPlaceholder

				out = append(out, e)
			}
		}
	} else {
		// if a session is not completed, we work on the assumption that the finishing grid is equal to the entrylist
		out = entryList.Sorted()
	}

	return out, nil
}

// RemoveParent removes a parent RaceWeekendSession from this session
func (rws *RaceWeekendSession) RemoveParent(parentID string) {
	foundIndex := -1

	for index, id := range rws.ParentIDs {
		if id.String() == parentID {
			foundIndex = index
		}
	}

	if foundIndex < 0 {
		return
	}

	rws.ParentIDs = append(rws.ParentIDs[:foundIndex], rws.ParentIDs[foundIndex+1:]...)
}

// ParentsDataAttr returns a html-safe data attribute for identifying parent sessions of a given RaceWeekendSession in the frontend
func (rws *RaceWeekendSession) ParentsDataAttr(raceWeekend *RaceWeekend) template.HTMLAttr {
	parentIDs := rws.ParentIDs

	if len(parentIDs) == 0 {
		// sessions with no parent are shown as having the race weekend as their parent.
		parentIDs = []uuid.UUID{raceWeekend.ID}
	}

	return template.HTMLAttr(fmt.Sprintf("data-parent-ids='%s'", jsonEncode(parentIDs)))
}

// InProgress indicates whether a RaceWeekendSession has been started but not stopped
func (rws *RaceWeekendSession) InProgress() bool {
	return !rws.StartedTime.IsZero() && rws.CompletedTime.IsZero()
}

// Completed RaceWeekendSessions have a non-zero CompletedTime
func (rws *RaceWeekendSession) Completed() bool {
	return !rws.CompletedTime.IsZero() && rws.Results != nil
}

// IsBase indicates that a RaceWeekendSession has no parent
func (rws *RaceWeekendSession) IsBase() bool {
	return rws.isBase
}

// HasParent determines if a RaceWeekendSession has a parent with id
func (rws *RaceWeekendSession) HasParent(id string) bool {
	for _, parentID := range rws.ParentIDs {
		if parentID.String() == id {
			return true
		}
	}

	return false
}

var ErrRaceWeekendSessionDependencyIncomplete = errors.New("servermanager: race weekend session dependency incomplete")

// GetRaceWeekendEntryList returns the RaceWeekendEntryList for the given session, built from the parent session(s) results and applied filters.
func (rws *RaceWeekendSession) GetRaceWeekendEntryList(rw *RaceWeekend, overrideFilter *RaceWeekendSessionToSessionFilter, overrideFilterSessionID string) (RaceWeekendEntryList, error) {
	var entryList RaceWeekendEntryList

	if rws.IsBase() {
		entryList = EntryListToRaceWeekendEntryList(rw.GetEntryList(), rws.ID)

		if overrideFilter == nil {
			var err error

			overrideFilter, err = rw.GetFilterOrUseDefault(rw.ID.String(), rws.ID.String())

			if err != nil {
				return nil, err
			}
		}

		err := overrideFilter.Filter(rw, rws, rws, entryList, &entryList)

		if err != nil {
			return nil, err
		}
	} else {
		entryList = make(RaceWeekendEntryList, 0)

		for _, parentSessionID := range rws.ParentIDs {
			parentSession, err := rw.FindSessionByID(parentSessionID.String())

			if err != nil {
				return nil, err
			}

			finishingGrid, err := parentSession.FinishingGrid(rw)

			if err != nil {
				return nil, err
			}

			if overrideFilter != nil && parentSessionID.String() == overrideFilterSessionID {
				// override filters are provided when users are modifying filters for their race weekend setups
				err = overrideFilter.Filter(rw, parentSession, rws, finishingGrid, &entryList)

				if err != nil {
					return nil, err
				}
			} else {
				sessionToSessionFilter, err := rw.GetFilterOrUseDefault(parentSessionID.String(), rws.ID.String())

				if err != nil {
					return nil, err
				}

				err = sessionToSessionFilter.Filter(rw, parentSession, rws, finishingGrid, &entryList)

				if err != nil {
					return nil, err
				}
			}
		}
	}

	if rw.SessionCanBeRun(rws) {
		// sorting can only be run if a session is ready to be run.
		sorter := GetRaceWeekendEntryListSort(rws.SortType)

		if err := sorter.Sort(rw, rws, entryList, nil); err != nil {
			return nil, err
		}

		// amend pitboxes post-sort
		for i, entrant := range entryList {
			entrant.PitBox = i
		}
	}

	return entryList, nil
}

// EntryListToRaceWeekendEntryList converts an EntryList to a RaceWeekendEntryList for a given RaceWeekendSession
func EntryListToRaceWeekendEntryList(e EntryList, sessionID uuid.UUID) RaceWeekendEntryList {
	out := make(RaceWeekendEntryList, 0, len(e))

	for _, entrant := range e.AsSlice() {
		rwe := NewRaceWeekendSessionEntrant(sessionID, entrant.AsSessionCar(), entrant.AsSessionResult(), nil)
		rwe.IsPlaceholder = entrant.IsPlaceHolder

		out.AddInPitBox(rwe, entrant.PitBox)
	}

	return out
}

// A RaceWeekendEntryList is a collection of RaceWeekendSessionEntrants
type RaceWeekendEntryList []*RaceWeekendSessionEntrant

// Add an Entrant to the EntryList
func (e *RaceWeekendEntryList) Add(entrant *RaceWeekendSessionEntrant) {
	e.AddInPitBox(entrant, len(*e))
}

// AddInPitBox adds an Entrant in a specific pitbox - overwriting any entrant that was in that pitbox previously.
func (e *RaceWeekendEntryList) AddInPitBox(entrant *RaceWeekendSessionEntrant, pitBox int) {
	entrant.PitBox = pitBox
	*e = append(*e, entrant)
}

// Remove an Entrant from the EntryList
func (e *RaceWeekendEntryList) Delete(entrant *RaceWeekendSessionEntrant) {
	toDelete := -1

	for k, v := range *e {
		if v == entrant {
			toDelete = k
			break
		}
	}

	if toDelete >= 0 {
		*e = append((*e)[:toDelete], (*e)[toDelete+1:]...)
	}
}

// Sorted returns the RaceWeekendEntryList ordered by Entrant PitBoxes
func (e *RaceWeekendEntryList) Sorted() []*RaceWeekendSessionEntrant {
	var entrants []*RaceWeekendSessionEntrant

	for _, x := range *e {
		entrants = append(entrants, x)
	}

	sort.Slice(entrants, func(i, j int) bool {
		return entrants[i].PitBox < entrants[j].PitBox
	})

	return entrants
}

// AsEntryList returns a RaceWeekendEntryList as an EntryList
func (e RaceWeekendEntryList) AsEntryList() EntryList {
	entryList := make(EntryList)

	for _, entrant := range e {
		x := entrant.GetEntrant()

		if entrant.OverrideSetupFile != "" {
			x.FixedSetup = entrant.OverrideSetupFile
		}

		entryList.AddInPitBox(entrant.GetEntrant(), entrant.PitBox)
	}

	return entryList
}

// ActiveRaceWeekend indicates which RaceWeekend and RaceWeekendSession are currently running on the server.
type ActiveRaceWeekend struct {
	Name                     string
	RaceWeekendID, SessionID uuid.UUID
	OverridePassword         bool
	ReplacementPassword      string
	Description              string
	IsPracticeSession        bool
	RaceConfig               CurrentRaceConfig
	EntryList                EntryList
}

func (a ActiveRaceWeekend) GetRaceConfig() CurrentRaceConfig {
	return a.RaceConfig
}

func (a ActiveRaceWeekend) GetEntryList() EntryList {
	return a.EntryList
}

func (a ActiveRaceWeekend) IsLooping() bool {
	return false
}

func (a ActiveRaceWeekend) IsChampionship() bool {
	return false
}

func (a ActiveRaceWeekend) IsRaceWeekend() bool {
	return true
}

func (a ActiveRaceWeekend) IsPractice() bool {
	return a.IsPracticeSession
}

func (a ActiveRaceWeekend) OverrideServerPassword() bool {
	return a.OverridePassword
}

func (a ActiveRaceWeekend) ReplacementServerPassword() string {
	return a.ReplacementPassword
}

func (a ActiveRaceWeekend) EventName() string {
	name := "Race Weekend: " + a.Name

	if a.IsPracticeSession {
		name += " - Practice Session"
	}

	return name
}

func (a ActiveRaceWeekend) EventDescription() string {
	return a.Description
}

func (a ActiveRaceWeekend) GetURL() string {
	if config.HTTP.BaseURL != "" {
		return config.HTTP.BaseURL + "/race-weekend/" + a.RaceWeekendID.String()
	}

	return ""
}

func (a ActiveRaceWeekend) GetForceStopTime() time.Duration {
	return 0
}

func (a ActiveRaceWeekend) GetForceStopWithDrivers() bool {
	return false
}
