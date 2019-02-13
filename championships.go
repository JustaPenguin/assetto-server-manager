package servermanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
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

	Events []*ChampionshipEvent
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
	numRaces := float64(len(c.Events))

	if numRaces == 0 {
		return 0
	}

	numCompletedRaces := float64(0)

	for _, race := range c.Events {
		if race.Completed() {
			numCompletedRaces++
		}
	}

	return (numCompletedRaces / numRaces) * 100
}

type ChampionshipStanding struct {
	Entrant Entrant
	Points  int
}

func (c *Championship) PointForPos(i int) int {
	if i >= len(c.Points.Places) {
		return 0
	}

	return c.Points.Places[i]
}

func (c *Championship) Standings() []*ChampionshipStanding {
	var out []*ChampionshipStanding

	entrants := make(map[string]*ChampionshipStanding)

	if len(c.Events) > 0 {
		for _, entrant := range c.Entrants {
			if entrants[entrant.GUID] == nil {
				entrants[entrant.GUID] = &ChampionshipStanding{
					Entrant: entrant,
				}
			}
		}
	}

	for _, event := range c.Events {
		race, ok := event.Results[SessionTypeRace]

		if !ok {
			continue
		}

		for pos, driver := range race.Result {
			entrants[driver.DriverGUID].Points += c.PointForPos(pos)
		}
	}

	for _, entrant := range entrants {
		out = append(out, entrant)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Points > out[j].Points
	})

	return out
}

type TeamStanding struct {
	Team   string
	Points int
}

func (c *Championship) TeamStandings() []*TeamStanding {
	teams := make(map[string]int)

	for _, driver := range c.Standings() {
		if _, ok := teams[driver.Entrant.Team]; !ok {
			teams[driver.Entrant.Team] = driver.Points
		} else {
			teams[driver.Entrant.Team] += driver.Points
		}
	}

	var out []*TeamStanding

	for name, pts := range teams {
		out = append(out, &TeamStanding{
			Team:   name,
			Points: pts,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Points > out[j].Points
	})

	return out
}

type ChampionshipEvent struct {
	RaceSetup CurrentRaceConfig
	Results   map[SessionType]*SessionResults

	CompletedTime time.Time
}

func (cr *ChampionshipEvent) Completed() bool {
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

func exportChampionshipHandler(w http.ResponseWriter, r *http.Request) {
	championship, err := championshipManager.LoadChampionship(mux.Vars(r)["championshipID"])

	if err != nil {
		panic(err)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(championship)
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
			prettifyName(championship.Events[len(championship.Events)-1].RaceSetup.Track, false),
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
