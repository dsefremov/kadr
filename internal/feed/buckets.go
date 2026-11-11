package feed

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type TimelineBucket struct {
	Month     string `json:"month"`
	Label     string `json:"label"`
	Count     int    `json:"count"`
	AnchorID  string `json:"anchor_id"`
	TakenSort string `json:"taken_sort"`
}

var ruMonths = [...]string{
	"", "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
	"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

func monthLabel(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return fmt.Sprintf("%s %d", ruMonths[int(t.Month())], t.Year())
}

func TimelineBuckets(ctx context.Context, db *sql.DB, feedPublicID string) ([]TimelineBucket, error) {
	var feedDBID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM feeds WHERE public_id = ?`, feedPublicID).Scan(&feedDBID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT DATE_FORMAT(m.taken_date, '%Y-%m') AS ym,
			COUNT(*) AS cnt,
			SUBSTRING_INDEX(GROUP_CONCAT(m.public_id ORDER BY m.taken_sort DESC, m.id DESC), ',', 1) AS anchor,
			MAX(m.taken_sort) AS newest
		FROM media m
		WHERE m.feed_id = ? AND m.state = 'ready' AND m.trashed_at IS NULL
		GROUP BY ym
		ORDER BY ym DESC`, feedDBID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TimelineBucket
	for rows.Next() {
		var ym, anchor string
		var cnt int
		var newest time.Time
		if err := rows.Scan(&ym, &cnt, &anchor, &newest); err != nil {
			return nil, err
		}
		out = append(out, TimelineBucket{
			Month: ym, Label: monthLabel(ym), Count: cnt, AnchorID: anchor,
			TakenSort: newest.UTC().Format(time.RFC3339Nano),
		})
	}
	return out, rows.Err()
}
