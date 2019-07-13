package servermanager

import (
	"encoding/json"
	"github.com/blevesearch/bleve/search/query"
	"github.com/davecgh/go-spew/spew"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/blevesearch/bleve"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/spkg/bom"
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

type Cars []*Car

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

		car, err := loadCarDetails(carFile.Name(), tyres)

		if err != nil && os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}

		cars = append(cars, car)
	}

	sort.Slice(cars, func(i, j int) bool {
		return cars[i].PrettyName() < cars[j].PrettyName()
	})

	return cars, nil
}

type CarDetails struct {
	Author      string          `json:"author"`
	Brand       string          `json:"brand"`
	Class       string          `json:"class"`
	Country     string          `json:"country"`
	Description string          `json:"description"`
	Name        string          `json:"name"`
	PowerCurve  [][]json.Number `json:"powerCurve"`
	Specs       CarSpecs        `json:"specs"`
	Tags        []string        `json:"tags"`
	TorqueCurve [][]json.Number `json:"torqueCurve"`
	URL         string          `json:"url"`
	Version     string          `json:"version"`
	Year        int64           `json:"year"`
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

const searchPageSize = 50

func carsHandler(w http.ResponseWriter, r *http.Request) {
	var q query.Query

	searchTerm := r.URL.Query().Get("q")

	if searchTerm == "" {
		q = bleve.NewMatchAllQuery()
	} else {
		q = bleve.NewQueryStringQuery(searchTerm)
	}

	results, err := carIndex.Search(bleve.NewSearchRequestOptions(q, searchPageSize, 0, false))

	if err != nil {
		logrus.Errorf("could not get car list, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	cars := make(map[string]*Car)

	for _, hit := range results.Hits {
		cars[hit.ID], err = loadCarDetails(hit.ID, nil)

		if err != nil {
			panic(err)
		}
	}

	//currentPage := formValueAsInt(r.URL.Query().Get("p"))

	ViewRenderer.MustLoadTemplate(w, r, "content/cars.html", map[string]interface{}{
		"Results": results,
		"Cars":    cars,
		"Query":   searchTerm,
	})
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
		AddFlash(w, r, "Car successfully deleted!")
	} else {
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
		"Car":     car,
		"Results": results,
	})
}

var carIndex bleve.Index

func createCarDetailsIndex() bleve.Index {
	indexMapping := bleve.NewIndexMapping()

	index, err := bleve.NewMemOnly(indexMapping)

	if err != nil {
		panic(err)
	}

	return index
}

func InitCarIndex() {
	var err error

	carIndex = createCarDetailsIndex()

	cars, err := ListCars()

	if err != nil {
		panic(err)
	}

	for _, car := range cars {
		err := carIndex.Index(car.Name, car.Details)

		if err != nil {
			panic(err)
		}
	}

	rq := bleve.NewSearchRequest(bleve.NewQueryStringQuery(`tags:"rss"`))
	rq.Size = 100

	deets, err := carIndex.Search(rq)

	spew.Dump(deets)
}
