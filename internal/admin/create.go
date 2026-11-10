package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/dsefremov/kadr/internal/password"
	"github.com/dsefremov/kadr/internal/token"
	"github.com/dsefremov/kadr/internal/units"
)

func Create(ctx context.Context, db *sql.DB, email, pw string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email")
	}
	if err := password.Validate(pw); err != nil {
		return err
	}
	hash, err := password.Hash(pw)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var userID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email).Scan(&userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO users (email, password_hash, role, email_verified_at, quota_bytes)
			VALUES (?, ?, 'admin', NOW(), ?)`, email, hash, units.DefaultQuotaBytes)
		if err != nil {
			return err
		}
		userID, err = res.LastInsertId()
		if err != nil {
			return err
		}
		handle, err := uniqueHandle(ctx, tx, email)
		if err != nil {
			return err
		}
		pubID, err := token.RandomURLSafe(16)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO feeds (public_id, owner_id, handle) VALUES (?, ?, ?)`, pubID, userID, handle)
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE users SET role = 'admin', password_hash = ?, email_verified_at = COALESCE(email_verified_at, NOW())
		WHERE id = ?`, hash, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func uniqueHandle(ctx context.Context, tx *sql.Tx, email string) (string, error) {
	base := strings.Split(email, "@")[0]
	base = authSanitize(base)
	for i := 0; i < 8; i++ {
		h := base
		if i > 0 {
			suf, _ := token.RandomURLSafe(4)
			h = fmt.Sprintf("%s-%s", base, suf)
		}
		var n int
		err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM feeds WHERE handle = ?`, h).Scan(&n)
		if err != nil {
			return "", err
		}
		if n == 0 {
			return h, nil
		}
	}
	return "", fmt.Errorf("handle exhausted")
}

func authSanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() < 3 {
		return "user"
	}
	if b.Len() > 32 {
		return b.String()[:32]
	}
	return b.String()
}
