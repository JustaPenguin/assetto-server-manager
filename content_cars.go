package servermanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/cj123/watcher"
	"github.com/dimchansky/utfbom"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func (c Car) IsPaidDLC() bool {
	if _, ok := isCarPaidDLC[c.Name]; ok {
		return isCarPaidDLC[c.Name]
	}

	return false
}

func (c Car) IsMod() bool {
	_, ok := isCarPaidDLC[c.Name]

	return !ok
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
	Author        string          `json:"author"`
	Brand         string          `json:"brand"`
	Class         string          `json:"class"`
	Country       string          `json:"country"`
	Description   string          `json:"description"`
	Name          string          `json:"name"`
	PowerCurve    [][]json.Number `json:"powerCurve"`
	Specs         CarSpecs        `json:"specs"`
	SpecsNumeric  CarSpecsNumeric `json:"spec"`
	Tags          []string        `json:"tags"`
	TorqueCurve   [][]json.Number `json:"torqueCurve"`
	URL           string          `json:"url"`
	Version       string          `json:"version"`
	Year          ShouldBeAnInt   `json:"year"`
	IsStock       bool            `json:"stock"`
	IsDLC         bool            `json:"dlc"`
	IsMod         bool            `json:"mod"`
	Key           string          `json:"key"`
	PrettifiedKey string          `json:"prettified_key"`

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
	f, err := os.Open(filepath.Join(ServerInstallPath, "content", "cars", carName, "ui", "ui_car.json"))

	if err != nil {
		return err
	}

	defer f.Close()

	carDetailsBytes, err := ioutil.ReadAll(utfbom.SkipOnly(f))

	if err != nil {
		return err
	}

	carDetailsBytes = regexp.MustCompile(`\t*\r*\n*`).ReplaceAll(carDetailsBytes, []byte(""))

	err = json.Unmarshal(carDetailsBytes, &cd)

	if err != nil {
		return err
	}

	cd.SpecsNumeric = cd.Specs.Numeric()

	isDLC, isStock := isCarPaidDLC[carName]
	cd.IsStock = isStock
	cd.IsDLC = isDLC
	cd.IsMod = !isStock && !isDLC

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
	carIndex                     bleve.Index
	watchFilesystemForCarChanges bool

	searchMutex  sync.Mutex
	trackManager *TrackManager
}

func NewCarManager(trackManager *TrackManager, watchForCarChanges bool) *CarManager {
	cm := &CarManager{trackManager: trackManager, watchFilesystemForCarChanges: watchForCarChanges}

	return cm
}

// watchForChanges looks for created/removed files in the cars folder and (de-)indexes them as necessary
func (cm *CarManager) watchForCarChanges() error {
	w := watcher.New()

	err := w.Add(filepath.Join(ServerInstallPath, "content", "cars"))

	if err != nil {
		return err
	}

	w.SetMaxEvents(1)
	w.FilterOps(watcher.Create, watcher.Remove)
	w.AddFilterHook(func(info os.FileInfo, fullPath string) error {
		if info.IsDir() && info.Name() != "cars" {
			split := strings.Split(fullPath, fmt.Sprintf("%c", os.PathSeparator))

			if len(split) > 0 && split[len(split)-2] == "cars" {
				return nil // only fire the event for the car folder itself
			}
		}

		return watcher.ErrSkip
	})

	go panicCapture(func() {
		for {
			select {
			case event := <-w.Event:
				var err error
				var carName string

				switch event.Op {
				case watcher.Create, watcher.Write:
					carName = filepath.Base(event.Path)
					logrus.Infof("Indexing car: %s", carName)
					car, err := cm.LoadCar(carName, nil)

					if err != nil {
						logrus.WithError(err).Errorf("Could not find car to index: %s", carName)
						continue
					}

					err = cm.IndexCar(car)

					if err != nil {
						logrus.WithError(err).Errorf("Could not index car: %s", carName)
						continue
					}
				case watcher.Remove:
					carName = filepath.Base(event.OldPath)
					logrus.Infof("De-indexing car: %s", carName)
					err = cm.DeIndexCar(carName)
				}

				if err != nil {
					logrus.WithError(err).Errorf("Could not update index for car: %s", carName)
					continue
				}
			case err := <-w.Error:
				logrus.WithError(err).Error("Car content watcher error")
				continue
			case <-w.Closed:
				return
			}
		}
	})

	return w.Start(time.Second * 15)
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
		if os.IsNotExist(err) {
			if err := os.Mkdir(filepath.Join(carDirectory, "skins"), 0755); err != nil {
				logrus.WithError(err).Warnf("Could not create skins directory for car: %s", name)
			} else {
				logrus.Infof("Created empty skins directory for car: %s", name)
			}
		} else {
			logrus.WithError(err).Warnf("Could not load skins for car: %s", name)
		}
	}

	carDetails := CarDetails{}

	if err := carDetails.Load(name); err != nil {
		if !os.IsNotExist(err) {
			logrus.WithError(err).Errorf("could not parse car details json for: %s (likely this is invalid/malformed JSON). falling back to empty car details", name)
		}

		// the car details don't exist or can't be loaded, just create some fake ones.
		carDetails.Name = prettifyName(name, true)
	}

	carDetails.Key = name
	carDetails.PrettifiedKey = prettifyName(name, true)

	return &Car{
		Name:    name,
		Skins:   skins,
		Tyres:   tyres[name],
		Details: carDetails,
	}, nil
}

