package servermanager

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cj123/ini"
	"github.com/google/uuid"
)

const entryListFilename = "entry_list.ini"

type EntryList map[string]*Entrant

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
func (e EntryList) Add(entrant *Entrant) {
	e[fmt.Sprintf("CAR_%d", len(e))] = entrant
}

// Remove an Entrant from the EntryList
func (e EntryList) Delete(entrant *Entrant) {
	for k, v := range e {
		if v == entrant {
			delete(e, k)
			return
		}
	}
}

func (e EntryList) AsSlice() []*Entrant {
	var entrants []*Entrant

	for _, x := range e {
		entrants = append(entrants, x)
	}

	sort.Slice(entrants, func(i, j int) bool {
		if entrants[i].Team == entrants[j].Team {
			return entrants[i].Name < entrants[j].Name
		} else {
			return entrants[i].Team < entrants[j].Team
		}
	})

	return entrants
}

func (e EntryList) Entrants() string {
	var entrants []string

	numOpenSlots := 0

	for _, x := range e {
		if x.Name == "" {
			numOpenSlots++
		} else {
			entrants = append(entrants, x.Name)
		}
	}

	if numOpenSlots > 0 {
		entrants = append(entrants, fmt.Sprintf("%d open slots", numOpenSlots))
	}

	return strings.Join(entrants, ", ")
}

func (e EntryList) FindEntrantByInternalUUID(internalUUID uuid.UUID) *Entrant {
	for _, entrant := range e {
		if entrant.InternalUUID == internalUUID {
			return entrant
		}
	}

	return &Entrant{}
}

func NewEntrant() *Entrant {
	return &Entrant{
		InternalUUID: uuid.New(),
	}
}

type Entrant struct {
	InternalUUID uuid.UUID `ini:"-"`

	Name string `ini:"DRIVERNAME"`
	Team string `ini:"TEAM"`
	GUID string `ini:"GUID"`

	Model string `ini:"MODEL"`
	Skin  string `ini:"SKIN"`

	Ballast       int    `ini:"BALLAST"`
	SpectatorMode int    `ini:"SPECTATOR_MODE"`
	Restrictor    int    `ini:"RESTRICTOR"`
	FixedSetup    string `ini:"FIXED_SETUP"`

	TransferTeamPoints bool `ini:"-" json:"-"`
	OverwriteAllEvents bool `ini:"-" json:"-"`
}

func (e Entrant) ID() string {
	if e.GUID != "" {
		return e.GUID
	} else {
		return e.Name
	}
}

func (e Entrant) OverwriteProperties(other *Entrant) {
	e.FixedSetup = other.FixedSetup
	e.Restrictor = other.Restrictor
	e.SpectatorMode = other.SpectatorMode
	e.Ballast = other.Ballast
	e.Skin = other.Skin
}
