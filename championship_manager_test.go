package servermanager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cj123/assetto-server-manager/pkg/udp/replay"
)

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

var championshipEventFixtures = []string{
	"barbagello.json",
	"red-bull-ring.json",
	"barbagello-no-end-sessions.json",
}

func TestChampionshipManager_ChampionshipEventCallback(t *testing.T) {
	SetupRaceManager(NewJSONRaceStore(os.TempDir()))

	AssettoProcess = dummyServerProcess{}
	ServerInstallPath = filepath.Join("cmd", "server-manager", "assetto")

	// make a championship
	champ := NewChampionship("Test Championship")
	champ.Entrants = TestEntryList

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
			if err := championshipManager.StartEvent(champ.ID.String(), i); err != nil {
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
