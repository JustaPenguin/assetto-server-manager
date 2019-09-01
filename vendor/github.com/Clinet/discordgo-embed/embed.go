package embed

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

//Embed ...
type Embed struct {
	*discordgo.MessageEmbed
}

// Constants for message embed character limits
const (
	EmbedLimitTitle       = 256
	EmbedLimitDescription = 2048
	EmbedLimitFieldValue  = 1024
	EmbedLimitFieldName   = 256
	EmbedLimitField       = 25
	EmbedLimitFooter      = 2048
	EmbedLimit            = 4000
)

//NewEmbed returns a new embed object
func NewEmbed() *Embed {
	return &Embed{&discordgo.MessageEmbed{}}
}

//SetTitle ...
func (e *Embed) SetTitle(name string) *Embed {
	e.Title = name
	return e
}

//SetDescription [desc]
func (e *Embed) SetDescription(description string) *Embed {
	if len(description) > 2048 {
		description = description[:2048]
	}
	e.Description = description
	return e
}

//AddField [name] [value]
func (e *Embed) AddField(name, value string) *Embed {
	fields := make([]*discordgo.MessageEmbedField, 0)

	if len(name) > EmbedLimitFieldName {
		name = name[:EmbedLimitFieldName]
	}

	if len(value) > EmbedLimitFieldValue {
		i := EmbedLimitFieldValue
		extended := false
		for i = EmbedLimitFieldValue; i < len(value); {
			if i != EmbedLimitFieldValue && extended == false {
				name += " (extended)"
				extended = true
			}
			if value[i] == []byte(" ")[0] || value[i] == []byte("\n")[0] || value[i] == []byte("-")[0] {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:  name,
					Value: value[i-EmbedLimitFieldValue : i],
				})
			} else {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:  name,
					Value: value[i-EmbedLimitFieldValue:i-1] + "-",
				})
				i--
			}

			if (i + EmbedLimitFieldValue) < len(value) {
				i += EmbedLimitFieldValue
			} else {
				break
			}
		}
		if i < len(value) {
			name += " (extended)"
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:  name,
				Value: value[i:],
			})
		}
	} else {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  name,
			Value: value,
		})
	}

	e.Fields = append(e.Fields, fields...)

	return e
}

//SetFooter [Text] [iconURL]
func (e *Embed) SetFooter(args ...string) *Embed {
	iconURL := ""
	text := ""
	proxyURL := ""

	switch {
	case len(args) > 2:
		proxyURL = args[2]
		fallthrough
	case len(args) > 1:
		iconURL = args[1]
		fallthrough
	case len(args) > 0:
		text = args[0]
	case len(args) == 0:
		return e
	}

	e.Footer = &discordgo.MessageEmbedFooter{
		IconURL:      iconURL,
		Text:         text,
		ProxyIconURL: proxyURL,
	}

	return e
}

//SetImage ...
func (e *Embed) SetImage(args ...string) *Embed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}
	e.Image = &discordgo.MessageEmbedImage{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

//SetThumbnail ...
func (e *Embed) SetThumbnail(args ...string) *Embed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}
	e.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

//SetAuthor ...
func (e *Embed) SetAuthor(args ...string) *Embed {
	var (
		name     string
		iconURL  string
		URL      string
		proxyURL string
	)

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		name = args[0]
	}
	if len(args) > 1 {
		iconURL = args[1]
	}
	if len(args) > 2 {
		URL = args[2]
	}
	if len(args) > 3 {
		proxyURL = args[3]
	}

	e.Author = &discordgo.MessageEmbedAuthor{
		Name:         name,
		IconURL:      iconURL,
		URL:          URL,
		ProxyIconURL: proxyURL,
	}

	return e
}

//SetURL ...
func (e *Embed) SetURL(URL string) *Embed {
	e.URL = URL
	return e
}

//SetColor ...
func (e *Embed) SetColor(clr int) *Embed {
	e.Color = clr
	return e
}

// InlineAllFields sets all fields in the embed to be inline
func (e *Embed) InlineAllFields() *Embed {
	for _, v := range e.Fields {
		v.Inline = true
	}
	return e
}

// Truncate truncates any embed value over the character limit.
func (e *Embed) Truncate() *Embed {
	e.TruncateDescription()
	e.TruncateFields()
	e.TruncateFooter()
	e.TruncateTitle()
	return e
}

// TruncateFields truncates fields that are too long
func (e *Embed) TruncateFields() *Embed {
	if len(e.Fields) > 25 {
		e.Fields = e.Fields[:EmbedLimitField]
	}

	for _, v := range e.Fields {

		if len(v.Name) > EmbedLimitFieldName {
			v.Name = v.Name[:EmbedLimitFieldName]
		}

		if len(v.Value) > EmbedLimitFieldValue {
			v.Value = v.Value[:EmbedLimitFieldValue]
		}

	}
	return e
}

// TruncateDescription ...
func (e *Embed) TruncateDescription() *Embed {
	if len(e.Description) > EmbedLimitDescription {
		e.Description = e.Description[:EmbedLimitDescription]
	}
	return e
}

// TruncateTitle ...
func (e *Embed) TruncateTitle() *Embed {
	if len(e.Title) > EmbedLimitTitle {
		e.Title = e.Title[:EmbedLimitTitle]
	}
	return e
}

// TruncateFooter ...
func (e *Embed) TruncateFooter() *Embed {
	if e.Footer != nil && len(e.Footer.Text) > EmbedLimitFooter {
		e.Footer.Text = e.Footer.Text[:EmbedLimitFooter]
	}
	return e
}

// NewGenericEmbed creates a new generic embed
func NewGenericEmbed(embedTitle, embedMsg string, replacements ...interface{}) *discordgo.MessageEmbed {
	genericEmbed := NewEmbed().
		SetTitle(embedTitle).
		SetDescription(fmt.Sprintf(embedMsg, replacements...)).
		SetColor(0x1c1c1c).MessageEmbed
	return genericEmbed
}

// NewGenericEmbedAdvanced creates a new generic embed with a custom color
func NewGenericEmbedAdvanced(embedTitle, embedMsg string, embedColor int) *discordgo.MessageEmbed {
	genericEmbed := NewEmbed().
		SetTitle(embedTitle).
		SetDescription(embedMsg).
		SetColor(embedColor).MessageEmbed
	return genericEmbed
}

// NewErrorEmbed creates a new error embed
func NewErrorEmbed(errorTitle, errorMsg string, replacements ...interface{}) *discordgo.MessageEmbed {
	errorEmbed := NewEmbed().
		SetTitle(errorTitle).
		SetDescription(fmt.Sprintf(errorMsg, replacements...)).
		SetColor(0xb40000).MessageEmbed
	return errorEmbed
}

// NewErrorEmbedAdvanced creates a new error embed with a custom color
func NewErrorEmbedAdvanced(errorTitle, errorMsg string, errorColor int) *discordgo.MessageEmbed {
	errorEmbed := NewEmbed().
		SetTitle(errorTitle).
		SetDescription(errorMsg).
		SetColor(errorColor).MessageEmbed
	return errorEmbed
}
