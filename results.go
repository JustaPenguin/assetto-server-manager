package servermanager

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type SessionResults struct {
	Cars           []*SessionCar    `json:"Cars"`
	Events         []*SessionEvent  `json:"Events"`
	Laps           []*SessionLap    `json:"Laps"`
	Result         []*SessionResult `json:"Result"`
	TrackConfig    string           `json:"TrackConfig"`
	TrackName      string           `json:"TrackName"`
	Type           SessionType      `json:"Type"`
	Date           time.Time        `json:"Date"`
	SessionFile    string           `json:"SessionFile"`
	ChampionshipID string           `json:"ChampionshipID"`
	RaceWeekendID  string           `json:"RaceWeekendID"`
}

var ErrSessionCarNotFound = errors.New("servermanager: session car not found")

func (s *SessionResults) FindCarByGUIDAndModel(guid, model string) (*SessionCar, error) {
	for _, car := range s.Cars {
		if car.GetGUID() == guid && car.GetCar() == model {
			return car, nil
		}
	}

	return nil, ErrSessionCarNotFound
}

func (s *SessionResults) Anonymize() {
	for _, car := range s.Cars {
		car.Driver.GUID = AnonymiseDriverGUID(car.Driver.GUID)
		car.Driver.Name = shortenDriverName(car.Driver.Name)

		for index := range car.Driver.GuidsList {
			car.Driver.GuidsList[index] = AnonymiseDriverGUID(car.Driver.GuidsList[index])
		}
	}

	for _, event := range s.Events {
		event.Driver.GUID = AnonymiseDriverGUID(event.Driver.GUID)
		event.OtherDriver.GUID = AnonymiseDriverGUID(event.OtherDriver.GUID)

		event.Driver.Name = shortenDriverName(event.Driver.Name)
		event.OtherDriver.Name = shortenDriverName(event.OtherDriver.Name)

		for i, guid := range event.Driver.GuidsList {
			event.Driver.GuidsList[i] = AnonymiseDriverGUID(guid)
		}

		for i, guid := range event.OtherDriver.GuidsList {
			event.Driver.GuidsList[i] = AnonymiseDriverGUID(guid)
		}
	}

	for _, lap := range s.Laps {
		lap.DriverGUID = AnonymiseDriverGUID(lap.DriverGUID)

		lap.DriverName = shortenDriverName(lap.DriverName)
	}

	for _, result := range s.Result {
		result.DriverGUID = AnonymiseDriverGUID(result.DriverGUID)

		result.DriverName = shortenDriverName(result.DriverName)
	}
}

func AnonymiseDriverGUID(guid string) string {
	hasher := md5.New()
	_, _ = hasher.Write([]byte(guid))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *SessionResults) MaskDriverNames() {
	for _, car := range s.Cars {
		car.Driver.Name = driverName(car.Driver.Name)
	}

	for _, event := range s.Events {
		event.Driver.Name = driverName(event.Driver.Name)
		event.OtherDriver.Name = driverName(event.OtherDriver.Name)
	}

	for _, lap := range s.Laps {
		lap.DriverName = driverName(lap.DriverName)
	}

	for _, result := range s.Result {
		result.DriverName = driverName(result.DriverName)
	}
}

func (s *SessionResults) RenameDriver(guid, newName string) {
	for _, car := range s.Cars {
		if car.Driver.GUID == guid {
			car.Driver.Name = newName
		}
	}

	for _, event := range s.Events {
		if event.Driver.GUID == guid {
			event.Driver.Name = newName
		}

		if event.OtherDriver.GUID == guid {
			event.OtherDriver.Name = newName
		}
	}

	for _, lap := range s.Laps {
		if lap.DriverGUID == guid {
			lap.DriverName = newName
		}
	}

	for _, result := range s.Result {
		if result.DriverGUID == guid {
			result.DriverName = newName
		}
	}
}

