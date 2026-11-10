package media

import (
	"context"
	"crypto/rand"
	"database/sql"
	"time"

	"github.com/dsefremov/kadr/internal/token"
)

// InsertFailed creates a failed media row for owner-visible errors (e.g. animated).
func InsertFailed(ctx context.Context, db *sql.DB, ownerID, uploadID int64, kind, code, msg string, sourceBytes uint64) error {
	var feedID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM feeds WHERE owner_id = ?`, ownerID).Scan(&feedID); err != nil {
		return err
	}
	pubID, err := token.RandomURLSafe(16)
	if err != nil {
		return err
	}
	var dedup [32]byte
	_, _ = rand.Read(dedup[:])
	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO media (public_id, feed_id, owner_id, upload_id, kind, dedup_sha256, source_bytes,
			state, taken_sort, taken_date, src_kind, error_code, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'failed', ?, ?, 'upload', ?, ?)`,
		pubID, feedID, ownerID, uploadID, kind, dedup[:], sourceBytes, now, now.Format("2006-01-02"), code, msg)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE users SET reserved_bytes = reserved_bytes - ? WHERE id = ? AND reserved_bytes >= ?`,
		sourceBytes, ownerID, sourceBytes)
	if err != nil {
		return err
	}
	return tx.Commit()
}
