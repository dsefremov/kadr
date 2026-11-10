package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Service     string
	LogLevel    string
	LogFormat   string // json or text
	HTTPAddr    string
	MySQLDSN    string
	MigrateOnly bool
	WorkerPoll  time.Duration
	WorkerJobs  int
	MediaDir    string
	SMTPHost      string
	SMTPPort      int
	MailFrom      string
	PublicURL     string
	CookieSecure  bool
}

func Load(service string) (Config, error) {
	cfg := Config{
		Service:    service,
		LogLevel:   envOr("KADR_LOG_LEVEL", "info"),
		LogFormat:  envOr("KADR_LOG_FORMAT", "json"),
		HTTPAddr:   envOr("KADR_HTTP_ADDR", ":8080"),
		MySQLDSN:   os.Getenv("KADR_MYSQL_DSN"),
		WorkerPoll: durationEnv("KADR_WORKER_POLL_INTERVAL", 5*time.Second),
		WorkerJobs: intEnv("KADR_WORKER_CONCURRENCY", 1),
		MediaDir:   envOr("KADR_MEDIA_DIR", "/data"),
		SMTPHost:     envOr("KADR_SMTP_HOST", ""),
		SMTPPort:     intEnv("KADR_SMTP_PORT", 1025),
		MailFrom:     envOr("KADR_MAIL_FROM", "noreply@kadr.local"),
		PublicURL:    envOr("KADR_PUBLIC_URL", "http://127.0.0.1:8088"),
		CookieSecure: os.Getenv("KADR_COOKIE_SECURE") == "1",
	}
	if cfg.WorkerJobs < 1 {
		cfg.WorkerJobs = 1
	}
	if cfg.MySQLDSN == "" && service != "media-runner" {
		return cfg, fmt.Errorf("KADR_MYSQL_DSN is required")
	}
	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func intEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func durationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
