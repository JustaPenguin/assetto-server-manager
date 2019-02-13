package servermanager

import (
	"net/http"
	"strings"

	"github.com/etcd-io/bbolt"
	"github.com/gorilla/mux"
)

type ChampionshipManager struct {
	*RaceManager
}

func NewChampionshipManager(rm *RaceManager) *ChampionshipManager {
	return &ChampionshipManager{
		RaceManager: rm,
	}
}

func (cm *ChampionshipManager) LoadChampionship(id string) (*Championship, error) {
	return cm.raceStore.LoadChampionship(id)
}

func (cm *ChampionshipManager) UpsertChampionship(c *Championship) error {
	return cm.raceStore.UpsertChampionship(c)
}

func (cm *ChampionshipManager) ListChampionships() ([]*Championship, error) {
	championships, err := cm.raceStore.ListChampionships()

	if err == bbolt.ErrBucketNotFound {
		return nil, nil
	}

	return championships, err
}

func (cm *ChampionshipManager) BuildChampionshipOpts(r *http.Request) (map[string]interface{}, error) {
	raceOpts, err := cm.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	raceOpts["DefaultPoints"] = DefaultChampionshipPoints

	return raceOpts, nil
}

func (cm *ChampionshipManager) HandleCreateChampionship(r *http.Request) (*Championship, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	championship := NewChampionship(r.FormValue("ChampionshipName"))

	var err error

	championship.Entrants, err = cm.BuildEntryList(r)

	if err != nil {
		return nil, err
	}

	for i := 0; i < len(r.Form["Points.Place"]); i++ {
		championship.Points.Places = append(championship.Points.Places, formValueAsInt(r.Form["Points.Place"][i]))
	}

	championship.Points.PolePosition = formValueAsInt(r.FormValue("Points.PolePosition"))
	championship.Points.BestLap = formValueAsInt(r.FormValue("Points.BestLap"))

	return championship, cm.raceStore.UpsertChampionship(championship)
}

func (cm *ChampionshipManager) BuildChampionshipRaceOpts(r *http.Request) (map[string]interface{}, error) {
	opts, err := cm.BuildRaceOpts(r)

	if err != nil {
		return nil, err
	}

	// here we customise the opts to tell the template that this is a championship race.
	championship, err := cm.raceStore.LoadChampionship(mux.Vars(r)["championshipID"])

	if err != nil {
		return nil, err
	}

	opts["IsChampionship"] = true
	opts["Championship"] = championship

	// override Current race config if there is a previous championship race configured
	if len(championship.Races) > 0 {
		opts["Current"] = championship.Races[len(championship.Races)-1].RaceSetup
		opts["ChampionshipHasAtLeastOnceRace"] = true
	} else {
		opts["Current"] = ConfigIniDefault.CurrentRaceConfig
		opts["ChampionshipHasAtLeastOnceRace"] = false
	}

	return opts, nil
}

func (cm *ChampionshipManager) SaveChampionshipRace(r *http.Request) (*Championship, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	championship, err := cm.raceStore.LoadChampionship(mux.Vars(r)["championshipID"])

	if err != nil {
		return nil, err
	}

	raceConfig, err := cm.BuildCustomRaceFromForm(r)

	if err != nil {
		return nil, err
	}

	raceConfig.Cars = strings.Join(championship.ValidCarIDs(), ";")

	championship.Races = append(championship.Races, &ChampionshipRace{
		RaceSetup: *raceConfig,
	})

	return championship, cm.raceStore.UpsertChampionship(championship)
}
