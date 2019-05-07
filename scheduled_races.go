package servermanager

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/heindl/caldav-go/icalendar"
	"github.com/heindl/caldav-go/icalendar/components"
	"github.com/heindl/caldav-go/icalendar/values"
	"github.com/sirupsen/logrus"
)

var CustomRaceStartTimers map[string]*time.Timer
var ChampionshipEventStartTimers map[string]*time.Timer

func InitialiseScheduledCustomRaces() error {
	CustomRaceStartTimers = make(map[string]*time.Timer)

	races, err := raceManager.raceStore.ListCustomRaces()

	if err != nil {
		return err
	}

	for _, race := range races {
		race := race

		if race.Scheduled.After(time.Now()) {
			// add a scheduled event on date
			duration := time.Until(race.Scheduled)

			CustomRaceStartTimers[race.UUID.String()] = time.AfterFunc(duration, func() {
				err := raceManager.StartCustomRace(race.UUID.String(), false)

				if err != nil {
					logrus.Errorf("couldn't start scheduled race, err: %s", err)
				}
			})

			err := raceManager.raceStore.UpsertCustomRace(race)

			if err != nil {
				return err
			}
		} else {
			emptyTime := time.Time{}
			if race.Scheduled != emptyTime {
				logrus.Infof("Looks like the server was offline whilst a scheduled event was meant to start!"+
					" Start time: %s. The schedule has been cleared. Start the event manually if you wish to run it.", race.Scheduled.String())

				race.Scheduled = emptyTime

				err := raceManager.raceStore.UpsertCustomRace(race)

				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func InitialiseScheduledChampionshipEvents() error {
	ChampionshipEventStartTimers = make(map[string]*time.Timer)

	championships, err := championshipManager.ListChampionships()

	if err != nil {
		return err
	}

	for _, championship := range championships {
		championship := championship

		for _, event := range championship.Events {
			event := event

			if event.Scheduled.After(time.Now()) {
				// add a scheduled event on date
				duration := time.Until(event.Scheduled)

				ChampionshipEventStartTimers[event.ID.String()] = time.AfterFunc(duration, func() {
					err := championshipManager.StartEvent(championship.ID.String(), event.ID.String())

					if err != nil {
						logrus.Errorf("couldn't start scheduled race, err: %s", err)
					}
				})

				return championshipManager.UpsertChampionship(championship)
			} else {
				emptyTime := time.Time{}
				if event.Scheduled != emptyTime {
					logrus.Infof("Looks like the server was offline whilst a scheduled event was meant to start!"+
						" Start time: %s. The schedule has been cleared. Start the event manually if you wish to run it.", event.Scheduled.String())

					event.Scheduled = emptyTime

					return championshipManager.UpsertChampionship(championship)
				}

			}
		}
	}

	return nil
}

type ScheduledEvent interface {
	GetID() uuid.UUID
	GetRaceSetup() CurrentRaceConfig
	GetScheduledTime() time.Time
	GetSummary() string
	GetURL() string
	GetEntryList() EntryList
}

func BuildICalEvent(event ScheduledEvent) *components.Event {
	icalEvent := components.NewEvent(event.GetID().String(), event.GetScheduledTime().UTC())

	raceSetup := event.GetRaceSetup()

	trackInfo := trackInfo(raceSetup.Track, raceSetup.TrackLayout)

	if trackInfo == nil {
		icalEvent.Summary = "Race at " + prettifyName(raceSetup.Track, false)

		if raceSetup.TrackLayout != "" {
			icalEvent.Summary += fmt.Sprintf(" (%s)", prettifyName(raceSetup.TrackLayout, true))
		}
	} else {
		icalEvent.Summary = fmt.Sprintf("Race at %s, %s, %s", trackInfo.Name, trackInfo.City, trackInfo.Country)
		icalEvent.Location = values.NewLocation(fmt.Sprintf("%s, %s", trackInfo.City, trackInfo.Country))
	}

	icalEvent.Summary += " " + event.GetSummary()

	if config.HTTP.BaseURL != "" {
		u, err := url.Parse(config.HTTP.BaseURL + event.GetURL())

		if err == nil {
			icalEvent.Url = values.NewUrl(*u)
		}
	}

	var totalDuration time.Duration
	var description string

	for _, session := range raceSetup.Sessions.AsSlice() {
		if session.Time > 0 {
			sessionDuration := time.Minute * time.Duration(session.Time)

			totalDuration += sessionDuration
			description += fmt.Sprintf("%s: %s\n", session.Name, sessionDuration.String())
		} else if session.Laps > 0 {
			description += fmt.Sprintf("%s: %d laps\n", session.Name, session.Laps)
			totalDuration += time.Minute * 30 // just add a 30 min buffer so it shows in the calendar
		}
	}

	entryList := event.GetEntryList()

	description += fmt.Sprintf("\n%d entrants in: %s", len(entryList), carList(entryList))

	icalEvent.Description = description
	icalEvent.Duration = values.NewDuration(totalDuration)

	return icalEvent
}

func buildScheduledRaces(w io.Writer) error {
	_, _, _, customRaces, err := raceManager.ListCustomRaces()

	if err != nil {
		return err
	}

	var scheduled []ScheduledEvent

	for _, race := range customRaces {
		scheduled = append(scheduled, race)
	}

	championships, err := championshipManager.ListChampionships()

	if err != nil {
		return err
	}

	for _, championship := range championships {
		for _, event := range championship.Events {
			if event.Scheduled.IsZero() {
				continue
			}

			event.championship = championship
			scheduled = append(scheduled, event)
		}
	}

	cal := components.NewCalendar()

	for _, event := range scheduled {
		icalEvent := BuildICalEvent(event)

		cal.Events = append(cal.Events, icalEvent)
	}

	str, err := icalendar.Marshal(cal)

	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, str)

	return err
}

func allScheduledRacesICalHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Add("Content-Disposition", "inline; filename=championship.ics")

	err := buildScheduledRaces(w)

	if err != nil {
		logrus.WithError(err).Error("could not build scheduled races feed")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
