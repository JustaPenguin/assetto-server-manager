package servermanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type SessionResults struct {
	Cars           []SessionCar     `json:"Cars"`
	Events         []SessionEvent   `json:"Events"`
	Laps           []SessionLap     `json:"Laps"`
	Result         []*SessionResult `json:"Result"`
	TrackConfig    string           `json:"TrackConfig"`
	TrackName      string           `json:"TrackName"`
	Type           string           `json:"Type"`
	Date           time.Time        `json:"Date"`
	SessionFile    string           `json:"SessionFile"`
	ChampionshipID string           `json:"ChampionshipID"`
}

func (s *SessionResults) GetCrashes(guid string) int {
	var num int

	for _, event := range s.Events {
		if event.Driver.GUID == guid {
			num++
		}
	}

	return num
}

func (s *SessionResults) GetAverageLapTime(guid string) time.Duration {
	var totalTime, driverLapCount, lapsForAverage int

	for _, lap := range s.Laps {
		if lap.DriverGUID == guid  {
			avgSoFar := (float64(totalTime) / float64(lapsForAverage)) * 1.07

			// if lap doesnt cut and if lap is < 107% of average for that driver so far and if lap isn't lap 1
			if lap.Cuts == 0 && driverLapCount != 0 && (float64(lap.LapTime) < avgSoFar || totalTime == 0) {
				totalTime += lap.LapTime
				lapsForAverage++
			}

			driverLapCount++
		}
	}

	return s.GetTime(int(float64(totalTime)/float64(lapsForAverage)), guid, false)
}

// lapNum is the drivers current lap
func (s *SessionResults) GetPosForLap(guid string, lapNum int64) int {
	var pos int

	driverLap := make(map[string]int)

	for overallLapNum, lap := range s.Laps {
		overallLapNum++
		driverLap[lap.DriverGUID]++

		if driverLap[lap.DriverGUID] == int(lapNum) && lap.DriverGUID == guid {
			return pos + 1
		} else if driverLap[lap.DriverGUID] == int(lapNum) {
			pos++
		}
	}

	return 0
}

