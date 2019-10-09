package servermanager

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type NotificationDispatcher interface {
	HasNotificationReminders() bool
	GetNotificationReminders() []int
	SendMessage(msg string) error
	SendMessageWithLink(msg string, linkText string, link *url.URL) error
	SendRaceStartMessage(config ServerConfig, event RaceEvent) error
	SendRaceScheduledMessage(event *CustomRace, date time.Time) error
	SendRaceReminderMessage(event *CustomRace, timer int) error
	SendChampionshipReminderMessage(championship *Championship, event *ChampionshipEvent, timer int) error
	SendRaceWeekendReminderMessage(raceWeekend *RaceWeekend, session *RaceWeekendSession, timer int) error
	SaveServerOptions(oldServerOpts *GlobalServerConfig, newServerOpts *GlobalServerConfig) error
}

// NotificationManager is the generic notification handler, which calls the individual notification
// managers. Initially, only a Discord manager is implemented.
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
		testing:        os.Getenv("NOTIFICATION_TEST_MODE") == "true",
	}
}

// check to see if any notification handlers need to process option changes
func (nm *NotificationManager) SaveServerOptions(oldServerOpts *GlobalServerConfig, newServerOpts *GlobalServerConfig) error {
	return nm.discordManager.SaveServerOptions(oldServerOpts, newServerOpts)
}

func (nm *NotificationManager) Stop() error {
	return nm.discordManager.Stop()
}

// HasNotificationReminders just tells us if we need to do any reminder scheduling
func (nm *NotificationManager) HasNotificationReminders() bool {
	reminders := nm.GetNotificationReminders()

	return len(reminders) > 0
}

// GetNotificationReminders returns an array of int timers
// Doesn't return errors, just omits anything it doesn't like and logs errors
func (nm *NotificationManager) GetNotificationReminders() []int {
	var reminders []int

	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
		return reminders
	}

	timers := strings.Split(serverOpts.NotificationReminderTimers, ",")

	for _, a := range timers {
		if strings.TrimSpace(a) == "" {
			logrus.WithError(err).Infof("couldn't convert notification time to int")
			continue
		}

		i, err := strconv.Atoi(strings.TrimSpace(a))

		if err != nil {
			logrus.WithError(err).Errorf("couldn't convert notification time to int")
			continue
		}

		if i == 0 {
			continue
		}

		reminders = append(reminders, i)
	}

	return reminders
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
		err = nm.discordManager.SendMessageWithLink(msg, linkText, link)
	}

	return err
}

// SendRaceStartMessage sends a message as a race session is started
func (nm *NotificationManager) SendRaceStartMessage(config ServerConfig, event RaceEvent) error {
	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options, skipping notification")
		return err
	}

	msg := ""
	eventName := event.EventName()
	trackInfo := trackSummary(config.CurrentRaceConfig.Track, config.CurrentRaceConfig.TrackLayout)

	if eventName != "" {
		msg = fmt.Sprintf("%s race at %s is starting now", eventName, trackInfo)
	} else {
		msg = fmt.Sprintf("Race at %s is starting now", trackInfo)
	}

	msg += fmt.Sprintf("\nServer: %s", serverOpts.Name)

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
		link, err := getContentManagerJoinLink(config)
		linkText := ""

		if err != nil {
			logrus.WithError(err).Errorf("could not get CM join link")

			return nm.SendMessage(msg)
		} else {
			linkText = "Content Manager join link"

			// delay sending message by 20 seconds to give server time to register with lobby so CM link works
			time.AfterFunc(time.Duration(20)*time.Second, func() {
				_ = nm.SendMessageWithLink(msg, linkText, link)
			})

			return nil
		}
	} else {
		return nm.SendMessage(msg)
	}
}

// SendRaceScheduledMessage sends a notification when a race is scheduled
func (nm *NotificationManager) SendRaceScheduledMessage(event *CustomRace, date time.Time) error {
	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options, skipping notification")
		return err
	}

	dateStr := date.Format("Mon, 02 Jan 2006 15:04:05 MST")

	var aCarNames []string

	for _, carName := range strings.Split(event.RaceConfig.Cars, ";") {
		car, err := nm.carManager.LoadCar(carName, nil)

		if err != nil {
			logrus.WithError(err).Warnf("Could not load car details for: %s", carName)
			continue
		}

		aCarNames = append(aCarNames, car.Details.Name)
	}

	carNames := strings.Join(aCarNames, ", ")

	msg := "A new event has been scheduled\n"
	msg += fmt.Sprintf("Server: %s\n", serverOpts.Name)
	eventName := event.EventName()
	trackInfo := trackSummary(event.RaceConfig.Track, event.RaceConfig.TrackLayout)

	if eventName != "" {
		msg += fmt.Sprintf("Event name: %s\n", eventName)
	}

	msg += fmt.Sprintf("Date: %s\n", dateStr)
	msg += fmt.Sprintf("Track: %s\n", trackInfo)
	msg += fmt.Sprintf("Car(s): %s\n", carNames)

	return nm.SendMessage(msg)
}

// SendRaceReminderMessage sends a reminder a configurable number of minutes prior to a race starting
func (nm *NotificationManager) SendRaceReminderMessage(event *CustomRace, timer int) error {
	msg := ""
	trackInfo := trackSummary(event.RaceConfig.Track, event.RaceConfig.TrackLayout)
	eventName := event.EventName()

	if eventName != "" {
		msg = fmt.Sprintf("%s race at %s starts in %d minutes", eventName, trackInfo, timer)
	} else {
		msg = fmt.Sprintf("Race at %s starts in %d minutes", trackInfo, timer)
	}

	return nm.SendMessage(msg)
}

// SendChampionshipReminderMessage sends a reminder a configurable number of minutes prior to a championship race starting
func (nm *NotificationManager) SendChampionshipReminderMessage(championship *Championship, event *ChampionshipEvent, timer int) error {
	return nm.SendMessage(fmt.Sprintf("%s race at %s starts in %d minutes", championship.Name, trackSummary(event.RaceSetup.Track, event.RaceSetup.TrackLayout), timer))
}

// SendRaceWeekendReminderMessage sends a reminder a configurable number of minutes prior to a RaceWeekendSession starting
func (nm *NotificationManager) SendRaceWeekendReminderMessage(raceWeekend *RaceWeekend, session *RaceWeekendSession, timer int) error {
	trackInfo := trackSummary(session.RaceConfig.Track, session.RaceConfig.TrackLayout)
	return nm.SendMessage(fmt.Sprintf("%s at %s (%s Race Weekend) starts in %d minutes", session.Name(), raceWeekend.Name, trackInfo, timer))
}
