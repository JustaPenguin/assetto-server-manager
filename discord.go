package servermanager

import (
	"errors"
	"net/url"
	"time"

	"github.com/Clinet/discordgo-embed"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type DiscordManager struct {
	store                 Store
	discord               *discordgo.Session
	scheduledRacesManager *ScheduledRacesManager
	enabled               bool
}

// NewDiscordManager instantiates the DiscordManager type.  On error, it will log the error and return the type
// flagged as disabled
func NewDiscordManager(store Store, scheduledRacesManager *ScheduledRacesManager) (*DiscordManager, error) {
	discordManager := &DiscordManager{
		store:                 store,
		scheduledRacesManager: scheduledRacesManager,
		discord:               nil,
		enabled:               false,
	}

	opts, err := store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
		return discordManager, err
	}

	var session *discordgo.Session

	if opts.DiscordAPIToken != "" {
		session, err = discordgo.New("Bot " + opts.DiscordAPIToken)

		if err == nil {
			err = session.Open()
		}

		if err != nil {
			logrus.WithError(err).Errorf("couldn't open discord session")
			return discordManager, err
		}
	} else {
		logrus.Debugf("Discord notification bot not enabled")
		return discordManager, err
	}

	logrus.Infof("Discord notification bot connected")

	discordManager.enabled = true
	discordManager.discord = session

	session.AddHandler(discordManager.CommandHandler)

	return discordManager, nil
}

func (dm *DiscordManager) SaveServerOptions(oldServerOpts *GlobalServerConfig, newServerOpts *GlobalServerConfig) error {
	if newServerOpts.DiscordAPIToken != "" && (oldServerOpts.DiscordAPIToken != newServerOpts.DiscordAPIToken) {
		// existing token changed, so stop
		if oldServerOpts.DiscordAPIToken != "" && dm.enabled {
			_ = dm.Stop()
		}

		// token added (or changed), so attempt to connect
		session, err := discordgo.New("Bot " + newServerOpts.DiscordAPIToken)

		if err == nil {
			err = session.Open()
		}

		if err != nil {
			logrus.WithError(err).Errorf("couldn't open discord session")
			return err
		}

		dm.discord = session
		dm.enabled = true

		session.AddHandler(dm.CommandHandler)

		logrus.Infof("Discord notification bot reconnected")
	} else if newServerOpts.DiscordAPIToken == "" && oldServerOpts.DiscordAPIToken != "" {
		// token removed, so close session (also sets enabled to false)
		_ = dm.Stop()
		logrus.Infof("Discord notification bot stopped")
	}

	return nil
}

func (dm *DiscordManager) CommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!schedule" {
		start := time.Now()
		end := start.AddDate(0, 0, 7)

		calendar, err := dm.scheduledRacesManager.buildCalendar(start, end)

		if err != nil {
			return
		}

		var msg = ""

		for _, event := range calendar {
			msg += event.Start.Format("Mon, 02 Jan 2006 15:04:05 MST") + "\n"
			msg += event.Title + "\n"
			msg += event.Description + "\n\n"
		}

		_, err = s.ChannelMessageSend(m.ChannelID, msg)

		if err != nil {
			logrus.WithError(err).Errorf("couldn't open discord session")
		}
	}
}

func (dm *DiscordManager) Stop() error {
	if dm.enabled {
		dm.enabled = false
		return dm.discord.Close()
	}

	return nil
}

// SendMessage sends a message to the configured channel and logs any errors
func (dm *DiscordManager) SendMessage(msg string) error {
	if dm.enabled {
		opts, err := dm.store.LoadServerOptions()

		if err != nil {
			logrus.WithError(err).Errorf("couldn't load server options")
			return err
		}

		// could check DiscordChannelID in new, but plan is to allow per-championship channels, so will need to pass
		// it in as an arg and check it here anyway
		if opts.DiscordChannelID != "" {
			_, err = dm.discord.ChannelMessageSend(opts.DiscordChannelID, msg)

			if err != nil {
				logrus.WithError(err).Errorf("couldn't send discord message")
				return err
			}
		} else {
			err = errors.New("no channel ID set in config")
			logrus.WithError(err).Errorf("couldn't send discord message")
			return err
		}
	}

	return nil
}

// SendMessage sends a message to the configured channel and logs any errors
func (dm *DiscordManager) SendEmbed(msg string, linkText string, link *url.URL) error {
	if !dm.enabled {
		return nil
	}

	opts, err := dm.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
		return err
	}

	// could check DiscordChannelID in new, but plan is to allow per-championship channels, so will need to pass
	// it in as an arg and check it here anyway
	if opts.DiscordChannelID != "" {
		linkMsg := "[" + linkText + "](" + link.String() + ")"
		_, err = dm.discord.ChannelMessageSendEmbed(opts.DiscordChannelID, embed.NewGenericEmbed(msg, "%s", linkMsg))

		if err != nil {
			logrus.WithError(err).Errorf("couldn't send discord message")
			return err
		}
	} else {
		err = errors.New("no channel ID set in config")
		logrus.WithError(err).Errorf("couldn't send discord message")
		return err
	}

	return err
}
