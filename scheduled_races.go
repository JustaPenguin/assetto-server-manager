package servermanager

import (
	"encoding/json"
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

type ScheduledEvent interface {
	GetID() uuid.UUID
	GetRaceSetup() CurrentRaceConfig
	GetScheduledTime() time.Time
	GetSummary() string
	GetURL() string
	ReadOnlyEntryList() EntryList
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

	entryList := event.ReadOnlyEntryList()

	description += fmt.Sprintf("\n%d entrants in: %s", len(entryList), carList(entryList))

	icalEvent.Description = description
	icalEvent.Duration = values.NewDuration(totalDuration)

	return icalEvent
}

type ScheduledRacesHandler struct {
	*BaseHandler

	store               Store
	raceManager         *RaceManager
	championshipManager *ChampionshipManager
}

func NewScheduledRacesHandler(baseHandler *BaseHandler, store Store, raceManager *RaceManager, championshipManager *ChampionshipManager) *ScheduledRacesHandler {
	return &ScheduledRacesHandler{
		BaseHandler: 		 baseHandler,
		store:               store,
		raceManager:         raceManager,
		championshipManager: championshipManager,
	}
}

func (rs *ScheduledRacesHandler) calendar(w http.ResponseWriter, r *http.Request) {
	rs.viewRenderer.MustLoadTemplate(w, r, "calendar.html", nil)
}

func (rs *ScheduledRacesHandler) calendarJSON(w http.ResponseWriter, r *http.Request) {
	err := rs.generateJSON(w)

	if err != nil {
		logrus.Errorf("could not find scheduled events, err: %s", err)
		return
	}
}

func (rs *ScheduledRacesHandler) getScheduledRaces() ([]ScheduledEvent, error) {
	_, _, _, customRaces, err := rs.raceManager.ListCustomRaces()

	if err != nil {
		return nil, err
	}

	var scheduled []ScheduledEvent

	for _, race := range customRaces {
		scheduled = append(scheduled, race)
	}

	championships, err := rs.championshipManager.ListChampionships()

	if err != nil {
		return nil, err
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

	return scheduled, nil
}

func (rs *ScheduledRacesHandler) buildScheduledRaces(w io.Writer) error {
	scheduled, err := rs.getScheduledRaces()

	if err != nil {
		return err
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

func (rs *ScheduledRacesHandler) allScheduledRacesICalHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Add("Content-Disposition", "inline; filename=championship.ics")

	err := rs.buildScheduledRaces(w)

	if err != nil {
		logrus.WithError(err).Error("could not build scheduled races feed")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

type calendarObject struct {
	ID string `json:"id"`
	GroupID string `json:"groupId"`
	AllDay bool `json:"allDay"`
	Start time.Time `json:"start"`
	End time.Time `json:"end"`
	Title string `json:"title"`
	URL string `json:"url"` // @TODO for onclick, see eventClick callback
	ClassNames []string `json:"classNames"`
	Editable bool `json:"editable"`
	StartEditable bool `json:"startEditable"`
	DurationEditable bool `json:"durationEditable"`
	ResourceEditable bool `json:"resourceEditable"`
	Rendering string `json:"rendering"`
	Overlap bool `json:"overlap"`

	Constraint string `json:"constraint"`
	BackgroundColor string `json:"backgroundColor"`
	BorderColor string `json:"borderColor"`
	TextColor string `json:"textColor"`
}

func (rs *ScheduledRacesHandler) generateJSON(w http.ResponseWriter) error {
	scheduled, err := rs.getScheduledRaces()

	if err != nil {
		return err
	}

	var calendarObjects []calendarObject

	if len(scheduled) == 0 {
		calendarObjects = append(calendarObjects, calendarObject{
			ID: "no-events",
			GroupID: "no-events",
			AllDay: false,
			Start: time.Now(),
			End: time.Now().Add(time.Hour*3),
			Title: "Looks like there are no scheduled events!",
			URL: "",
			ClassNames: nil,
			Editable: false,
			StartEditable: false,
			DurationEditable: false,
			ResourceEditable: false,
			Rendering: "",
			Overlap: true,
			Constraint: "",
			BackgroundColor: "red",
			BorderColor: "white",
			TextColor: "white",
		})
	}

	for _, scheduledEvent := range scheduled {

		var prevSessionTime time.Duration
		start := scheduledEvent.GetScheduledTime()
		end := scheduledEvent.GetScheduledTime()

		for _, session := range scheduledEvent.GetRaceSetup().Sessions.AsSlice() {
			// calculate session start/end
			start = start.Add(prevSessionTime)

			if session.Time > 0 {
				prevSessionTime = time.Minute * time.Duration(session.Time)
			} else {
				// approximate, probably fine
				prevSessionTime = 3 * time.Minute * time.Duration(session.Laps)
			}

			end = end.Add(prevSessionTime)

			// select colours
			var backgroundColor, borderColor, textColor string

			switch session.Name {
			case "Practice":
				backgroundColor = "#0a84ff"
			case "Qualify":
				backgroundColor = "#23a127"
			case "Race":
				backgroundColor = "#FF4136"
			}

			// @TODO make clear that championships are championships
			// @TODO hover background color change?
			if scheduledEvent.GetURL() != "" {
				borderColor = "white"
				textColor = "white"
			} else {
				borderColor = "black"
				textColor = "black"
			}

			calendarObjects = append(calendarObjects, calendarObject{
				ID:               scheduledEvent.GetID().String() + session.Name,
				GroupID:          scheduledEvent.GetID().String(),
				AllDay:           false,
				Start:            start,
				End:              end,
				Title:            generateSummary(scheduledEvent.GetRaceSetup(), session.Name) + " " + scheduledEvent.GetSummary(),
				URL:              scheduledEvent.GetURL(),
				ClassNames:       nil,
				Editable:         false,
				StartEditable:    false,
				DurationEditable: false,
				ResourceEditable: false,
				Rendering:        "",
				Overlap:          true,
				Constraint:       "",
				BackgroundColor:  backgroundColor,
				BorderColor:      borderColor,
				TextColor:        textColor,
			})
		}
	}

	return json.NewEncoder(w).Encode(calendarObjects)
}


func generateSummary(raceSetup CurrentRaceConfig, eventType string) string {
	var summary string

	trackInfo := trackInfo(raceSetup.Track, raceSetup.TrackLayout)

	if trackInfo == nil {
		summary = eventType + " at " + prettifyName(raceSetup.Track, false)

		if raceSetup.TrackLayout != "" {
			summary += fmt.Sprintf(" (%s)", prettifyName(raceSetup.TrackLayout, true))
		}
	} else {
		summary = fmt.Sprintf(eventType + " at %s", trackInfo.Name)
	}

	return summary
}
