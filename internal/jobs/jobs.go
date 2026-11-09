package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	KindSendEmail         = "send_email"
	KindExpireUpload      = "expire_upload"
	KindCleanupUnverified = "cleanup_unverified"
	KindDerive            = "derive"
	LeaseDuration         = 2 * time.Minute
	MaxAttempts           = 5
)

type Job struct {
	ID       int64
	Kind     string
	Payload  json.RawMessage
	Attempts int
}

func EnqueueTx(ctx context.Context, tx *sql.Tx, kind string, payload any, idemKey string) error {
	return EnqueueTxAfter(ctx, tx, kind, payload, idemKey, time.Now())
}

func EnqueueAfter(ctx context.Context, db *sql.DB, kind string, payload any, idemKey string, runAfter time.Time) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO jobs (kind, payload, idem_key, state, run_after)
		VALUES (?, ?, ?, 'pending', ?)
		ON DUPLICATE KEY UPDATE id = id`, kind, body, idemKey, runAfter)
	return err
}

func EnqueueTxAfter(ctx context.Context, tx *sql.Tx, kind string, payload any, idemKey string, runAfter time.Time) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO jobs (kind, payload, idem_key, state, run_after)
		VALUES (?, ?, ?, 'pending', ?)
		ON DUPLICATE KEY UPDATE id = id`, kind, body, idemKey, runAfter)
	return err
}

func Claim(ctx context.Context, db *sql.DB) (*Job, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var j Job
	var payload []byte
	err = tx.QueryRowContext(ctx, `
		SELECT id, kind, payload, attempts FROM jobs
		WHERE state = 'pending' AND run_after <= NOW()
		ORDER BY id
		LIMIT 1
		FOR UPDATE SKIP LOCKED`).Scan(&j.ID, &j.Kind, &payload, &j.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	j.Payload = payload
	leaseUntil := time.Now().Add(LeaseDuration)
	_, err = tx.ExecContext(ctx, `
		UPDATE jobs SET state = 'running', lease_until = ?, attempts = attempts + 1
		WHERE id = ?`, leaseUntil, j.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &j, nil
}

func Complete(ctx context.Context, db *sql.DB, jobID int64) error {
	_, err := db.ExecContext(ctx, `UPDATE jobs SET state = 'done', lease_until = NULL, last_error = NULL WHERE id = ?`, jobID)
	return err
}

func Fail(ctx context.Context, db *sql.DB, jobID int64, jobErr error, attempts int) error {
	msg := jobErr.Error()
	if attempts >= MaxAttempts {
		_, err := db.ExecContext(ctx, `
			UPDATE jobs SET state = 'dead', lease_until = NULL, last_error = ? WHERE id = ?`, msg, jobID)
		return err
	}
	delay := retryDelay(attempts)
	_, err := db.ExecContext(ctx, `
		UPDATE jobs SET state = 'pending', lease_until = NULL, last_error = ?,
			run_after = DATE_ADD(NOW(), INTERVAL ? SECOND)
		WHERE id = ?`, msg, int(delay.Seconds()), jobID)
	return err
}

func retryDelay(attempts int) time.Duration {
	switch attempts {
	case 1:
		return time.Minute
	case 2:
		return 5 * time.Minute
	case 3:
		return 30 * time.Minute
	default:
		return 2 * time.Hour
	}
}

func InsertOutboxTx(ctx context.Context, tx *sql.Tx, event string, payload any) (int64, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO outbox (event, payload) VALUES (?, ?)`, event, body)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func EnqueueMailForOutbox(ctx context.Context, tx *sql.Tx, outboxID int64) error {
	return EnqueueTx(ctx, tx, KindSendEmail, map[string]int64{"outbox_id": outboxID}, fmt.Sprintf("mail:%d", outboxID))
}
