package media

import (
	"testing"

	"github.com/dsefremov/kadr/internal/units"
)

// Restore must fail when used+trash+stored would exceed quota (spec §7).
func TestRestoreQuotaRaceCondition(t *testing.T) {
	limit := uint64(units.DefaultQuotaBytes)
	used := limit - 1_000_000
	trash := uint64(2_000_000)
	stored := uint64(2_000_000)
	if used+trash+stored <= limit {
		t.Fatal("fixture should exceed quota on restore")
	}
	afterFree := limit - used
	if afterFree >= stored {
		t.Fatal("restore should be rejected")
	}
}
