package db

import (
	"database/sql"
	"time"
)

// LastCommandAt returns the newest command timestamp in local time.
func (db *DB) LastCommandAt() (time.Time, bool, error) {
	var raw sql.NullString
	if err := db.conn.QueryRow(`SELECT MAX(created_at) FROM commands`).Scan(&raw); err != nil {
		return time.Time{}, false, err
	}
	if !raw.Valid || raw.String == "" {
		return time.Time{}, false, nil
	}
	t, err := time.ParseInLocation(sqliteTimeLayout, raw.String, time.UTC)
	if err != nil {
		return time.Time{}, false, err
	}
	return t.Local(), true, nil
}
