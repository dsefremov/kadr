package quota

// Package quota implements Den's decimal quota accounting helpers (spec §7).

const WarningPercent = 90

// Status describes UI quota state for the owner dashboard.
type Status string

const (
	StatusOK       Status = "ok"
	StatusWarning  Status = "warning"
	StatusExceeded Status = "exceeded"
)

func StatusFor(used, trash, reserved, limit uint64) Status {
	if limit == 0 {
		return StatusOK
	}
	accounted := used + trash + reserved
	if accounted >= limit {
		return StatusExceeded
	}
	if accounted*100/limit >= WarningPercent {
		return StatusWarning
	}
	return StatusOK
}

func FreeBytes(used, trash, reserved, limit uint64) uint64 {
	accounted := used + trash + reserved
	if accounted >= limit {
		return 0
	}
	return limit - accounted
}

func Accounted(used, trash, reserved uint64) uint64 {
	return used + trash + reserved
}
