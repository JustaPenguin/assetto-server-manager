package servermanager

import (
	"fmt"
	"html/template"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RaceWeekends are a collection of sessions, where one session influences the EntryList of the next.
type RaceWeekend struct {
	ID      uuid.UUID
	Created time.Time
	Updated time.Time
	Deleted time.Time

	// Filters is a map of Parent ID -> Child ID -> Filter
	Filters map[string]map[string]*RaceWeekendSessionToSessionFilter

	Name string

	EntryList EntryList
	Sessions  []*RaceWeekendSession
}

// NewRaceWeekend creates a RaceWeekend
func NewRaceWeekend() *RaceWeekend {
	return &RaceWeekend{
		ID:      uuid.New(),
		Created: time.Now(),
	}
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
		if parentSession.Completed() && parentSession.Results != nil {
			filter = &RaceWeekendSessionToSessionFilter{
				ResultStart:          1,
				ResultEnd:            len(parentSession.Results.Cars),
				NumEntrantsToReverse: 0,
				EntryListStart:       1,
			}
		} else {
			filter = &RaceWeekendSessionToSessionFilter{
				ResultStart:          1,
				ResultEnd:            len(rw.EntryList),
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

	rw.Sessions = append(rw.Sessions[:toDelete], rw.Sessions[toDelete+1:]...)

	for _, session := range rw.Sessions {
		session.RemoveParent(sessionID)
	}
}

// SessionCanBeRun determines whether a RaceWeekendSession has all of its parent dependencies met to be allowed to run
// (i.e. all parent RaceWeekendSessions must be complete to allow it to run)
func (rw *RaceWeekend) SessionCanBeRun(s *RaceWeekendSession) bool {
	if s.IsBase() {
		return true
	}

	for _, parentID := range s.ParentIDs {
		parent, err := rw.FindSessionByID(parentID.String())

		if err == RaceWeekendSessionNotFound {
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
	for _, entrant := range rw.EntryList {
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

var (
	ErrRaceWeekendNotFound     = errors.New("servermanager: race weekend not found")
	RaceWeekendSessionNotFound = errors.New("servermanager: race weekend session not found")
)

// FindSessionByID finds a RaceWeekendSession by its unique identifier
func (rw *RaceWeekend) FindSessionByID(id string) (*RaceWeekendSession, error) {
	for _, sess := range rw.Sessions {
		if sess.ID.String() == id {
			return sess, nil
		}
	}

	return nil, RaceWeekendSessionNotFound
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

	return e
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

	RaceConfig CurrentRaceConfig

	StartedTime   time.Time
	CompletedTime time.Time
	Results       *SessionResults
}

// NewRaceWeekendSession creates an empty RaceWeekendSession
func NewRaceWeekendSession() *RaceWeekendSession {
	return &RaceWeekendSession{
		ID:      uuid.New(),
		Created: time.Now(),
	}
}

// Name of the RaceWeekendSession
func (rws *RaceWeekendSession) Name() string {
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
func (rws *RaceWeekendSession) FinishingGrid(raceWeekend *RaceWeekend) []*RaceWeekendSessionEntrant {
	var out []*RaceWeekendSessionEntrant

	entryList, err := rws.GetRaceWeekendEntryList(raceWeekend, nil, "")

	if err != nil {
		panic(err)
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

			if !foundEntrant && entrant.Car.GetGUID() != "" {
				out = append(out, NewRaceWeekendSessionEntrant(rws.ID, entrant.Car, entrant.EntrantResult, rws.Results))
			}
		}
	} else {
		// if a session is not completed, we work on the assumption that the finishing grid is equal to the entrylist
		out = entryList.Sorted()
	}

	return out
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
func (rws *RaceWeekendSession) ParentsDataAttr() template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf("data-parent-ids='%s'", jsonEncode(rws.ParentIDs)))
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
	return rws.ParentIDs == nil || len(rws.ParentIDs) == 0
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
	if rws.IsBase() {
		// base race weekend sessions just return the race weekend EntryList
		return EntryListToRaceWeekendEntryList(rw.EntryList, rws.ID), nil
	}

	entryList := make(RaceWeekendEntryList, 0)

	for _, parentSessionID := range rws.ParentIDs {
		parentSession, err := rw.FindSessionByID(parentSessionID.String())

		if err != nil {
			return nil, err
		}

		if overrideFilter != nil && parentSessionID.String() == overrideFilterSessionID {
			// override filters are provided when users are modifying filters for their race weekend setups
			err = overrideFilter.Filter(parentSession, rws, parentSession.FinishingGrid(rw), &entryList)

			if err != nil {
				return nil, err
			}
		} else {
			sessionToSessionFilter, err := rw.GetFilterOrUseDefault(parentSessionID.String(), rws.ID.String())

			if err != nil {
				return nil, err
			}

			err = sessionToSessionFilter.Filter(parentSession, rws, parentSession.FinishingGrid(rw), &entryList)

			if err != nil {
				return nil, err
			}
		}
	}

	if rw.SessionCanBeRun(rws) {
		// sorting can only be run if a session is ready to be run.
		sorter := GetRaceWeekendEntryListSort(rws.SortType)

		if err := sorter(rws, entryList); err != nil {
			return nil, err
		}

		reverseEntrants(rws.NumEntrantsToReverse, entryList)

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

	for _, entrant := range e {
		rwe := NewRaceWeekendSessionEntrant(sessionID, entrant.AsSessionCar(), entrant.AsSessionResult(), nil)

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
}

func (a ActiveRaceWeekend) IsChampionship() bool {
	return false // @TODO
}

func (a ActiveRaceWeekend) IsRaceWeekend() bool {
	return true
}

func (a ActiveRaceWeekend) OverrideServerPassword() bool {
	return a.OverridePassword
}

func (a ActiveRaceWeekend) ReplacementServerPassword() string {
	return a.ReplacementPassword
}

func (a ActiveRaceWeekend) EventName() string {
	return "Race Weekend: " + a.Name
}

func (a ActiveRaceWeekend) EventDescription() string {
	return a.Description
}

func (a ActiveRaceWeekend) GetURL() string {
	if config.HTTP.BaseURL != "" {
		return config.HTTP.BaseURL + "/race-weekend/" + a.RaceWeekendID.String()
	} else {
		return ""
	}
}
