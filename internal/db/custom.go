package db

import (
	"errors"
	"strings"
	"time"
)

var ErrCommandExists = errors.New("custom command already exists")
var ErrCommandNotFound = errors.New("custom command not found")

type CustomCommandRow struct {
	ID        int64
	Name      string
	Command   string
	CreatedAt time.Time
}

func (db *DB) AddCustomCommand(name, command string) error {
	_, err := db.conn.Exec(
		`INSERT INTO custom_commands (name, command) VALUES (?, ?)`,
		name, command,
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return ErrCommandExists
	}
	return err
}

func (db *DB) ListCustomCommands() ([]CustomCommandRow, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, command, created_at FROM custom_commands ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CustomCommandRow
	for rows.Next() {
		var r CustomCommandRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Command, &r.CreatedAt); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (db *DB) RemoveCustomCommand(name string) error {
	res, err := db.conn.Exec(`DELETE FROM custom_commands WHERE name = ?`, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrCommandNotFound
	}
	return nil
}
