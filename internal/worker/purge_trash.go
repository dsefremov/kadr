package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dsefremov/kadr/internal/media"
)

func (p *Processor) purgeTrash(ctx context.Context, payload json.RawMessage) error {
	var body struct {
		MediaID int64 `json:"media_id"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return err
	}
	if body.MediaID == 0 {
		return fmt.Errorf("purge_trash: missing media_id")
	}
	var trashed sql.NullTime
	var purgeAfter sql.NullTime
	err := p.db.QueryRowContext(ctx, `
		SELECT trashed_at, purge_after FROM media WHERE id = ?`, body.MediaID).
		Scan(&trashed, &purgeAfter)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !trashed.Valid {
		return nil
	}
	if purgeAfter.Valid && purgeAfter.Time.After(time.Now().UTC()) {
		return fmt.Errorf("purge_trash: not yet due for media %d", body.MediaID)
	}
	return media.PurgePermanent(ctx, p.db, p.uploads.RootDir, body.MediaID)
}
