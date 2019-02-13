package servermanager

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"path/filepath"
	"time"
)

var championshipManager *ChampionshipManager

// DefaultChampionshipPoints is the Formula 1 points system.
var DefaultChampionshipPoints = ChampionshipPoints{
	Places: []int{
		25,
		18,
		15,
		12,
		10,
		8,
		6,
		4,
		2,
		1,
	},
	BestLap:      0,
	PolePosition: 0,
}

type ChampionshipPoints struct {
	Places       []int
	BestLap      int
	PolePosition int
}

func NewChampionship(name string) *Championship {
	return &Championship{
		ID:   uuid.New(),
		Name: name,
	}
}

type Championship struct {
	ID   uuid.UUID
	Name string

	Entrants EntryList

	Races  []ChampionshipRace
	Points ChampionshipPoints
}

func (c *Championship) ValidCarIDs() []string {
	cars := make(map[string]bool)

	for _, e := range c.Entrants {
		cars[e.Model] = true
	}

	var out []string

	for car := range cars {
		out = append(out, car)
	}

	return out
}

func (c *Championship) Progress() float64 {
	numRaces := float64(len(c.Races))

	if numRaces == 0 {
		return 0
	}

	numCompletedRaces := float64(0)

	for _, race := range c.Races {
		if race.Completed() {
			numCompletedRaces++
		}
	}

	return (numCompletedRaces / numRaces) * 100
}

type ChampionshipRace struct {
	RaceSetup CustomRace
	Results   map[SessionType]SessionResults

	CompletedTime time.Time
}

func (cr *ChampionshipRace) Completed() bool {
	return !cr.CompletedTime.IsZero()
}

func listChampionshipsHandler(w http.ResponseWriter, r *http.Request) {
	championships, err := championshipManager.ListChampionships()

	if err != nil {
		panic(err)
	}

	ViewRenderer.MustLoadTemplate(w, r, filepath.Join("championships", "index.html"), map[string]interface{}{
		"championships": championships,
	})
}

func newChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	opts, err := championshipManager.BuildChampionshipOpts(r)

	if err != nil {
		panic(err)
	}

	ViewRenderer.MustLoadTemplate(w, r, filepath.Join("championships", "new.html"), opts)
}

func submitNewChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.HandleCreateChampionship(r)

	if err != nil {
		logrus.Errorf("couldn't create championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/championship/"+championship.ID.String()+"/race", http.StatusFound)
}

func championshipRaceConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	championshipRaceOpts, err := championshipManager.BuildChampionshipRaceOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build championship race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, filepath.Join("custom-race", "new.html"), championshipRaceOpts)
}
