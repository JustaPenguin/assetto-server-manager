package servermanager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
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

func (cd *CarDetails) AddTag(name string) {
	for _, tag := range cd.Tags {
		if tag == name {
			// tag exists
			return
		}
	}

	cd.Tags = append(cd.Tags, name)
}

func (cd *CarDetails) DelTag(name string) {
	deleteIndex := -1

	for index, tag := range cd.Tags {
		if tag == name {
			deleteIndex = index
		}
	}

	if deleteIndex == -1 {
		return
	}

	cd.Tags = append(cd.Tags[:deleteIndex], cd.Tags[deleteIndex+1:]...)
}

func (cd *CarDetails) Save(carName string) error {
	f, err := os.Create(filepath.Join(ServerInstallPath, "content", "cars", carName, "ui", "ui_car.json"))

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "   ")

	return enc.Encode(cd)
}

func (cd *CarDetails) Load(carName string) error {
	carDetailsBytes, err := ioutil.ReadFile(filepath.Join(ServerInstallPath, "content", "cars", carName, "ui", "ui_car.json"))

	if err != nil {
		return err
	}

	carDetailsBytes = bom.Clean(regexp.MustCompile(`\t*\r*\n*`).ReplaceAll(carDetailsBytes, []byte("")))

	return json.Unmarshal(carDetailsBytes, &cd)
}

type CarSpecs struct {
	Acceleration string `json:"acceleration"`
	Bhp          string `json:"bhp"`
	Pwratio      string `json:"pwratio"`
	Topspeed     string `json:"topspeed"`
	Torque       string `json:"torque"`
	Weight       string `json:"weight"`
}

type CarManager struct {
	carIndex bleve.Index
}

func NewCarManager() *CarManager {
	return &CarManager{}
}

