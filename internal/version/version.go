package version

// Set at link time via -ldflags; defaults for local dev.
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func Info() map[string]string {
	return map[string]string{
		"version":   Version,
		"commit":    Commit,
		"buildTime": BuildTime,
		"goVersion": "go1.22",
	}
}
