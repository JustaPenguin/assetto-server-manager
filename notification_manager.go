package servermanager

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/url"
	"strings"
	"time"
)

// NotificationManager is the generic notification handler, which calls the individual notification
// managers.  Initially, only a Discord manager is implemented.
type NotificationManager struct {
	discordManager *DiscordManager
	carManager     *CarManager
	store          Store
	testing        bool
}

func NewNotificationManager(discord *DiscordManager, cars *CarManager, store Store) *NotificationManager {
	return &NotificationManager{
		discordManager: discord,
		carManager:     cars,
		store:          store,
		testing:        false,
	}
}

// check to see if any notification handlers need to process option changes
func (nm *NotificationManager) SaveServerOptions(soOld *GlobalServerConfig, soNew *GlobalServerConfig) error {
	return nm.discordManager.SaveServerOptions(soOld, soNew)
}

func (nm *NotificationManager) Stop() error {
	return nm.discordManager.Stop()
}

// SendMessage sends a message (surprise surprise)
func (nm *NotificationManager) SendMessage(msg string) error {
	var err error

	// Call all message senders here ... atm just discord.  The manager will know if it's enabled or not, so just call it
	if !nm.testing {
		err = nm.discordManager.SendMessage(msg)
	}

	return err
}

// SendMessageWithLink sends a message with an embedded CM join link
func (nm *NotificationManager) SendMessageWithLink(msg string, linkText string, link *url.URL) error {
	var err error

	// Call all message senders here ... atm just discord.  The manager will know if it's enabled or not, so just call it
	if !nm.testing {
		err = nm.discordManager.SendEmbed(msg, linkText, link)
	}

	return err
}

// SendRaceStartMessage sends a message as a race session is started
func (nm *NotificationManager) SendRaceStartMessage(config ServerConfig, event RaceEvent) error {
	trackInfo, err := GetTrackInfo(config.CurrentRaceConfig.Track, config.CurrentRaceConfig.TrackLayout)

	if err != nil {
		logrus.WithError(err).Warnf("Could not load track details, skipping notification: %s, %s", config.CurrentRaceConfig.Track, config.CurrentRaceConfig.TrackLayout)
		return err
	}

	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		logrus.Errorf("couldn't load server options, skipping notification, err: %s", err)
		return err
	}

	msg := ""
	eventName := event.EventName()

	if eventName != "" {
		msg = fmt.Sprintf("%s race at %s is starting now", eventName, trackInfo.Name)
	} else {
		msg = fmt.Sprintf("Race at %s is starting now", trackInfo.Name)
	}

	if serverOpts.ShowPasswordInNotifications == 1 {
		passwordString := "\nNo password"

		if event.OverrideServerPassword() {
			if event.ReplacementServerPassword() != "" {
				passwordString = fmt.Sprintf("\nPassword is '%s' (no quotes)", event.ReplacementServerPassword())
			}
		} else if config.GlobalServerConfig.Password != "" {
			passwordString = fmt.Sprintf("\nPassword is '%s' (no quotes)", config.GlobalServerConfig.Password)
		}

		msg += passwordString
	}

	if config.GlobalServerConfig.ShowContentManagerJoinLink == 1 {
		link, err := getCMJoinLink(config)
		linkText := ""

		if err != nil {
			logrus.Errorf("could not get CM join link, err: %s", err)
			return nm.SendMessage(msg)
		} else {
			linkText = "Content Manager join link"
			return nm.SendMessageWithLink(msg, linkText, link)
		}
	} else {
		return nm.SendMessage(msg)
	}
}

// SendRaceScheduledMessage sends a notification when a race is scheduled
func (nm *NotificationManager) SendRaceScheduledMessage(event *CustomRace, date time.Time) error {
	var dateStr = date.Format("Mon, 02 Jan 2006 15:04:05 MST")

	var aCarNames = []string{}

	for _, carName := range strings.Split(event.RaceConfig.Cars, ";") {
		car, err := nm.carManager.LoadCar(carName, nil)

		if err != nil {
			logrus.WithError(err).Warnf("Could not load car details for: %s", carName)
			continue
		}

		aCarNames = append(aCarNames, car.Details.Name)
	}

	carNames := strings.Join(aCarNames, ", ")

	trackInfo, err := GetTrackInfo(event.RaceConfig.Track, event.RaceConfig.TrackLayout)

	if err != nil {
		logrus.WithError(err).Warnf("Could not load track details, skipping notification: %s, %s", event.RaceConfig.Track, event.RaceConfig.TrackLayout)
		return err
	}

	var msg = "A new event has been scheduled\n"
	eventName := event.EventName()

	if eventName != "" {
		msg += fmt.Sprintf("Event name: %s\n", eventName)
	}

	msg += fmt.Sprintf("Date: %s\n", dateStr)
	msg += fmt.Sprintf("Track: %s\n", trackInfo.Name)
	msg += fmt.Sprintf("Car(s): %s\n", carNames)

	return nm.SendMessage(msg)
}

// SendRaceReminderMessage sends a reminder a configurable number of minutes prior to a race starting
func (nm *NotificationManager) SendRaceReminderMessage(event *CustomRace) error {
	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		logrus.Errorf("couldn't load server options, skipping notification, err: %s", err)
		return err
	}

	msg := ""
	eventName := event.EventName()
	trackInfo, err := GetTrackInfo(event.RaceConfig.Track, event.RaceConfig.TrackLayout)

	if err != nil {
		logrus.WithError(err).Warnf("Could not load track details, skipping notification: %s, %s", event.RaceConfig.Track, event.RaceConfig.TrackLayout)
		return err
	}

	if eventName != "" {
		msg = fmt.Sprintf("%s race at %s starts in %d minutes", eventName, trackInfo.Name, serverOpts.NotificationReminderTimer)
	} else {
		msg = fmt.Sprintf("Race at %s starts in %d minutes", trackInfo.Name, serverOpts.NotificationReminderTimer)
	}

	return nm.SendMessage(msg)
}

// SendChampionshipReminderMessage sends a reminder a configurable number of minutes prior to a championship race starting
func (nm *NotificationManager) SendChampionshipReminderMessage(championship *Championship, event *ChampionshipEvent) error {
	var err error
	trackInfo, err := GetTrackInfo(event.RaceSetup.Track, event.RaceSetup.TrackLayout)

	if err != nil {
		logrus.WithError(err).Warnf("Could not load track details, skipping notification: %s, %s", event.RaceSetup.Track, event.RaceSetup.TrackLayout)
		return err
	}

	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		logrus.Errorf("couldn't load server options, skipping notification, err: %s", err)
		return err
	}

	return nm.SendMessage(fmt.Sprintf("%s race at %s starts in %d minutes", championship.Name, trackInfo.Name, serverOpts.NotificationReminderTimer))
}
