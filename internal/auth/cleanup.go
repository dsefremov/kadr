package auth

import (
	"context"
	"database/sql"
	"time"
)

const unverifiedMaxAge = 7 * 24 * time.Hour

// CleanupUnverified deletes accounts with no verified email older than 7 days (spec §7).
func CleanupUnverified(ctx context.Context, db *sql.DB) (int64, error) {
	res, err := db.ExecContext(ctx, `
		DELETE FROM users
		WHERE email_verified_at IS NULL
		  AND created_at < DATE_SUB(NOW(), INTERVAL 7 DAY)`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