func (cm *CarManager) RandomSkin(model string) string {
	car, err := cm.LoadCar(model, nil)

	switch {
	case err != nil:
		logrus.WithError(err).Errorf("Could not load car %s. No skin will be specified", model)
		return ""
	case len(car.Skins) == 0:
		logrus.Warnf("Car %s has no skins uploaded. No skin will be specified", model)
		return ""
	default:
		return car.Skins[rand.Intn(len(car.Skins))]
	}
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
	cm.searchMutex.Lock()
	indexPath := filepath.Join(ServerInstallPath, "search-index", "cars")

	var err error

	cm.carIndex, err = bleve.Open(indexPath)
	cm.searchMutex.Unlock()

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

	if cm.watchFilesystemForCarChanges {
		go panicCapture(func() {
			err := cm.watchForCarChanges()

			if err != nil {
				logrus.WithError(err).Error("Could not watch for changes in the content/cars directory")
			}
		})
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

	logrus.Infof("Search index build is complete (took: %s)", time.Since(started).String())

	return nil
}

var (
	positiveCarTypeRegex = regexp.MustCompile(`\+(mod|dlc|stock)`)
	negativeCarTypeRegex = regexp.MustCompile(`-(mod|dlc|stock)`)
)

func (cm *CarManager) rebuildTerm(term string) string {
	// bleve only allows searching for true/false via the ugly terms
	// e.g. dlc:T* - make these a bit more user friendly (e.g. +dlc)
	term = positiveCarTypeRegex.ReplaceAllString(term, "$1:T*")
	term = negativeCarTypeRegex.ReplaceAllString(term, "$1:F*")

	return term
}

// Search looks for cars in the search index.
func (cm *CarManager) Search(ctx context.Context, term string, from, size int) (*bleve.SearchResult, Cars, error) {
	cm.searchMutex.Lock()
	defer cm.searchMutex.Unlock()

	var q query.Query

	term = cm.rebuildTerm(term)

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
		if hit.ID == "cars" {
			continue
		}

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

type carDetailsTemplateVars struct {
	BaseTemplateVars

	Car       *Car
	Results   []SessionResults
	Setups    map[string][]string
	TrackOpts []Track
}

// loadCarDetailsForTemplate loads all necessary items to generate the car details template.
func (cm *CarManager) loadCarDetailsForTemplate(carName string) (*carDetailsTemplateVars, error) {
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

	tracks, err := cm.trackManager.ListTracks()

	if err != nil {
		return nil, err
	}

	return &carDetailsTemplateVars{
		Car:       car,
		Results:   results,
		Setups:    setups,
		TrackOpts: tracks,
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

type carListTemplateVars struct {
	BaseTemplateVars

	Results     *bleve.SearchResult
	Cars        Cars
	Query       string
	CurrentPage int
	NumPages    int
	PageSize    int
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

	numPages := int(math.Ceil(float64(results.Total) / float64(searchPageSize)))

	ch.viewRenderer.MustLoadTemplate(w, r, "content/cars.html", &carListTemplateVars{
		Results:     results,
		Cars:        cars,
		Query:       searchTerm,
		CurrentPage: page,
		NumPages:    numPages,
		PageSize:    searchPageSize,
	})
}

type carSearchResult struct {
	CarName string `json:"CarName"`
	CarID   string `json:"CarID"`
	Class   string `json:"Class"`
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
		var class string

		if car.IsPaidDLC() {
			class = "bg-dlc"
		}

		if car.IsMod() {
			class = "bg-mod"
		}

		searchResults = append(searchResults, carSearchResult{
			CarName: car.Details.Name,
			CarID:   car.Name,
			Class:   class,
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
	skinPath := filepath.Join("content", "cars", car, "skins", url.PathEscape(skin), "preview.jpg")

	// look to see if the car preview image exists
	_, err := os.Stat(filepath.Join(ServerInstallPath, filepath.Join("content", "cars", car, "skins", skin, "preview.jpg")))

	if err != nil {
		return defaultSkinURL
	}

	return "/" + filepath.ToSlash(skinPath)
}

func (ch *CarsHandler) view(w http.ResponseWriter, r *http.Request) {
	carName := chi.URLParam(r, "car_id")
	templateParams, err := ch.carManager.loadCarDetailsForTemplate(carName)

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
	go panicCapture(func() {
		err := ch.carManager.IndexAllCars()

		if err != nil {
			logrus.WithError(err).Error("could not rebuild search index")
		}
	})

	AddFlash(w, r, "Started re-indexing cars!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

var isCarPaidDLC = map[string]bool{
	"abarth500":                          false,
	"abarth500_s1":                       false,
	"alfa_romeo_giulietta_qv":            false,
	"alfa_romeo_giulietta_qv_le":         false,
	"bmw_1m":                             false,
	"bmw_1m_s3":                          false,
	"bmw_m3_e30":                         false,
	"bmw_m3_e30_drift":                   false,
	"bmw_m3_e30_dtm":                     false,
	"bmw_m3_e30_gra":                     false,
	"bmw_m3_e30_s1":                      false,
	"bmw_m3_e92":                         false,
	"bmw_m3_e92_drift":                   false,
	"bmw_m3_e92_s1":                      false,
	"bmw_m3_gt2":                         false,
	"bmw_z4":                             false,
	"bmw_z4_drift":                       false,
	"bmw_z4_gt3":                         false,
	"bmw_z4_s1":                          false,
	"ferrari_312t":                       false,
	"ferrari_458":                        false,
	"ferrari_458_gt2":                    false,
	"ferrari_458_s3":                     false,
	"ferrari_599xxevo":                   false,
	"ferrari_f40":                        false,
	"ferrari_f40_s3":                     false,
	"ferrari_laferrari":                  false,
	"ks_abarth500_assetto_corse":         false,
	"ks_abarth_595ss":                    false,
	"ks_abarth_595ss_s1":                 false,
	"ks_abarth_595ss_s2":                 false,
	"ks_alfa_33_stradale":                false,
	"ks_alfa_giulia_qv":                  false,
	"ks_alfa_mito_qv":                    false,
	"ks_alfa_romeo_155_v6":               false,
	"ks_alfa_romeo_4c":                   false,
	"ks_alfa_romeo_gta":                  false,
	"ks_audi_a1s1":                       true,
	"ks_audi_r18_etron_quattro":          true,
	"ks_audi_r8_lms":                     false,
	"ks_audi_r8_lms_2016":                true,
	"ks_audi_r8_plus":                    true,
	"ks_audi_sport_quattro":              false,
	"ks_audi_sport_quattro_rally":        false,
	"ks_audi_sport_quattro_s1":           false,
	"ks_audi_tt_cup":                     true,
	"ks_audi_tt_vln":                     true,
	"ks_bmw_m235i_racing":                false,
	"ks_bmw_m4":                          true,
	"ks_bmw_m4_akrapovic":                false,
	"ks_corvette_c7_stingray":            true,
	"ks_corvette_c7r":                    false,
	"ks_ferrari_250_gto":                 true,
	"ks_ferrari_288_gto":                 true,
	"ks_ferrari_312_67":                  true,
	"ks_ferrari_330_p4":                  true,
	"ks_ferrari_488_gt3":                 true,
	"ks_ferrari_488_gtb":                 true,
	"ks_ferrari_812_superfast":           true,
	"ks_ferrari_f138":                    true,
	"ks_ferrari_f2004":                   true,
	"ks_ferrari_fxx_k":                   false,
	"ks_ferrari_sf15t":                   true,
	"ks_ferrari_sf70h":                   true,
	"ks_ford_escort_mk1":                 false,
	"ks_ford_gt40":                       false,
	"ks_ford_mustang_2015":               true,
	"ks_glickenhaus_scg003":              false,
	"ks_lamborghini_aventador_sv":        true,
	"ks_lamborghini_countach":            false,
	"ks_lamborghini_countach_s1":         false,
	"ks_lamborghini_gallardo_sl":         true,
	"ks_lamborghini_gallardo_sl_s3":      false,
	"ks_lamborghini_huracan_gt3":         false,
	"ks_lamborghini_huracan_performante": false,
	"ks_lamborghini_huracan_st":          false,
	"ks_lamborghini_miura_sv":            false,
	"ks_lamborghini_sesto_elemento":      false,
	"ks_lotus_25":                        false,
	"ks_lotus_3_eleven":                  true,
	"ks_lotus_72d":                       false,
	"ks_maserati_250f_12cyl":             true,
	"ks_maserati_250f_6cyl":              true,
	"ks_maserati_alfieri":                false,
	"ks_maserati_gt_mc_gt4":              true,
	"ks_maserati_levante":                false,
	"ks_maserati_mc12_gt1":               true,
	"ks_maserati_quattroporte":           false,
	"ks_mazda_787b":                      false,
	"ks_mazda_miata":                     false,
	"ks_mazda_mx5_cup":                   true,
	"ks_mazda_mx5_nd":                    true,
	"ks_mazda_rx7_spirit_r":              true,
	"ks_mazda_rx7_tuned":                 true,
	"ks_mclaren_570s":                    true,
	"ks_mclaren_650_gt3":                 false,
	"ks_mclaren_f1_gtr":                  false,
	"ks_mclaren_p1":                      false,
	"ks_mclaren_p1_gtr":                  true,
	"ks_mercedes_190_evo2":               false,
	"ks_mercedes_amg_gt3":                false,
	"ks_mercedes_c9":                     false,
	"ks_nissan_370z":                     true,
	"ks_nissan_gtr":                      true,
	"ks_nissan_gtr_gt3":                  false,
	"ks_nissan_skyline_r34":              true,
	"ks_pagani_huayra_bc":                false,
	"ks_porsche_718_boxster_s":           true,
	"ks_porsche_718_boxster_s_pdk":       true,
	"ks_porsche_718_cayman_s":            true,
	"ks_porsche_718_spyder_rs":           true,
	"ks_porsche_908_lh":                  true,
	"ks_porsche_911_carrera_rsr":         true,
	"ks_porsche_911_gt1":                 true,
	"ks_porsche_911_gt3_cup_2017":        true,
	"ks_porsche_911_gt3_r_2016":          true,
	"ks_porsche_911_gt3_rs":              true,
	"ks_porsche_911_r":                   true,
	"ks_porsche_911_rsr_2017":            true,
	"ks_porsche_917_30":                  true,
	"ks_porsche_917_k":                   true,
	"ks_porsche_918_spyder":              true,
	"ks_porsche_919_hybrid_2015":         true,
	"ks_porsche_919_hybrid_2016":         true,
	"ks_porsche_935_78_moby_dick":        true,
	"ks_porsche_962c_longtail":           true,
	"ks_porsche_962c_shorttail":          true,
	"ks_porsche_991_carrera_s":           true,
	"ks_porsche_991_turbo_s":             true,
	"ks_porsche_cayenne":                 false,
	"ks_porsche_cayman_gt4_clubsport":    true,
	"ks_porsche_cayman_gt4_std":          true,
	"ks_porsche_macan":                   false,
	"ks_porsche_panamera":                false,
	"ks_praga_r1":                        false,
	"ks_ruf_rt12r":                       false,
	"ks_ruf_rt12r_awd":                   false,
	"ks_toyota_ae86":                     true,
	"ks_toyota_ae86_drift":               true,
	"ks_toyota_ae86_tuned":               true,
	"ks_toyota_celica_st185":             true,
	"ks_toyota_gt86":                     true,
	"ks_toyota_supra_mkiv":               true,
	"ks_toyota_supra_mkiv_drift":         true,
	"ks_toyota_supra_mkiv_tuned":         true,
	"ks_toyota_ts040":                    true,
	"ktm_xbow_r":                         false,
	"lotus_2_eleven":                     false,
	"lotus_2_eleven_gt4":                 false,
	"lotus_49":                           false,
	"lotus_98t":                          false,
	"lotus_elise_sc":                     false,
	"lotus_elise_sc_s1":                  false,
	"lotus_elise_sc_s2":                  false,
	"lotus_evora_gtc":                    false,
	"lotus_evora_gte":                    false,
	"lotus_evora_gte_carbon":             false,
	"lotus_evora_gx":                     false,
	"lotus_evora_s":                      false,
	"lotus_evora_s_s2":                   false,
	"lotus_exige_240":                    false,
	"lotus_exige_240_s3":                 false,
	"lotus_exige_s":                      false,
	"lotus_exige_s_roadster":             false,
	"lotus_exige_scura":                  false,
	"lotus_exige_v6_cup":                 false,
	"lotus_exos_125":                     false,
	"lotus_exos_125_s1":                  false,
	"mclaren_mp412c":                     false,
	"mclaren_mp412c_gt3":                 false,
	"mercedes_sls":                       false,
	"mercedes_sls_gt3":                   false,
	"p4-5_2011":                          false,
	"pagani_huayra":                      false,
	"pagani_zonda_r":                     false,
	"ruf_yellowbird":                     false,
	"shelby_cobra_427sc":                 false,
	"tatuusfa1":                          false,
}
