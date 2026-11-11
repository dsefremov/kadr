package media

import (
	"context"
	"testing"
)

func TestSetTakenDateBadInput(t *testing.T) {
	err := SetTakenDate(context.Background(), nil, 1, "x", "not-a-date")
	if err != ErrBadDate {
		t.Fatalf("want ErrBadDate, got %v", err)
	}
}
