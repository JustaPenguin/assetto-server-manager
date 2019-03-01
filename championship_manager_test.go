package servermanager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cj123/assetto-server-manager/pkg/udp/replay"
)

var TestEntryList = EntryList{
	"CAR_0": {
		Name:  "",
		GUID:  "",
		Team:  "Team 1",
		Model: "rss_formula_rss_4",
		Skin:  "",
	},
	"CAR_1": {
		Name:  "",
		GUID:  "",
		Team:  "Team 1",
		Model: "rss_formula_rss_4",
		Skin:  "",
	},
	"CAR_2": {
		Name:  "",
		GUID:  "",
		Team:  "Team 2",
		Model: "rss_formula_rss_4",
		Skin:  "",
	},
	"CAR_3": {
		Name:  "",
		GUID:  "",
		Team:  "Team 2",
		Model: "rss_formula_rss_4",
		Skin:  "",
	},
	"CAR_4": {
		Name:  "",
		GUID:  "",
		Team:  "Team 2",
		Model: "rss_formula_rss_4",
		Skin:  "",
	},
	"CAR_5": {
		Name:  "",
		GUID:  "",
		Team:  "Team 2",
		Model: "rss_formula_rss_4",
		Skin:  "",
	},
}

type dummyServerProcess struct{}

func (dummyServerProcess) Logs() string {
	return ""
}

func (dummyServerProcess) Start() error {
	return nil
}

func (dummyServerProcess) Stop() error {
	return nil
}

func (dummyServerProcess) Restart() error {
	return nil
}

func (dummyServerProcess) IsRunning() bool {
	return true
}

var championshipEventFixtures = []string{
	"barbagello.json",
	"red-bull-ring.json",
	"barbagello-no-end-sessions.json",
}

