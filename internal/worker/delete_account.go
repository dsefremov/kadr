package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dsefremov/kadr/internal/media"
)

func (p *Processor) deleteAccount(ctx context.Context, payload json.RawMessage) error {
	var body struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return err
	}
	if body.UserID == 0 {
		return fmt.Errorf("delete_account: missing user_id")
	}
	var status string
	err := p.db.QueryRowContext(ctx, `SELECT status FROM users WHERE id = ?`, body.UserID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if status != "deleting" {
		return nil
	}
	rows, err := p.db.QueryContext(ctx, `SELECT id FROM media WHERE owner_id = ?`, body.UserID)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	root := p.uploads.RootDir
	for _, id := range ids {
		if err := media.PurgeAny(ctx, p.db, root, id); err != nil {
			return err
		}
	}
	_ = os.RemoveAll(filepath.Join(root, "stored", fmt.Sprintf("%d", body.UserID)))
	_, _ = p.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, body.UserID)
	return nil
}