func (s *SessionResults) DriversHaveTeams() bool {
	teams := make(map[string]string)

	for _, car := range s.Cars {
		teams[car.Driver.GUID] = car.Driver.Team
	}

	for _, driver := range s.Result {
		if driver.TotalTime > 0 {
			if team, ok := teams[driver.DriverGUID]; ok && team != "" {
				return true
			}
		}
	}

	return false
}

func (s *SessionResults) GetURL() string {
	return config.HTTP.BaseURL + "/results/download/" + s.SessionFile + ".json"
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

func (s *SessionResults) GetCrashesOfType(guid, collisionType string) int {
	var num int

	for _, event := range s.Events {
		if event.Driver.GUID == guid && event.Type == collisionType {
			num++
		}
	}

	return num
}

func (s *SessionResults) GetAverageLapTime(guid, model string) time.Duration {
	var totalTime, driverLapCount, lapsForAverage, totalTimeForAverage int

	for _, lap := range s.Laps {
		if lap.DriverGUID == guid && lap.CarModel == model {
			avgSoFar := (float64(totalTime) / float64(lapsForAverage)) * 1.07

			// if lap doesnt cut and if lap is < 107% of average for that driver so far and if lap isn't lap 1
			if lap.Cuts == 0 && driverLapCount != 0 && (float64(lap.LapTime) < avgSoFar || totalTime == 0) {
				totalTimeForAverage += lap.LapTime
				lapsForAverage++
			}

			driverLapCount++
			totalTime += lap.LapTime
		}
	}

	return s.GetTime(int(float64(totalTimeForAverage)/float64(lapsForAverage)), guid, model, false)
}

func (s *SessionResults) GetOverallAverageLapTime() time.Duration {
	var totalTime, driverLapCount, lapsForAverage, totalTimeForAverage int

	for _, lap := range s.Laps {
		avgSoFar := (float64(totalTime) / float64(lapsForAverage)) * 1.07

		// if lap doesnt cut and if lap is < 107% of average for that driver so far and if lap isn't lap 1
		if lap.Cuts == 0 && driverLapCount != 0 && (float64(lap.LapTime) < avgSoFar || totalTime == 0) {
			totalTimeForAverage += lap.LapTime
			lapsForAverage++
		}

		driverLapCount++
		totalTime += lap.LapTime
	}

	d, _ := time.ParseDuration(fmt.Sprintf("%dms", int(float64(totalTimeForAverage)/float64(lapsForAverage))))

	return d
}

func (s *SessionResults) GetConsistency(guid, model string) float64 {
	var bestLap int

	for _, lap := range s.Laps {
		if lap.DriverGUID == guid && lap.CarModel == model {
			if s.IsDriversFastestLap(guid, model, lap.LapTime, lap.Cuts) {
				bestLap = lap.LapTime
			}
		}
	}

	var percentage float64

	average := s.GetAverageLapTime(guid, model)
	best := s.GetTime(bestLap, guid, model, false)

	if average != 0 && best != 0 {
		consistency := average.Seconds() - best.Seconds()

		percentage = 100 - ((consistency / best.Seconds()) * 100)
	} else {
		percentage = 0
	}

	return math.Round(percentage*100) / 100
}

// lapNum is the drivers current lap
func (s *SessionResults) GetPosForLap(guid, model string, lapNum int64) int {
	var pos int

	driverLap := make(map[string]int)

	for _, lap := range s.Laps {
		driverLap[lap.DriverGUID+lap.CarModel]++

		if driverLap[lap.DriverGUID+lap.CarModel] == int(lapNum) && lap.DriverGUID == guid && lap.CarModel == model {
			return pos + 1
		} else if driverLap[lap.DriverGUID+lap.CarModel] == int(lapNum) {
			pos++
		}
	}

	return 0
}

func (s *SessionResults) IsFastestLap(time, cuts int) bool {
	if cuts != 0 {
		return false
	}

	fastest := true

	for _, lap := range s.Laps {
		if lap.LapTime < time && lap.Cuts == 0 {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) IsDriversFastestLap(guid, model string, time, cuts int) bool {
	if cuts != 0 {
		return false
	}

	fastest := true

	for _, lap := range s.Laps {
		if lap.LapTime < time && lap.DriverGUID == guid && lap.CarModel == model && lap.Cuts == 0 {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) GetDriversFastestLap(guid, model string) *SessionLap {
	var fastest *SessionLap

	for _, lap := range s.Laps {
		if lap.DriverGUID == guid && lap.CarModel == model && (fastest == nil || lap.LapTime < fastest.LapTime) && lap.Cuts == 0 {
			fastest = lap
		}
	}

	return fastest
}

func (s *SessionResults) IsFastestSector(sector, time, cuts int) bool {
	if cuts != 0 {
		return false
	}

	fastest := true

	for _, lap := range s.Laps {
		if lap.Sectors[sector] < time && lap.Cuts == 0 {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) IsDriversFastestSector(guid, model string, sector, time, cuts int) bool {
	if cuts != 0 {
		return false
	}

	fastest := true

	for _, lap := range s.Laps {
		if lap.Sectors[sector] < time && lap.DriverGUID == guid && lap.CarModel == model && lap.Cuts == 0 {
			fastest = false
			break
		}
	}

	return fastest
}

func (s *SessionResults) FallBackSort() {
	// sort the results by laps completed then race time
	// this is a fall back for when assetto's sorting is terrible
	// sort results.Result, if disqualified go to back, if time penalty sort by laps completed then lap time

cars:
	for i := range s.Cars {
		for z := range s.Result {
			if s.Cars[i].Driver.GUID == s.Result[z].DriverGUID {
				continue cars
			}
		}

		var bestLap int

		for y := range s.Laps {
			if (s.Cars[i].Driver.GUID == s.Laps[y].DriverGUID) && s.IsDriversFastestLap(s.Cars[i].Driver.GUID, s.Cars[i].Model, s.Laps[y].LapTime, s.Laps[y].Cuts) {
				bestLap = s.Laps[y].LapTime
				break
			}
		}

		s.Result = append(s.Result, &SessionResult{
			BallastKG:    s.Cars[i].BallastKG,
			BestLap:      bestLap,
			CarID:        s.Cars[i].CarID,
			CarModel:     s.Cars[i].Model,
			DriverGUID:   s.Cars[i].Driver.GUID,
			DriverName:   s.Cars[i].Driver.Name,
			Restrictor:   s.Cars[i].Restrictor,
			TotalTime:    0,
			HasPenalty:   false,
			PenaltyTime:  time.Duration(0),
			LapPenalty:   0,
			Disqualified: false,
		})
	}

	for i := range s.Result {
		s.Result[i].TotalTime = 0

		for _, lap := range s.Laps {
			if lap.DriverGUID == s.Result[i].DriverGUID {
				s.Result[i].TotalTime += lap.LapTime
			}
		}

		if s.Result[i].HasPenalty {
			s.Result[i].TotalTime += int(s.Result[i].PenaltyTime.Seconds())
		}
	}

	sort.Slice(s.Result, func(i, j int) bool {
		if (!s.Result[i].Disqualified && !s.Result[j].Disqualified) || (s.Result[i].Disqualified && s.Result[j].Disqualified) {

			if s.Type == SessionTypeQualifying || s.Type == SessionTypePractice {

				if s.Result[i].BestLap == 0 {
					return false
				}

				if s.Result[j].BestLap == 0 {
					return true
				}

				return s.GetTime(s.Result[i].BestLap, s.Result[i].DriverGUID, s.Result[i].CarModel, true) <
					s.GetTime(s.Result[j].BestLap, s.Result[j].DriverGUID, s.Result[j].CarModel, true)
			}

			// if both drivers aren't/are disqualified
			if s.GetNumLaps(s.Result[i].DriverGUID, s.Result[i].CarModel) == s.GetNumLaps(s.Result[j].DriverGUID, s.Result[j].CarModel) {
				if s.Result[i].HasPenalty || s.Result[j].HasPenalty {
					return s.GetTime(s.Result[i].TotalTime, s.Result[i].DriverGUID, s.Result[i].CarModel, true) <
						s.GetTime(s.Result[j].TotalTime, s.Result[j].DriverGUID, s.Result[j].CarModel, true)
				}

				// if their number of laps are equal, compare last lap pos
				return s.GetLastLapPos(s.Result[i].DriverGUID, s.Result[i].CarModel) < s.GetLastLapPos(s.Result[j].DriverGUID, s.Result[j].CarModel)
			}

			return s.GetNumLaps(s.Result[i].DriverGUID, s.Result[i].CarModel) >= s.GetNumLaps(s.Result[j].DriverGUID, s.Result[j].CarModel)

		}

		// driver i is closer to the front than j if they are not disqualified and j is
		return s.Result[j].Disqualified
	})
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

	for _, car := range s.Result {
		if car.DriverName != "" {
			drivers = append(drivers, driverName(car.DriverName))
		}
	}

	return strings.Join(drivers, ", ")
}

func (s *SessionResults) GetTime(timeINT int, driverGUID, model string, penalty bool) time.Duration {
	if i := s.GetNumLaps(driverGUID, model); i == 0 {
		return time.Duration(0)
	}

	d, _ := time.ParseDuration(fmt.Sprintf("%dms", timeINT))

	if penalty {
		for _, driver := range s.Result {
			if driver.DriverGUID == driverGUID && driver.CarModel == model && driver.HasPenalty {
				d += driver.PenaltyTime

				if s.Type == SessionTypeRace {
					d -= time.Duration(driver.LapPenalty) * s.GetLastLapTime(driverGUID, model)
				}
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

	return ""
}

func (s *SessionResults) GetNumLaps(driverGUID, model string) int {
	var i int

	for _, lap := range s.Laps {
		if lap.DriverGUID == driverGUID && lap.CarModel == model {
			i++
		}
	}

	// Apply lap penalty
	for _, driver := range s.Result {
		if driver.DriverGUID == driverGUID && driver.CarModel == model && driver.HasPenalty {
			i -= driver.LapPenalty
		}
	}

	return i
}

func (s *SessionResults) GetLastLapTime(driverGUID, model string) time.Duration {
	for i := len(s.Laps) - 1; i >= 0; i-- {
		if s.Laps[i].DriverGUID == driverGUID && s.Laps[i].CarModel == model {
			return s.Laps[i].GetLapTime()
		}
	}

	return 1
}

func (s *SessionResults) HasHandicaps() bool {
	for _, car := range s.Result {
		if car.BallastKG > 0 || car.Restrictor > 0 {
			return true
		}
	}

	return false
}

func (s *SessionResults) GetPotentialLap(driverGUID, model string) time.Duration {
	sectors := make([]int, len(s.GetNumSectors()))

	for _, lap := range s.Laps {
		if lap.DriverGUID != driverGUID || lap.CarModel != model || lap.Cuts > 0 {
			continue
		}

		for i, sector := range lap.Sectors {
			if sectors[i] == 0 || sector < sectors[i] {
				sectors[i] = sector
			}
		}
	}

	var totalSectorTime time.Duration

	for _, sector := range sectors {
		totalSectorTime += time.Duration(sector) * time.Millisecond
	}

	return totalSectorTime
}

func (s *SessionResults) GetLastLapPos(driverGUID, model string) int {
	var driverLaps int

	for i := range s.Laps {
		if s.Laps[i].DriverGUID == driverGUID && s.Laps[i].CarModel == model {
			driverLaps++
		}
	}

	return s.GetPosForLap(driverGUID, model, int64(driverLaps))
}

func (s *SessionResults) GetDriverPosition(driverGUID, model string) int {
	for i := range s.Result {
		if s.Result[i].DriverGUID == driverGUID && s.Result[i].CarModel == model {
			return i + 1
		}
	}

	return 0
}

func (s *SessionResults) GetCuts(driverGUID, model string) int {
	var i int

	for _, lap := range s.Laps {
		if lap.DriverGUID == driverGUID && lap.CarModel == model {
			i += lap.Cuts
		}
	}

	return i
}

func (s *SessionResults) FastestLap() *SessionLap {
	if len(s.Laps) == 0 {
		return nil
	}

	laps := make([]*SessionLap, len(s.Laps))

	copy(laps, s.Laps)

	sort.Slice(laps, func(i, j int) bool {
		if laps[i].Cuts != 0 {
			return false
		}

		if laps[j].Cuts != 0 {
			return true
		}

		return laps[i].LapTime < laps[j].LapTime
	})

	return laps[0]
}

func (s *SessionResults) FastestLapInClass(classID uuid.UUID) *SessionLap {
	if len(s.Laps) == 0 || s.Laps == nil {
		return nil
	}

	var laps []*SessionLap

	for _, lap := range s.Laps {
		if lap.ClassID == classID {
			laps = append(laps, lap)
		}
	}

	if len(laps) == 0 {
		return nil
	}

	sort.Slice(laps, func(i, j int) bool {
		if laps[i].Cuts != 0 {
			return false
		}

		if laps[j].Cuts != 0 {
			return true
		}

		return laps[i].LapTime < laps[j].LapTime
	})

	return laps[0]
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
	ClassID      uuid.UUID     `json:"ClassID"`
}

func (s *SessionResult) BestLapTyre(results *SessionResults) string {
	for _, lap := range results.Laps {
		if lap.LapTime == s.BestLap {
			return lap.Tyre
		}
	}

	return "?"
}

type SessionLap struct {
	BallastKG  int       `json:"BallastKG"`
	CarID      int       `json:"CarId"`
	CarModel   string    `json:"CarModel"`
	Cuts       int       `json:"Cuts"`
	DriverGUID string    `json:"DriverGuid"`
	DriverName string    `json:"DriverName"`
	LapTime    int       `json:"LapTime"`
	Restrictor int       `json:"Restrictor"`
	Sectors    []int     `json:"Sectors"`
	Timestamp  int       `json:"Timestamp"`
	Tyre       string    `json:"Tyre"`
	ClassID    uuid.UUID `json:"ClassID"`
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

func (c *SessionCar) GetName() string {
	return c.Driver.Name
}

func (c *SessionCar) GetCar() string {
	return c.Model
}

func (c *SessionCar) GetSkin() string {
	return c.Skin
}

func (c *SessionCar) GetGUID() string {
	return c.Driver.GUID
}

func (c *SessionCar) GetTeam() string {
	return c.Driver.Team
}

type SessionEvent struct {
	CarID         int            `json:"CarId"`
	Driver        *SessionDriver `json:"Driver"`
	ImpactSpeed   float64        `json:"ImpactSpeed"`
	OtherCarID    int            `json:"OtherCarId"`
	OtherDriver   *SessionDriver `json:"OtherDriver"`
	RelPosition   *SessionPos    `json:"RelPosition"`
	Type          string         `json:"Type"`
	WorldPosition *SessionPos    `json:"WorldPosition"`
}

func (se *SessionEvent) GetRelPosition() string {
	return fmt.Sprintf("X: %.1f Y: %.1f Z: %.1f", se.RelPosition.X, se.RelPosition.Y, se.RelPosition.Z)
}

func (se *SessionEvent) GetWorldPosition() string {
	return fmt.Sprintf("X: %.1f Y: %.1f Z: %.1f", se.WorldPosition.X, se.WorldPosition.Y, se.WorldPosition.Z)
}

type SessionDriver struct {
	GUID      string    `json:"Guid"`
	GuidsList []string  `json:"GuidsList"`
	Name      string    `json:"Name"`
	Nation    string    `json:"Nation"`
	Team      string    `json:"Team"`
	ClassID   uuid.UUID `json:"ClassID"`
}

func (sd *SessionDriver) AssignEntrant(entrant *Entrant, classID uuid.UUID) {
	if sd.GUID != entrant.GUID {
		return
	}

	sd.Name = entrant.Name
	sd.Team = entrant.Team
	sd.ClassID = classID
}

type SessionPos struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
	Z float64 `json:"Z"`
}

const pageSize = 20

var ErrResultsPageNotFound = errors.New("servermanager: results page not found")

func listResults(page int) ([]SessionResults, []int, error) {
	resultsPath := filepath.Join(ServerInstallPath, "results")
	resultFiles, err := ioutil.ReadDir(resultsPath)

	if err != nil {
		return nil, nil, err
	}

	sort.Slice(resultFiles, func(i, j int) bool {
		d1, _ := GetResultDate(resultFiles[i].Name())
		d2, _ := GetResultDate(resultFiles[j].Name())

		return d1.After(d2)
	})

	pages := float64(len(resultFiles)) / float64(pageSize)
	pagesRound := math.Ceil(pages)

	if page > int(pages) || page < 0 {
		return nil, nil, ErrResultsPageNotFound
	}

	var pagesSlice []int

	for x := 0; x < int(pagesRound); x++ {
		pagesSlice = append(pagesSlice, x)
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

func ListAllResults() ([]SessionResults, error) {
	resultsPath := filepath.Join(ServerInstallPath, "results")
	resultFiles, err := ioutil.ReadDir(resultsPath)

	if err != nil {
		return nil, err
	}

	sort.Slice(resultFiles, func(i, j int) bool {
		d1, _ := GetResultDate(resultFiles[i].Name())
		d2, _ := GetResultDate(resultFiles[j].Name())

		return d1.After(d2)
	})

	var results []SessionResults

	for _, resultFile := range resultFiles {
		result, err := LoadResult(resultFile.Name(), LoadResultWithoutPluginFire)

		if err != nil {
			return nil, err
		}

		results = append(results, *result)
	}

	return results, nil
}

func GetResultDate(name string) (time.Time, error) {
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

var UseFallBackSorting = false

type LoadResultOpts int

const LoadResultWithoutPluginFire LoadResultOpts = 0

func LoadResult(fileName string, opts ...LoadResultOpts) (*SessionResults, error) {
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

	date, err := GetResultDate(fileName)

	if err != nil {
		return nil, err
	}

	result.Date = date
	result.SessionFile = strings.Trim(fileName, ".json")

	var validResults []*SessionResult

	// filter out invalid results
	for _, driver := range result.Result {
		if driver.TotalTime > 0 {
			validResults = append(validResults, driver)
		}
	}

	result.Result = validResults

	if UseFallBackSorting {
		result.FallBackSort()
	}

	var skipLua bool

	for _, opt := range opts {
		if opt == LoadResultWithoutPluginFire {
			skipLua = true
		}
	}

	if !skipLua && config.Lua.Enabled && Premium() {
		err = resultsLoadPlugin(result)

		if err != nil {
			logrus.WithError(err).Error("results load plugin script failed")
		}
	}

	return result, nil
}

func resultsLoadPlugin(results *SessionResults) error {
	p := &LuaPlugin{}

	newSessionResults := &SessionResults{}

	p.Inputs(results).Outputs(newSessionResults)
	err := p.Call("./plugins/results.lua", "onResultsLoad")

	if err != nil {
		return err
	}

	*results = *newSessionResults

	return nil
}

type ResultsHandler struct {
	*BaseHandler

	store Store
}

func NewResultsHandler(baseHandler *BaseHandler, store Store) *ResultsHandler {
	return &ResultsHandler{
		BaseHandler: baseHandler,
		store:       store,
	}
}

type resultsListTemplateVars struct {
	BaseTemplateVars

	Results     []SessionResults
	Pages       []int
	CurrentPage int
}

func (rh *ResultsHandler) list(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))

	if err != nil {
		page = 0
	}

	results, pages, err := listResults(page)

	if err == ErrResultsPageNotFound {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if err != nil {
		logrus.WithError(err).Errorf("could not get result list")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rh.viewRenderer.MustLoadTemplate(w, r, "results/index.html", &resultsListTemplateVars{
		Results:     results,
		Pages:       pages,
		CurrentPage: page,
	})
}

func (rh *ResultsHandler) uploadHandler(w http.ResponseWriter, r *http.Request) {
	matched, err := rh.upload(r)

	if err != nil {
		logrus.WithError(err).Errorf("could not parse results form")
		AddErrorFlash(w, r, "Sorry, we couldn't parse that results file! Please make sure the format is correct.")
	}

	if !matched {
		AddErrorFlash(w, r, "Your results file content was correct, but the file name is incorrect! Please make sure the file name matches the AC standard then try again.")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

const uploadFileSizeLimit = 5e6

func (rh *ResultsHandler) upload(r *http.Request) (bool, error) {
	err := r.ParseMultipartForm(10 << 20)

	if err != nil {
		return true, err
	}

	file, header, err := r.FormFile("resultsFile")
	if err != nil {
		return true, err
	}
	defer file.Close()

	if header.Size > (uploadFileSizeLimit) {
		return true, fmt.Errorf("servermanager: file size too large, limit is: %d, this file is: %d", int64(uploadFileSizeLimit), header.Size)
	}

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return true, err
	}

	// make sure what we've been given is actually a results file
	var resultTest *SessionResults

	err = json.Unmarshal(fileBytes, &resultTest)

	if err != nil {
		return true, err
	}

	matched, err := regexp.MatchString(`\d{4}_\d{1,2}_\d{1,2}_\d{1,2}_\d{1,2}_(RACE|QUALIFY|PRACTICE|BOOK)\.json`, header.Filename)

	if err != nil {
		return matched, err
	}

	if !matched {
		return matched, nil
	}

	path := filepath.Join(ServerInstallPath, "results", header.Filename)

	err = ioutil.WriteFile(path, fileBytes, 0644)

	if err != nil {
		return matched, err
	}

	return matched, nil
}

type resultsViewTemplateVars struct {
	BaseTemplateVars

	Result  *SessionResults
	Account *Account
	UseMPH  bool
}

func (rh *ResultsHandler) view(w http.ResponseWriter, r *http.Request) {
	fileName := chi.URLParam(r, "fileName")

	result, err := LoadResult(fileName + ".json")

	if os.IsNotExist(err) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if err != nil {
		logrus.WithError(err).Errorf("could not get result")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	serverOpts, err := rh.store.LoadServerOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load server options")
	}

	rh.viewRenderer.MustLoadTemplate(w, r, "results/result.html", &resultsViewTemplateVars{
		BaseTemplateVars: BaseTemplateVars{
			WideContainer: true,
		},
		Result:  result,
		Account: AccountFromRequest(r),
		UseMPH:  serverOpts.UseMPH == 1,
	})
}

func (rh *ResultsHandler) file(w http.ResponseWriter, r *http.Request) {
	fileName := chi.URLParam(r, "fileName")

	result, err := LoadResult(fileName)

	if os.IsNotExist(err) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if err != nil {
		logrus.WithError(err).Errorf("could not get result")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if UseShortenedDriverNames {
		result.MaskDriverNames()
	}

	w.Header().Add("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
}

func (rh *ResultsHandler) edit(w http.ResponseWriter, r *http.Request) {
	fileName := chi.URLParam(r, "fileName")

	results, err := LoadResult(fileName + ".json")

	if os.IsNotExist(err) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if err != nil {
		logrus.WithError(err).Error("could not load results")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	for key, vals := range r.Form {
		if strings.HasPrefix(key, "guid:") {
			guid := strings.TrimPrefix(key, "guid:")
			name := vals[0]

			results.RenameDriver(guid, name)
		}
	}

	err = saveResults(results.SessionFile+".json", results)

	if err != nil {
		logrus.WithError(err).Error("could not load results")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Drivers successfully edited")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

type vec struct {
	x    float64
	z    float64
	relX float64
	relZ float64

	color color.RGBA
}

func (rh *ResultsHandler) renderCollisions(w http.ResponseWriter, r *http.Request) {
	var collisionsToLoad []int

	fileName := chi.URLParam(r, "fileName")

	result, err := LoadResult(fileName + ".json")

	if os.IsNotExist(err) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if err != nil {
		logrus.WithError(err).Errorf("could not get result")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	collisions := r.URL.Query().Get("collisions")

	if collisions == "all" {
		for z := range result.Events {
			collisionsToLoad = append(collisionsToLoad, z)
		}
	} else {
		splitCollisions := strings.Split(collisions, ",")

		for i := range splitCollisions {

			collisionNum, err := strconv.Atoi(splitCollisions[i])

			if err != nil {
				continue
			}

			collisionsToLoad = append(collisionsToLoad, collisionNum)
		}
	}

	trackMapData, err := LoadTrackMapData(result.TrackName, result.TrackConfig)

	if err != nil {
		logrus.WithError(err).Error("could not load track map data")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	trackMapImage, err := LoadTrackMapImage(result.TrackName, result.TrackConfig)

	if err != nil {
		logrus.WithError(err).Error("could not load track map image")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var collisionVectors []vec

	for i, collision := range result.Events {

		var mainColor color.RGBA

		if collision.Type == "COLLISION_WITH_ENV" {
			mainColor = color.RGBA{R: 255, G: 125, B: 0, A: 0xff}
		} else {
			mainColor = color.RGBA{R: 255, G: 0, B: 0, A: 0xff}
		}

		for z := range collisionsToLoad {
			if collisionsToLoad[z] == i {
				collisionVectors = append(collisionVectors, vec{
					x:     (collision.WorldPosition.X + trackMapData.OffsetX) / trackMapData.ScaleFactor,
					z:     (collision.WorldPosition.Z + trackMapData.OffsetZ) / trackMapData.ScaleFactor,
					relX:  collision.RelPosition.X,
					relZ:  collision.RelPosition.Z,
					color: mainColor,
				})
			}
		}
	}

	img := image.NewRGBA(image.Rectangle{Min: image.Pt(0, 0), Max: image.Pt(trackMapImage.Bounds().Max.X, trackMapImage.Bounds().Max.Y)})
	radius := 4

	for _, collisionVector := range collisionVectors {
		if collisionVector.relX > 0 || collisionVector.relZ > 0 {
			// show the relative collision position
			draw.Draw(img, img.Bounds(), &circle{image.Pt(int(collisionVector.x+collisionVector.relX), int(collisionVector.z+collisionVector.relZ)), radius, color.RGBA{R: 0, G: 0, B: 255, A: 0xff}}, image.Pt(0, 0), draw.Over)
		}

		draw.Draw(img, img.Bounds(), &circle{image.Pt(int(collisionVector.x), int(collisionVector.z)), radius, collisionVector.color}, image.Pt(0, 0), draw.Over)
	}

	err = png.Encode(w, img)

	if err != nil {
		logrus.WithError(err).Error("could not encode image")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

type circle struct {
	p     image.Point
	r     int
	color color.RGBA
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return c.color
	}
	return color.RGBA{0, 0, 0, 0}
}

// saveResults takes a full json filepath (including the json extension) and saves the results to that file.
func saveResults(jsonFileName string, results *SessionResults) error {
	path := filepath.Join(ServerInstallPath, "results", jsonFileName)

	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")

	return encoder.Encode(results)
}
