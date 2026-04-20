package db

import (
	"fmt"
	"time"
)

// HourlyBucket holds command count for one hour of the day.
type HourlyBucket struct {
	Hour  int
	Count int64
}

// DailyBucket holds command count for one calendar day.
type DailyBucket struct {
	Date  string
	Count int64
}

// ProjectSummary is a full per-project stats row used by `pulse projects`.
type ProjectSummary struct {
	Name        string
	TotalTimeMS int64
	Commands    int64
	SuccessRate float64
}

// GetHourlyStats returns command counts for each hour of the given date (YYYY-MM-DD).
//
// 🧠 Go Lesson #30: strftime is SQLite's time formatter.
// CAST(...AS INTEGER) converts the "09" string to integer 9.
// We pass a plain string date so the query is parameterised (safe from injection).
func (db *DB) GetHourlyStats(date string) ([]HourlyBucket, error) {
	rows, err := db.conn.Query(`
		SELECT CAST(strftime('%H', created_at) AS INTEGER) AS hr, COUNT(*) AS cnt
		FROM commands
		WHERE DATE(created_at) = ?
		GROUP BY hr
		ORDER BY hr`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HourlyBucket
	for rows.Next() {
		var b HourlyBucket
		if err := rows.Scan(&b.Hour, &b.Count); err != nil {
			continue
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetDailyActivity returns command count per day for the last N days, oldest first.
func (db *DB) GetDailyActivity(days int) ([]DailyBucket, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	rows, err := db.conn.Query(`
		SELECT DATE(created_at) AS day, COUNT(*) AS cnt
		FROM commands
		WHERE created_at >= ?
		GROUP BY day
		ORDER BY day ASC`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DailyBucket
	for rows.Next() {
		var b DailyBucket
		if err := rows.Scan(&b.Date, &b.Count); err != nil {
			continue
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetTodayStats returns a Stats summary scoped to today only.
func (db *DB) GetTodayStats() (*Stats, error) {
	today := time.Now().Format("2006-01-02")
	var s Stats
	err := db.conn.QueryRow(`
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(duration_ms), 0) AS total_ms,
			COALESCE(AVG(CASE WHEN exit_code = 0 THEN 1.0 ELSE 0.0 END) * 100, 0) AS success_rate
		FROM commands
		WHERE DATE(created_at) = ?`, today).
		Scan(&s.TotalCommands, &s.TotalTimeMS, &s.SuccessRate)
	return &s, err
}

// GetProjectList returns all projects with full stats for the last N days.
func (db *DB) GetProjectList(days int) ([]ProjectSummary, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	rows, err := db.conn.Query(`
		SELECT
			project,
			COALESCE(SUM(duration_ms), 0) AS total_ms,
			COUNT(*) AS total_cmds,
			COALESCE(AVG(CASE WHEN exit_code = 0 THEN 1.0 ELSE 0.0 END) * 100, 0) AS success_rate
		FROM commands
		WHERE created_at >= ? AND project != ''
		GROUP BY project
		ORDER BY total_ms DESC`, since)
	if err != nil {
		return nil, fmt.Errorf("querying project list: %w", err)
	}
	defer rows.Close()

	var out []ProjectSummary
	for rows.Next() {
		var p ProjectSummary
		if err := rows.Scan(&p.Name, &p.TotalTimeMS, &p.Commands, &p.SuccessRate); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
