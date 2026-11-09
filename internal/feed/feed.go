package feed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("feed not found")

type Meta struct {
	PublicID   string
	Handle     string
	Visibility string
	OwnerID    int64
}

func ByHandle(ctx context.Context, db *sql.DB, handle string) (*Meta, error) {
	var m Meta
	err := db.QueryRowContext(ctx, `
		SELECT public_id, handle, visibility, owner_id FROM feeds WHERE handle = ?`, handle).
		Scan(&m.PublicID, &m.Handle, &m.Visibility, &m.OwnerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func ByPublicID(ctx context.Context, db *sql.DB, publicID string) (*Meta, error) {
	var m Meta
	err := db.QueryRowContext(ctx, `
		SELECT public_id, handle, visibility, owner_id FROM feeds WHERE public_id = ?`, publicID).
		Scan(&m.PublicID, &m.Handle, &m.Visibility, &m.OwnerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func ForOwner(ctx context.Context, db *sql.DB, ownerID int64) (*Meta, error) {
	var m Meta
	err := db.QueryRowContext(ctx, `
		SELECT public_id, handle, visibility, owner_id FROM feeds WHERE owner_id = ?`, ownerID).
		Scan(&m.PublicID, &m.Handle, &m.Visibility, &m.OwnerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func SetVisibility(ctx context.Context, db *sql.DB, ownerID int64, vis string) error {
	if vis != "public" && vis != "private" {
		return fmt.Errorf("invalid visibility")
	}
	res, err := db.ExecContext(ctx, `
		UPDATE feeds SET visibility = ? WHERE owner_id = ?`, vis, ownerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type TimelinePage struct {
	Items  []any `json:"items"`
	Cursor *string `json:"cursor"`
}

func Timeline(ctx context.Context, db *sql.DB, _ string, _ string, limit int) (*TimelinePage, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	_ = limit
	// Media rows land in stage 4; timeline is empty until then.
	return &TimelinePage{Items: []any{}}, nil
}
