package servermanager

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/url"
)

// NotificationHandler is the generic notification handler, which calls the individual notification
// managers.  Initially, only a Discord manager is implemented.
type NotificationHandler struct {
	discordManager *DiscordManager
	store          Store
	testing        bool
}

func NewNotificationManager(discord *DiscordManager, store Store) *NotificationHandler {
	return &NotificationHandler{
		discordManager: discord,
		store:          store,
		testing:        false,
	}
}

// SendMessage sends a message (surprise surprise)
func (nm *NotificationHandler) SendMessage(msg string) error {
	var err error

	// Call all message senders here ... atm just discord.  The manager will know if it's enabled or not, so just call it
	if !nm.testing {
		err = nm.discordManager.SendMessage(msg)
	}

	return err
}

// SendMessage sends a message (surprise surprise)
func (nm *NotificationHandler) SendMessageWithLink(msg string, linkText string, link *url.URL) error {
	var err error

	// Call all message senders here ... atm just discord.  The manager will know if it's enabled or not, so just call it
	if !nm.testing {
		err = nm.discordManager.SendEmbed(msg, linkText, link)
	}

	return err
}

// SendRaceStartMessage sends a message as a race session is started
func (nm *NotificationHandler) SendRaceStartMessage(config ServerConfig, event RaceEvent) error {
	trackInfo, err := GetTrackInfo(config.CurrentRaceConfig.Track, config.CurrentRaceConfig.TrackLayout)

	if err != nil {
		return err
	}

	// @TODO add option for displaying password
	passwordString := "\nNo password"

	if event.OverrideServerPassword() {
		if event.ReplacementServerPassword() != "" {
			passwordString = fmt.Sprintf("\nPassword is '%s' (no quotes)", event.ReplacementServerPassword())
		}
	} else if config.GlobalServerConfig.Password != "" {
		passwordString = fmt.Sprintf("\nPassword is '%s' (no quotes)", config.GlobalServerConfig.Password)
	}

	// @TODO figure out how to show links in Discord messages

	if config.GlobalServerConfig.ShowContentManagerJoinLink == 1 {
		link, err := getCMJoinLink(config)
		linkText := ""

		if err != nil {
			logrus.Errorf("could not get CM join link, err: %s", err)
		} else {
			linkText = "Content Manager join link"
		}

		return nm.SendMessageWithLink(fmt.Sprintf("Race %s at %s is starting now%s", event.EventName(), trackInfo.Name, passwordString), linkText, link)
	} else {
		return nm.SendMessage(fmt.Sprintf("Race %s at %s is starting now%s", event.EventName(), trackInfo.Name, passwordString))
	}
}

// SendRaceReminderMessage sends a reminder a configurable number of minutes prior to a race starting
func (nm *NotificationHandler) SendRaceReminderMessage(event RaceEvent) error {
	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		return err
	}

	return nm.SendMessage(fmt.Sprintf("Race %s starts in %s minutes", event.EventName(), serverOpts.NotificationReminderTimer))
}

// SendRaceReminderMessage sends a reminder a configurable number of minutes prior to a championship race starting
func (nm *NotificationHandler) SendChampionshipReminderMessage(championship *Championship, event *ChampionshipEvent) error {
	var err error
	trackInfo, err := GetTrackInfo(event.RaceSetup.Track, event.RaceSetup.TrackLayout)

	if err != nil {
		return err
	}

	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		return err
	}

	return nm.SendMessage(fmt.Sprintf("%s race at %s starts in %s minutes", championship.Name, trackInfo.Name, serverOpts.NotificationReminderTimer))
}
