package media

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrBadDate = errors.New("bad taken_date")

func SetTakenDate(ctx context.Context, db *sql.DB, ownerID int64, publicID, takenDate string) error {
	takenDate = strings.TrimSpace(takenDate)
	d, err := time.Parse("2006-01-02", takenDate)
	if err != nil {
		return ErrBadDate
	}
	sortAt := time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.UTC)
	res, err := db.ExecContext(ctx, `
		UPDATE media SET taken_date = ?, taken_sort = ?
		WHERE public_id = ? AND owner_id = ? AND state = 'ready' AND trashed_at IS NULL`,
		takenDate, sortAt, publicID, ownerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func DetailByPublicID(ctx context.Context, db *sql.DB, publicID string) (*Row, string, error) {
	var takenDate string
	r, err := ByPublicID(ctx, db, publicID)
	if err != nil {
		return nil, "", err
	}
	err = db.QueryRowContext(ctx, `SELECT taken_date FROM media WHERE public_id = ?`, publicID).Scan(&takenDate)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	return r, takenDate, nil
}
