package servermanager

import (
	"github.com/bwmarrin/discordgo"
)

type DiscordManager struct {
	discord *discordgo.Session
	enabled bool
}

func NewDiscordManager() *DiscordManager {
	cfg := ConfigIniDefault
	var session *discordgo.Session

	if cfg.GlobalServerConfig.DiscordAPIToken != "" {
		var err error

		session, _ = discordgo.New("Bot " + cfg.GlobalServerConfig.DiscordAPIToken)
		err = session.Open()

		if err != nil {
			return &DiscordManager{
				discord: session,
				enabled: false,
			}
		}
	}

	return &DiscordManager{
		discord: session,
		enabled: true,
	}
}

func (dm *DiscordManager) SendMessage(msg string) error {
	if dm.enabled {
		var err error
		_, err = dm.discord.ChannelMessageSend(ConfigIniDefault.GlobalServerConfig.DiscordChannelID, msg)
		//err = nil

		if err != nil {
			return err
		}
	}

	return nil
}
