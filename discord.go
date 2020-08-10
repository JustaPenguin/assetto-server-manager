package servermanager

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	embed "github.com/Clinet/discordgo-embed"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
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

// CommandSessions outputs a full list of all scheduled sessions (P, Q & R), using buildCalendar as a base
func (dm *DiscordManager) CommandSessions() (string, error) {
	serverOpts, err := dm.store.LoadServerOptions()

	if err != nil {
		return "A server error occurred, please try again later", err
	}

	start := time.Now()
	end := start.AddDate(0, 0, 7)

	calendar, err := dm.scheduledRacesManager.buildCalendar(start, end)

	if err != nil {
		return "A server error occurred, please try again later", err
	}

	msg := fmt.Sprintf("Upcoming sessions on server %s\n", serverOpts.Name)

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
		return "A server error occurred, please try again later", err
	}

	start := time.Now()
	end := start.AddDate(0, 0, 7)
	scheduled, err := dm.scheduledRacesManager.getScheduledRaces(true)

	if err != nil {
		return "A server error occurred, please try again later", err
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

	msg := fmt.Sprintf("\nUpcoming events on server %s\n\n", serverOpts.Name)

	for _, scheduledEvent := range scheduled {
		raceSetup := scheduledEvent.GetRaceSetup()
		cars := carList(scheduledEvent.GetRaceSetup().Cars)
		msg += fmt.Sprintf("Date: %s\n", scheduledEvent.GetScheduledTime().Format("Mon, 02 Jan 2006 15:04:05 MST"))
		msg += fmt.Sprintf("Track: %s\n", trackSummary(raceSetup.Track, raceSetup.TrackLayout))
		msg += fmt.Sprintf("Cars: %s\n", cars)
		msg += "\n\n"
	}

	return msg, nil
}

// CommandNotify attempts to add a role ID (if configured) to the user issuing the !notify command
// The role will be added as a mention on all Discord notifications
func (dm *DiscordManager) CommandNotify(s *discordgo.Session, m *discordgo.MessageCreate) (string, error) {
	serverOpts, err := dm.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Infof("couldn't get server options")
		return "A server error occurred, try again later", err
	}

	if serverOpts.DiscordRoleID == "" || serverOpts.DiscordRoleCommand == "" {
		return "", nil
	}

	// get the member
	member, err := s.State.Member(m.GuildID, m.Author.ID)

	if err != nil {
		// if it's not in the state, punt to asking the server
		if err == discordgo.ErrStateNotFound {
			member, err = s.GuildMember(m.GuildID, m.Author.ID)
		}

		if err != nil {
			return "You don't seem to exist, so I can't assign you that role.  Try again later.", err
		}
	}

	// get the role name from ID, for use in user feedback
	roleName := "notification"
	roles, err := s.GuildRoles(m.GuildID)

	if err != nil {
		// meh, just log it and carry on
		logrus.WithError(err).Infof("failed to get Discord roles (make sure you have set the bot permissions to see roles)")
	} else {
		for _, role := range roles {
			if strings.TrimSpace(role.ID) == strings.TrimSpace(serverOpts.DiscordRoleID) {
				roleName = role.Name
				break
			}
		}
	}

	for _, roleID := range member.Roles {
		if roleID == serverOpts.DiscordRoleID {
			// they have the role, so remove it
			err = s.GuildMemberRoleRemove(m.GuildID, m.Author.ID, serverOpts.DiscordRoleID)

			if err != nil {
				// meh, log the error here, and just return some feedback to the user
				logrus.WithError(err).Infof("failed to remove Discord role (make sure you have set the bot permissions to manage roles)")
				return fmt.Sprintf("You already have the %s role, and an error occurred trying to remove it.  Try again later.", roleName), nil
			}

			return fmt.Sprintf("You already had the %s role, it has now been removed.  Type the command again to add it back.", roleName), nil
		}
	}

	// they didn't have the role, so add it
	err = s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, serverOpts.DiscordRoleID)

	if err != nil {
		// meh, log the error here and return feedback to the user
		logrus.WithError(err).Infof("failed to set Discord role (make sure you have set the bot permissions to manage roles)")
		return fmt.Sprintf("A server error occurred trying to assign you the %s role, try again later", roleName), nil
	}

	// w00t!
	return fmt.Sprintf("The %s role has been assigned, you will now get pinged with notifications.  Type the command again to remove it.", roleName), nil
}

func (dm *DiscordManager) CommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	serverOpts, err := dm.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server opts")
		return
	}

	if m.Author.ID == s.State.User.ID {
		return
	}

	msg := ""

	switch m.Content {
	case "!schedule":
		msg, err = dm.CommandSchedule()
	case "!sessions":
		msg, err = dm.CommandSessions()
	case "!" + serverOpts.DiscordRoleCommand:
		msg, err = dm.CommandNotify(s, m)
	default:
		return
	}

	// if error, log it, but continue with message sending, handler may have put user feedback in msg
	if err != nil {
		logrus.WithError(err).Errorf("Error during handling of Discord command")
	}

	if msg != "" {
		_, err = s.ChannelMessageSend(m.ChannelID, msg)

		if err != nil {
			logrus.WithError(err).Errorf("couldn't send Discord msg")
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
func (dm *DiscordManager) SendMessage(title string, msg string) error {
	if dm.enabled {
		opts, err := dm.store.LoadServerOptions()

		if err != nil {
			logrus.WithError(err).Errorf("couldn't load server options")
			return err
		}

		// could check DiscordChannelID in new, but plan is to allow per-championship channels, so will need to pass
		// it in as an arg and check it here anyway
		if opts.DiscordChannelID != "" {
			if opts.DiscordRoleID != "" {
				mention := fmt.Sprintf("Attention <@&%s> - %s\n", opts.DiscordRoleID, title)
				messageSend := &discordgo.MessageSend{
					Content: mention,
					Embed:   embed.NewEmbed().SetDescription(msg).SetColor(0x1c1c1c).MessageEmbed,
				}
				_, err = dm.discord.ChannelMessageSendComplex(opts.DiscordChannelID, messageSend)
			} else {

				_, err = dm.discord.ChannelMessageSendEmbed(opts.DiscordChannelID, embed.NewGenericEmbed(title, msg))
			}

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
func (dm *DiscordManager) SendMessageWithLink(title string, msg string, linkText string, link *url.URL) error {
	if !dm.enabled {
		return nil
	}

	opts, err := dm.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
		return err
	}

	linkMsg := "[" + linkText + "](" + link.String() + ")"

	// could check DiscordChannelID in new, but plan is to allow per-championship channels, so will need to pass
	// it in as an arg and check it here anyway
	if opts.DiscordChannelID != "" {
		if opts.DiscordRoleID != "" {
			mention := fmt.Sprintf("Attention <@&%s> - %s\n", opts.DiscordRoleID, title)
			messageSend := &discordgo.MessageSend{
				Content: mention,
				Embed:   embed.NewEmbed().SetDescription(msg + "\n" + linkMsg).SetColor(0x1c1c1c).MessageEmbed,
			}
			_, err = dm.discord.ChannelMessageSendComplex(opts.DiscordChannelID, messageSend)
		} else {

			_, err = dm.discord.ChannelMessageSendEmbed(opts.DiscordChannelID, embed.NewGenericEmbed(title, msg+"\n"+linkMsg))
		}

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
