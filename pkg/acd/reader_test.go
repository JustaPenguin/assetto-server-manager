package acd

import "testing"

func TestEncryptionKey(t *testing.T) {
	expected := "247-76-61-254-174-237-57-53"
	input := "rss_formula_rss_4"

	if cipherKey(input) != expected {
		t.Logf("wanted %s, got %s", expected, cipherKey(input))
		t.Fail()
	}
}
