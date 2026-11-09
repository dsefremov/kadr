// Package units uses Den's decimal byte convention (2026-11-09): 1 GB = 10^9 bytes.
package units

const (
	MB = 1_000_000
	GB = 1_000_000_000

	DefaultQuotaBytes = 5 * GB
	PhotoMaxBytes     = 50 * MB
	VideoMaxBytes     = 2 * GB
)
