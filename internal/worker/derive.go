package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dsefremov/kadr/internal/media"
	"github.com/dsefremov/kadr/internal/upload"
)

func (p *Processor) derive(ctx context.Context, payload []byte) error {
	var pl struct {
		UploadID int64 `json:"upload_id"`
	}
	if err := json.Unmarshal(payload, &pl); err != nil {
		return err
	}
	var row upload.Row
	err := p.db.QueryRowContext(ctx, `
		SELECT id, public_id, owner_id, kind_declared, upload_length, offset_bytes, incoming_key, state
		FROM uploads WHERE id = ?`, pl.UploadID).
		Scan(&row.ID, &row.PublicID, &row.OwnerID, &row.Kind, &row.UploadLength, &row.Offset, &row.IncomingKey, &row.State)
	if err != nil {
		return err
	}
	if row.State != "complete" {
		return nil
	}
	src := filepath.Join(p.uploads.RootDir, row.IncomingKey)
	kind, err := media.SniffKind(src)
	if err != nil {
		p.failUpload(ctx, &row, "unsupported", err.Error())
		return nil
	}
	if kind != row.Kind {
		kind = row.Kind
	}
	if anim, err := media.IsAnimatedImage(src); err == nil && anim {
		p.failUpload(ctx, &row, "animated", media.ErrAnimated.Error())
		return nil
	}
	dedup, err := media.HashFile(src)
	if err != nil {
		return err
	}
	mediaID, mediaPub, err := media.CreateProcessing(ctx, p.db, row.OwnerID, row.ID, kind, row.UploadLength, dedup, row.IncomingKey)
	if errors.Is(err, media.ErrDuplicate) {
		_ = media.ReleaseReserved(ctx, p.db, row.OwnerID, row.UploadLength)
		media.DeleteIncoming(p.uploads, &row)
		p.log.Info("duplicate upload", "upload_id", row.ID)
		return nil
	}
	if err != nil {
		return err
	}
	spool := media.SpoolRoot()
	jobID := fmt.Sprintf("u%d", row.ID)
	_, err = media.PrepareJob(spool, jobID, kind, src)
	if err != nil {
		p.failMedia(ctx, mediaID, &row, "spool", err.Error())
		return nil
	}
	result, err := media.WaitResult(spool, jobID, 30*time.Minute)
	if err != nil || result == nil || !result.OK {
		media.CleanupJob(spool, jobID)
		msg := "processing failed"
		code := "processing"
		if result != nil {
			msg = result.ErrorMessage
			code = result.ErrorCode
		}
		if err != nil {
			return err
		}
		p.failMedia(ctx, mediaID, &row, code, msg)
		return nil
	}
	outDir := filepath.Join(spool, "out", jobID)
	if err := media.MoveRunnerOutputs(p.uploads.RootDir, row.OwnerID, mediaPub, outDir, result.Variants); err != nil {
		p.failMedia(ctx, mediaID, &row, "store", err.Error())
		return nil
	}
	var stored uint64
	for _, v := range result.Variants {
		stored += v.Bytes
	}
	err = media.CommitReady(ctx, p.db, uint64(mediaID), uint64(row.OwnerID), row.UploadLength, stored,
		result.Variants, mediaPub, result.Width, result.Height, result.DurationMs)
	if errors.Is(err, media.ErrWaitingQuota) {
		p.log.Info("media waiting quota", "media_id", mediaID)
		return nil
	}
	if err != nil {
		p.failMedia(ctx, mediaID, &row, "quota", err.Error())
		return nil
	}
	media.DeleteIncoming(p.uploads, &row)
	media.CleanupJob(spool, jobID)
	p.log.Info("derive complete", "upload_id", row.ID, "media_id", mediaID, "pub", mediaPub)
	return nil
}

func (p *Processor) failUpload(ctx context.Context, row *upload.Row, code, msg string) {
	_ = media.InsertFailed(ctx, p.db, row.OwnerID, row.ID, row.Kind, code, msg, row.UploadLength)
	media.DeleteIncoming(p.uploads, row)
}

func (p *Processor) failMedia(ctx context.Context, mediaID int64, row *upload.Row, code, msg string) {
	_ = media.MarkFailed(ctx, p.db, mediaID, row.OwnerID, row.UploadLength, code, msg)
	media.DeleteIncoming(p.uploads, row)
}

func spoolWritable() bool {
	spool := media.SpoolRoot()
	if err := os.MkdirAll(filepath.Join(spool, "in"), 0o750); err != nil {
		return false
	}
	return true
}
