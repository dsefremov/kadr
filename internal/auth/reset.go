package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/dsefremov/kadr/internal/jobs"
	"github.com/dsefremov/kadr/internal/password"
	"github.com/dsefremov/kadr/internal/token"
)

const resetTTL = 24 * time.Hour

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil // uniform response
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var userID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	_, _ = tx.ExecContext(ctx, `
		UPDATE email_tokens SET used_at = NOW()
		WHERE user_id = ? AND purpose = 'reset' AND used_at IS NULL`, userID)
	plain, err := token.RandomURLSafe(32)
	if err != nil {
		return err
	}
	th := token.Hash(plain)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO email_tokens (token_hash, user_id, purpose, expires_at)
		VALUES (?, ?, 'reset', ?)`, th[:], userID, time.Now().Add(resetTTL))
	if err != nil {
		return err
	}
	link := fmt.Sprintf("/reset?token=%s", plain)
	body := "Сброс пароля: {{link}}"
	oid, err := jobs.InsertOutboxTx(ctx, tx, "email.reset", map[string]string{
		"to": email, "subject": "Кадр — сброс пароля", "body": body, "link": link,
	})
	if err != nil {
		return err
	}
	if err := jobs.EnqueueMailForOutbox(ctx, tx, oid); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ResetPassword(ctx context.Context, plain, newPassword string) error {
	if err := password.Validate(newPassword); err != nil {
		return errInvalidToken()
	}
	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}
	th := token.Hash(plain)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var tokenID, userID int64
	var expires time.Time
	var used sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, used_at FROM email_tokens
		WHERE token_hash = ? AND purpose = 'reset'`, th[:]).Scan(&tokenID, &userID, &expires, &used)
	if err != nil {
		return errInvalidToken()
	}
	if used.Valid || time.Now().After(expires) {
		return errInvalidToken()
	}
	_, err = tx.ExecContext(ctx, `UPDATE email_tokens SET used_at = NOW() WHERE id = ?`, tokenID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, hash, userID)
	if err != nil {
		return err
	}
	_, _ = tx.ExecContext(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = ? AND revoked_at IS NULL`, userID)
	return tx.Commit()
}
