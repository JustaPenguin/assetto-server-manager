package servermanager

import (
	"math/rand"
	"testing"
)

type lastSessionTest struct {
	expected SessionType

	sessions []SessionType
}

func TestChampionshipEvent_LastSession(t *testing.T) {
	lastSessionTests := []lastSessionTest{
		{
			expected: SessionTypeRace,

			sessions: []SessionType{
				SessionTypeRace,
				SessionTypeQualifying,
				SessionTypePractice,
				SessionTypeBooking,
			},
		},
		{
			expected: SessionTypeQualifying,

			sessions: []SessionType{
				SessionTypeQualifying,
				SessionTypePractice,
				SessionTypeBooking,
			},
		},
		{
			expected: SessionTypePractice,

			sessions: []SessionType{
				SessionTypeBooking,
				SessionTypePractice,
			},
		},
		{
			expected: SessionTypeBooking,

			sessions: []SessionType{
				SessionTypeBooking,
			},
		},
		{
			expected: SessionTypeBooking,

			sessions: []SessionType{},
		},
		{
			expected: SessionTypeRace,

			sessions: []SessionType{
				SessionTypeRace,
			},
		},
		{
			expected: SessionTypePractice,

			sessions: []SessionType{
				SessionTypePractice,
			},
		},
		{
			expected: SessionTypeQualifying,

			sessions: []SessionType{
				SessionTypeQualifying,
				SessionTypeBooking,
			},
		},
	}

	for _, x := range lastSessionTests {
		e := NewChampionshipEvent()

		for _, session := range x.sessions {
			e.RaceSetup.AddSession(session, &SessionConfig{})
		}

		if e.LastSession() != x.expected {
			t.Logf("Expected: %s, got %s", x.expected, e.LastSession())
			t.Fail()
		}
	}
}

type carClassModelTest struct {
	searchModel       string
	expectedClassName string

	classesToModels map[string][]string
}

func TestChampionship_FindClassForCarModel(t *testing.T) {
	fixtures := []carClassModelTest{
		{
			searchModel:       "ks_lambo",
			expectedClassName: "KS",

			classesToModels: map[string][]string{
				"KS": {
					"ks_merc",
					"ks_ferrari",
					"ks_lambo",
				},
			},
		},
		{
			searchModel:       "ks_lambo",
			expectedClassName: "KS",

			classesToModels: map[string][]string{
				"KS": {
					"ks_merc",
					"ks_ferrari",
					"ks_lambo",
				},
				"GT3": {
					"fast_gt3",
					"porsche_gt3",
					"ks_gt3_rs",
				},
			},
		},
		{
			searchModel:       "ks_lambo2",
			expectedClassName: "",

			classesToModels: map[string][]string{
				"KS": {
					"ks_merc",
					"ks_ferrari",
					"ks_lambo",
				},
				"GT3": {
					"fast_gt3",
					"porsche_gt3",
					"ks_gt3_rs",
				},
			},
		},
		{
			searchModel:       "porsche_gt3",
			expectedClassName: "GT3",

			classesToModels: map[string][]string{
				"KS": {
					"ks_merc",
					"ks_ferrari",
					"ks_lambo",
				},
				"GT3": {
					"fast_gt3",
					"porsche_gt3",
					"ks_gt3_rs",
				},
			},
		},
	}

	for _, fixture := range fixtures {
		c := NewChampionship(fixture.searchModel)

		for className, models := range fixture.classesToModels {
			class := NewChampionshipClass(className)
			class.AvailableCars = models

			for _, model := range models {
				for i := 0; i < rand.Intn(20)+1; i++ {
					class.Entrants.AddToBackOfGrid(&Entrant{
						Model: model,
					})
				}
			}

			c.AddClass(class)
		}

		foundClass, err := c.FindClassForCarModel(fixture.searchModel)

		if err != nil && fixture.expectedClassName != "" {
			t.Fail()
		}

		if foundClass == nil && fixture.expectedClassName == "" {
			continue // pass
		}

		if foundClass.Name != fixture.expectedClassName {
			t.Fail()
		}
	}
}

