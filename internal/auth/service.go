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
	"github.com/dsefremov/kadr/internal/units"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrBlocked            = errors.New("account blocked")
)

const (
	verifyTTL     = 24 * time.Hour
	resendMinGap  = 60 * time.Second
	resendHourMax = 5
)

type Service struct {
	DB         *sql.DB
	PublicPath string // e.g. https://app.example
}

type RegisterInput struct {
	Email    string
	Password string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) error {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email")
	}
	if err := password.Validate(in.Password); err != nil {
		return err
	}
	hash, err := password.Hash(in.Password)
	if err != nil {
		return err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID int64
	var verified sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT id, email_verified_at FROM users WHERE email = ?`, email).
		Scan(&userID, &verified)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if errors.Is(err, sql.ErrNoRows) {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO users (email, password_hash, quota_bytes)
			VALUES (?, ?, ?)`, email, hash, units.DefaultQuotaBytes)
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
		if err := issueVerifyToken(ctx, tx, s, userID, email, false); err != nil {
			return err
		}
	} else {
		if err := sendAlreadyRegistered(ctx, tx, s, email); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func sendAlreadyRegistered(ctx context.Context, tx *sql.Tx, s *Service, email string) error {
	body := "Вы уже зарегистрированы в Кадр. Если это не вы — смените пароль."
	oid, err := jobs.InsertOutboxTx(ctx, tx, "email.already_registered", map[string]string{
		"to": email, "subject": "Кадр — регистрация", "body": body,
	})
	if err != nil {
		return err
	}
	return jobs.EnqueueMailForOutbox(ctx, tx, oid)
}

func issueVerifyToken(ctx context.Context, tx *sql.Tx, s *Service, userID int64, email string, resend bool) error {
	if resend {
		var recent int
		_ = tx.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM email_tokens
			WHERE user_id = ? AND purpose = 'verify' AND created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)`,
			userID).Scan(&recent)
		if recent >= resendHourMax {
			return fmt.Errorf("rate limited")
		}
		var last time.Time
		_ = tx.QueryRowContext(ctx, `
			SELECT created_at FROM email_tokens
			WHERE user_id = ? AND purpose = 'verify'
			ORDER BY id DESC LIMIT 1`, userID).Scan(&last)
		if time.Since(last) < resendMinGap {
			return fmt.Errorf("rate limited")
		}
		_, _ = tx.ExecContext(ctx, `
			UPDATE email_tokens SET used_at = NOW()
			WHERE user_id = ? AND purpose = 'verify' AND used_at IS NULL`, userID)
	}
	plain, err := token.RandomURLSafe(32)
	if err != nil {
		return err
	}
	th := token.Hash(plain)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO email_tokens (token_hash, user_id, purpose, expires_at)
		VALUES (?, ?, 'verify', ?)`, th[:], userID, time.Now().Add(verifyTTL))
	if err != nil {
		return err
	}
	link := fmt.Sprintf("/verify?token=%s", plain)
	body := "Подтвердите почту: {{link}}"
	oid, err := jobs.InsertOutboxTx(ctx, tx, "email.verify", map[string]string{
		"to": email, "subject": "Кадр — подтвердите почту", "body": body, "link": link,
	})
	if err != nil {
		return err
	}
	return jobs.EnqueueMailForOutbox(ctx, tx, oid)
}

func (s *Service) Verify(ctx context.Context, plain string) error {
	th := token.Hash(plain)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var tokenID int64
	var userID int64
	var expires time.Time
	var used sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, used_at FROM email_tokens
		WHERE token_hash = ? AND purpose = 'verify'`, th[:]).Scan(&tokenID, &userID, &expires, &used)
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
	_, err = tx.ExecContext(ctx, `UPDATE users SET email_verified_at = NOW() WHERE id = ? AND email_verified_at IS NULL`, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func errInvalidToken() error {
	return fmt.Errorf("invalid token")
}

func (s *Service) ResendVerify(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var userID int64
	var verified sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT id, email_verified_at FROM users WHERE email = ?`, email).Scan(&userID, &verified)
	if errors.Is(err, sql.ErrNoRows) {
		// uniform response
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	if verified.Valid {
		return tx.Commit()
	}
	if err := issueVerifyToken(ctx, tx, s, userID, email, true); err != nil {
		if err.Error() == "rate limited" {
			return tx.Commit()
		}
		return err
	}
	return tx.Commit()
}

func (s *Service) Login(ctx context.Context, email, pw string) (int64, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var userID int64
	var hash string
	var status string
	var verified sql.NullTime
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, password_hash, status, email_verified_at FROM users WHERE email = ?`, email).
		Scan(&userID, &hash, &status, &verified)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidCredentials
	}
	if err != nil {
		return 0, err
	}
	if status == "blocked" {
		return 0, ErrBlocked
	}
	ok, err := password.Compare(hash, pw)
	if err != nil || !ok {
		return 0, ErrInvalidCredentials
	}
	return userID, nil
}

func uniqueHandle(ctx context.Context, tx *sql.Tx, email string) (string, error) {
	base := strings.Split(email, "@")[0]
	base = sanitizeHandle(base)
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

func sanitizeHandle(s string) string {
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
