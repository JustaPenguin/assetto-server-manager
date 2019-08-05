package servermanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spkg/bom"
	"golang.org/x/sync/errgroup"
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
	Author       string          `json:"author"`
	Brand        string          `json:"brand"`
	Class        string          `json:"class"`
	Country      string          `json:"country"`
	Description  string          `json:"description"`
	Name         string          `json:"name"`
	PowerCurve   [][]json.Number `json:"powerCurve"`
	Specs        CarSpecs        `json:"specs"`
	SpecsNumeric CarSpecsNumeric `json:"spec"`
	Tags         []string        `json:"tags"`
	TorqueCurve  [][]json.Number `json:"torqueCurve"`
	URL          string          `json:"url"`
	Version      string          `json:"version"`
	Year         ShouldBeAnInt   `json:"year"`

	DownloadURL string `json:"downloadURL"`
	Notes       string `json:"notes"`
}

// ShouldBeAnInt can be used in JSON struct definitions in places where the value provided should be an int, but isn't.
type ShouldBeAnInt int

func (i *ShouldBeAnInt) UnmarshalJSON(b []byte) error {
	var number int

	err := json.Unmarshal(b, &number)

	if err != nil {
		var str string

		err := json.Unmarshal(b, &str)

		if err != nil {
			return err
		}

		*i = ShouldBeAnInt(formValueAsInt(str))
	} else {
		*i = ShouldBeAnInt(number)
	}

	return nil
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
	uiDirectory := filepath.Join(ServerInstallPath, "content", "cars", carName, "ui")

	err := os.MkdirAll(uiDirectory, 0755)

	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(uiDirectory, "ui_car.json"))

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

	err = json.Unmarshal(carDetailsBytes, &cd)

	if err != nil {
		return err
	}

	cd.SpecsNumeric = cd.Specs.Numeric()

	return nil
}

type CarSpecs struct {
	Acceleration string `json:"acceleration"`
	BHP          string `json:"bhp"`
	PWRatio      string `json:"pwratio"`
	TopSpeed     string `json:"topspeed"`
	Torque       string `json:"torque"`
	Weight       string `json:"weight"`
}

type CarSpecsNumeric struct {
	Acceleration int `json:"acceleration"`
	BHP          int `json:"bhp"`
	PWRatio      int `json:"pwratio"`
	TopSpeed     int `json:"topspeed"`
	Torque       int `json:"torque"`
	Weight       int `json:"weight"`
}

var keepNumericRegex = regexp.MustCompile(`[0-9]+`)

func toNumber(str string) int {
	str = keepNumericRegex.FindString(str)

	return formValueAsInt(str)
}

func (cs CarSpecs) Numeric() CarSpecsNumeric {
	return CarSpecsNumeric{
		Acceleration: toNumber(cs.Acceleration),
		BHP:          toNumber(cs.BHP),
		PWRatio:      toNumber(cs.PWRatio),
		TopSpeed:     toNumber(cs.TopSpeed),
		Torque:       toNumber(cs.Torque),
		Weight:       toNumber(cs.Weight),
	}
}

