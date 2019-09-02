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

	Name string

	EntryList EntryList
	Sessions  []*RaceWeekendSession
}

func NewRaceWeekend() *RaceWeekend {
	return &RaceWeekend{
		ID:      uuid.New(),
		Created: time.Now(),
	}
}

func (rw *RaceWeekend) AddSession(s *RaceWeekendSession, parent *RaceWeekendSession) {
	if parent != nil {
		s.ParentIDs = append(s.ParentIDs, parent.ID)
	}

	rw.Sessions = append(rw.Sessions, s)
}

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

func (rw *RaceWeekend) NumParentsLeft(session *RaceWeekendSession) int {
	return len(session.ParentIDs) - rw.NumParentsAbove(session)
}

func (rw *RaceWeekend) NumParentsAbove(session *RaceWeekendSession) int {
	numParentsAbove := 0

	// a parent is above a session if the parent and this session share no similar children
	sessionChildren := rw.FindChildren(session)

	for _, otherSessionID := range session.ParentIDs {
		otherSession, err := rw.FindSessionByID(otherSessionID.String())

		if err != nil {
			continue
		}

		otherSessionChildren := rw.FindChildren(otherSession)

		childrenInCommon := 0

		for _, child := range sessionChildren {
			for _, otherChild := range otherSessionChildren {
				if child.ID == otherChild.ID {
					childrenInCommon++
				}
			}
		}

		if childrenInCommon == 0 {
			numParentsAbove++
		}
	}

	return numParentsAbove
}

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

func (rw *RaceWeekend) FindChildren(session *RaceWeekendSession) []*RaceWeekendSession {
	var children []*RaceWeekendSession

	for _, otherSession := range rw.Sessions {
		if otherSession.HasParent(session.ID.String()) {
			children = append(children, otherSession)
		}
	}

	return children
}

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

func (rw *RaceWeekend) FindSessionByID(id string) (*RaceWeekendSession, error) {
	for _, sess := range rw.Sessions {
		if sess.ID.String() == id {
			return sess, nil
		}
	}

	return nil, RaceWeekendSessionNotFound
}

func (rw *RaceWeekend) GetNumSiblings(session *RaceWeekendSession) int {
	numSiblings := 0

	for _, sess := range rw.Sessions {
		for _, parentID := range sess.ParentIDs {

			if parentID == session.ID && len(sess.ParentIDs) > 1 {
				numSiblings += len(sess.ParentIDs)
				break
			}
		}
	}

	return numSiblings
}

type RaceWeekendSession struct {
	ID      uuid.UUID
	Created time.Time
	Updated time.Time
	Deleted time.Time

	ParentIDs []uuid.UUID

	Filters []EntryListFilter

	RaceConfig CurrentRaceConfig

	StartedTime   time.Time
	CompletedTime time.Time
	Results       *SessionResults
}

func NewRaceWeekendSession() *RaceWeekendSession {
	return &RaceWeekendSession{
		ID:      uuid.New(),
		Created: time.Now(),
	}
}

func (rws *RaceWeekendSession) Name() string {
	return rws.SessionInfo().Name
}

func (rws *RaceWeekendSession) SessionInfo() *SessionConfig {
	for _, sess := range rws.RaceConfig.Sessions {
		return sess
	}

	return &SessionConfig{}
}

func (rws *RaceWeekendSession) SessionType() SessionType {
	for sessType := range rws.RaceConfig.Sessions {
		return sessType
	}

	return ""
}

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

func (rws *RaceWeekendSession) ParentsDataAttr() template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf("data-parent-ids='%s'", jsonEncode(rws.ParentIDs)))
}

// InProgress indicates whether a RaceWeekendSession has been started but not stopped
func (rws *RaceWeekendSession) InProgress() bool {
	return !rws.StartedTime.IsZero() && rws.CompletedTime.IsZero()
}

// Completed RaceWeekendSessions have a non-zero CompletedTime
func (rws *RaceWeekendSession) Completed() bool {
	return !rws.CompletedTime.IsZero()
}

func (rws *RaceWeekendSession) IsBase() bool {
	return rws.ParentIDs == nil
}

func (rws *RaceWeekendSession) HasParent(id string) bool {
	for _, parentID := range rws.ParentIDs {
		if parentID.String() == id {
			return true
		}
	}

	return false
}

var ErrRaceWeekendSessionDependencyIncomplete = errors.New("servermanager: race weekend session dependency incomplete")

func (rws *RaceWeekendSession) GetEntryList(rw *RaceWeekend) (EntryList, error) {
	if rws.IsBase() {
		// base race weekend sessions just return the race weekend EntryList
		return rw.EntryList, nil
	}

	entryList := make(EntryList)

	for _, inheritedID := range rws.ParentIDs {
		// find previous event
		previousEvent, err := rw.FindSessionByID(inheritedID.String())

		if err != nil {
			continue
		}

		if previousEvent.Results == nil {
			return nil, ErrRaceWeekendSessionDependencyIncomplete
		}

		results := previousEvent.Results.Result

		for _, filter := range rws.Filters {
			results, err = filter.Filter(results)

			if err != nil {
				return nil, err
			}
		}

		for pos, result := range results {
			e := NewEntrant()

			car, err := previousEvent.Results.FindCarByGUID(result.DriverGUID)

			if err != nil {
				return nil, err
			}

			e.AssignFromResult(result, car)
			e.PitBox = pos

			entryList.Add(e)
		}
	}

	// @TODO what do we do if there are duplicate drivers in the entrylist?

	return entryList, nil
}
