# discordgo-embed

[![GoDoc](https://godoc.org/github.com/Clinet/discordgo-embed?status.svg)](https://godoc.org/github.com/Clinet/discordgo-embed)
[![Go Report Card](https://goreportcard.com/badge/github.com/Clinet/discordgo-embed)](https://goreportcard.com/report/github.com/Clinet/discordgo-embed)

An embed helper library for DiscordGo.

# Installing
`go get github.com/clinet/discordgo-embed`

# Example
```
package main

import (
    "github.com/bwmarrin/discordgo"
    "github.com/Clinet/discordgo-embed"
)

// ...

discordSession.ChannelSendEmbed(channelID, embed.NewGenericEmbed("Example", "This is an example embed!"))
discordSession.ChannelSendEmbed(channelID, embed.NewErrorEmbed("Example Error", "This is an example error embed!"))
```

## License
The source code for discordgo-embed is released under the MIT License. See [LICENSE](https://raw.githubusercontent.com/clinet/discordgo-embed/master/LICENSE) for more details.

## Donations
All donations are appreciated and help me stay awake at night to work on this more. Even if it's not much, it helps a lot in the long run!

[![Donate](https://img.shields.io/badge/Donate-PayPal-green.svg)](https://paypal.me/JoshuaDoes)
