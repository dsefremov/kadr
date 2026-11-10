package feed

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

)

const timelineProfileHash = "v1sdr20261110"

type TimelineItem struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Width      int               `json:"width,omitempty"`
	Height     int               `json:"height,omitempty"`
	DurationMs *int64            `json:"duration_ms,omitempty"`
	TakenSort  string            `json:"taken_sort"`
	TakenDate  string            `json:"taken_date"`
	URLs       map[string]string `json:"urls,omitempty"`
}

func TimelineForFeed(ctx context.Context, db *sql.DB, feedPublicID string, cursor string, limit int) (*TimelinePage, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	var feedDBID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM feeds WHERE public_id = ?`, feedPublicID).Scan(&feedDBID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	args := []any{feedDBID}
	q := `
		SELECT m.id, m.public_id, m.kind, m.width, m.height, m.duration_ms, m.taken_sort, m.taken_date
		FROM media m
		WHERE m.feed_id = ? AND m.state = 'ready' AND m.trashed_at IS NULL`
	if cursor != "" {
		taken, id, err := decodeCursor(cursor)
		if err == nil {
			q += ` AND (m.taken_sort < ? OR (m.taken_sort = ? AND m.id < ?))`
			args = append(args, taken, taken, id)
		}
	}
	q += ` ORDER BY m.taken_sort DESC, m.id DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []any
	var lastID int64
	var lastTaken time.Time
	for rows.Next() {
		var id int64
		var pub, kind, takenDate string
		var w, h sql.NullInt64
		var dur sql.NullInt64
		var taken time.Time
		if err := rows.Scan(&id, &pub, &kind, &w, &h, &dur, &taken, &takenDate); err != nil {
			return nil, err
		}
		item := TimelineItem{
			ID: pub, Kind: kind, TakenSort: taken.UTC().Format(time.RFC3339Nano), TakenDate: takenDate,
			URLs: buildURLs(pub, kind),
		}
		if w.Valid {
			item.Width = int(w.Int64)
		}
		if h.Valid {
			item.Height = int(h.Int64)
		}
		if dur.Valid {
			d := dur.Int64
			item.DurationMs = &d
		}
		items = append(items, item)
		lastID = id
		lastTaken = taken
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		c := encodeCursor(lastTaken, lastID)
		next = &c
	}
	return &TimelinePage{Items: items, Cursor: next}, nil
}

func buildURLs(pub, kind string) map[string]string {
	ph := timelineProfileHash
	if kind == "video" {
		return map[string]string{
			"video":  fmt.Sprintf("/m/%s/video.%s", pub, ph),
			"poster": fmt.Sprintf("/m/%s/poster.%s", pub, ph),
		}
	}
	return map[string]string{
		"master":      fmt.Sprintf("/m/%s/master.%s", pub, ph),
		"preview_640": fmt.Sprintf("/m/%s/preview-640.%s", pub, ph),
	}
}

func encodeCursor(taken time.Time, id int64) string {
	raw := fmt.Sprintf("%s|%d", taken.UTC().Format(time.RFC3339Nano), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(c string) (time.Time, int64, error) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, 0, err
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("bad cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, 0, err
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	return t, id, err
}
