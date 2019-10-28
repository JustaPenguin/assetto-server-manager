package changelog

import (
	"html/template"

	"github.com/russross/blackfriday"
)

// Pack changelog into this package
//go:generate esc -o changelog_embed.go -pkg=changelog ../../CHANGELOG.md

func LoadChangelog() (template.HTML, error) {
	changelog, err := FSByte(false, "/CHANGELOG.md")

	if err != nil {
		return "", err
	}

	return template.HTML(blackfriday.Run(changelog)), nil
}
