package upload

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dsefremov/kadr/internal/jobs"
	"github.com/dsefremov/kadr/internal/token"
	"github.com/dsefremov/kadr/internal/units"
)

const (
	TusVersion   = "1.0.0"
	UploadExpiry = 24 * time.Hour
)

var (
	ErrNotFound     = errors.New("upload not found")
	ErrLength       = errors.New("upload length exceeds limit")
	ErrQuota        = errors.New("quota exceeded")
	ErrOffset       = errors.New("upload offset conflict")
	ErrUnverified   = errors.New("email not verified")
	ErrInvalidState = errors.New("invalid upload state")
)

type Service struct {
	DB       *sql.DB
	RootDir  string
	PublicURL string
}

type Row struct {
	ID           int64
	PublicID     string
	OwnerID      int64
	Kind         string
	UploadLength uint64
	Offset       uint64
	IncomingKey  string
	State        string
}

func (s *Service) Create(ctx context.Context, ownerID int64, length uint64, kind string) (*Row, error) {
	if kind != "photo" && kind != "video" {
		return nil, fmt.Errorf("unknown kind")
	}
	max := units.PhotoMaxBytes
	if kind == "video" {
		max = units.VideoMaxBytes
	}
	if length == 0 || length > uint64(max) {
		return nil, ErrLength
	}
	pubID, err := token.RandomURLSafe(16)
	if err != nil {
		return nil, err
	}
	key := filepath.Join("incoming", fmt.Sprintf("%d", ownerID), pubID)
	full := filepath.Join(s.RootDir, key)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(full, os.O_CREATE|os.O_RDWR, 0o640)
	if err != nil {
		return nil, err
	}
	_ = f.Close()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `
		UPDATE users SET reserved_bytes = reserved_bytes + ?
		WHERE id = ? AND (used_bytes + trash_bytes + reserved_bytes + ?) <= quota_bytes`,
		length, ownerID, length)
	if err != nil {
		return nil, err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		_ = os.Remove(full)
		return nil, ErrQuota
	}
	exp := time.Now().Add(UploadExpiry)
	res2, err := tx.ExecContext(ctx, `
		INSERT INTO uploads (public_id, owner_id, kind_declared, upload_length, offset_bytes, incoming_key, state, last_activity_at, expires_at)
		VALUES (?, ?, ?, ?, 0, ?, 'uploading', NOW(), ?)`,
		pubID, ownerID, kind, length, key, exp)
	if err != nil {
		return nil, err
	}
	id, _ := res2.LastInsertId()
	_ = jobs.EnqueueTxAfter(ctx, tx, jobs.KindExpireUpload, map[string]int64{"upload_id": id},
		fmt.Sprintf("exp:%d", id), time.Now().Add(UploadExpiry))
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &Row{
		ID: id, PublicID: pubID, OwnerID: ownerID, Kind: kind,
		UploadLength: length, Offset: 0, IncomingKey: key, State: "uploading",
	}, nil
}

func ParseKindFromMetadata(header string) string {
	if header == "" {
		return "photo"
	}
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, " ", 2)
		if len(kv) != 2 || kv[0] != "filetype" {
			continue
		}
		raw, err := base64.RawStdEncoding.DecodeString(kv[1])
		if err != nil {
			raw, err = base64.StdEncoding.DecodeString(kv[1])
		}
		if err != nil {
			raw = []byte(kv[1])
		}
		k := strings.ToLower(strings.TrimSpace(string(raw)))
		if k == "video" || k == "photo" {
			return k
		}
	}
	return "photo"
}

func (s *Service) Get(ctx context.Context, publicID string, ownerID int64) (*Row, error) {
	var row Row
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, public_id, owner_id, kind_declared, upload_length, offset_bytes, incoming_key, state
		FROM uploads WHERE public_id = ? AND owner_id = ?`, publicID, ownerID).
		Scan(&row.ID, &row.PublicID, &row.OwnerID, &row.Kind, &row.UploadLength, &row.Offset, &row.IncomingKey, &row.State)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) Patch(ctx context.Context, publicID string, ownerID int64, expectOffset int64, body io.Reader) (*Row, error) {
	row, err := s.Get(ctx, publicID, ownerID)
	if err != nil {
		return nil, err
	}
	if row.State != "uploading" {
		return nil, ErrInvalidState
	}
	if int64(row.Offset) != expectOffset {
		return nil, ErrOffset
	}
	path := filepath.Join(s.RootDir, row.IncomingKey)
	f, err := os.OpenFile(path, os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.Seek(int64(row.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	written, err := io.Copy(f, body)
	if err != nil {
		return nil, err
	}
	newOff := row.Offset + uint64(written)
	if newOff > row.UploadLength {
		return nil, ErrLength
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
		UPDATE uploads SET offset_bytes = ?, last_activity_at = NOW()
		WHERE id = ? AND offset_bytes = ?`, newOff, row.ID, row.Offset)
	if err != nil {
		return nil, err
	}
	if newOff == row.UploadLength {
		_, err = tx.ExecContext(ctx, `UPDATE uploads SET state = 'complete' WHERE id = ?`, row.ID)
		if err != nil {
			return nil, err
		}
		if err := jobs.EnqueueDeriveTx(ctx, tx, row.ID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	row.Offset = newOff
	if newOff == row.UploadLength {
		row.State = "complete"
	}
	return row, nil
}

func (s *Service) Delete(ctx context.Context, publicID string, ownerID int64) error {
	row, err := s.Get(ctx, publicID, ownerID)
	if err != nil {
		return err
	}
	return s.expireRow(ctx, row, "cancelled")
}

func (s *Service) ExpireByID(ctx context.Context, uploadID int64) error {
	var row Row
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, public_id, owner_id, kind_declared, upload_length, offset_bytes, incoming_key, state
		FROM uploads WHERE id = ?`, uploadID).
		Scan(&row.ID, &row.PublicID, &row.OwnerID, &row.Kind, &row.UploadLength, &row.Offset, &row.IncomingKey, &row.State)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.State != "uploading" {
		return nil
	}
	return s.expireRow(ctx, &row, "expired")
}

func (s *Service) expireRow(ctx context.Context, row *Row, state string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if row.State == "uploading" {
		_, err = tx.ExecContext(ctx, `
			UPDATE users SET reserved_bytes = reserved_bytes - ?
			WHERE id = ? AND reserved_bytes >= ?`, row.UploadLength, row.OwnerID, row.UploadLength)
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE uploads SET state = ? WHERE id = ?`, state, row.ID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(s.RootDir, row.IncomingKey))
	return nil
}

func (s *Service) ListActive(ctx context.Context, ownerID int64) ([]Row, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, public_id, owner_id, kind_declared, upload_length, offset_bytes, incoming_key, state
		FROM uploads WHERE owner_id = ? AND state IN ('uploading', 'complete')
		ORDER BY id DESC LIMIT 50`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.PublicID, &r.OwnerID, &r.Kind, &r.UploadLength, &r.Offset, &r.IncomingKey, &r.State); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
