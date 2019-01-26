package servermanager

import (
	"fmt"
	"path/filepath"

	"gopkg.in/ini.v1"
)

const entryListFilename = "entry_list.ini"

type EntryList map[string]Entrant

// Write the EntryList to the server location
func (e EntryList) Write() error {
	f := ini.Empty()

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

type Entrant struct {
	Name string `ini:"DRIVERNAME"`
	Team string `ini:"TEAM"`
	GUID string `ini:"GUID"`

	Model string `ini:"MODEL"`
	Skin  string `ini:"SKIN"`

	Ballast       int `ini:"BALLAST"`
	SpectatorMode int `ini:"SPECTATOR_MODE"`
}
