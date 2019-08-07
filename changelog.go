package servermanager

// Pack changelog into this package
//go:generate esc -o changelog_embed.go -pkg=servermanager CHANGELOG.md

func LoadChangelog() ([]byte, error) {
	return FSByte(false, "/CHANGELOG.md")
}
