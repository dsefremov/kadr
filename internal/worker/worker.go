package worker

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/dsefremov/kadr/internal/mail"
)

// Worker runs the job queue in a single goroutine (KADR_WORKER_CONCURRENCY=1 on small VPS).
type Worker struct {
	log       *slog.Logger
	db        *sql.DB
	interval  time.Duration
	processor *Processor
}

func New(log *slog.Logger, database *sql.DB, interval time.Duration, smtp *mail.SMTP, publicURL string) *Worker {
	return &Worker{
		log:       log,
		db:        database,
		interval:  interval,
		processor: NewProcessor(log, database, smtp, publicURL),
	}
}

func (w *Worker) Run(ctx context.Context) error {
	w.log.Info("worker started", "poll", w.interval.String(), "concurrency", 1)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker stopped")
			return ctx.Err()
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) Health(ctx context.Context) error {
	return w.db.PingContext(ctx)
}
