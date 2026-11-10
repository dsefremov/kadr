package quota

import (
	"testing"

	"github.com/dsefremov/kadr/internal/units"
)

func TestStatusFor(t *testing.T) {
	limit := uint64(units.DefaultQuotaBytes)
	if StatusFor(0, 0, 0, limit) != StatusOK {
		t.Fatal("empty should be ok")
	}
	warn := limit * 91 / 100
	if StatusFor(warn, 0, 0, limit) != StatusWarning {
		t.Fatalf("91%% should warn, got %s", StatusFor(warn, 0, 0, limit))
	}
	if StatusFor(limit, 0, 0, limit) != StatusExceeded {
		t.Fatal("full should exceed")
	}
	if StatusFor(limit-1, 0, 1, limit) != StatusExceeded {
		t.Fatal("reserved tip should exceed")
	}
}

func TestFreeBytes(t *testing.T) {
	limit := uint64(1000)
	if FreeBytes(400, 100, 200, limit) != 300 {
		t.Fatalf("free=%d", FreeBytes(400, 100, 200, limit))
	}
}
