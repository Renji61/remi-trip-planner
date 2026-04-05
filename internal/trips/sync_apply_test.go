package trips

import "testing"

func TestParseSyncBaseCursor(t *testing.T) {
	if _, ok := parseSyncBaseCursor(""); ok {
		t.Fatal("empty")
	}
	if _, ok := parseSyncBaseCursor("abc"); ok {
		t.Fatal("non-numeric")
	}
	n, ok := parseSyncBaseCursor("42")
	if !ok || n != 42 {
		t.Fatalf("got %v %v", n, ok)
	}
}
