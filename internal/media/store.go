package media

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"os"
	"path/filepath"
	"time"

	"github.com/dsefremov/kadr/internal/token"
	"github.com/dsefremov/kadr/internal/upload"
)

type Row struct {
	ID          int64
	PublicID    string
	FeedID      int64
	OwnerID     int64
	Kind        string
	State       string
	Width       sql.NullInt64
	Height      sql.NullInt64
	DurationMs  sql.NullInt64
	TakenSort   time.Time
	ErrorCode   sql.NullString
	ErrorMessage sql.NullString
}

type VariantRow struct {
	Variant     string
	ProfileHash string
	StorageKey  string
	Bytes       uint64
	Mime        string
	Width       int
	Height      int
}

func HashFile(path string) ([32]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return [32]byte{}, err
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return sum, nil
}

func CreateProcessing(ctx context.Context, db *sql.DB, ownerID, uploadID int64, kind string, sourceBytes uint64, dedup [32]byte, incomingKey string,
	takenSort time.Time, takenDate, srcKind string) (int64, string, error) {
	var feedID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM feeds WHERE owner_id = ?`, ownerID).Scan(&feedID); err != nil {
		return 0, "", err
	}
	pubID, err := token.RandomURLSafe(16)
	if err != nil {
		return 0, "", err
	}
	if takenSort.IsZero() {
		takenSort = time.Now().UTC()
	}
	if takenDate == "" {
		takenDate = takenSort.Format("2006-01-02")
	}
	if srcKind == "" {
		srcKind = "upload"
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO media (public_id, feed_id, owner_id, upload_id, kind, dedup_sha256, source_bytes,
			state, taken_sort, taken_date, src_kind)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'processing', ?, ?, ?)`,
		pubID, feedID, ownerID, uploadID, kind, dedup[:], sourceBytes, takenSort, takenDate, srcKind)
	if err != nil {
		if isDupKey(err) {
			return 0, "", ErrDuplicate
		}
		return 0, "", err
	}
	mediaID, _ := res.LastInsertId()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO source_files (upload_id, media_id, incoming_key, state, delete_after)
		VALUES (?, ?, ?, 'held', DATE_ADD(NOW(), INTERVAL 7 DAY))`,
		uploadID, mediaID, incomingKey)
	if err != nil {
		return 0, "", err
	}
	if err := tx.Commit(); err != nil {
		return 0, "", err
	}
	return mediaID, pubID, nil
}

func isDupKey(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return errors.Is(err, ErrDuplicate) || strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "1062")
}

func ReleaseReserved(ctx context.Context, db *sql.DB, ownerID int64, uploadLen uint64) error {
	_, err := db.ExecContext(ctx, `
		UPDATE users SET reserved_bytes = reserved_bytes - ? WHERE id = ? AND reserved_bytes >= ?`,
		uploadLen, ownerID, uploadLen)
	return err
}

func MarkFailed(ctx context.Context, db *sql.DB, mediaID, ownerID int64, uploadLen uint64, code, msg string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if mediaID > 0 {
		_, err = tx.ExecContext(ctx, `
			UPDATE media SET state = 'failed', error_code = ?, error_message = ? WHERE id = ?`,
			code, msg, mediaID)
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE users SET reserved_bytes = reserved_bytes - ? WHERE id = ? AND reserved_bytes >= ?`,
		uploadLen, ownerID, uploadLen)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func nullDuration(ms uint64) interface{} {
	if ms == 0 {
		return nil
	}
	return ms
}

