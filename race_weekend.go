package servermanager

import (
	"html/template"
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

func (rw *RaceWeekend) FindChildren(session *RaceWeekendSession) []*RaceWeekendSession {
	var children []*RaceWeekendSession

	for _, otherSession := range rw.Sessions {
		if otherSession.HasParent(session.ID.String()) {
			children = append(children, otherSession)
		}
	}

	return children
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

func (rw *RaceWeekend) GetColumnWidth(session *RaceWeekendSession) int {
	numSiblings := rw.GetNumSiblings(session)

	return int(12.0 / float64(numSiblings))
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
+

func NewRaceWeekendSession() *RaceWeekendSession {
	return &RaceWeekendSession{
		ID:      uuid.New(),
		Created: time.Now(),
	}
}

func (rws *RaceWeekendSession) SessionInfo() SessionConfig {
	for _, sess := range rws.RaceConfig.Sessions {
		return sess
	}

	return SessionConfig{}
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
	var entryList EntryList

	if rws.IsBase() {
		entryList = rw.EntryList
	} else {
		entryList = make(EntryList)

		for _, inheritedID := range rws.ParentIDs {
			// find previous event
			previousEvent, err := rw.FindSessionByID(inheritedID.String())

			if err != nil {
				return nil, err
			}

			if previousEvent.Results == nil {
				return nil, ErrRaceWeekendSessionDependencyIncomplete
			}

			for pos, result := range previousEvent.Results.Result {
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
	}

	for _, filter := range rws.Filters {
		err := filter.Filter(entryList)

		if err != nil {
			return nil, errors.Wrapf(err, "could not apply filter: %s", filter.Name())
		}
	}

	return entryList, nil
}

// An EntryListFilter takes a given EntryList, and (based on some criteria) filters out invalid Entrants
type EntryListFilter interface {
	Name() string
	Filter(e EntryList) error
	Render() *template.HTML
}
