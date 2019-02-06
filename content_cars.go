package servermanager

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/sirupsen/logrus"
)

type Car struct {
	Name  string
	Skins []string
}

func (c Car) PrettyName() string {
	return prettifyName(c.Name, true)
}

type Cars []Car

func (cs Cars) AsMap() map[string][]string {
	out := make(map[string][]string)

	for _, car := range cs {
		out[car.Name] = car.Skins
	}

	return out
}

func ListCars() (Cars, error) {
	var cars Cars

	carFiles, err := ioutil.ReadDir(filepath.Join(ServerInstallPath, "content", "cars"))

	if err != nil {
		return nil, err
	}

	for _, carFile := range carFiles {
		if !carFile.IsDir() {
			continue
		}

		skinFiles, err := ioutil.ReadDir(filepath.Join(ServerInstallPath, "content", "cars", carFile.Name(), "skins"))

		if err != nil && !os.IsNotExist(err) {
			// just load without skins. non-fatal
			logrus.Errorf("couldn't read car dir, err: %s", err)
		}

		var skins []string

		for _, skinFile := range skinFiles {
			if !skinFile.IsDir() {
				continue
			}

			skins = append(skins, skinFile.Name())
		}

		cars = append(cars, Car{
			Name:  carFile.Name(),
			Skins: skins,
		})
	}

	sort.Slice(cars, func(i, j int) bool {
		return cars[i].PrettyName() < cars[j].PrettyName()
	})

	return cars, nil
}
