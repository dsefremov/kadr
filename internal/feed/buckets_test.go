package feed

import "testing"

func TestMonthLabel(t *testing.T) {
	if monthLabel("2026-11") != "Ноябрь 2026" {
		t.Fatalf("got %q", monthLabel("2026-11"))
	}
}
