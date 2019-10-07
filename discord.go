package servermanager

import (
	"errors"
	"fmt"
	"github.com/Clinet/discordgo-embed"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/url"
	"time"
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
		logrus.Errorf("couldn't load server options, err: %s", err)
		return discordManager, err
	}

	var session *discordgo.Session

	if opts.DiscordAPIToken != "" {
		session, err = discordgo.New("Bot " + opts.DiscordAPIToken)

		if err == nil {
			err = session.Open()
		}

		if err != nil {
			logrus.Errorf("couldn't open discord session, err: %s", err)
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
			logrus.Errorf("couldn't open discord session, err: %s", err)
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

// CommandSessions outputs a full list of all scheduled sessions (P, Q & R), using buildCalendar as a base
func (dm *DiscordManager) CommandSessions() (string, error) {
	serverOpts, err := dm.store.LoadServerOptions()

	if err != nil {
		return "", err
	}

	start := time.Now()
	end := start.AddDate(0, 0, 7)

	calendar, err := dm.scheduledRacesManager.buildCalendar(start, end)

	if err != nil {
		return "", err
	}

	var msg = fmt.Sprintf("Upcoming sessions on server %s\n", serverOpts.Name)

	for _, event := range calendar {
		msg += event.Start.Format("Mon, 02 Jan 2006 15:04:05 MST") + "\n"
		msg += event.Title + "\n"
		msg += event.Description + "\n\n"
	}

	return msg, nil
}

// CommandSchedule outputs an abbreviated list of all scheduled events
func (dm *DiscordManager) CommandSchedule() (string, error) {
	serverOpts, err := dm.store.LoadServerOptions()

	if err != nil {
		return "", err
	}

	start := time.Now()
	end := start.AddDate(0, 0, 7)
	scheduled, err := dm.scheduledRacesManager.getScheduledRaces()

	if err != nil {
		return "", err
	}

	var recurring []ScheduledEvent

	for _, scheduledEvent := range scheduled {
		if scheduledEvent.HasRecurrenceRule() {
			customRace, ok := scheduledEvent.(*CustomRace)

			if !ok {
				continue
			}

			rule, err := customRace.GetRecurrenceRule()

			if err != nil {
				continue
			}

			for _, startTime := range rule.Between(start, end, true) {
				newEvent := *customRace
				newEvent.Scheduled = startTime
				newEvent.UUID = uuid.New()

				if customRace.GetScheduledTime() == newEvent.GetScheduledTime() {
					continue
				}

				recurring = append(recurring, &newEvent)
			}
		}
	}

	scheduled = append(scheduled, recurring...)

	var msg = fmt.Sprintf("\nUpcoming events on server %s\n\n", serverOpts.Name)

	for _, scheduledEvent := range scheduled {
		raceSetup := scheduledEvent.GetRaceSetup()
		trackInfo := trackInfo(raceSetup.Track, raceSetup.TrackLayout)
		cars := carList(scheduledEvent.GetRaceSetup().Cars)
		msg += fmt.Sprintf("Date: %s\n", scheduledEvent.GetScheduledTime().Format("Mon, 02 Jan 2006 15:04:05 MST"))
		msg += fmt.Sprintf("Track: %s\n", trackInfo.Name)
		msg += fmt.Sprintf("Cars: %s\n", cars)
		msg += "\n\n"
	}

	return msg, nil
}

// CommandSchedule outputs an abbreviated list of all scheduled events
func (dm *DiscordManager) CommandNotify(s *discordgo.Session, m *discordgo.MessageCreate) (string, error) {
	serverOpts, err := dm.store.LoadServerOptions()

	if err != nil {
		return "", err
	}

	if serverOpts.DiscordRoleID == "" {
		return "", nil
	}

	member, err := s.State.Member(m.GuildID, m.Author.ID)

	if err != nil {
		if err == discordgo.ErrStateNotFound {
			member, err = s.GuildMember(m.GuildID, m.Author.ID)
		}

		if err != nil {
			return "", err
		}
	}

	for _, roleID := range member.Roles {
		if roleID == serverOpts.DiscordRoleID {
			return "You already have this role", nil
		}
	}

	err = s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, serverOpts.DiscordRoleID)

	if err != nil {
		return fmt.Sprintf("I'm sorry Dave, I can't do that (%s)", err.Error()), err
	}

	return "Your role has been assigned", nil
}

func (dm *DiscordManager) CommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	var msg = ""
	var err error

	switch m.Content {
	case "!schedule":
		msg, err = dm.CommandSchedule()
	case "!sessions":
		msg, err = dm.CommandSessions()
	case "!spamificate":
		msg, err = dm.CommandNotify(s, m)
	default:
		return
	}

	_, err = s.ChannelMessageSend(m.ChannelID, msg)

	if err != nil {
		logrus.Errorf("couldn't open discord session, err: %s", err)
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
			logrus.Errorf("couldn't load server options, err: %s", err)
			return err
		}

		// could check DiscordChannelID in new, but plan is to allow per-championship channels, so will need to pass
		// it in as an arg and check it here anyway
		if opts.DiscordChannelID != "" {
			if opts.DiscordRoleID != "" {
				mention := fmt.Sprintf("Attention <@&%s>\n", opts.DiscordRoleID)
				messageSend := &discordgo.MessageSend{
					Content: mention,
					Embed:   embed.NewGenericEmbed(msg, ""),
				}
				_, err = dm.discord.ChannelMessageSendComplex(opts.DiscordChannelID, messageSend)
			} else {

				_, err = dm.discord.ChannelMessageSendEmbed(opts.DiscordChannelID, embed.NewGenericEmbed(msg, ""))
			}

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
func (dm *DiscordManager) SendMessageWithLink(msg string, linkText string, link *url.URL) error {
	if !dm.enabled {
		return nil
	}

	opts, err := dm.store.LoadServerOptions()

	if err != nil {
		logrus.Errorf("couldn't load server options, err: %s", err)
		return err
	}

	linkMsg := "[" + linkText + "](" + link.String() + ")"

	// could check DiscordChannelID in new, but plan is to allow per-championship channels, so will need to pass
	// it in as an arg and check it here anyway
	if opts.DiscordChannelID != "" {
		if opts.DiscordRoleID != "" {
			mention := fmt.Sprintf("Attention <@&%s>\n", opts.DiscordRoleID)
			messageSend := &discordgo.MessageSend{
				Content: mention,
				Embed:   embed.NewGenericEmbed(msg, "%s", linkMsg),
			}
			_, err = dm.discord.ChannelMessageSendComplex(opts.DiscordChannelID, messageSend)
		} else {

			_, err = dm.discord.ChannelMessageSendEmbed(opts.DiscordChannelID, embed.NewGenericEmbed(msg, "%s", linkMsg))
		}

		if err != nil {
			logrus.Errorf("couldn't send discord message, err: %s", err)
			return err
		}
	} else {
		err = errors.New("no channel ID set in config")
		logrus.Errorf("couldn't send discord message, err: %s", err)
		return err
	}

	return err
}
