package servermanager

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/sirupsen/logrus"
)

type NotificationDispatcher interface {
	HasNotificationReminders() bool
	GetNotificationReminders() []int
	SendMessage(title string, msg string) error
	SendMessageWithLink(title string, msg string, linkText string, link *url.URL) error
	SendRaceStartMessage(config ServerConfig, event RaceEvent) error
	SendRaceScheduledMessage(event *CustomRace, date time.Time) error
	SendRaceCancelledMessage(event *CustomRace, date time.Time) error
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
func (nm *NotificationManager) SendMessage(title string, msg string) error {
	var err error

	// Call all message senders here ... atm just discord.  The manager will know if it's enabled or not, so just call it
	if !nm.testing {
		err = nm.discordManager.SendMessage(title, msg)
	}

	return err
}

// SendMessageWithLink sends a message with an embedded CM join link
func (nm *NotificationManager) SendMessageWithLink(title string, msg string, linkText string, link *url.URL) error {
	var err error

	// Call all message senders here ... atm just discord.  The manager will know if it's enabled or not, so just call it
	if !nm.testing {
		err = nm.discordManager.SendMessageWithLink(title, msg, linkText, link)
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
		msg = fmt.Sprintf("%s event at %s is starting now", eventName, trackInfo)
	} else {
		msg = fmt.Sprintf("Event at %s is starting now", trackInfo)
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

	title := fmt.Sprintf("Event starting at %s", trackInfo)

	if config.GlobalServerConfig.ShowContentManagerJoinLink == 1 {
		link, err := getContentManagerJoinLink(config.GlobalServerConfig)
		linkText := ""

		if err != nil {
			logrus.WithError(err).Errorf("could not get CM join link")

			return nm.SendMessage(title, msg)
		}

		linkText = "Content Manager join link"

		// delay sending message by 20 seconds to give server time to register with lobby so CM link works
		time.AfterFunc(time.Duration(20)*time.Second, func() {
			_ = nm.SendMessageWithLink(title, msg, linkText, link)
		})

		return nil
	}

	return nm.SendMessage(title, msg)
}

// GetCarList takes a ; sep string of cars from a race config, returns , sep of UI names with download links added
func (nm *NotificationManager) GetCarList(cars string) string {
	var aCarNames []string

	for _, carName := range strings.Split(cars, ";") {
		car, err := nm.carManager.LoadCar(carName, nil)

		if err != nil {
			logrus.WithError(err).Warnf("Could not load car details for: %s", carName)
			continue
		}

		if car.Details.DownloadURL != "" {
			aCarNames = append(aCarNames, car.Details.Name+" ([download]("+car.Details.DownloadURL+"))")
		} else {
			aCarNames = append(aCarNames, car.Details.Name)
		}
	}

	return strings.Join(aCarNames, ", ")
}

// GetTrackInfo returns the track summary with any download link appended
func (nm *NotificationManager) GetTrackInfo(track string, layout string, download bool) string {
	trackInfo := trackSummary(track, layout)

	if download {
		trackLink := trackDownloadLink(track)

		if trackLink != "" {
			trackInfo += " ([download](" + trackLink + "))"
		}
	}

	return trackInfo
}

// SendRaceScheduledMessage sends a notification when a race is scheduled
func (nm *NotificationManager) SendRaceScheduledMessage(event *CustomRace, date time.Time) error {
	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options, skipping notification")
		return err
	}

	if serverOpts.NotifyWhenScheduled != 1 {
		return nil
	}

	msg := "A new event has been scheduled\n"
	msg += fmt.Sprintf("Server: %s\n", serverOpts.Name)
	eventName := event.EventName()

	if eventName != "" {
		msg += fmt.Sprintf("Event name: %s\n", eventName)
	}

	msg += fmt.Sprintf("Date: %s\n", date.Format("Mon, 02 Jan 2006 15:04:05 MST"))
	carNames := nm.GetCarList(event.RaceConfig.Cars)
	trackInfo := nm.GetTrackInfo(event.RaceConfig.Track, event.RaceConfig.TrackLayout, true)
	msg += fmt.Sprintf("Track: %s\n", trackInfo)
	msg += fmt.Sprintf("Car(s): %s\n", carNames)
	title := fmt.Sprintf("Event scheduled at %s", nm.GetTrackInfo(event.RaceConfig.Track, event.RaceConfig.TrackLayout, false))

	return nm.SendMessage(title, msg)
}

// SendRaceCancelledMessage sends a notification when a race is cancelled
func (nm *NotificationManager) SendRaceCancelledMessage(event *CustomRace, date time.Time) error {
	serverOpts, err := nm.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options, skipping notification")
		return err
	}

	if serverOpts.NotifyWhenScheduled != 1 {
		return nil
	}

	dateStr := date.Format("Mon, 02 Jan 2006 15:04:05 MST")

	msg := "The following scheduled Event has been cancelled\n"
	msg += fmt.Sprintf("Server: %s\n", serverOpts.Name)
	eventName := event.EventName()
	trackInfo := trackSummary(event.RaceConfig.Track, event.RaceConfig.TrackLayout)

	if eventName != "" {
		msg += fmt.Sprintf("Event name: %s\n", eventName)
	}

	msg += fmt.Sprintf("Date: %s\n", dateStr)
	msg += fmt.Sprintf("Track: %s\n", trackInfo)
	title := fmt.Sprintf("Event cancelled at %s", trackInfo)

	return nm.SendMessage(title, msg)
}

