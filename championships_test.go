package servermanager

import "testing"

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