func TestChampionship_AddEntrantFromSession(t *testing.T) {
	t.Run("Added entrant with team name", func(t *testing.T) {
		potentialEntrant := &SessionCar{
			Driver: SessionDriver{
				GUID:      "78987656782716273",
				GuidsList: []string{"78987656782716273"},
				Name:      "Driver 1",
				Team:      "Team Name",
			},
			Model: "ferrari_fxx_k",
			Skin:  "skin_01",
		}

		class := NewChampionshipClass("FXX K")
		class.Entrants.AddToBackOfGrid(&Entrant{
			Model: "ferrari_fxx_k",
		})

		champ := &Championship{}
		champ.AddClass(class)

		foundSlot, _, _, err := champ.AddEntrantFromSession(potentialEntrant)

		if err != nil {
			t.Error(err)
		}

		if !foundSlot {
			t.Log("Expected to find slot, did not")
			t.Fail()
			return
		}

		for _, entrant := range class.Entrants {
			if entrant.GUID == "78987656782716273" {
				if entrant.Team == "Team Name" {
					return // pass
				}
			}
		}

		t.Log("Could not find entrant for guid")
		t.Fail()
	})

	t.Run("Added entrant with team name, then added them again without. Team name should persist (#386)", func(t *testing.T) {
		potentialEntrant := &SessionCar{
			Driver: SessionDriver{
				GUID:      "78987656782716273",
				GuidsList: []string{"78987656782716273"},
				Name:      "Driver 1",
				Team:      "Team Name",
			},
			Model: "ferrari_fxx_k",
			Skin:  "skin_01",
		}

		class := NewChampionshipClass("FXX K")
		class.Entrants.AddToBackOfGrid(&Entrant{
			Model: "ferrari_fxx_k",
		})

		champ := &Championship{}
		champ.AddClass(class)

		foundSlot, _, _, err := champ.AddEntrantFromSession(potentialEntrant)

		if err != nil {
			t.Error(err)
		}

		if !foundSlot {
			t.Log("Expected to find slot, did not")
			t.Fail()
			return
		}

		ok := false

		for _, entrant := range class.Entrants {
			if entrant.GUID == "78987656782716273" {
				if entrant.Team == "Team Name" {
					ok = true
					break
				}
			}
		}

		if !ok {
			t.Log("Could not find entrant for guid")
			t.Fail()
		}

		// re-add the entrant, this time with no team
		potentialEntrant.Driver.Team = ""

		foundSlot, _, _, err = champ.AddEntrantFromSession(potentialEntrant)

		if err != nil {
			t.Error(err)
		}

		if !foundSlot {
			t.Log("Expected to find slot, did not")
			t.Fail()
			return
		}

		for _, entrant := range class.Entrants {
			if entrant.GUID == "78987656782716273" {
				if entrant.Team == "" {
					t.Log("Entrant had a team name, it has now been removed")
					t.Fail()
					return
				}
			}
		}
	})

	t.Run("Added entrant with team name, then added them again with a new team name. New team name should persist (#386)", func(t *testing.T) {
		potentialEntrant := &SessionCar{
			Driver: SessionDriver{
				GUID:      "78987656782716273",
				GuidsList: []string{"78987656782716273"},
				Name:      "Driver 1",
				Team:      "Team Name",
			},
			Model: "ferrari_fxx_k",
			Skin:  "skin_01",
		}

		class := NewChampionshipClass("FXX K")
		class.Entrants.AddToBackOfGrid(&Entrant{
			Model: "ferrari_fxx_k",
		})

		champ := &Championship{}
		champ.AddClass(class)

		foundSlot, _, _, err := champ.AddEntrantFromSession(potentialEntrant)

		if err != nil {
			t.Error(err)
		}

		if !foundSlot {
			t.Log("Expected to find slot, did not")
			t.Fail()
			return
		}

		ok := false

		for _, entrant := range class.Entrants {
			if entrant.GUID == "78987656782716273" {
				if entrant.Team == "Team Name" {
					ok = true
					break
				}
			}
		}

		if !ok {
			t.Log("Could not find entrant for guid")
			t.Fail()
		}

		// re-add the entrant, this time with a new team name
		potentialEntrant.Driver.Team = "New Team Name"

		foundSlot, _, _, err = champ.AddEntrantFromSession(potentialEntrant)

		if err != nil {
			t.Error(err)
		}

		if !foundSlot {
			t.Log("Expected to find slot, did not")
			t.Fail()
			return
		}

		for _, entrant := range class.Entrants {
			if entrant.GUID == "78987656782716273" {
				if entrant.Team != "New Team Name" {
					t.Log("Entrant team name was not correctly updated")
					t.Fail()
					return
				}
			}
		}
	})

	t.Run("Entrant joined and was placed in a different grid slot. Their extra properties (especially InternalUUID) should persist (#441)", func(t *testing.T) {
		potentialEntrant := &SessionCar{
			Driver: SessionDriver{
				GUID:      "78987656782716273",
				GuidsList: []string{"78987656782716273"},
				Name:      "Driver 1",
				Team:      "Team Name",
			},
			Model: "ferrari_fxx_k",
			Skin:  "skin_01",
		}

		class := NewChampionshipClass("FXX K")
		e := NewEntrant()
		e.Model = "ferrari_fxx_k"

		class.Entrants.AddToBackOfGrid(e)

		e2 := NewEntrant()
		e2.Model = "ferrari_fxx_k"
		e2.GUID = "78987656782716273"

		class.Entrants.AddToBackOfGrid(e2)

		champ := &Championship{}
		champ.AddClass(class)

		emptySlotUUID := e.InternalUUID
		currentInternalUUID := e2.InternalUUID

		_, _, _, err := champ.AddEntrantFromSession(potentialEntrant)

		if err != nil {
			t.Error(err)
		}

		for _, entrant := range class.Entrants {
			if entrant.GUID == "78987656782716273" && entrant.InternalUUID != currentInternalUUID {
				t.Log("Expected entrant guid to have carried over, it did not")
				t.Fail()
			}

			if entrant.GUID == "" && entrant.InternalUUID != emptySlotUUID {
				t.Log("Expected entrant guid swap")
				t.Fail()
			}
		}
	})
}
