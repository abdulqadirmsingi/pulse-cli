package db

import (
	"database/sql"
	"fmt"
	"time"
	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type Stats struct {
	TotalCommands int64
	TotalTimeMS   int64
	SuccessRate   float64
	StreakDays    int
}

type TopEntry struct {
	Name  string
	Count int64
	MS    int64
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS commands (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			command     TEXT     NOT NULL,
			directory   TEXT     NOT NULL DEFAULT '',
			project     TEXT     NOT NULL DEFAULT '',
			exit_code   INTEGER  NOT NULL DEFAULT 0,
			duration_ms INTEGER  NOT NULL DEFAULT 0,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_cmd_created ON commands(created_at);
		CREATE INDEX IF NOT EXISTS idx_cmd_project  ON commands(project);
	`)
	return err
}

func (db *DB) InsertCommand(command, dir, project string, exitCode int, durationMS int64) error {
	_, err := db.conn.Exec(
		`INSERT INTO commands (command, directory, project, exit_code, duration_ms)
		 VALUES (?, ?, ?, ?, ?)`,
		command, dir, project, exitCode, durationMS,
	)
	return err
}

func (db *DB) GetStats(days int) (*Stats, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	var s Stats
	err := db.conn.QueryRow(`
		SELECT
			COUNT(*)                                                AS total,
			COALESCE(SUM(duration_ms), 0)                          AS total_ms,
			COALESCE(AVG(CASE WHEN exit_code = 0 THEN 1.0 ELSE 0.0 END) * 100, 0) AS success_rate
		FROM commands
		WHERE created_at >= ?`, since).
		Scan(&s.TotalCommands, &s.TotalTimeMS, &s.SuccessRate)
	if err != nil {
		return nil, err
	}

	s.StreakDays = db.calcStreak()
	return &s, nil
}

func (db *DB) GetTopCommands(days, limit int) ([]TopEntry, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	rows, err := db.conn.Query(`
		SELECT
			TRIM(SUBSTR(command, 1, INSTR(command || ' ', ' ') - 1)) AS base_cmd,
			COUNT(*) AS cnt
		FROM commands
		WHERE created_at >= ? AND command != ''
		GROUP BY base_cmd
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
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

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
		SELECT DISTINCT DATE(created_at) AS day
		FROM commands
		WHERE created_at >= DATE('now', '-365 days')
		ORDER BY day DESC`)
	if err != nil {
		return 0
	}
	defer rows.Close()

	streak := 0
	expected := time.Now().Format("2006-01-02")
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
