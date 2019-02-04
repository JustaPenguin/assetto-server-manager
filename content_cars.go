package servermanager

import (
	"io/ioutil"
	"path/filepath"
	"sort"
)

type Car struct {
	Name  string
	Skins []string
}

func (c Car) PrettyName() string {
	return prettifyName(c.Name, true)
}

func ListCars() ([]Car, error) {
	var cars []Car

	carFiles, err := ioutil.ReadDir(filepath.Join(ServerInstallPath, "content", "cars"))

	if err != nil {
		return nil, err
	}

	for _, carFile := range carFiles {
		cars = append(cars, Car{
			Name: carFile.Name(),
		})
	}

	sort.Slice(cars, func(i, j int) bool {
		return cars[i].PrettyName() < cars[j].PrettyName()
	})

	return cars, nil
}
