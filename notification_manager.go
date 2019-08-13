package servermanager

import (
	"fmt"
)

// NotificationHandler is the generic notification handler, which calls the individual notification
// managers.  Initially, only a Discord manager is implemented.
type NotificationHandler struct {
	discordManager *DiscordManager
}

func NewNotificationManager(discord *DiscordManager) *NotificationHandler {
	return &NotificationHandler{
		discordManager: discord,
	}
}

// SendMessage sends a message (surprise surprise)
func (nm *NotificationHandler) SendMessage(msg string) error {
	var err error

	// Call all message senders here ... atm just discord.  The manager will know if it's enabled or not, so just call it
	err = nm.discordManager.SendMessage(msg)

	return err
}

// SendRaceStartMessage sends a message as a race session is started
func (nm *NotificationHandler) SendRaceStartMessage(config ServerConfig, event RaceEvent) error {
	trackInfo, err := GetTrackInfo(config.CurrentRaceConfig.Track, config.CurrentRaceConfig.TrackLayout)

	if err != nil {
		return err
	}

	return nm.SendMessage(fmt.Sprintf("Race %s at %s is starting now", event.EventName(), trackInfo.Name))
}

// SendRaceReminderMessage sends a reminder a configurable number of minutes prior to a race starting
func (nm *NotificationHandler) SendRaceReminderMessage(event RaceEvent) error {
	return nm.SendMessage(fmt.Sprintf("Race %s starts in 10 minutes", event.EventName()))
}

// SendRaceReminderMessage sends a reminder a configurable number of minutes prior to a championship race starting
func (nm *NotificationHandler) SendChampionshipReminderMessage(championship *Championship, event *ChampionshipEvent) error {
	var err error
	trackInfo, err := GetTrackInfo(event.RaceSetup.Track, event.RaceSetup.TrackLayout)

	if err != nil {
		return err
	}

	return nm.SendMessage(fmt.Sprintf("%s race at %s starts in 10 minutes", championship.Name, trackInfo.Name))
}
