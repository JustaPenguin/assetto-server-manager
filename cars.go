package servermanager

import (
	"io/ioutil"
	"path/filepath"
)

type Car struct {
	Name  string
	Skins []string
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

	return cars, nil
}
