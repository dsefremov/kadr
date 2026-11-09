package worker

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// Worker runs the job queue in a single goroutine (KADR_WORKER_CONCURRENCY=1 on small VPS).
type Worker struct {
	log      *slog.Logger
	db       *sql.DB
	interval time.Duration
}

func New(log *slog.Logger, database *sql.DB, interval time.Duration) *Worker {
	return &Worker{log: log, db: database, interval: interval}
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

func (w *Worker) tick(ctx context.Context) {
	// Stage 0: prove SKIP LOCKED wiring without product jobs.
	var id int64
	err := w.db.QueryRowContext(ctx, `
		SELECT id FROM jobs
		WHERE state = 'pending' AND run_after <= NOW()
		ORDER BY id
		LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return
	}
	if err != nil {
		w.log.Debug("job poll", "err", err)
		return
	}
	w.log.Debug("would process job", "id", id)
}

func (w *Worker) Health(ctx context.Context) error {
	return w.db.PingContext(ctx)
}
