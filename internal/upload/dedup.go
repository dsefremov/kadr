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

type dedupHit struct {
	publicID string
	state    string
	inTrash  bool
}

func findExistingMedia(ctx context.Context, db *sql.DB, ownerID int64, dedup [32]byte, sourceBytes uint64) (*dedupHit, error) {
	var pub, state string
	err := db.QueryRowContext(ctx, `
		SELECT public_id, state FROM media
		WHERE owner_id = ? AND dedup_sha256 = ? AND source_bytes = ?
		  AND trashed_at IS NULL
		  AND state IN ('processing', 'ready', 'waiting_quota')
		ORDER BY id DESC LIMIT 1`,
		ownerID, dedup[:], sourceBytes).Scan(&pub, &state)
	if err == nil {
		return &dedupHit{publicID: pub, state: state}, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	err = db.QueryRowContext(ctx, `
		SELECT public_id, state FROM media
		WHERE owner_id = ? AND dedup_sha256 = ? AND source_bytes = ?
		  AND trashed_at IS NOT NULL
		ORDER BY id DESC LIMIT 1`,
		ownerID, dedup[:], sourceBytes).Scan(&pub, &state)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &dedupHit{publicID: pub, state: state, inTrash: true}, nil
}