type CarManager struct {
	carIndex bleve.Index

	searchIndexRebuildMutex sync.Mutex
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

	var skins []string

	if err == nil {
		for _, skinFile := range skinFiles {
			if !skinFile.IsDir() {
				continue
			}

			skins = append(skins, skinFile.Name())
		}
	} else {
		logrus.WithError(err).Warnf("Could not load skins for car: %s", name)
	}

	carDetails := CarDetails{}

	if err := carDetails.Load(name); err != nil && os.IsNotExist(err) {
		// the car details don't exist, just create some fake ones.
		carDetails.Name = prettifyName(name, true)
	} else if err != nil {
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
func (cm *CarManager) CreateOrOpenSearchIndex() error {
	indexPath := filepath.Join(ServerInstallPath, "search-index", "cars")

	var err error

	cm.carIndex, err = bleve.Open(indexPath)

	if err == bleve.ErrorIndexPathDoesNotExist {
		logrus.Infof("Creating car search index")
		indexMapping := bleve.NewIndexMapping()
		cm.carIndex, err = bleve.New(indexPath, indexMapping)

		if err != nil {
			return err
		}

		err = cm.IndexAllCars()

		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

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
func (cm *CarManager) IndexAllCars() error {
	cm.searchIndexRebuildMutex.Lock()
	defer cm.searchIndexRebuildMutex.Unlock()

	logrus.Infof("Building search index for all cars")
	started := time.Now()

	results, _, err := cm.Search(context.Background(), "", 0, 100000)

	if err == nil {
		errs, _ := errgroup.WithContext(context.Background())

		for _, result := range results.Hits {
			result := result

			errs.Go(func() error {
				return cm.DeIndexCar(result.ID)
			})
		}

		if err := errs.Wait(); err != nil {
			return err
		}
	} else {
		logrus.WithError(err).Warnf("could not de-index cars")
	}

	cars, err := cm.ListCars()

	if err != nil {
		return err
	}

	errs, _ := errgroup.WithContext(context.Background())

	for _, car := range cars {
		car := car

		errs.Go(func() error {
			return cm.IndexCar(car)
		})
	}

	if err := errs.Wait(); err != nil {
		return err
	}

	logrus.Infof("Search index build is complete (took: %s)", time.Now().Sub(started).String())

	return nil
}

// Search looks for cars in the search index.
func (cm *CarManager) Search(ctx context.Context, term string, from, size int) (*bleve.SearchResult, Cars, error) {
	var q query.Query

	if term == "" {
		q = bleve.NewMatchAllQuery()
	} else {
		q = bleve.NewQueryStringQuery(term)
	}

	request := bleve.NewSearchRequestOptions(q, size, from, false)
	results, err := cm.carIndex.SearchInContext(ctx, request)

	if err != nil {
		return nil, nil, err
	}

	var cars Cars

	for _, hit := range results.Hits {
		car, err := cm.LoadCar(hit.ID, nil)

		if err != nil {
			return nil, nil, errors.Wrap(err, hit.ID)
		}

		cars = append(cars, car)
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

func (cm *CarManager) UpdateCarMetadata(carName string, r *http.Request) error {
	car, err := cm.LoadCar(carName, nil)

	if err != nil {
		return err
	}

	car.Details.Notes = r.FormValue("Notes")
	car.Details.DownloadURL = r.FormValue("DownloadURL")

	return car.Details.Save(carName)
}

func (cm *CarManager) UploadSkin(carName string, files map[string][]*multipart.FileHeader) error {
	carDirectory := filepath.Join(ServerInstallPath, "content", "cars", carName, "skins")

	for _, files := range files {
		for _, fh := range files {
			if err := cm.uploadSkinFile(carDirectory, fh); err != nil {
				return err
			}
		}
	}

	return nil
}

func (cm *CarManager) uploadSkinFile(carDirectory string, header *multipart.FileHeader) error {
	r, err := header.Open()

	if err != nil {
		return err
	}

	defer r.Close()

	fileDirectory := filepath.Join(carDirectory, filepath.Dir(header.Filename))

	if err := os.MkdirAll(fileDirectory, 0755); err != nil {
		return err
	}

	w, err := os.Create(filepath.Join(fileDirectory, filepath.Base(header.Filename)))

	if err != nil {
		return err
	}

	defer w.Close()

	_, err = io.Copy(w, r)

	return err
}

func (cm *CarManager) DeleteSkin(car, skin string) error {
	return os.RemoveAll(filepath.Join(ServerInstallPath, "content", "cars", car, "skins", skin))
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
	page := formValueAsInt(r.URL.Query().Get("page"))
	results, cars, err := ch.carManager.Search(r.Context(), searchTerm, page*searchPageSize, searchPageSize)

	if err != nil {
		logrus.WithError(err).Error("Could not perform search")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	numPages := int(math.Ceil(float64(float64(results.Total)) / float64(searchPageSize)))

	ch.viewRenderer.MustLoadTemplate(w, r, "content/cars.html", map[string]interface{}{
		"Results":     results,
		"Cars":        cars,
		"Query":       searchTerm,
		"CurrentPage": page,
		"NumPages":    numPages,
		"PageSize":    searchPageSize,
	})
}

type carSearchResult struct {
	CarName string `json:"CarName"`
	CarID   string `json:"CarID"`
	// Tags    []string `json:"Tags"`
}

func (ch *CarsHandler) searchJSON(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")

	_, cars, err := ch.carManager.Search(r.Context(), searchTerm, 0, 100000)

	if err != nil {
		logrus.WithError(err).Error("Could not perform search")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var searchResults []carSearchResult

	for _, car := range cars {
		searchResults = append(searchResults, carSearchResult{
			CarName: car.Details.Name,
			CarID:   car.Name,
			// Tags:    car.Details.Tags,
		})
	}

	enc := json.NewEncoder(w)
	if Debug {
		enc.SetIndent("", "    ")
	}
	_ = enc.Encode(searchResults)
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
	http.Redirect(w, r, "/cars", http.StatusFound)
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

	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		logrus.WithError(err).Errorf("Could not load car details for: %s", carName)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "content/car-details.html", templateParams)
}

func (ch *CarsHandler) tags(w http.ResponseWriter, r *http.Request) {
	car := chi.URLParam(r, "name")

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

func (ch *CarsHandler) saveMetadata(w http.ResponseWriter, r *http.Request) {
	car := chi.URLParam(r, "name")

	if err := ch.carManager.UpdateCarMetadata(car, r); err != nil {
		logrus.WithError(err).Errorf("Could not update car metadata for %s", car)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Car metadata updated successfully!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ch *CarsHandler) uploadSkin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	car := chi.URLParam(r, "name")

	err = ch.carManager.UploadSkin(car, r.MultipartForm.File)

	if err != nil {
		logrus.WithError(err).Errorf("could not upload car skin for %s", car)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Car skin uploaded successfully!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ch *CarsHandler) deleteSkin(w http.ResponseWriter, r *http.Request) {
	car := chi.URLParam(r, "name")
	skin := r.FormValue("skin-delete")

	if skin == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := ch.carManager.DeleteSkin(car, skin); err != nil {
		logrus.WithError(err).Errorf("could not delete car skin for %s", car)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Car skin deleted successfully!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ch *CarsHandler) rebuildSearchIndex(w http.ResponseWriter, r *http.Request) {
	go func() {
		err := ch.carManager.IndexAllCars()

		if err != nil {
			logrus.WithError(err).Error("could not rebuild search index")
		}
	}()

	AddFlash(w, r, "Started re-indexing cars!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
