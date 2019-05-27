package servermanager

import (
	"encoding/json"
	"github.com/spkg/bom"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type Car struct {
	Name    string
	Skins   []string
	Tyres   map[string]string
	Details CarDetails
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

	tyres, err := ListTyres()

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
			continue
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
			Tyres: tyres[carFile.Name()],
		})
	}

	sort.Slice(cars, func(i, j int) bool {
		return cars[i].PrettyName() < cars[j].PrettyName()
	})

	return cars, nil
}

type CarDetails struct {
	Author      string    `json:"author"`
	Brand       string    `json:"brand"`
	Class       string    `json:"class"`
	Country     string    `json:"country"`
	Description string    `json:"description"`
	Name        string    `json:"name"`
	PowerCurve  [][]json.Number `json:"powerCurve"`
	Specs       CarSpecs  `json:"specs"`
	Tags        []string  `json:"tags"`
	TorqueCurve [][]json.Number `json:"torqueCurve"`
	URL         string    `json:"url"`
	Version     string    `json:"version"`
	Year        int64     `json:"year"`
}

type CarSpecs struct {
	Acceleration string `json:"acceleration"`
	Bhp          string `json:"bhp"`
	Pwratio      string `json:"pwratio"`
	Topspeed     string `json:"topspeed"`
	Torque       string `json:"torque"`
	Weight       string `json:"weight"`
}

func loadCarDetails(name string, tyres Tyres) (*Car, error) {
	carDirectory := filepath.Join(ServerInstallPath, "content", "cars", name)
	skinFiles, err := ioutil.ReadDir(filepath.Join(carDirectory, "skins"))

	if err != nil {
		return nil, err
	}

	var skins []string

	for _, skinFile := range skinFiles {
		if !skinFile.IsDir() {
			continue
		}

		skins = append(skins, skinFile.Name())
	}

	carDetailsBytes, err := ioutil.ReadFile(filepath.Join(carDirectory, "ui", "ui_car.json"))

	if err != nil {
		return nil, err
	}

	carDetailsBytes = bom.Clean(regexp.MustCompile(`\t*\r*\n*`).ReplaceAll(carDetailsBytes, []byte("")))

	var carDetails CarDetails

	if err := json.Unmarshal(carDetailsBytes, &carDetails); err != nil {
		return nil, err
	}

	return &Car{
		Name:    name,
		Skins:   skins,
		Tyres:   tyres[name],
		Details: carDetails,
	}, nil
}

func ResultsForCar(car string) ([]SessionResults, error) {
	results, err := ListAllResults()

	if err != nil {
		return nil, err
	}

	var out []SessionResults

	for _, result := range results {
		hasCar := false

		for _, driver := range result.Result {
			if driver.CarModel == car {
				hasCar = true
				break
			}
		}

		if hasCar {
			out = append(out, result)
		}
	}

	return out, nil
}

func carsHandler(w http.ResponseWriter, r *http.Request) {
	opts, err := raceManager.BuildRaceOpts(r)

	if err != nil {
		logrus.Errorf("could not get car list, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	opts["ShowSetups"] = r.URL.Query().Get("tab") == "setups"

	ViewRenderer.MustLoadTemplate(w, r, "content/cars.html", opts)
}

func apiCarUploadHandler(w http.ResponseWriter, r *http.Request) {
	uploadHandler(w, r, "Car")
}

func carDeleteHandler(w http.ResponseWriter, r *http.Request) {
	carName := chi.URLParam(r, "name")
	carsPath := filepath.Join(ServerInstallPath, "content", "cars")

	existingCars, err := ListCars()

	if err != nil {
		logrus.Errorf("could not get car list, err: %s", err)

		AddErrorFlash(w, r, "couldn't get car list")

		http.Redirect(w, r, r.Referer(), http.StatusFound)

		return
	}

	var found bool

	for _, car := range existingCars {
		if car.Name == carName {
			// Delete car
			found = true

			err := os.RemoveAll(filepath.Join(carsPath, carName))

			if err != nil {
				found = false
				logrus.Errorf("could not remove car files, err: %s", err)
			}

			break
		}
	}

	if found {
		// confirm deletion
		AddFlash(w, r, "Car successfully deleted!")
	} else {
		// inform car wasn't found
		AddErrorFlash(w, r, "Sorry, car could not be deleted. Are you sure it was installed?")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

const defaultSkinURL = "/static/img/no-preview-car.png"

func carSkinURL(car, skin string) string {
	skinPath := filepath.Join("content", "cars", car, "skins", skin, "preview.jpg")

	// look to see if the car preview image exists
	_, err := os.Stat(filepath.Join(ServerInstallPath, skinPath))

	if err != nil {
		return defaultSkinURL
	}

	return "/" + filepath.ToSlash(skinPath)
}

func carDetailsHandler(w http.ResponseWriter, r *http.Request) {
	tyres, err := ListTyres()

	if err != nil {
		panic(err)
	}

	carName := chi.URLParam(r, "car_id")

	car, err := loadCarDetails(carName, tyres)

	if err != nil {
		panic(err)
	}

	results, err := ResultsForCar(carName)

	if err != nil {
		panic(err)
	}

	ViewRenderer.MustLoadTemplate(w, r, "content/car-details.html", map[string]interface{}{
		"Car": car,
		"Results": results,
	})
}
