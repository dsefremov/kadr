package upload

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"io"
	"os"
)

func hashFile(path string) ([32]byte, error) {
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

func findExistingMedia(ctx context.Context, db *sql.DB, ownerID int64, dedup [32]byte, sourceBytes uint64) (publicID, state string, err error) {
	err = db.QueryRowContext(ctx, `
		SELECT public_id, state FROM media
		WHERE owner_id = ? AND dedup_sha256 = ? AND source_bytes = ?
		  AND trashed_at IS NULL
		  AND state IN ('processing', 'ready', 'waiting_quota')
		ORDER BY id DESC LIMIT 1`,
		ownerID, dedup[:], sourceBytes).Scan(&publicID, &state)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return publicID, state, err
}
