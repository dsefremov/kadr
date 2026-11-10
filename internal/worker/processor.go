package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dsefremov/kadr/internal/auth"
	"github.com/dsefremov/kadr/internal/media"
	"github.com/dsefremov/kadr/internal/jobs"
	"github.com/dsefremov/kadr/internal/mail"
	"github.com/dsefremov/kadr/internal/upload"
)

type Processor struct {
	log      *slog.Logger
	db       *sql.DB
	smtp     *mail.SMTP
	pubPath  string
	uploads  *upload.Service
}

func NewProcessor(log *slog.Logger, db *sql.DB, smtp *mail.SMTP, publicURL, mediaDir string) *Processor {
	return &Processor{
		log: log, db: db, smtp: smtp, pubPath: publicURL,
		uploads: &upload.Service{DB: db, RootDir: mediaDir, PublicURL: publicURL},
	}
}

func (p *Processor) Process(ctx context.Context, j *jobs.Job) error {
	switch j.Kind {
	case jobs.KindSendEmail:
		return p.sendEmail(ctx, j)
	case jobs.KindExpireUpload:
		return p.expireUpload(ctx, j)
	case jobs.KindCleanupUnverified:
		return p.cleanupUnverified(ctx)
	case jobs.KindDerive:
		if !spoolWritable() {
			return fmt.Errorf("spool not writable at %s", media.SpoolRoot())
		}
		return p.derive(ctx, j.Payload)
	default:
		p.log.Warn("unknown job kind", "kind", j.Kind)
		return nil
	}
}

func (p *Processor) expireUpload(ctx context.Context, j *jobs.Job) error {
	var payload struct {
		UploadID int64 `json:"upload_id"`
	}
	if err := json.Unmarshal(j.Payload, &payload); err != nil {
		return err
	}
	return p.uploads.ExpireByID(ctx, payload.UploadID)
}

func (p *Processor) cleanupUnverified(ctx context.Context) error {
	n, err := auth.CleanupUnverified(ctx, p.db)
	if err != nil {
		return err
	}
	if n > 0 {
		p.log.Info("cleanup_unverified", "deleted", n)
	}
	return nil
}

func (p *Processor) sendEmail(ctx context.Context, j *jobs.Job) error {
	var payload struct {
		OutboxID int64 `json:"outbox_id"`
	}
	if err := json.Unmarshal(j.Payload, &payload); err != nil {
		return err
	}
	var row mail.OutboxRow
	err := p.db.QueryRowContext(ctx, `
		SELECT id, event, payload FROM outbox WHERE id = ?`, payload.OutboxID).
		Scan(&row.ID, &row.Event, &row.Payload)
	if err != nil {
		return err
	}
	to, subject, body, err := mail.RenderOutbox(row, p.pubPath)
	if err != nil {
		return err
	}
	if err := p.smtp.Send(ctx, to, subject, body); err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx, `UPDATE outbox SET dispatched_at = NOW() WHERE id = ?`, row.ID)
	return err
}

func (w *Worker) tick(ctx context.Context) {
	job, err := jobs.Claim(ctx, w.db)
	if err != nil {
		w.log.Debug("claim", "err", err)
		return
	}
	if job == nil {
		return
	}
	proc := w.processor
	if proc == nil {
		return
	}
	err = proc.Process(ctx, job)
	if err != nil {
		w.log.Warn("job failed", "id", job.ID, "kind", job.Kind, "err", err)
		_ = jobs.Fail(ctx, w.db, job.ID, fmt.Errorf("%w", err), job.Attempts)
		return
	}
	_ = jobs.Complete(ctx, w.db, job.ID)
}
