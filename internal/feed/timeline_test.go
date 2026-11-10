package feed

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	taken := time.Date(2024, 3, 2, 15, 4, 5, 0, time.UTC)
	c := encodeCursor(taken, 42)
	t2, id, err := decodeCursor(c)
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 || !t2.Equal(taken) {
		t.Fatalf("cursor mismatch %v %d", t2, id)
	}
}
