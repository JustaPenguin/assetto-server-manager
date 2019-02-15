package servermanager

import (
	"encoding/json"
	"fmt"
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func init() {
	err := SetupRaceManager(filepath.Join(os.TempDir(), "assetto-store.db"))

	if err != nil {
		panic(err)
	}

	AssettoProcess = dummyServerProcess{}
	ServerInstallPath = filepath.Join("cmd", "server-manager", "assetto")
}

var TestEntryList = EntryList{
	"CAR_0": {
		Name:  "John Doe",
		GUID:  "8623812638761238",
		Team:  "Team 1",
		Model: "ks_mazda_miata",
		Skin:  "00_classic_red",
	},
	"CAR_1": {
		Name:  "Jane Doe",
		GUID:  "8623812638761222",
		Team:  "Team 1",
		Model: "ks_mazda_miata",
		Skin:  "00_classic_red",
	},
	"CAR_2": {
		Name:  "Steve Smith",
		GUID:  "8655432638761222",
		Team:  "Team 2",
		Model: "ks_mazda_miata",
		Skin:  "00_classic_red",
	},
	"CAR_3": {
		Name:  "Sue Smith",
		GUID:  "8655432638761222",
		Team:  "Team 2",
		Model: "ks_mazda_miata",
		Skin:  "00_classic_red",
	},
	"CAR_4": {
		Name:  "Michael Scott",
		GUID:  "8655432638761222",
		Team:  "Team 2",
		Model: "ks_mazda_miata",
		Skin:  "00_classic_red",
	},
	"CAR_5": {
		Name:  "Dwight Schrute",
		GUID:  "8655432638761222",
		Team:  "Team 2",
		Model: "ks_mazda_miata",
		Skin:  "00_classic_red",
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

func TestChampionshipManager_ChampionshipEventCallback(t *testing.T) {
	// make a championship
	champ := NewChampionship("Test Championship")
	champ.Entrants = TestEntryList

	e := NewChampionshipEvent()
	e.RaceSetup = ConfigIniDefault.CurrentRaceConfig

	e1 := NewChampionshipEvent()
	e1.RaceSetup = ConfigIniDefault.CurrentRaceConfig

	e2 := NewChampionshipEvent()
	e2.RaceSetup = ConfigIniDefault.CurrentRaceConfig

	champ.Events = append(champ.Events, e, e1, e2)

	if err := championshipManager.UpsertChampionship(champ); err != nil {
		t.Error(err)
		return
	}

	t.Run("First event has EndSession message", func(t *testing.T) {
		startedCh := make(chan struct{})

		if err := championshipManager.StartEvent(champ.ID.String(), 0, startedCh); err != nil {
			t.Error(err)
			return
		}

		// send a new session event
		go func() {
			// for some reason assetto sends two of these, so lets do the same
			for i := 0; i < 2; i++ {
				championshipManager.ChampionshipEventCallback(udp.SessionInfo{
					EventType:   udp.EventNewSession,
					Name:        "Practice",
					Type:        1,
					Track:       e.RaceSetup.Track,
					TrackConfig: e.RaceSetup.TrackLayout,
					Time:        15,
				})
			}
		}()

		<-startedCh

		fname, err := createResultsFile("PRACTICE")

		if err != nil {
			t.Error(err)
			return
		}

		// now pretend we've finished the session
		championshipManager.ChampionshipEventCallback(udp.EndSession(fname))

		// now do a qualifying
		championshipManager.ChampionshipEventCallback(udp.SessionInfo{
			EventType:   udp.EventNewSession,
			Name:        "Qualify",
			Type:        2,
			Track:       e.RaceSetup.Track,
			TrackConfig: e.RaceSetup.TrackLayout,
			Time:        15,
		})

		fname, err = createResultsFile("QUALIFY")

		if err != nil {
			t.Error(err)
			return
		}

		// now pretend we've finished the session
		championshipManager.ChampionshipEventCallback(udp.EndSession(fname))

		// now do a race
		championshipManager.ChampionshipEventCallback(udp.SessionInfo{
			EventType:   udp.EventNewSession,
			Name:        "Race",
			Type:        3,
			Track:       e.RaceSetup.Track,
			TrackConfig: e.RaceSetup.TrackLayout,
			Time:        15,
		})

		fname, err = createResultsFile("RACE")

		if err != nil {
			t.Error(err)
			return
		}

		// now pretend we've finished the session
		championshipManager.ChampionshipEventCallback(udp.EndSession(fname))

		// now pretend the session has looped
		championshipManager.ChampionshipEventCallback(udp.SessionInfo{
			EventType:   udp.EventNewSession,
			Name:        "Practice",
			Type:        1,
			Track:       e.RaceSetup.Track,
			TrackConfig: e.RaceSetup.TrackLayout,
			Time:        15,
		})

		checkChampionshipEventCompletion(t, champ.ID.String(), 0)
	})

	t.Run("Second event doesn't have EndSession message for Race", func(t *testing.T) {
		startedCh := make(chan struct{})

		if err := championshipManager.StartEvent(champ.ID.String(), 1, startedCh); err != nil {
			t.Error(err)
			return
		}

		// send a new session event
		go func() {
			// for some reason assetto sends two of these, so lets do the same
			for i := 0; i < 2; i++ {
				championshipManager.ChampionshipEventCallback(udp.SessionInfo{
					EventType:   udp.EventNewSession,
					Name:        "Practice",
					Type:        1,
					Track:       e.RaceSetup.Track,
					TrackConfig: e.RaceSetup.TrackLayout,
					Time:        15,
				})
			}
		}()

		<-startedCh

		fname, err := createResultsFile("PRACTICE")

		if err != nil {
			t.Error(err)
			return
		}

		// now pretend we've finished the session
		championshipManager.ChampionshipEventCallback(udp.EndSession(fname))

		// now do a qualifying
		championshipManager.ChampionshipEventCallback(udp.SessionInfo{
			EventType:   udp.EventNewSession,
			Name:        "Qualify",
			Type:        2,
			Track:       e.RaceSetup.Track,
			TrackConfig: e.RaceSetup.TrackLayout,
			Time:        15,
		})

		fname, err = createResultsFile("QUALIFY")

		if err != nil {
			t.Error(err)
			return
		}

		// now pretend we've finished the session
		championshipManager.ChampionshipEventCallback(udp.EndSession(fname))

		// now do a race
		championshipManager.ChampionshipEventCallback(udp.SessionInfo{
			EventType:   udp.EventNewSession,
			Name:        "Race",
			Type:        3,
			Track:       e.RaceSetup.Track,
			TrackConfig: e.RaceSetup.TrackLayout,
			Time:        15,
		})

		fname, err = createResultsFile("RACE")

		if err != nil {
			t.Error(err)
			return
		}

		// loop
		championshipManager.ChampionshipEventCallback(udp.SessionInfo{
			EventType:   udp.EventNewSession,
			Name:        "Practice",
			Type:        1,
			Track:       e.RaceSetup.Track,
			TrackConfig: e.RaceSetup.TrackLayout,
			Time:        15,
		})

		checkChampionshipEventCompletion(t, champ.ID.String(), 1)
	})

	t.Run("Third event doesn't have end session messages for any sessions", func(t *testing.T) {
		startedCh := make(chan struct{})

		if err := championshipManager.StartEvent(champ.ID.String(), 2, startedCh); err != nil {
			t.Error(err)
			return
		}

		// send a new session event
		go func() {
			// for some reason assetto sends two of these, so lets do the same
			for i := 0; i < 2; i++ {
				championshipManager.ChampionshipEventCallback(udp.SessionInfo{
					EventType:   udp.EventNewSession,
					Name:        "Practice",
					Type:        1,
					Track:       e.RaceSetup.Track,
					TrackConfig: e.RaceSetup.TrackLayout,
					Time:        15,
				})
			}
		}()

		<-startedCh

		_, err := createResultsFile("PRACTICE")

		if err != nil {
			t.Error(err)
			return
		}

		// now do a qualifying
		championshipManager.ChampionshipEventCallback(udp.SessionInfo{
			EventType:   udp.EventNewSession,
			Name:        "Qualify",
			Type:        2,
			Track:       e.RaceSetup.Track,
			TrackConfig: e.RaceSetup.TrackLayout,
			Time:        15,
		})

		_, err = createResultsFile("QUALIFY")

		if err != nil {
			t.Error(err)
			return
		}

		// now do a race
		championshipManager.ChampionshipEventCallback(udp.SessionInfo{
			EventType:   udp.EventNewSession,
			Name:        "Race",
			Type:        3,
			Track:       e.RaceSetup.Track,
			TrackConfig: e.RaceSetup.TrackLayout,
			Time:        15,
		})

		_, err = createResultsFile("RACE")

		if err != nil {
			t.Error(err)
			return
		}

		// loop
		championshipManager.ChampionshipEventCallback(udp.SessionInfo{
			EventType:   udp.EventNewSession,
			Name:        "Practice",
			Type:        1,
			Track:       e.RaceSetup.Track,
			TrackConfig: e.RaceSetup.TrackLayout,
			Time:        15,
		})

		checkChampionshipEventCompletion(t, champ.ID.String(), 2)
	})
}

func checkChampionshipEventCompletion(t *testing.T, championshipID string, eventID int) {
	// now look at the championship event and see if it has a start/end time
	loadedChampionship, err := championshipManager.LoadChampionship(championshipID)

	if err != nil {
		t.Error(err)
		return
	}

	if eventID > len(loadedChampionship.Events) || eventID < 0 {
		t.Fail()
		return
	}

	if loadedChampionship.Events[eventID].StartedTime.IsZero() || loadedChampionship.Events[eventID].CompletedTime.IsZero() {
		t.Fail()
		return
	}

	for _, sess := range loadedChampionship.Events[eventID].Sessions {
		if sess.StartedTime.IsZero() || sess.CompletedTime.IsZero() {
			t.Fail()
			return
		}
	}
}

func createResultsFile(sess string) (string, error) {
	now := time.Now()

	minute := now.Minute()

	if sess == "QUALIFY" {
		minute += 20
	}

	if sess == "RACE" {
		minute += 40
	}

	fname := fmt.Sprintf("results/%d_%d_%d_%d_%d_%s.json", now.Year(), now.Month(), now.Day(), now.Hour(), minute, sess)

	f, err := os.Create(filepath.Join(ServerInstallPath, fname))

	if err != nil {
		return "", err
	}

	defer f.Close()

	return fname, json.NewEncoder(f).Encode(&SessionResults{})
}