func (cm *CarManager) ListCars() (Cars, error) {
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

		car, err := cm.LoadCar(carFile.Name(), tyres)

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

// LoadCar reads a car from the content folder on the filesystem
func (cm *CarManager) LoadCar(name string, tyres Tyres) (*Car, error) {
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

	carDetails := CarDetails{}

	if err := carDetails.Load(name); err != nil {
		return nil, err
	}

	return &Car{
		Name:    name,
		Skins:   skins,
		Tyres:   tyres[name],
		Details: carDetails,
	}, nil
}

// ResultsForCar finds results for a given car.
func (cm *CarManager) ResultsForCar(car string) ([]SessionResults, error) {
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

// DeleteCar removes a car from the file system and search index.
func (cm *CarManager) DeleteCar(carName string) error {
	carsPath := filepath.Join(ServerInstallPath, "content", "cars")

	existingCars, err := cm.ListCars()

	if err != nil {
		return err
	}

	for _, car := range existingCars {
		if car.Name != carName {
			continue
		}

		err := os.RemoveAll(filepath.Join(carsPath, carName))

		if err != nil {
			return err
		}

		break
	}

	return cm.DeIndexCar(carName)
}

const searchPageSize = 50

// CreateSearchIndex builds a search index for the cars
// @TODO this really should use a filesystem index.
func (cm *CarManager) CreateSearchIndex() error {
	indexMapping := bleve.NewIndexMapping()

	index, err := bleve.NewMemOnly(indexMapping)

	if err != nil {
		return err
	}

	cm.carIndex = index

	return nil
}

// IndexCar indexes an individual car.
func (cm *CarManager) IndexCar(car *Car) error {
	return cm.carIndex.Index(car.Name, car.Details)
}

// DeIndexCar removes a car from the index.
func (cm *CarManager) DeIndexCar(name string) error {
	return cm.carIndex.Delete(name)
}

// IndexAllCars loads all current cars and adds them to the search index
// @TODO building the index and doing this should be a migration
func (cm *CarManager) IndexAllCars() error {
	cars, err := cm.ListCars()

	if err != nil {
		return err
	}

	for _, car := range cars {
		err := cm.IndexCar(car)

		if err != nil {
			return err
		}
	}

	return nil
}

// Search looks for cars in the search index.
func (cm *CarManager) Search(term string, from int) (*bleve.SearchResult, map[string]*Car, error) {
	var q query.Query

	if term == "" {
		q = bleve.NewMatchAllQuery()
	} else {
		q = bleve.NewQueryStringQuery(term)
	}

	results, err := cm.carIndex.Search(bleve.NewSearchRequestOptions(q, searchPageSize, from, false))

	if err != nil {
		return nil, nil, err
	}

	cars := make(map[string]*Car)

	for _, hit := range results.Hits {
		cars[hit.ID], err = cm.LoadCar(hit.ID, nil)

		if err != nil {
			return nil, nil, err
		}
	}

	return results, cars, nil
}

func (cm *CarManager) AddTag(carName, tag string) error {
	car, err := cm.LoadCar(carName, nil)

	if err != nil {
		return err
	}

	car.Details.AddTag(tag)

	return cm.SaveCarDetails(carName, car)
}

func (cm *CarManager) DelTag(carName, tag string) error {
	car, err := cm.LoadCar(carName, nil)

	if err != nil {
		return err
	}

	car.Details.DelTag(tag)

	return cm.SaveCarDetails(carName, car)
}

// SaveCarDetails saves a car's details, and indexes that car.
func (cm *CarManager) SaveCarDetails(carName string, car *Car) error {
	if err := car.Details.Save(carName); err != nil {
		return err
	}

	return cm.IndexCar(car)
}

// LoadCarDetailsForTemplate loads all necessary items to generate the car details template.
func (cm *CarManager) LoadCarDetailsForTemplate(carName string) (map[string]interface{}, error) {
	tyres, err := ListTyres()

	if err != nil {
		return nil, err
	}

	car, err := cm.LoadCar(carName, tyres)

	if err != nil {
		return nil, err
	}

	results, err := cm.ResultsForCar(carName)

	if err != nil {
		return nil, err
	}

	setups, err := ListSetupsForCar(carName)

	if err != nil {
		return nil, err
	}

	tracks, err := ListTracks()

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"Car":       car,
		"Results":   results,
		"Setups":    setups,
		"TrackOpts": tracks,
	}, nil
}

type CarsHandler struct {
	*BaseHandler

	carManager *CarManager
}

func NewCarsHandler(baseHandler *BaseHandler, carManager *CarManager) *CarsHandler {
	return &CarsHandler{
		BaseHandler: baseHandler,
		carManager:  carManager,
	}
}

func (ch *CarsHandler) list(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")
	results, cars, err := ch.carManager.Search(searchTerm, 0) // @TODO pagination

	if err != nil {
		logrus.WithError(err).Error("Could not perform search")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "content/cars.html", map[string]interface{}{
		"Results": results,
		"Cars":    cars,
		"Query":   searchTerm,
	})
}

func (ch *CarsHandler) delete(w http.ResponseWriter, r *http.Request) {
	carName := chi.URLParam(r, "name")
	err := ch.carManager.DeleteCar(carName)

	if err != nil {
		logrus.WithError(err).Errorf("Could not delete car: %s", carName)
		AddErrorFlash(w, r, "couldn't get car list")
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	AddFlash(w, r, fmt.Sprintf("Car %s successfully deleted!", carName))
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

func (ch *CarsHandler) view(w http.ResponseWriter, r *http.Request) {
	carName := chi.URLParam(r, "car_id")
	templateParams, err := ch.carManager.LoadCarDetailsForTemplate(carName)

	if err != nil {
		logrus.WithError(err).Errorf("Could not load car details for: %s", carName)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "content/car-details.html", templateParams)
}

func (ch *CarsHandler) tags(w http.ResponseWriter, r *http.Request) {
	car := chi.URLParam(r, "car_id")

	if r.Method == http.MethodPost {
		tag := r.FormValue("new-tag")
		err := ch.carManager.AddTag(car, tag)

		if err == nil {
			AddFlash(w, r, fmt.Sprintf("Successfully added the tag: %s", tag))
		} else {
			AddFlash(w, r, "Could not add tag")
		}
	} else {
		tag := r.URL.Query().Get("delete")
		err := ch.carManager.DelTag(car, tag)

		if err == nil {
			AddFlash(w, r, fmt.Sprintf("Successfully deleted the tag: %s", tag))
		} else {
			AddFlash(w, r, "Could not delete tag")
		}
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