// ApplyTaken updates timeline date after better metadata is known (e.g. video creation_time).
func ApplyTaken(ctx context.Context, db *sql.DB, mediaID int64, takenSort time.Time, takenDate, srcKind string) error {
	if takenSort.IsZero() || takenDate == "" {
		return nil
	}
	if srcKind == "" {
		srcKind = "upload"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE media SET taken_sort = ?, taken_date = ?, src_kind = ? WHERE id = ?`,
		takenSort, takenDate, srcKind, mediaID)
	return err
}

func CommitReady(ctx context.Context, db *sql.DB, mediaID, ownerID, uploadLen, stored uint64,
	variants []VariantResult, mediaPub string, width, height int, durationMs uint64) error {
	var total uint64
	for _, v := range variants {
		total += v.Bytes
	}
	if stored == 0 {
		stored = total
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var quota, used, trash uint64
	err = tx.QueryRowContext(ctx, `
		SELECT quota_bytes, used_bytes, trash_bytes FROM users WHERE id = ? FOR UPDATE`,
		ownerID).Scan(&quota, &used, &trash)
	if err != nil {
		return err
	}
	if used+trash+stored > quota {
		_, err = tx.ExecContext(ctx, `UPDATE media SET state = 'waiting_quota', stored_bytes = ? WHERE id = ?`, stored, mediaID)
		if err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrWaitingQuota
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE media SET state = 'ready', stored_bytes = ?, width = ?, height = ?, duration_ms = ?,
			error_code = NULL, error_message = NULL WHERE id = ?`,
		stored, width, height, nullDuration(durationMs), mediaID)
	if err != nil {
		return err
	}
	for _, v := range variants {
		hexHash := v.SHA256
		if len(hexHash) != 64 {
			hexHash = "0000000000000000000000000000000000000000000000000000000000000000"
		}
		key := filepath.ToSlash(filepath.Join("stored", fmt.Sprintf("%d", ownerID), mediaPub, v.File))
		_, err = tx.ExecContext(ctx, `
			INSERT INTO media_variants (media_id, variant, profile_hash, storage_key, bytes, sha256, width, height, mime)
			VALUES (?, ?, ?, ?, ?, UNHEX(?), ?, ?, ?)`,
			mediaID, v.Variant, ProfileHashV1SDR, key, v.Bytes, hexHash, v.Width, v.Height, v.Mime)
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE users SET used_bytes = used_bytes + ?, reserved_bytes = reserved_bytes - ?
		WHERE id = ? AND reserved_bytes >= ?`,
		stored, uploadLen, ownerID, uploadLen)
	if err != nil {
		return err
	}
	_, _ = tx.ExecContext(ctx, `UPDATE source_files SET state = 'deleting' WHERE media_id = ?`, mediaID)
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func ByPublicID(ctx context.Context, db *sql.DB, publicID string) (*Row, error) {
	var r Row
	err := db.QueryRowContext(ctx, `
		SELECT m.id, m.public_id, m.feed_id, m.owner_id, m.kind, m.state,
			m.width, m.height, m.duration_ms, m.taken_sort, m.error_code, m.error_message
		FROM media m WHERE m.public_id = ?`, publicID).
		Scan(&r.ID, &r.PublicID, &r.FeedID, &r.OwnerID, &r.Kind, &r.State,
			&r.Width, &r.Height, &r.DurationMs, &r.TakenSort, &r.ErrorCode, &r.ErrorMessage)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func VariantPath(ctx context.Context, db *sql.DB, mediaPublicID, variant, profileHash string) (string, string, error) {
	var storageKey, mime string
	err := db.QueryRowContext(ctx, `
		SELECT mv.storage_key, mv.mime FROM media_variants mv
		JOIN media m ON m.id = mv.media_id
		WHERE m.public_id = ? AND mv.variant = ? AND mv.profile_hash = ?`,
		mediaPublicID, variant, profileHash).Scan(&storageKey, &mime)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return storageKey, mime, err
}

func ListByOwner(ctx context.Context, db *sql.DB, ownerID int64, state string) ([]Row, error) {
	q := `
		SELECT id, public_id, feed_id, owner_id, kind, state, width, height, duration_ms, taken_sort, error_code, error_message
		FROM media WHERE owner_id = ? AND trashed_at IS NULL`
	args := []any{ownerID}
	if state != "" {
		q += ` AND state = ?`
		args = append(args, state)
	}
	q += ` ORDER BY id DESC LIMIT 50`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.PublicID, &r.FeedID, &r.OwnerID, &r.Kind, &r.State,
			&r.Width, &r.Height, &r.DurationMs, &r.TakenSort, &r.ErrorCode, &r.ErrorMessage); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func DeleteIncoming(svc *upload.Service, row *upload.Row) {
	_ = os.Remove(filepath.Join(svc.RootDir, row.IncomingKey))
}

func StoreVariant(mediaRoot string, ownerID int64, mediaPub, fileName string, data []byte) (string, error) {
	dir := filepath.Join(mediaRoot, "stored", fmt.Sprintf("%d", ownerID), mediaPub)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, fileName)
	if err := os.WriteFile(dest, data, 0o640); err != nil {
		return "", err
	}
	return filepath.Join("stored", fmt.Sprintf("%d", ownerID), mediaPub, fileName), nil
}

func MoveRunnerOutputs(mediaRoot string, ownerID int64, mediaPub string, spoolOut string, variants []VariantResult) error {
	dir := filepath.Join(mediaRoot, "stored", fmt.Sprintf("%d", ownerID), mediaPub)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	for _, v := range variants {
		src := filepath.Join(spoolOut, v.File)
		dst := filepath.Join(dir, v.File)
		if err := os.Rename(src, dst); err != nil {
			if err2 := copyFileSpool(src, dst); err2 != nil {
				return err
			}
		}
	}
	return nil
}

func copyFileSpool(src, dst string) error {
	return copyFile(src, dst)
}
