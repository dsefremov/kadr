package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/dsefremov/kadr/internal/jobs"
	"github.com/dsefremov/kadr/internal/password"
)

var (
	ErrDeleteBadPassword = errors.New("invalid password")
	ErrDeleteInProgress  = errors.New("account deletion in progress")
)

// RequestDelete marks the account for async wipe (spec §15): revoke sessions, hide feed, cancel uploads.
func (s *Service) RequestDelete(ctx context.Context, userID int64, plainPassword string) error {
	var hash, status string
	err := s.DB.QueryRowContext(ctx, `SELECT password_hash, status FROM users WHERE id = ?`, userID).
		Scan(&hash, &status)
	if err != nil {
		return err
	}
	if status == "deleting" {
		return ErrDeleteInProgress
	}
	ok, err := password.Compare(hash, plainPassword)
	if err != nil || !ok {
		return ErrDeleteBadPassword
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `UPDATE users SET status = 'deleting' WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = ? AND revoked_at IS NULL`, userID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE feeds SET visibility = 'private' WHERE owner_id = ?`, userID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE uploads SET state = 'cancelled'
		WHERE owner_id = ? AND state IN ('uploading', 'complete')`, userID)
	if err != nil {
		return err
	}
	idem := fmt.Sprintf("acct:%d", userID)
	if err := jobs.EnqueueTx(ctx, tx, jobs.KindDeleteAccount, map[string]int64{"user_id": userID}, idem); err != nil {
		return err
	}
	return tx.Commit()
}
