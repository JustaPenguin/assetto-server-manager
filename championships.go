package servermanager

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
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
		ID:      uuid.New(),
		Name:    name,
		Created: time.Now(),
	}
}

type Championship struct {
	ID      uuid.UUID
	Name    string
	Created time.Time
	Updated time.Time
	Deleted time.Time

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
	RaceSetup CurrentRaceConfig
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

func viewChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.LoadChampionship(mux.Vars(r)["championshipID"])

	if err != nil {
		panic(err)
	}

	ViewRenderer.MustLoadTemplate(w, r, filepath.Join("championships", "view.html"), map[string]interface{}{
		"Championship": championship,
	})
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

func championshipSubmitRaceConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.SaveChampionshipRace(r)

	if err != nil {
		logrus.Errorf("couldn't build championship race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlashQuick(w, r,
		fmt.Sprintf(
			"Championship race at %s was successfully added!",
			prettifyName(championship.Races[len(championship.Races)-1].RaceSetup.Track, false),
		),
	)

	if r.FormValue("action") == "saveChampionship" {
		// end the race creation flow
		http.Redirect(w, r, "/championship/"+championship.ID.String(), http.StatusFound)
		return
	} else {
		// add another race
		http.Redirect(w, r, "/championship/"+championship.ID.String()+"/race", http.StatusFound)
	}
}
