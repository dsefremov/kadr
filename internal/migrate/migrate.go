package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed sql/*.sql
var files embed.FS

func Up(ctx context.Context, db *sql.DB) error {
	entries, err := files.ReadDir("sql")
	if err != nil {
		return err
	}
	var ups []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			ups = append(ups, name)
		}
	}
	sort.Strings(ups)
	for _, name := range ups {
		ver, err := parseVersion(name)
		if err != nil {
			return err
		}
		applied, err := isApplied(ctx, db, ver)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := files.ReadFile("sql/" + name)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT IGNORE INTO schema_migrations (version) VALUES (?)`, ver); err != nil {
			return fmt.Errorf("record migration %d: %w", ver, err)
		}
	}
	return nil
}

func parseVersion(name string) (int64, error) {
	prefix := strings.Split(name, "_")[0]
	return strconv.ParseInt(prefix, 10, 64)
}

func isApplied(ctx context.Context, db *sql.DB, ver int64) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, ver).Scan(&n)
	if err != nil {
		if isMissingTable(err) {
			return false, nil
		}
		return false, err
	}
	return n > 0, nil
}

func isMissingTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "1146") || strings.Contains(msg, "doesn't exist")
}
