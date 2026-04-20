package db

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type Stats struct {
	TotalCommands int64
	NoiseCommands int64
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
	db.prune()
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	db.conn.Exec(`PRAGMA journal_mode=WAL`)
	db.conn.Exec(`PRAGMA synchronous=NORMAL`)
	db.conn.Exec(`PRAGMA foreign_keys=ON`)
	db.conn.Exec(`PRAGMA auto_vacuum=INCREMENTAL`)

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS commands (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			command     TEXT     NOT NULL,
			directory   TEXT     NOT NULL DEFAULT '',
			project     TEXT     NOT NULL DEFAULT '',
			exit_code   INTEGER  NOT NULL DEFAULT 0,
			duration_ms INTEGER  NOT NULL DEFAULT 0,
			noise       INTEGER  NOT NULL DEFAULT 0,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cmd_created         ON commands(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cmd_project         ON commands(project)`,
		`CREATE INDEX IF NOT EXISTS idx_cmd_project_created ON commands(project, created_at)`,
		`CREATE TABLE IF NOT EXISTS git_events (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			command_id  INTEGER  REFERENCES commands(id) ON DELETE CASCADE,
			subcommand  TEXT     NOT NULL,
			branch      TEXT     NOT NULL DEFAULT '',
			is_force    INTEGER  NOT NULL DEFAULT 0,
			remote      TEXT     NOT NULL DEFAULT '',
			message     TEXT     NOT NULL DEFAULT '',
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_git_branch  ON git_events(branch)`,
		`CREATE INDEX IF NOT EXISTS idx_git_sub     ON git_events(subcommand)`,
		`CREATE INDEX IF NOT EXISTS idx_git_created ON git_events(created_at)`,
	}
	for _, s := range stmts {
		if _, err := db.conn.Exec(s); err != nil {
			return err
		}
	}
	// additive column migrations for existing databases
	db.conn.Exec(`ALTER TABLE commands ADD COLUMN noise INTEGER NOT NULL DEFAULT 0`)
	return nil
}

// prune deletes old rows and reclaims disk space. Noise commands are kept for
// 7 days; all other commands for 90 days. git_events are pruned via CASCADE
// when their parent command row is deleted.
func (db *DB) prune() {
	db.conn.Exec(`DELETE FROM commands WHERE noise = 1 AND created_at < datetime('now', '-7 days')`)
	db.conn.Exec(`DELETE FROM commands WHERE noise = 0 AND created_at < datetime('now', '-90 days')`)
	db.conn.Exec(`DELETE FROM git_events WHERE command_id NOT IN (SELECT id FROM commands)`)
	db.conn.Exec(`PRAGMA incremental_vacuum`)
	db.conn.Exec(`PRAGMA wal_checkpoint(PASSIVE)`)
}
