package servermanager

import (
	"fmt"
)

// NotificationHandler is the generic notification handler, which calls the individual notification
// managers.  Initially, only a Discord manager is implemented.
type NotificationHandler struct {
	discordManager *DiscordManager
}

func NewNotificationManager(resolver *Resolver) *NotificationHandler {
	return &NotificationHandler{
		discordManager: resolver.resolveDiscordManager(),
	}
}

// SendMessage sends a message (surprise surprise)
func (nm *NotificationHandler) SendMessage(msg string) {
	var err error
	err = nm.discordManager.SendMessage(msg)

	if err != nil {
		// @TODO log error
		err = nil
	}
}

// SendRaceStartMessage sends a message as a race session is started
func (nm *NotificationHandler) SendRaceStartMessage(config ServerConfig, event RaceEvent) {
	var err error
	trackInfo, err := GetTrackInfo(config.CurrentRaceConfig.Track, config.CurrentRaceConfig.TrackLayout)

	if err == nil {
		var msg = fmt.Sprintf("Race %s at %s is starting now", event.EventName(), trackInfo.Name)
		err = nm.discordManager.SendMessage(msg)
	}

	if err != nil {
		// @TODO log error
		err = nil
	}
}

// SendRaceReminderMessage sends a reminder a configurable number of minutes prior to a race starting
func (nm *NotificationHandler) SendRaceReminderMessage(event RaceEvent) {
	var err error
	var msg = fmt.Sprintf("Race %s starts in 10 minutes", event.EventName())
	err = nm.discordManager.SendMessage(msg)

	if err != nil {
		// @TODO log error
		err = nil
	}
}

// SendRaceReminderMessage sends a reminder a configurable number of minutes prior to a championship race starting
func (nm *NotificationHandler) SendChampionshipReminderMessage(championship *Championship, event *ChampionshipEvent) {
	var err error
	trackInfo, err := GetTrackInfo(event.RaceSetup.Track, event.RaceSetup.TrackLayout)

	if err == nil {
		var msg = fmt.Sprintf("%s race at %s starts in 10 minutes", championship.Name, trackInfo.Name)
		err = nm.discordManager.SendMessage(msg)
	}

	if err != nil {
		// @TODO log error
		err = nil
	}
}
