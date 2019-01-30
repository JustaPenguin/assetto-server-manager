package servermanager

import "io/ioutil"

type Car struct {
	Name  string
	Skins []string
}

func ListCars() ([]Car, error) {
	var cars []Car

	carFiles, err := ioutil.ReadDir(ServerInstallPath + "/content/cars")

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
