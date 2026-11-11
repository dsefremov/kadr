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
	"strconv"
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

// DuplicateMediaError is returned when the uploaded bytes match existing media (§12).
type DuplicateMediaError struct {
	MediaPublicID string
	MediaState    string
	InTrash       bool
}

func (e *DuplicateMediaError) Error() string {
	return "duplicate media"
}

type Service struct {
	DB       *sql.DB
	RootDir  string
	PublicURL string
}

type Row struct {
	ID             int64
	PublicID       string
	OwnerID        int64
	Kind           string
	ClientFilename string
	ClientMTime    time.Time // zero if unknown
	UploadLength   uint64
	Offset         uint64
	IncomingKey    string
	State          string
}

// CreateOpts carries optional client metadata from tus Upload-Metadata.
type CreateOpts struct {
	Filename string
	MTime    time.Time
}

func (s *Service) Create(ctx context.Context, ownerID int64, length uint64, kind string, opts CreateOpts) (*Row, error) {
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
	var filename any
	if opts.Filename != "" {
		filename = truncateRunes(opts.Filename, 255)
	}
	var mtime any
	if !opts.MTime.IsZero() {
		mtime = opts.MTime.UTC()
	}
	res2, err := tx.ExecContext(ctx, `
		INSERT INTO uploads (public_id, owner_id, kind_declared, client_filename, client_mtime,
			upload_length, offset_bytes, incoming_key, state, last_activity_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?, 'uploading', NOW(), ?)`,
		pubID, ownerID, kind, filename, mtime, length, key, exp)
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
		ClientFilename: opts.Filename, ClientMTime: opts.MTime,
		UploadLength: length, Offset: 0, IncomingKey: key, State: "uploading",
	}, nil
}

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func ParseKindFromMetadata(header string) string {
	meta := ParseMetadata(header)
	return meta.Kind
}

// Metadata is decoded tus Upload-Metadata.
type Metadata struct {
	Kind     string
	Filename string
	MTime    time.Time
}

func ParseMetadata(header string) Metadata {
	out := Metadata{Kind: "photo"}
	if header == "" {
		return out
	}
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, " ", 2)
		if len(kv) != 2 {
			continue
		}
		raw, err := base64.RawStdEncoding.DecodeString(kv[1])
		if err != nil {
			raw, err = base64.StdEncoding.DecodeString(kv[1])
		}
		if err != nil {
			raw = []byte(kv[1])
		}
		val := strings.TrimSpace(string(raw))
		switch kv[0] {
		case "filetype":
			k := strings.ToLower(val)
			if k == "video" || k == "photo" {
				out.Kind = k
			}
		case "filename":
			out.Filename = val
		case "mtime":
			if ms, err := strconv.ParseInt(val, 10, 64); err == nil && ms > 0 {
				out.MTime = time.UnixMilli(ms).UTC()
			}
		}
	}
	return out
}

func (s *Service) Get(ctx context.Context, publicID string, ownerID int64) (*Row, error) {
	var row Row
	var filename sql.NullString
	var mtime sql.NullTime
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, public_id, owner_id, kind_declared, client_filename, client_mtime,
			upload_length, offset_bytes, incoming_key, state
		FROM uploads WHERE public_id = ? AND owner_id = ?`, publicID, ownerID).
		Scan(&row.ID, &row.PublicID, &row.OwnerID, &row.Kind, &filename, &mtime,
			&row.UploadLength, &row.Offset, &row.IncomingKey, &row.State)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if filename.Valid {
		row.ClientFilename = filename.String
	}
	if mtime.Valid {
		row.ClientMTime = mtime.Time
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
	if newOff == row.UploadLength {
		dedup, derr := hashFile(path)
		if derr != nil {
			return nil, derr
		}
		hit, derr := findExistingMedia(ctx, s.DB, row.OwnerID, dedup, row.UploadLength)
		if derr != nil {
			return nil, derr
		}
		if hit != nil {
			if err := s.abortDuplicate(ctx, row); err != nil {
				return nil, err
			}
			return nil, &DuplicateMediaError{
				MediaPublicID: hit.publicID,
				MediaState:    hit.state,
				InTrash:       hit.inTrash,
			}
		}
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

func (s *Service) abortDuplicate(ctx context.Context, row *Row) error {
	return s.expireRow(ctx, row, "cancelled")
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
	// Hide complete uploads once media exists (ready/processing/failed) — otherwise Загрузки
	// keeps "Отменить" forever after the photo already appears in the feed.
	rows, err := s.DB.QueryContext(ctx, `
		SELECT u.id, u.public_id, u.owner_id, u.kind_declared, u.upload_length, u.offset_bytes, u.incoming_key, u.state
		FROM uploads u
		WHERE u.owner_id = ?
		  AND u.state IN ('uploading', 'complete')
		  AND NOT EXISTS (
		    SELECT 1 FROM media m WHERE m.upload_id = u.id
		  )
		ORDER BY u.id DESC LIMIT 50`, ownerID)
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
