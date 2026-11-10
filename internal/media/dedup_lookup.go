package media

import (
	"context"
	"database/sql"
)

// DedupMatch describes an existing media row with same content hash for an owner.
type DedupMatch struct {
	PublicID string
	State    string
	InTrash  bool
}

// LookupDedup checks active timeline media first, then trashed (spec §12).
func LookupDedup(ctx context.Context, db *sql.DB, ownerID int64, dedup [32]byte, sourceBytes uint64) (*DedupMatch, error) {
	var pub, state string
	err := db.QueryRowContext(ctx, `
		SELECT public_id, state FROM media
		WHERE owner_id = ? AND dedup_sha256 = ? AND source_bytes = ?
		  AND trashed_at IS NULL
		  AND state IN ('processing', 'ready', 'waiting_quota')
		ORDER BY id DESC LIMIT 1`,
		ownerID, dedup[:], sourceBytes).Scan(&pub, &state)
	if err == nil {
		return &DedupMatch{PublicID: pub, State: state, InTrash: false}, nil
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
	return &DedupMatch{PublicID: pub, State: state, InTrash: true}, nil
}
