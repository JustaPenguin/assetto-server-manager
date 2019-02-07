package servermanager

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cj123/ini"
)

const entryListFilename = "entry_list.ini"

type EntryList map[string]Entrant

// Write the EntryList to the server location
func (e EntryList) Write() error {
	f := ini.NewFile([]ini.DataSource{nil}, ini.LoadOptions{
		IgnoreInlineComment: true,
	})

	// making and throwing away a default section due to the utter insanity of ini or assetto. i don't know which.
	_, err := f.NewSection("DEFAULT")

	if err != nil {
		return err
	}

	for k, v := range e {
		s, err := f.NewSection(k)

		if err != nil {
			return err
		}

		err = s.ReflectFrom(&v)

		if err != nil {
			return err
		}
	}

	return f.SaveTo(filepath.Join(ServerInstallPath, ServerConfigPath, entryListFilename))
}

// Add an Entrant to the EntryList
func (e EntryList) Add(entrant Entrant) {
	e[fmt.Sprintf("CAR_%d", len(e))] = entrant
}

// Remove an Entrant from the EntryList
func (e EntryList) Delete(entrant Entrant) {
	for k, v := range e {
		if v == entrant {
			delete(e, k)
			return
		}
	}
}

func (e EntryList) Entrants() string {
	var entrants []string

	for _, x := range e {
		entrants = append(entrants, x.Name)
	}

	return strings.Join(entrants, ", ")
}

type Entrant struct {
	Name string `ini:"DRIVERNAME"`
	Team string `ini:"TEAM"`
	GUID string `ini:"GUID"`

	Model string `ini:"MODEL"`
	Skin  string `ini:"SKIN"`

	Ballast       int `ini:"BALLAST"`
	SpectatorMode int `ini:"SPECTATOR_MODE"`
	Restrictor    int `ini:"RESTRICTOR"`
}

func (e Entrant) ID() string {
	if e.GUID != "" {
		return e.GUID
	} else {
		return e.Name
	}
}
