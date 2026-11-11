package quota

import (
	"fmt"
	"math"

	"github.com/dsefremov/kadr/internal/units"
)

// FormatDecimalGB renders bytes as short decimal GB/MB (mockups: 1 GB = 1000 MB).
func FormatDecimalGB(b uint64) string {
	if b >= units.GB {
		g := float64(b) / float64(units.GB)
		if g >= 10 {
			return fmt.Sprintf("%.0f ГБ", math.Round(g*10)/10)
		}
		return fmt.Sprintf("%.1f ГБ", math.Round(g*10)/10)
	}
	mb := float64(b) / float64(units.MB)
	if mb >= 100 {
		return fmt.Sprintf("%.0f МБ", math.Round(mb))
	}
	return fmt.Sprintf("%.1f МБ", math.Round(mb*10)/10)
}
