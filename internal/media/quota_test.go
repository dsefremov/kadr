package media

import (
	"testing"

	"github.com/dsefremov/kadr/internal/units"
)

func TestQuotaWouldExceed(t *testing.T) {
	quota := uint64(units.DefaultQuotaBytes)
	used := uint64(900_000_000)
	trash := uint64(50_000_000)
	stored := uint64(200_000_000)
	if used+trash+stored <= quota {
		t.Fatal("expected exceed")
	}
	storedSmall := uint64(30_000_000)
	if used+trash+storedSmall > quota {
		t.Fatal("should fit under quota")
	}
}