// SendRaceReminderMessage sends a reminder a configurable number of minutes prior to a race starting
func (nm *NotificationManager) SendRaceReminderMessage(event *CustomRace, timer int) error {
	msg := ""
	trackInfo := nm.GetTrackInfo(event.RaceConfig.Track, event.RaceConfig.TrackLayout, true)
	eventName := event.EventName()
	carList := nm.GetCarList(event.RaceConfig.Cars)
	reminder := durafmt.Parse(time.Duration(timer) * time.Minute).String()

	if eventName != "" {
		msg = fmt.Sprintf("%s event at %s starts in %s\nCars: %s", eventName, trackInfo, reminder, carList)
	} else {
		msg = fmt.Sprintf("Event at %s starts in %s\nCars: %s", trackInfo, reminder, carList)
	}

	title := fmt.Sprintf("Event reminder - %s", reminder)
	return nm.SendMessage(title, msg)
}

// SendChampionshipReminderMessage sends a reminder a configurable number of minutes prior to a championship race starting
func (nm *NotificationManager) SendChampionshipReminderMessage(championship *Championship, event *ChampionshipEvent, timer int) error {
	reminder := durafmt.Parse(time.Duration(timer) * time.Minute).String()
	title := fmt.Sprintf("Event reminder - %s", reminder)
	trackInfo := nm.GetTrackInfo(event.RaceSetup.Track, event.RaceSetup.TrackLayout, true)
	msg := fmt.Sprintf("%s event at %s starts in %s", championship.Name, trackInfo, reminder)
	return nm.SendMessage(title, msg)
}

// SendRaceWeekendReminderMessage sends a reminder a configurable number of minutes prior to a RaceWeekendSession starting
func (nm *NotificationManager) SendRaceWeekendReminderMessage(raceWeekend *RaceWeekend, session *RaceWeekendSession, timer int) error {
	reminder := durafmt.Parse(time.Duration(timer) * time.Minute).String()
	title := fmt.Sprintf("Event reminder - %s", reminder)
	trackInfo := nm.GetTrackInfo(session.RaceConfig.Track, session.RaceConfig.TrackLayout, true)
	msg := fmt.Sprintf("%s at %s (%s Race Weekend) starts in %s", session.Name(), raceWeekend.Name, trackInfo, reminder)
	return nm.SendMessage(title, msg)
}
