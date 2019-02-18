package servermanager

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cj123/assetto-server-manager/pkg/acd"

	"github.com/cj123/ini"
	"github.com/sirupsen/logrus"
)

var tyreFiles = []string{
	"ks_tyres.ini",
	"mod_tyres.ini",
}

type Tyres map[string]map[string]string

func (t Tyres) Name(search string, cars []string) string {
	carExists := make(map[string]bool)

	for _, car := range cars {
		carExists[car] = true
	}

	for car, carTyres := range t {
		if _, ok := carExists[car]; !ok {
			continue
		}

		if name, ok := carTyres[search]; ok {
			return name
		}
	}

	return search
}

// ListTyres reads tyres from the TyreFiles and returns them as a map of car => tyre short name => tyre long name
func ListTyres() (Tyres, error) {
	out := make(Tyres)

	for _, file := range tyreFiles {
		tyres, err := loadTyresFromFile(file)

		if os.IsNotExist(err) {
			logrus.Debugf("Skipping loading tyre data from %s, file does not exist", file)
			continue
		} else if err != nil {
			return nil, err
		}

		for car, carTyres := range tyres {
			if _, ok := out[car]; !ok {
				out[car] = make(map[string]string)
			}

			for key, tyreName := range carTyres {
				out[car][key] = tyreName
			}
		}
	}

	return out, nil
}

func loadTyresFromFile(name string) (map[string]map[string]string, error) {
	f, err := os.Open(filepath.Join(ServerInstallPath, "manager", name))

	if err != nil {
		return nil, err
	}

	defer f.Close()

	i, err := ini.Load(f)

	if err != nil {
		return nil, err
	}

	out := make(map[string]map[string]string)

	for _, car := range i.Sections() {
		out[car.Name()] = car.KeysHash()
	}

	return out, nil
}

// CarNameFromFilepath takes a filepath e.g. content/cars/rss_formula_rss_4/data.acd and returns the car name, e.g.
// in this case: rss_formula_rss_4.
func CarNameFromFilepath(path string) (string, error) {
	parts := strings.Split(filepath.ToSlash(filepath.Dir(path)), "/")

	if len(parts) == 0 {
		return "", fmt.Errorf("servermanager: can't get car name from path: %s", path)
	}

	return parts[len(parts)-1], nil
}

// addTyresForNewCar looks for tyres within the data.acd file and adds them to mod_tyres.ini if any are found
func addTyresForNewCar(filename string, dataACD []byte) error {
	carName, err := CarNameFromFilepath(filename)

	if err != nil {
		return err
	}

	currentTyres, err := ListTyres()

	if err != nil {
		return err
	}

	r, err := acd.NewReader(bytes.NewReader(dataACD), carName)

	if err != nil {
		return err
	}

	for _, file := range r.Files {
		if file.Name() != "tyres.ini" {
			continue
		}

		newTyres, err := LoadTyresFromACDINI(file.Bytes())

		if err != nil {
			return err
		}

		if carTyres, ok := currentTyres[carName]; ok {
			hasNew := false

			for newTyre := range newTyres {
				if _, ok := carTyres[newTyre]; !ok {
					hasNew = true
				}
			}

			if !hasNew {
				logrus.Infof("New car: %s already has all of its tyres configured", carName)
				return nil
			}
		}

		return addModTyres(carName, newTyres)
	}

	logrus.Warnf("Couldn't find tyres.ini within filepath: '%s'. Cannot add mod_tyres.ini", filename)

	return nil
}

// addModTyres writes a set of tyres to the mod_tyres.ini file
func addModTyres(model string, tyres map[string]string) error {
	modTyresFilename := filepath.Join(ServerInstallPath, "manager", "mod_tyres.ini")

	if _, err := os.Stat(modTyresFilename); os.IsNotExist(err) {
		f, err := os.Create(modTyresFilename)

		if err != nil {
			return err
		}

		err = f.Close()

		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	i, err := ini.Load(modTyresFilename)

	if err != nil {
		return err
	}

	sec, err := i.NewSection(model)

	if err != nil {
		return err
	}

	for k, v := range tyres {
		_, err := sec.NewKey(k, v)

		if err != nil {
			return err
		}
	}

	return i.SaveTo(modTyresFilename)
}

// LoadTyresFromACDINI reads the tyres.ini file from within the data.acd of a car and finds all available tyre compounds.
func LoadTyresFromACDINI(data []byte) (map[string]string, error) {
	f, err := ini.Load(data)

	if err != nil {
		return nil, err
	}

	outTyres := make(map[string]string)

	for _, sec := range f.Sections() {
		vals := sec.KeysHash()

		name, hasName := vals["NAME"]
		key, hasKey := vals["SHORT_NAME"]

		if hasName && hasKey {
			outTyres[key] = name
		}
	}

	return outTyres, nil
}

type TrackSurfacePreset struct {
	Name            string
	Description     string
	SessionStart    int
	SessionTransfer int
	Randomness      int
	LapGain         int
}

var DefaultTrackSurfacePresets = []TrackSurfacePreset{
	{
		Name:            "Dusty",
		SessionStart:    86,
		SessionTransfer: 50,
		Randomness:      1,
		LapGain:         30,
		Description:     "A very slippery track, improves fast with more laps.",
	},
	{
		Name:            "Old",
		SessionStart:    89,
		SessionTransfer: 80,
		Randomness:      3,
		LapGain:         50,
		Description:     "Old tarmac. Bad grip that won't get better soon.",
	},
	{
		Name:            "Slow",
		SessionStart:    96,
		SessionTransfer: 80,
		Randomness:      1,
		LapGain:         300,
		Description:     "A slow track that doesn't improve much.",
	},
	{
		Name:            "Green",
		SessionStart:    95,
		SessionTransfer: 90,
		Randomness:      2,
		LapGain:         132,
		Description:     "A clean track, gets better with more laps.",
	},
	{
		Name:            "Fast",
		SessionStart:    98,
		SessionTransfer: 80,
		Randomness:      2,
		LapGain:         700,
		Description:     "Very grippy track right from the start.",
	},
	{
		Name:            "Optimum",
		SessionStart:    100,
		SessionTransfer: 100,
		Randomness:      0,
		LapGain:         1,
		Description:     "Perfect track for hotlapping.",
	},
}
