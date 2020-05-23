package servermanager

import (
	"errors"
	"time"

	"github.com/sirupsen/logrus"
)

type Scheduler struct {
	raceManager         *RaceManager
	championshipManager *ChampionshipManager
	raceWeekendManager  *RaceWeekendManager
	notificationManager *NotificationManager
	store               Store

	timers         map[string]*time.Timer
	reminderTimers map[string]*time.Timer
}

func NewScheduler(store Store, raceManager *RaceManager, championshipManager *ChampionshipManager, raceWeekendManager *RaceWeekendManager, notificationManager *NotificationManager) *Scheduler {
	return &Scheduler{
		store:               store,
		notificationManager: notificationManager,
		raceManager:         raceManager,
		championshipManager: championshipManager,
		raceWeekendManager:  raceWeekendManager,

		timers:         make(map[string]*time.Timer),
		reminderTimers: make(map[string]*time.Timer),
	}
}

var ErrInvalidScheduleTime = errors.New("servermanager: invalid schedule time")

func (s *Scheduler) Init() error {
	// load all custom races, championships, race weekends. if scheduled is after now, then schedule it
	customRaces, err := s.store.ListCustomRaces()

	if err != nil {
		return err
	}

	for _, customRace := range customRaces {
		s.scheduleExistingEvent(customRace)
	}

	championships, err := s.store.ListChampionships()

	if err != nil {
		return err
	}

	for _, championship := range championships {
		for _, event := range championship.Events {
			s.scheduleExistingEvent(event)
		}
	}

	raceWeekends, err := s.store.ListRaceWeekends()

	if err != nil {
		return err
	}

	for _, raceWeekend := range raceWeekends {
		for _, session := range raceWeekend.Sessions {
			s.scheduleExistingEvent(session)
		}
	}

	return nil
}

func (s *Scheduler) scheduleExistingEvent(event ScheduledEvent) {
	if event.GetScheduledTime().IsZero() {
		// this event has no schedule
		return
	}

	if event.GetScheduledTime().After(time.Now()) {
		err := s.Schedule(event, event.GetScheduledTime())

		if err != nil {
			logrus.WithError(err).Error("Could not schedule event (%s)", event.EventName())
		}
	} else if event.HasRecurrenceRule() {
		if !event.GetScheduledTime().IsZero() {
			logrus.Infof("Looks like the server was offline whilst a recurring scheduled event was meant to start!"+
				" Start time: %s. The schedule has been cleared, and the next recurrence time has been set."+
				" Start the event manually if you wish to run it.", event.GetScheduledTime().String())
		}

		err := s.Schedule(event, s.findNextRecurrence(event, event.GetScheduledTime()))

		if err != nil {
			logrus.WithError(err).Errorf("Couldn't schedule next recurring event (%s)", event.EventName())
		}
	} else {
		logrus.Infof("Looks like the server was offline whilst a scheduled event was meant to start!"+
			" Start time: %s. The schedule has been cleared. Start the event manually if you wish to run it.", event.GetScheduledTime().String())

		err := s.clearScheduledTime(event)

		if err != nil {
			logrus.WithError(err).Error("Could not clear event scheduled time")
		}
	}
}

func (s *Scheduler) Schedule(event ScheduledEvent, startTime time.Time) error {
	if startTime.IsZero() {
		return ErrInvalidScheduleTime
	}

	if timer := s.timers[event.GetID().String()]; timer != nil {
		timer.Stop()
	}

	if reminderTimer := s.reminderTimers[event.GetID().String()]; reminderTimer != nil {
		reminderTimer.Stop()
	}

	s.timers[event.GetID().String()] = time.AfterFunc(time.Until(startTime), func() {
		err := s.startEvent(event)

		if err != nil {
			logrus.WithError(err).Errorf("Could not start scheduled event")
			return
		}

		if event.HasRecurrenceRule() {
			nextRecurrence := s.findNextRecurrence(event, startTime)

			if !nextRecurrence.IsZero() {
				err = s.Schedule(event, nextRecurrence)

				if err != nil {
					logrus.WithError(err).Errorf("Could not set next recurrence timer")
				}
			}
		}
	})

	serverOptions, err := s.store.LoadServerOptions()

	if err != nil {
		return err
	}

	if serverOptions.NotificationReminderTimer > 0 {
		err := s.notificationManager.SendRaceScheduledMessage(event, startTime)

		if err != nil {
			logrus.WithError(err).Errorf("Could not send race scheduled message")
		}

		duration := time.Until(startTime.Add(time.Duration(-serverOptions.NotificationReminderTimer)))

		s.reminderTimers[event.GetID().String()] = time.AfterFunc(duration, func() {
			err := s.notificationManager.SendRaceReminderMessage(event)

			if err != nil {
				logrus.WithError(err).Errorf("Could not send race reminder message")
			}
		})
	}

	return nil
}

func (s *Scheduler) DeSchedule(event ScheduledEvent) error {
	if timer := s.timers[event.GetID().String()]; timer != nil {
		timer.Stop()
	}

	event.ClearRecurrenceRule()

	return s.clearScheduledTime(event)
}

func (s *Scheduler) findNextRecurrence(event ScheduledEvent, start time.Time) time.Time {
	rule, err := event.GetRecurrenceRule()

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't get recurrence rule for event: %s", event.GetID())
		return time.Time{}
	}

	next := rule.After(start, false)

	if next.After(time.Now()) {
		return next
	}

	return time.Time{}
}

var ErrUnknownScheduledEvent = errors.New("servermanager: unknown scheduled event")

func (s *Scheduler) startEvent(event ScheduledEvent) error {
	switch e := event.(type) {
	case *RaceWeekendSession:
		raceWeekend, _, err := s.raceWeekendManager.FindRaceWeekendForSession(event.GetID().String())

		if err != nil {
			return err
		}

		err = s.raceWeekendManager.StartSession(raceWeekend.ID.String(), event.GetID().String())

		if err != nil {
			return err
		}
	case *ChampionshipEvent:
		championship, championshipEvent, err := s.championshipManager.FindChampionshipForEvent(event.GetID().String())

		if err != nil {
			return err
		}

		err = s.championshipManager.StartEvent(championship.ID.String(), championshipEvent.GetID().String(), false)

		if err != nil {
			return err
		}
	case *CustomRace:
		err := s.raceManager.StartCustomRace(e.GetID().String(), false)

		if err != nil {
			return err
		}
	default:
		return ErrUnknownScheduledEvent
	}

	return s.clearScheduledTime(event)
}

func (s *Scheduler) clearScheduledTime(event ScheduledEvent) error {
	switch e := event.(type) {
	case *RaceWeekendSession:
		raceWeekend, raceWeekendSession, err := s.raceWeekendManager.FindRaceWeekendForSession(event.GetID().String())

		if err != nil {
			return err
		}

		raceWeekendSession.Scheduled = time.Time{}

		return s.store.UpsertRaceWeekend(raceWeekend)
	case *ChampionshipEvent:
		championship, championshipEvent, err := s.championshipManager.FindChampionshipForEvent(event.GetID().String())

		if err != nil {
			return err
		}

		championshipEvent.Scheduled = time.Time{}

		return s.championshipManager.UpsertChampionship(championship)
	case *CustomRace:

		e.Scheduled = time.Time{}

		return s.store.UpsertCustomRace(e)
	default:
		return ErrUnknownScheduledEvent
	}
}
