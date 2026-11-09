package ratelimit

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"time"
)

var ErrLoginLocked = errors.New("too many login attempts")

const (
	loginWindow   = 15 * time.Minute
	loginMaxFails = 5
	resendIPHour  = 20
)

// LoginAllowed returns false when ≥5 failures for this email in 15 minutes.
func LoginAllowed(ctx context.Context, db *sql.DB, email string) error {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM auth_failures
		WHERE email = ? AND created_at > DATE_SUB(NOW(), INTERVAL 15 MINUTE)`, email).Scan(&n)
	if err != nil {
		return err
	}
	if n >= loginMaxFails {
		return ErrLoginLocked
	}
	return nil
}

func RecordLoginFailure(ctx context.Context, db *sql.DB, email string, ip net.IP) error {
	_, err := db.ExecContext(ctx, `INSERT INTO auth_failures (email, ip) VALUES (?, ?)`, email, ipSlice(ip))
	return err
}

func ClearLoginFailures(ctx context.Context, db *sql.DB, email string) {
	_, _ = db.ExecContext(ctx, `DELETE FROM auth_failures WHERE email = ?`, email)
}

func ResendIPAllowed(ctx context.Context, db *sql.DB, ip net.IP) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM auth_resend_log
	 WHERE ip = ? AND created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)`, ipSlice(ip)).Scan(&n)
	if err != nil {
		return false, err
	}
	return n < resendIPHour, nil
}

func RecordResend(ctx context.Context, db *sql.DB, email string, ip net.IP) error {
	_, err := db.ExecContext(ctx, `INSERT INTO auth_resend_log (email, ip) VALUES (?, ?)`, email, ipSlice(ip))
	return err
}

func ipSlice(ip net.IP) []byte {
	if ip == nil {
		return []byte{}
	}
	v4 := ip.To4()
	if v4 != nil {
		return v4
	}
	return ip.To16()
}
