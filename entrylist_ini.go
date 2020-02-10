package servermanager

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cj123/ini"
	"github.com/google/uuid"
)

const (
	AnyCarModel       = "any_car_model"
	entryListFilename = "entry_list.ini"
)

type EntryList map[string]*Entrant

// Write the EntryList to the server location
func (e EntryList) Write() error {
	setupDirectory := filepath.Join(ServerInstallPath, "setups")

	// belt and braces check to make sure setup file exists
	for _, entrant := range e.AsSlice() {
		if entrant.FixedSetup != "" {
			if _, err := os.Stat(filepath.Join(setupDirectory, entrant.FixedSetup)); os.IsNotExist(err) {
				return err
			}
		}
	}

	for i, entrant := range e.AsSlice() {
		entrant.PitBox = i
	}

	f := ini.NewFile([]ini.DataSource{nil}, ini.LoadOptions{
		IgnoreInlineComment: true,
	})

	// making and throwing away a default section due to the utter insanity of ini or assetto. i don't know which.
	_, err := f.NewSection("DEFAULT")

	if err != nil {
		return err
	}

	for _, v := range e {
		s, err := f.NewSection(fmt.Sprintf("CAR_%d", v.PitBox))

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
	e.AddInPitBox(entrant, len(e))
}

// AddInPitBox adds an Entrant in a specific pitbox - overwriting any entrant that was in that pitbox previously.
func (e EntryList) AddInPitBox(entrant *Entrant, pitBox int) {
	entrant.PitBox = pitBox
	e[fmt.Sprintf("CAR_%d", pitBox)] = entrant
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
		return entrants[i].PitBox < entrants[j].PitBox
	})

	return entrants
}

func (e EntryList) AlphaSlice() []*Entrant {
	var entrants []*Entrant

	for _, x := range e {
		entrants = append(entrants, x)
	}

	sort.Slice(entrants, func(i, j int) bool {
		return entrants[i].Name < entrants[j].Name
	})

	return entrants
}

func (e EntryList) PrettyList() []*Entrant {
	var entrants []*Entrant

	numOpenSlots := 0

	for _, x := range e {
		if x.GUID == "" {
			numOpenSlots++
			continue
		}

		if x.Model == AnyCarModel {
			continue
		}

		entrants = append(entrants, x)
	}

	sort.Slice(entrants, func(i, j int) bool {
		return entrants[i].Name < entrants[j].Name
	})

	entrants = append(entrants, &Entrant{
		Name: fmt.Sprintf("%d open slots", numOpenSlots),
		GUID: "OPEN_SLOTS",
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
			entrants = append(entrants, driverName(x.Name))
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

// CarIDs returns a unique list of car IDs used in the EntryList
func (e EntryList) CarIDs() []string {
	cars := make(map[string]bool)

	for _, entrant := range e {
		cars[entrant.Model] = true
	}

	var out []string

	for car := range cars {
		out = append(out, car)
	}

	return out
}

// returns the greatest ballast set on any entrant
func (e EntryList) FindGreatestBallast() int {
	var greatest int

	for _, entrant := range e {
		if entrant.Ballast > greatest {
			greatest = entrant.Ballast
		}
	}

	return greatest
}

func NewEntrant() *Entrant {
	return &Entrant{
		InternalUUID: uuid.New(),
	}
}

type Entrant struct {
	InternalUUID uuid.UUID `ini:"-"`
	PitBox       int       `ini:"-"`

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
	IsPlaceHolder      bool `ini:"-"`
}

func (e Entrant) ID() string {
	if e.GUID != "" {
		return e.GUID
	}

	return e.Name
}

func (e *Entrant) OverwriteProperties(other *Entrant) {
	e.FixedSetup = other.FixedSetup
	e.Restrictor = other.Restrictor
	e.SpectatorMode = other.SpectatorMode
	e.Ballast = other.Ballast
	e.Skin = other.Skin
	e.PitBox = other.PitBox
}

func (e *Entrant) SwapProperties(other *Entrant, entrantRemainedInClass bool) {
	if entrantRemainedInClass {
		e.Model, other.Model = other.Model, e.Model
		e.Skin, other.Skin = other.Skin, e.Skin
		e.FixedSetup, other.FixedSetup = other.FixedSetup, e.FixedSetup
		e.Restrictor, other.Restrictor = other.Restrictor, e.Restrictor
		e.Ballast, other.Ballast = other.Ballast, e.Ballast
	}

	e.Team, other.Team = other.Team, e.Team
	e.InternalUUID, other.InternalUUID = other.InternalUUID, e.InternalUUID
	e.PitBox, other.PitBox = other.PitBox, e.PitBox
}

func (e *Entrant) AssignFromResult(result *SessionResult, car *SessionCar) {
	e.Name = result.DriverName
	e.Team = car.Driver.Team
	e.GUID = result.DriverGUID
	e.Model = result.CarModel
	e.Skin = car.Skin
	e.Restrictor = car.Restrictor
	e.Ballast = car.BallastKG
}

func (e *Entrant) AsSessionCar() *SessionCar {
	return &SessionCar{
		BallastKG: e.Ballast,
		CarID:     e.PitBox,
		Driver: SessionDriver{
			GUID:      e.GUID,
			GuidsList: []string{e.GUID},
			Name:      e.Name,
			Team:      e.Team,
		},
		Model:      e.Model,
		Restrictor: e.Restrictor,
		Skin:       e.Skin,
	}
}

func (e *Entrant) AsSessionResult() *SessionResult {
	return &SessionResult{
		BallastKG:  e.Ballast,
		CarID:      e.PitBox,
		CarModel:   e.Model,
		DriverGUID: e.GUID,
		DriverName: e.Name,
		Restrictor: e.Restrictor,
	}
}
