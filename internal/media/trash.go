package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dsefremov/kadr/internal/jobs"
)

const TrashRetention = 30 * 24 * time.Hour

var (
	ErrTrashState     = errors.New("media not trashable")
	ErrNotTrashed     = errors.New("media not in trash")
	ErrAlreadyTrashed = errors.New("media already in trash")
	ErrTrashOwner     = errors.New("not owner")
)

type TrashItem struct {
	PublicID    string
	Kind        string
	StoredBytes uint64
	TrashedAt   time.Time
	PurgeAfter  time.Time
	Width       int
	Height      int
}

func TrashByPublicID(ctx context.Context, db *sql.DB, ownerID int64, publicID string) error {
	row, err := ByPublicID(ctx, db, publicID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if row.OwnerID != ownerID {
		return ErrTrashOwner
	}
	if row.State != "ready" && row.State != "waiting_quota" {
		return ErrTrashState
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var trashed sql.NullTime
	var stored uint64
	err = tx.QueryRowContext(ctx, `
		SELECT trashed_at, stored_bytes FROM media WHERE id = ? FOR UPDATE`, row.ID).
		Scan(&trashed, &stored)
	if err != nil {
		return err
	}
	if trashed.Valid {
		return ErrAlreadyTrashed
	}
	purgeAfter := time.Now().UTC().Add(TrashRetention)
	_, err = tx.ExecContext(ctx, `
		UPDATE media SET trashed_at = NOW(), purge_after = ? WHERE id = ?`,
		purgeAfter, row.ID)
	if err != nil {
		return err
	}
	if stored > 0 {
		_, err = tx.ExecContext(ctx, `
			UPDATE users SET used_bytes = used_bytes - ?, trash_bytes = trash_bytes + ?
			WHERE id = ? AND used_bytes >= ?`,
			stored, stored, ownerID, stored)
		if err != nil {
			return err
		}
	}
	idem := fmt.Sprintf("purge:%d", row.ID)
	if err := jobs.EnqueueTxAfter(ctx, tx, jobs.KindPurgeTrash, map[string]int64{"media_id": row.ID}, idem, purgeAfter); err != nil {
		return err
	}
	return tx.Commit()
}

func RestoreByPublicID(ctx context.Context, db *sql.DB, ownerID int64, publicID string) error {
	row, err := ByPublicID(ctx, db, publicID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if row.OwnerID != ownerID {
		return ErrTrashOwner
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var trashed sql.NullTime
	var stored uint64
	err = tx.QueryRowContext(ctx, `
		SELECT trashed_at, stored_bytes FROM media WHERE id = ? FOR UPDATE`, row.ID).
		Scan(&trashed, &stored)
	if err != nil {
		return err
	}
	if !trashed.Valid {
		return ErrNotTrashed
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE media SET trashed_at = NULL, purge_after = NULL WHERE id = ?`, row.ID)
	if err != nil {
		return err
	}
	if stored > 0 {
		var quota, used, trash uint64
		err = tx.QueryRowContext(ctx, `
			SELECT quota_bytes, used_bytes, trash_bytes FROM users WHERE id = ? FOR UPDATE`, ownerID).
			Scan(&quota, &used, &trash)
		if err != nil {
			return err
		}
		if used+trash+stored > quota {
			return ErrWaitingQuota
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE users SET trash_bytes = trash_bytes - ?, used_bytes = used_bytes + ?
			WHERE id = ? AND trash_bytes >= ?`,
			stored, stored, ownerID, stored)
		if err != nil {
			return err
		}
	}
	_, _ = tx.ExecContext(ctx, `
		UPDATE jobs SET state = 'done', lease_until = NULL
		WHERE idem_key = ? AND kind = ? AND state IN ('pending','running')`,
		fmt.Sprintf("purge:%d", row.ID), jobs.KindPurgeTrash)
	return tx.Commit()
}

func ListTrash(ctx context.Context, db *sql.DB, ownerID int64, limit int) ([]TrashItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := db.QueryContext(ctx, `
		SELECT public_id, kind, stored_bytes, trashed_at, purge_after, width, height
		FROM media
		WHERE owner_id = ? AND trashed_at IS NOT NULL
		ORDER BY trashed_at DESC LIMIT ?`, ownerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrashItem
	for rows.Next() {
		var it TrashItem
		var w, h sql.NullInt64
		if err := rows.Scan(&it.PublicID, &it.Kind, &it.StoredBytes, &it.TrashedAt, &it.PurgeAfter, &w, &h); err != nil {
			return nil, err
		}
		if w.Valid {
			it.Width = int(w.Int64)
		}
		if h.Valid {
			it.Height = int(h.Int64)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// PurgePermanent deletes trashed media row, variants on disk, and adjusts trash_bytes.
func PurgePermanent(ctx context.Context, db *sql.DB, mediaRoot string, mediaID int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var ownerID int64
	var pubID string
	var stored uint64
	var trashed sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT owner_id, public_id, stored_bytes, trashed_at FROM media WHERE id = ? FOR UPDATE`, mediaID).
		Scan(&ownerID, &pubID, &stored, &trashed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !trashed.Valid {
		return fmt.Errorf("purge: media %d not trashed", mediaID)
	}
	var keys []string
	rows, err := tx.QueryContext(ctx, `SELECT storage_key FROM media_variants WHERE media_id = ?`, mediaID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, k)
	}
	rows.Close()
	_, err = tx.ExecContext(ctx, `DELETE FROM media WHERE id = ?`, mediaID)
	if err != nil {
		return err
	}
	if stored > 0 {
		_, err = tx.ExecContext(ctx, `
			UPDATE users SET trash_bytes = trash_bytes - ? WHERE id = ? AND trash_bytes >= ?`,
			stored, ownerID, stored)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	dir := filepath.Join(mediaRoot, "stored", fmt.Sprintf("%d", ownerID), pubID)
	_ = os.RemoveAll(dir)
	for _, k := range keys {
		_ = os.Remove(filepath.Join(mediaRoot, filepath.FromSlash(k)))
	}
	return nil
}

// PurgeAny deletes media regardless of trash state (account wipe).
func PurgeAny(ctx context.Context, db *sql.DB, mediaRoot string, mediaID int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var ownerID int64
	var pubID string
	var stored uint64
	var trashed sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT owner_id, public_id, stored_bytes, trashed_at FROM media WHERE id = ? FOR UPDATE`, mediaID).
		Scan(&ownerID, &pubID, &stored, &trashed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM media WHERE id = ?`, mediaID)
	if err != nil {
		return err
	}
	if stored > 0 {
		if trashed.Valid {
			_, err = tx.ExecContext(ctx, `
				UPDATE users SET trash_bytes = trash_bytes - ? WHERE id = ? AND trash_bytes >= ?`,
				stored, ownerID, stored)
		} else {
			_, err = tx.ExecContext(ctx, `
				UPDATE users SET used_bytes = used_bytes - ? WHERE id = ? AND used_bytes >= ?`,
				stored, ownerID, stored)
		}
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(mediaRoot, "stored", fmt.Sprintf("%d", ownerID), pubID))
	return nil
}

func DeleteTrashedByPublicID(ctx context.Context, db *sql.DB, mediaRoot string, ownerID int64, publicID string) error {
	row, err := ByPublicID(ctx, db, publicID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if row.OwnerID != ownerID {
		return ErrTrashOwner
	}
	var trashed sql.NullTime
	err = db.QueryRowContext(ctx, `SELECT trashed_at FROM media WHERE id = ?`, row.ID).Scan(&trashed)
	if err != nil {
		return err
	}
	if !trashed.Valid {
		return ErrNotTrashed
	}
	return PurgePermanent(ctx, db, mediaRoot, row.ID)
}

func EmptyTrash(ctx context.Context, db *sql.DB, mediaRoot string, ownerID int64) (int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id FROM media WHERE owner_id = ? AND trashed_at IS NOT NULL`, ownerID)
	if err != nil {
		return 0, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	n := 0
	for _, id := range ids {
		if err := PurgePermanent(ctx, db, mediaRoot, id); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