func (s *SessionResults) IsFastestLap(time, cuts int) bool {
	if cuts != 0 {
		return false
	}

	var fastest = true

	for _, lap := range s.Laps {
		if lap.LapTime < time && lap.Cuts == 0 {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) IsDriversFastestLap(guid string, time, cuts int) bool {
	if cuts != 0 {
		return false
	}

	var fastest = true

	for _, lap := range s.Laps {
		if lap.LapTime < time && lap.DriverGUID == guid && lap.Cuts == 0 {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) IsFastestSector(sector, time, cuts int) bool {
	if cuts != 0 {
		return false
	}

	var fastest = true

	for _, lap := range s.Laps {
		if lap.Sectors[sector] < time && lap.Cuts == 0 {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) IsDriversFastestSector(guid string, sector, time, cuts int) bool {
	if cuts != 0 {
		return false
	}

	var fastest = true

	for _, lap := range s.Laps {
		if lap.Sectors[sector] < time && lap.DriverGUID == guid && lap.Cuts == 0 {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) GetDate() string {
	return s.Date.Format(time.RFC822)
}

func (s *SessionResults) GetNumSectors() []int {
	var num []int

	for range s.Laps[0].Sectors {
		num = append(num, 1)
	}

	return num
}

func (s *SessionResults) GetDrivers() string {
	var drivers []string

	numOpenSlots := 0

	for _, car := range s.Cars {
		if car.Driver.Name != "" {
			drivers = append(drivers, car.Driver.Name)
		} else {
			numOpenSlots++
		}
	}

	if numOpenSlots > 0 {
		drivers = append(drivers, fmt.Sprintf("%d open slots", numOpenSlots))
	}

	return strings.Join(drivers, ", ")
}

func (s *SessionResults) GetTime(timeINT int, driverGUID string, total bool) time.Duration {
	if i := s.GetLaps(driverGUID); i == 0 {
		return time.Duration(0)
	}

	d, _ := time.ParseDuration(fmt.Sprintf("%dms", timeINT))

	if total {
		for _, driver := range s.Result {
			if driver.DriverGUID == driverGUID && driver.HasPenalty {
				d += driver.PenaltyTime
				d -= time.Duration(driver.LapPenalty) * s.GetLastLapTime(driverGUID)
			}
		}
	}

	return d
}

func (s *SessionResults) GetTeamName(driverGUID string) string {
	for _, car := range s.Cars {
		if car.Driver.GUID == driverGUID {
			return car.Driver.Team
		}
	}

	return "Unknown"
}

func (s *SessionResults) GetLaps(driverGUID string) int {
	var i int

	for _, lap := range s.Laps {
		if lap.DriverGUID == driverGUID {
			i++
		}
	}

	// Apply lap penalty
	for _, driver := range s.Result {
		if driver.DriverGUID == driverGUID && driver.HasPenalty {
			i -= driver.LapPenalty
		}
	}

	return i
}

func (s *SessionResults) GetLastLapTime(driverGuid string) time.Duration {
	for i := len(s.Laps) - 1; i >= 0; i-- {
		if s.Laps[i].DriverGUID == driverGuid {
			return s.Laps[i].GetLapTime()
		}
	}

	return 1
}

func (s *SessionResults) GetCuts(driverGuid string) int {
	var i int

	for _, lap := range s.Laps {
		if lap.DriverGUID == driverGuid {
			i += lap.Cuts
		}
	}

	return i
}

func (s *SessionResults) FastestLap() *SessionLap {
	if len(s.Laps) == 0 {
		return nil
	}

	laps := make([]SessionLap, len(s.Laps))

	copy(laps, s.Laps)

	sort.Slice(laps, func(i, j int) bool {
		return laps[i].Cuts == 0 && laps[i].LapTime < laps[j].LapTime
	})

	return &laps[0]
}

type SessionResult struct {
	BallastKG    int           `json:"BallastKG"`
	BestLap      int           `json:"BestLap"`
	CarID        int           `json:"CarId"`
	CarModel     string        `json:"CarModel"`
	DriverGUID   string        `json:"DriverGuid"`
	DriverName   string        `json:"DriverName"`
	Restrictor   int           `json:"Restrictor"`
	TotalTime    int           `json:"TotalTime"`
	HasPenalty   bool          `json:"HasPenalty"`
	PenaltyTime  time.Duration `json:"PenaltyTime"`
	LapPenalty   int           `json:"LapPenalty"`
	Disqualified bool          `json:"Disqualified"`
}

type SessionLap struct {
	BallastKG  int    `json:"BallastKG"`
	CarID      int    `json:"CarId"`
	CarModel   string `json:"CarModel"`
	Cuts       int    `json:"Cuts"`
	DriverGUID string `json:"DriverGuid"`
	DriverName string `json:"DriverName"`
	LapTime    int    `json:"LapTime"`
	Restrictor int    `json:"Restrictor"`
	Sectors    []int  `json:"Sectors"`
	Timestamp  int    `json:"Timestamp"`
	Tyre       string `json:"Tyre"`
}

func (sl *SessionLap) GetSector(x int) time.Duration {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", sl.Sectors[x]))

	return d
}

func (sl *SessionLap) GetLapTime() time.Duration {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", sl.LapTime))

	return d
}

func (sl *SessionLap) DidCheat(averageTime time.Duration) bool {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", sl.LapTime))

	return d < averageTime && sl.Cuts > 0
}

type SessionCar struct {
	BallastKG  int           `json:"BallastKG"`
	CarID      int           `json:"CarId"`
	Driver     SessionDriver `json:"Driver"`
	Model      string        `json:"Model"`
	Restrictor int           `json:"Restrictor"`
	Skin       string        `json:"Skin"`
}

type SessionEvent struct {
	CarID         int           `json:"CarId"`
	Driver        SessionDriver `json:"Driver"`
	ImpactSpeed   float64       `json:"ImpactSpeed"`
	OtherCarID    int           `json:"OtherCarId"`
	OtherDriver   SessionDriver `json:"OtherDriver"`
	RelPosition   SessionPos    `json:"RelPosition"`
	Type          string        `json:"Type"`
	WorldPosition SessionPos    `json:"WorldPosition"`
}

func (se *SessionEvent) GetRelPosition() string {
	return fmt.Sprintf("X: %.1f Y: %.1f Z: %.1f", se.RelPosition.X, se.RelPosition.Y, se.RelPosition.Z)
}

func (se *SessionEvent) GetWorldPosition() string {
	return fmt.Sprintf("X: %.1f Y: %.1f Z: %.1f", se.WorldPosition.X, se.WorldPosition.Y, se.WorldPosition.Z)
}

type SessionDriver struct {
	GUID      string   `json:"Guid"`
	GuidsList []string `json:"GuidsList"`
	Name      string   `json:"Name"`
	Nation    string   `json:"Nation"`
	Team      string   `json:"Team"`
}

type SessionPos struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
	Z float64 `json:"Z"`
}

const pageSize = 10

var ErrResultsPageNotFound = errors.New("servermanager: results page not found")

func listResults(page int) ([]SessionResults, []int, error) {
	resultsPath := filepath.Join(ServerInstallPath, "results")
	resultFiles, err := ioutil.ReadDir(resultsPath)

	if err != nil {
		return nil, nil, err
	}

	sort.Slice(resultFiles, func(i, j int) bool {
		d1, _ := getResultDate(resultFiles[i].Name())
		d2, _ := getResultDate(resultFiles[j].Name())

		return d1.After(d2)
	})

	pages := float64(len(resultFiles)) / float64(pageSize)
	pagesRound := math.Ceil(pages)

	if page > int(pages) || page < 0 {
		return nil, nil, ErrResultsPageNotFound
	}

	var pagesSlice []int

	for x := 0; x < int(pagesRound); x++ {
		pagesSlice = append(pagesSlice, 0)
	}

	// get result files for selected page probably
	if len(resultFiles) > page*pageSize+pageSize {
		resultFiles = resultFiles[page*pageSize : page*pageSize+pageSize]
	} else {
		resultFiles = resultFiles[page*pageSize:]
	}

	var results []SessionResults

	for _, resultFile := range resultFiles {
		result, err := LoadResult(resultFile.Name())

		if err != nil {
			return nil, nil, err
		}

		results = append(results, *result)
	}

	return results, pagesSlice, nil
}

func getResultDate(name string) (time.Time, error) {
	dateSplit := strings.Split(name, "_")
	dateSplit = dateSplit[0 : len(dateSplit)-1]
	date := strings.Join(dateSplit, "_")

	var year, month, day, hour, minute int

	_, err := fmt.Sscanf(date, "%d_%d_%d_%d_%d", &year, &month, &day, &hour, &minute)

	if err != nil {
		return time.Time{}, err
	}

	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local), nil
}

func LoadResult(fileName string) (*SessionResults, error) {
	var result *SessionResults

	resultsPath := filepath.Join(ServerInstallPath, "results")

	data, err := ioutil.ReadFile(filepath.Join(resultsPath, fileName))

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &result)

	if err != nil {
		return nil, err
	}

	date, err := getResultDate(fileName)

	if err != nil {
		return nil, err
	}

	result.Date = date
	result.SessionFile = strings.Trim(fileName, ".json")

	return result, nil
}

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))

	if err != nil {
		page = 0
	}

	results, pages, err := listResults(page)

	if err == ErrResultsPageNotFound {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if err != nil {
		logrus.Errorf("could not get result list, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "results/index.html", map[string]interface{}{
		"results":     results,
		"pages":       pages,
		"currentPage": page,
	})
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	var result *SessionResults
	fileName := chi.URLParam(r, "fileName")

	result, err := LoadResult(fileName + ".json")

	if err != nil {
		logrus.Errorf("could not get result, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "results/result.html", map[string]interface{}{
		"result": result,
	})
}
