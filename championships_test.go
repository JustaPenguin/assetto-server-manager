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
			e.RaceSetup.AddSession(session, SessionConfig{})
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
	}

	for _, fixture := range fixtures {
		c := NewChampionship(fixture.searchModel)

		for className, models := range fixture.classesToModels {
			class := NewChampionshipClass(className)

			for _, model := range models {
				for i := 0; i < rand.Intn(20); i++ {
					class.Entrants.Add(&Entrant{
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