func TestChampionshipManager_ChampionshipEventCallback(t *testing.T) {
	SetupRaceManager(NewJSONRaceStore(filepath.Join(os.TempDir(), "asm-race-store")))

	AssettoProcess = dummyServerProcess{}
	ServerInstallPath = filepath.Join("cmd", "server-manager", "assetto")

	t.Run("Basic championship flow, closed entrylist", func(t *testing.T) {
		// make a championship
		champ := NewChampionship("Test Championship")
		champ.OpenEntrants = false
		cl := NewChampionshipClass("Default")
		cl.Entrants = TestEntryList
		champ.AddClass(cl)

		for range championshipEventFixtures {
			e := NewChampionshipEvent()
			e.RaceSetup = ConfigIniDefault.CurrentRaceConfig

			champ.Events = append(champ.Events, e)
		}

		if err := championshipManager.UpsertChampionship(champ); err != nil {
			t.Error(err)
			return
		}

		for i, sessionFile := range championshipEventFixtures {
			t.Run(sessionFile, func(t *testing.T) {
				if err := championshipManager.StartEvent(champ.ID.String(), champ.Events[i].ID.String()); err != nil {
					t.Error(err)
					return
				}

				err := replay.ReplayUDPMessages(filepath.Join("fixtures", sessionFile), 1000, championshipManager.ChampionshipEventCallback, false)

				if err != nil {
					t.Error(err)
					return
				}

				checkChampionshipEventCompletion(t, champ.ID.String(), i)
			})
		}

		checkAutoPopulatedEntryList(t, champ.ID.String(), 0)
	})

	t.Run("Basic championship flow, open entrylist", func(t *testing.T) {
		// make a championship
		champ := NewChampionship("Test Championship")
		champ.OpenEntrants = true
		cl := NewChampionshipClass("Default")
		cl.Entrants = TestEntryList
		champ.AddClass(cl)

		for range championshipEventFixtures {
			e := NewChampionshipEvent()
			e.RaceSetup = ConfigIniDefault.CurrentRaceConfig

			champ.Events = append(champ.Events, e)
		}

		if err := championshipManager.UpsertChampionship(champ); err != nil {
			t.Error(err)
			return
		}

		for i, sessionFile := range championshipEventFixtures {
			t.Run(sessionFile, func(t *testing.T) {
				if err := championshipManager.StartEvent(champ.ID.String(), champ.Events[i].ID.String()); err != nil {
					t.Error(err)
					return
				}

				err := replay.ReplayUDPMessages(filepath.Join("fixtures", sessionFile), 1000, championshipManager.ChampionshipEventCallback, false)

				if err != nil {
					t.Error(err)
					return
				}

				checkChampionshipEventCompletion(t, champ.ID.String(), i)
			})
		}

		checkAutoPopulatedEntryList(t, champ.ID.String(), 2)
	})

	t.Run("Basic championship flow, open entrylist with only one free slot", func(t *testing.T) {
		// make a championship
		champ := NewChampionship("Test Championship")
		champ.OpenEntrants = true
		cl := NewChampionshipClass("Default")
		cl.Entrants = EntryList{
			"CAR_0": {
				Name:  "",
				GUID:  "",
				Team:  "Team 1",
				Model: "rss_formula_rss_4",
				Skin:  "",
			},
		}
		champ.AddClass(cl)

		for range championshipEventFixtures {
			e := NewChampionshipEvent()
			e.RaceSetup = ConfigIniDefault.CurrentRaceConfig

			champ.Events = append(champ.Events, e)
		}

		if err := championshipManager.UpsertChampionship(champ); err != nil {
			t.Error(err)
			return
		}

		for i, sessionFile := range championshipEventFixtures {
			t.Run(sessionFile, func(t *testing.T) {
				if err := championshipManager.StartEvent(champ.ID.String(), champ.Events[i].ID.String()); err != nil {
					t.Error(err)
					return
				}

				err := replay.ReplayUDPMessages(filepath.Join("fixtures", sessionFile), 1000, championshipManager.ChampionshipEventCallback, false)

				if err != nil {
					t.Error(err)
					return
				}

				checkChampionshipEventCompletion(t, champ.ID.String(), i)
			})
		}

		checkAutoPopulatedEntryList(t, champ.ID.String(), 1)
	})

	t.Run("Basic championship flow, open entrylist with one free slot for a non-matching car", func(t *testing.T) {
		// make a championship
		champ := NewChampionship("Test Championship")
		champ.OpenEntrants = true
		cl := NewChampionshipClass("Default")
		cl.Entrants = EntryList{
			"CAR_0": {
				Name:  "",
				GUID:  "",
				Team:  "Team 1",
				Model: "bmw_m3",
				Skin:  "",
			},
		}
		champ.AddClass(cl)

		for range championshipEventFixtures {
			e := NewChampionshipEvent()
			e.RaceSetup = ConfigIniDefault.CurrentRaceConfig

			champ.Events = append(champ.Events, e)
		}

		if err := championshipManager.UpsertChampionship(champ); err != nil {
			t.Error(err)
			return
		}

		for i, sessionFile := range championshipEventFixtures {
			t.Run(sessionFile, func(t *testing.T) {
				if err := championshipManager.StartEvent(champ.ID.String(), champ.Events[i].ID.String()); err != nil {
					t.Error(err)
					return
				}

				err := replay.ReplayUDPMessages(filepath.Join("fixtures", sessionFile), 1000, championshipManager.ChampionshipEventCallback, false)

				if err != nil {
					t.Error(err)
					return
				}

				checkChampionshipEventCompletion(t, champ.ID.String(), i)
			})
		}

		checkAutoPopulatedEntryList(t, champ.ID.String(), 0)
	})
}

func checkAutoPopulatedEntryList(t *testing.T, championshipID string, expected int) {
	loadedChampionship, err := championshipManager.LoadChampionship(championshipID)

	if err != nil {
		t.Error(err)
		return
	}

	numPopulatedEntrants := 0

	for _, entrant := range loadedChampionship.AllEntrants() {
		if entrant.Name != "" && entrant.GUID != "" {
			numPopulatedEntrants++
		}
	}

	if numPopulatedEntrants != expected {
		t.Fail()
	}
}

func checkChampionshipEventCompletion(t *testing.T, championshipID string, eventID int) {
	// now look at the championship event and see if it has a start/end time
	loadedChampionship, err := championshipManager.LoadChampionship(championshipID)

	if err != nil {
		t.Error(err)
		return
	}

	if eventID >= len(loadedChampionship.Events) || eventID < 0 {
		t.Logf("Invalid event ID %d", eventID)
		t.Fail()
		return
	}

	if loadedChampionship.Events[eventID].StartedTime.IsZero() {
		t.Logf("Invalid championship event start time (zero)")
		t.Fail()
		return
	}

	if loadedChampionship.Events[eventID].CompletedTime.IsZero() {
		t.Logf("Invalid championship event completed time (zero)")
		t.Fail()
		return
	}

	for _, sess := range loadedChampionship.Events[eventID].Sessions {
		if sess.StartedTime.IsZero() {
			t.Logf("Invalid session start time (zero)")
			t.Fail()
			return
		}

		if sess.CompletedTime.IsZero() {
			t.Logf("Invalid session end time (zero)")
			t.Fail()
			return
		}
	}
}
