package servermanager

import (
	"testing"
)

type cleanGUIDTest struct {
	guid     string
	expected string
}

var cleanGUIDsTests = []cleanGUIDTest{
	{guid: "5463726354637263543", expected: "5463726354637263543"},
	{guid: "5463726354637263543 ", expected: "5463726354637263543"},
	{guid: "S5463726354637263543", expected: "5463726354637263543"},
}

func TestCleanGUIDs(t *testing.T) {
	for _, test := range cleanGUIDsTests {
		out := CleanGUIDs([]string{test.guid})

		if out[0] != test.expected {
			t.Fail()
		}
	}
}
