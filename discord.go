package servermanager

import (
	"errors"
	"github.com/Clinet/discordgo-embed"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"net/url"
	"time"
)

type DiscordManager struct {
	store   Store
	discord *discordgo.Session
	r       *Resolver
	enabled bool
}

// NewDiscordManager instantiates the DiscordManager type.  On error, it will log the error and return the type
// flagged as disabled
func NewDiscordManager(store Store, r *Resolver) (*DiscordManager, error) {
	opts, err := store.LoadServerOptions()

	if err != nil {
		logrus.Errorf("couldn't load server options, err: %s", err)
		return &DiscordManager{
			store:   store,
			r:       r,
			discord: nil,
			enabled: false,
		}, err
	}

	var session *discordgo.Session

	if opts.DiscordAPIToken != "" {
		session, err = discordgo.New("Bot " + opts.DiscordAPIToken)

		if err == nil {
			err = session.Open()
		}

		if err != nil {
			logrus.Errorf("couldn't open discord session, err: %s", err)
			return &DiscordManager{
				store:   store,
				discord: nil,
				r:       r,
				enabled: false,
			}, err
		}
	} else {
		logrus.Infof("Discord notification bot not enabled")

		return &DiscordManager{
			store:   store,
			discord: nil,
			r:       r,
			enabled: false,
		}, nil
	}

	logrus.Infof("Discord notification bot connected")

	dm := &DiscordManager{
		store:   store,
		discord: session,
		r:       r,
		enabled: true,
	}

	session.AddHandler(dm.CommandHandler)

	return dm, nil
}

func (dm *DiscordManager) CommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!schedule" {
		rs := dm.r.resolveScheduledRacesHandler()

		start := time.Now()
		end := start.AddDate(0, 0, 7)

		calendar, err := rs.buildCalendar(start, end)

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
			logrus.Errorf("couldn't open discord session, err: %s", err)
		}
	}
}

func (dm *DiscordManager) Stop() error {
	if dm.enabled {
		return dm.discord.Close()
	}

	return nil
}

// SendMessage sends a message to the configured channel and logs any errors
func (dm *DiscordManager) SendMessage(msg string) error {
	if dm.enabled {
		opts, err := dm.store.LoadServerOptions()

		if err != nil {
			logrus.Errorf("couldn't load server options, err: %s", err)
			return err
		}

		// could check DiscordChannelID in new, but plan is to allow per-championship channels, so will need to pass
		// it in as an arg and check it here anyway
		if opts.DiscordChannelID != "" {
			_, err = dm.discord.ChannelMessageSend(opts.DiscordChannelID, msg)

			if err != nil {
				logrus.Errorf("couldn't send discord message, err: %s", err)
				return err
			}
		} else {
			err = errors.New("no channel ID set in config")
			logrus.Errorf("couldn't send discord message, err: %s", err)
			return err
		}
	}

	return nil
}

// SendMessage sends a message to the configured channel and logs any errors
func (dm *DiscordManager) SendEmbed(msg string, linkText string, link *url.URL) error {
	if dm.enabled {
		opts, err := dm.store.LoadServerOptions()

		if err != nil {
			logrus.Errorf("couldn't load server options, err: %s", err)
			return err
		}

		// could check DiscordChannelID in new, but plan is to allow per-championship channels, so will need to pass
		// it in as an arg and check it here anyway
		if opts.DiscordChannelID != "" {
			linkMsg := "[" + linkText + "](" + link.String() + ")"
			_, err = dm.discord.ChannelMessageSendEmbed(opts.DiscordChannelID, embed.NewGenericEmbed(msg, linkMsg))

			if err != nil {
				logrus.Errorf("couldn't send discord message, err: %s", err)
				return err
			}
		} else {
			err = errors.New("no channel ID set in config")
			logrus.Errorf("couldn't send discord message, err: %s", err)
			return err
		}
	}

	return nil
}
