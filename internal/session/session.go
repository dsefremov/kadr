package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/dsefremov/kadr/internal/token"
)

const (
	CookieName      = "kadr_session"
	AbsoluteTTL     = 30 * 24 * time.Hour
	IdleTTL         = 14 * 24 * time.Hour
)

type Store struct {
	DB *sql.DB
}

type Data struct {
	UserID      int64
	Verified    bool
	Email       string
	IdleExpires time.Time
}

func (s *Store) Create(ctx context.Context, userID int64) (string, error) {
	plain, err := token.RandomURLSafe(32)
	if err != nil {
		return "", err
	}
	th := token.Hash(plain)
	now := time.Now()
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO sessions (id_hash, user_id, expires_at, idle_expires_at)
		VALUES (?, ?, ?, ?)`, th[:], userID, now.Add(AbsoluteTTL), now.Add(IdleTTL))
	if err != nil {
		return "", err
	}
	return plain, nil
}

func (s *Store) Revoke(ctx context.Context, plain string) error {
	th := token.Hash(plain)
	_, err := s.DB.ExecContext(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE id_hash = ?`, th[:])
	return err
}

func (s *Store) Load(ctx context.Context, plain string) (*Data, error) {
	th := token.Hash(plain)
	var userID int64
	var exp, idle time.Time
	var revoked sql.NullTime
	err := s.DB.QueryRowContext(ctx, `
		SELECT user_id, expires_at, idle_expires_at, revoked_at FROM sessions
		WHERE id_hash = ?`, th[:]).Scan(&userID, &exp, &idle, &revoked)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if revoked.Valid || now.After(exp) || now.After(idle) {
		return nil, sql.ErrNoRows
	}
	var email string
	var verified sql.NullTime
	err = s.DB.QueryRowContext(ctx, `SELECT email, email_verified_at FROM users WHERE id = ?`, userID).
		Scan(&email, &verified)
	if err != nil {
		return nil, err
	}
	_, _ = s.DB.ExecContext(ctx, `UPDATE sessions SET idle_expires_at = ? WHERE id_hash = ?`,
		now.Add(IdleTTL), th[:])
	return &Data{
		UserID: userID, Email: email, Verified: verified.Valid,
		IdleExpires: now.Add(IdleTTL),
	}, nil
}

// CSRF token helper for future forms (optional).
func RandomCSRF() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
