package db

import (
	"database/sql"
	"strings"
	"time"
)

func (db *DB) InsertCommand(command, dir, project string, exitCode int, durationMS int64, isNoise bool) error {
	noise := 0
	if isNoise {
		noise = 1
	}
	_, err := db.conn.Exec(
		`INSERT INTO commands (command, directory, project, exit_code, duration_ms, noise)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		command, dir, project, exitCode, durationMS, noise,
	)
	return err
}

func (db *DB) InsertCommandGetID(command, dir, project string, exitCode int, durationMS int64, isNoise bool) (int64, error) {
	noise := 0
	if isNoise {
		noise = 1
	}
	res, err := db.conn.Exec(
		`INSERT INTO commands (command, directory, project, exit_code, duration_ms, noise)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		command, dir, project, exitCode, durationMS, noise,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) GetStats(days int) (*Stats, error) {
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
	var s Stats
	err := db.conn.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN noise = 0 THEN 1 ELSE 0 END), 0)            AS dev_total,
			COALESCE(SUM(CASE WHEN noise = 1 THEN 1 ELSE 0 END), 0)            AS noise_total,
			COALESCE(SUM(CASE WHEN noise = 0 THEN duration_ms ELSE 0 END), 0)  AS total_ms,
			COALESCE(AVG(CASE WHEN noise = 0 AND exit_code = 0 THEN 1.0
			               WHEN noise = 0 THEN 0.0 END) * 100, 0)             AS success_rate
		FROM commands
		WHERE created_at >= ?`, since).
		Scan(&s.TotalCommands, &s.NoiseCommands, &s.TotalTimeMS, &s.SuccessRate)
	if err != nil {
		return nil, err
	}
	s.StreakDays = db.calcStreak()
	return &s, nil
}

func (db *DB) GetTopCommands(days, limit int) ([]TopEntry, error) {
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
	rows, err := db.conn.Query(`
		SELECT
			SUBSTR(TRIM(command), 1, INSTR(TRIM(command) || ' ', ' ') - 1) AS base_cmd,
			COUNT(*) AS cnt
		FROM commands
		WHERE created_at >= ? AND TRIM(command) != '' AND noise = 0
		GROUP BY base_cmd
		HAVING base_cmd != ''
		ORDER BY cnt DESC
		LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []TopEntry
	for rows.Next() {
		var e TopEntry
		if err := rows.Scan(&e.Name, &e.Count); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (db *DB) GetTopProjects(days, limit int) ([]TopEntry, error) {
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
	rows, err := db.conn.Query(`
		SELECT project, COUNT(*) AS cnt, COALESCE(SUM(duration_ms), 0) AS total_ms
		FROM commands
		WHERE created_at >= ? AND project != ''
		GROUP BY project
		ORDER BY total_ms DESC
		LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []TopEntry
	for rows.Next() {
		var e TopEntry
		if err := rows.Scan(&e.Name, &e.Count, &e.MS); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (db *DB) calcStreak() int {
	rows, err := db.conn.Query(`
		SELECT DISTINCT DATE(created_at, 'localtime') AS day
		FROM commands
		WHERE created_at >= datetime('now', '-3650 days')
		ORDER BY day DESC`)
	if err != nil {
		return 0
	}
	defer rows.Close()

	streak := 0
	expected := time.Now().Format("2006-01-02") // local date
	for rows.Next() {
		var day string
		if err := rows.Scan(&day); err != nil {
			break
		}
		if day != expected {
			break
		}
		streak++
		t, _ := time.Parse("2006-01-02", day)
		expected = t.AddDate(0, 0, -1).Format("2006-01-02")
	}
	return streak
}

// CommandRow is a single raw command entry for display in history.
type CommandRow struct {
	Command    string
	ExitCode   int
	DurationMS int64
	CreatedAt  time.Time
	Noise      bool
}

// GetTodayCommands returns every command logged today in chronological order.
// Noise commands are included but flagged so the caller can filter or style them.
func (db *DB) GetTodayCommands() ([]CommandRow, error) {
	rows, err := db.conn.Query(`
		SELECT command, exit_code, duration_ms, created_at, noise
		FROM commands
		WHERE DATE(created_at, 'localtime') = DATE('now', 'localtime') AND TRIM(command) != ''
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CommandRow
	for rows.Next() {
		var c CommandRow
		var noise int
		if err := rows.Scan(&c.Command, &c.ExitCode, &c.DurationMS, &c.CreatedAt, &noise); err != nil {
			continue
		}
		c.Noise = noise == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

// escapeLike escapes LIKE special characters so user input is treated as a literal string.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// SearchCommands returns commands matching query (case-insensitive LIKE).
// days=0 means all-time. Results are ordered newest-first.
func (db *DB) SearchCommands(query string, days, limit int) ([]CommandRow, error) {
	pattern := "%" + escapeLike(query) + "%"
	var (
		rows *sql.Rows
		err  error
	)
	if days > 0 {
		since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
		rows, err = db.conn.Query(`
			SELECT command, exit_code, duration_ms, created_at, noise
			FROM commands
			WHERE command LIKE ? ESCAPE '\' AND created_at >= ? AND TRIM(command) != ''
			ORDER BY created_at DESC LIMIT ?`, pattern, since, limit)
	} else {
		rows, err = db.conn.Query(`
			SELECT command, exit_code, duration_ms, created_at, noise
			FROM commands
			WHERE command LIKE ? ESCAPE '\' AND TRIM(command) != ''
			ORDER BY created_at DESC LIMIT ?`, pattern, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CommandRow
	for rows.Next() {
		var c CommandRow
		var noise int
		if err := rows.Scan(&c.Command, &c.ExitCode, &c.DurationMS, &c.CreatedAt, &noise); err != nil {
			continue
		}
		c.Noise = noise == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) ResetCommands() (int64, error) {
	// delete git_events first to handle databases created before ON DELETE CASCADE
	// was added to the schema — those tables have a plain FK that blocks the parent delete
	if _, err := d.conn.Exec(`DELETE FROM git_events`); err != nil {
		return 0, err
	}
	res, err := d.conn.Exec(`DELETE FROM commands`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
