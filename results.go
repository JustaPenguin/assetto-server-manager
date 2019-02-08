package servermanager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type SessionResults struct {
	Cars        []SessionCars   `json:"Cars"`
	Events      []SessionEvents `json:"Events"`
	Laps        []SessionLaps   `json:"Laps"`
	Result      []SessionResult `json:"Result"`
	TrackConfig string          `json:"TrackConfig"`
	TrackName   string          `json:"TrackName"`
	Type        string          `json:"Type"`
	Date        time.Time
	SessionFile string
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
	var totalTime int
	var laps int

	for _, lap := range s.Laps {
		if lap.DriverGUID == guid {
			totalTime += lap.LapTime
			laps++
		}
	}

	return s.GetTime(int(float64(totalTime)/float64(laps)), guid)
}

// lapNum is the drivers current lap
func (s *SessionResults) GetPosForLap(guid string, lapNum int64) int {
	var driverLap map[string]int
	var pos int

	driverLap = make(map[string]int)

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

func (s *SessionResults) IsFastestLap(time int) bool {
	var fastest = true

	for _, lap := range s.Laps {
		if lap.LapTime < time {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) IsDriversFastestLap(guid string, time int) bool {
	var fastest = true

	for _, lap := range s.Laps {
		if lap.LapTime < time && lap.DriverGUID == guid {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) IsFastestSector(sector, time int) bool {
	var fastest = true

	for _, lap := range s.Laps {
		if lap.Sectors[sector] < time {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) IsDriversFastestSector(guid string, sector, time int) bool {
	var fastest = true

	for _, lap := range s.Laps {
		if lap.Sectors[sector] < time && lap.DriverGUID == guid {
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
	var drivers string

	for i, car := range s.Cars {
		drivers += car.Driver.Name

		if i != len(s.Cars)-1 {
			drivers += ", "
		}
	}

	return drivers
}

func (s *SessionResults) GetTime(timeINT int, driverGUID string) time.Duration {
	if i := s.GetLaps(driverGUID); i == 0 {
		return time.Duration(0)
	}

	d, _ := time.ParseDuration(fmt.Sprintf("%dms", timeINT))

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

func (s *SessionResults) GetLaps(driverGuid string) int {
	var i int

	for _, lap := range s.Laps {
		if lap.DriverGUID == driverGuid {
			i++
		}
	}

	return i
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

type SessionResult struct {
	BallastKG  int    `json:"BallastKG"`
	BestLap    int    `json:"BestLap"`
	CarID      int    `json:"CarId"`
	CarModel   string `json:"CarModel"`
	DriverGUID string `json:"DriverGuid"`
	DriverName string `json:"DriverName"`
	Restrictor int    `json:"Restrictor"`
	TotalTime  int    `json:"TotalTime"`
}

type SessionLaps struct {
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

func (sl *SessionLaps) GetSector(x int) time.Duration {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", sl.Sectors[x]))

	return d
}

func (sl *SessionLaps) GetLapTime() time.Duration {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", sl.LapTime))

	return d
}

func (sl *SessionLaps) DidCheat(averageTime time.Duration) bool {
	d, _ := time.ParseDuration(fmt.Sprintf("%dms", sl.LapTime))

	return d < averageTime && sl.Cuts > 0
}

type SessionCars struct {
	BallastKG  int           `json:"BallastKG"`
	CarID      int           `json:"CarId"`
	Driver     SessionDriver `json:"Driver"`
	Model      string        `json:"Model"`
	Restrictor int           `json:"Restrictor"`
	Skin       string        `json:"Skin"`
}

type SessionEvents struct {
	CarID         int           `json:"CarId"`
	Driver        SessionDriver `json:"Driver"`
	ImpactSpeed   float64       `json:"ImpactSpeed"`
	OtherCarID    int           `json:"OtherCarId"`
	OtherDriver   SessionDriver `json:"OtherDriver"`
	RelPosition   SessionPos    `json:"RelPosition"`
	Type          string        `json:"Type"`
	WorldPosition SessionPos    `json:"WorldPosition"`
}

func (se *SessionEvents) GetRelPosition() string {
	return fmt.Sprintf("X: %.1f Y: %.1f Z: %.1f", se.RelPosition.X, se.RelPosition.Y, se.RelPosition.Z)
}

func (se *SessionEvents) GetWorldPosition() string {
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

func listResults(page int) ([]SessionResults, []int, error) {
	var results []SessionResults

	resultsPath := filepath.Join(ServerInstallPath, "results")

	resultFiles, err := ioutil.ReadDir(resultsPath)

	if err != nil {
		return nil, nil, err
	}

	sort.Slice(resultFiles, func(i, j int) bool {
		return resultFiles[i].ModTime().After(resultFiles[j].ModTime())
	})

	pages := float64(len(resultFiles)) / float64(pageSize)
	pagesRound := math.Round(pages)

	if pagesRound < pages {
		pagesRound++
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

	for _, resultFile := range resultFiles {
		result, err := getResult(resultFile.Name())

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

func getResult(fileName string) (*SessionResults, error) {
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

	if err != nil {
		logrus.Errorf("could not get result list, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "results.html", map[string]interface{}{
		"results":     results,
		"pages":       pages,
		"currentPage": page,
	})
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	var result *SessionResults
	fileName := mux.Vars(r)["fileName"]

	result, err := getResult(fileName + ".json")

	if err != nil {
		logrus.Errorf("could not get result, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "result.html", map[string]interface{}{
		"result": result,
	})
}
