package media

import (
	"context"
	"database/sql"
)

// FindExistingByDedup returns public_id and state for owner+content hash, or empty if none.
// Matches non-trashed media in processing, ready, or waiting_quota.
func FindExistingByDedup(ctx context.Context, db *sql.DB, ownerID int64, dedup [32]byte, sourceBytes uint64) (publicID, state string, err error) {
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
