package db

import "time"

type GitEvent struct {
	ID         int64
	CommandID  int64
	Subcommand string
	Branch     string
	IsForce    bool
	Remote     string
	Message    string
	CreatedAt  time.Time
}

func (db *DB) InsertGitEvent(commandID int64, subcommand, branch, remote, message string, isForce bool) error {
	force := 0
	if isForce {
		force = 1
	}
	_, err := db.conn.Exec(
		`INSERT INTO git_events (command_id, subcommand, branch, is_force, remote, message)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		commandID, subcommand, branch, force, remote, message,
	)
	return err
}

func (db *DB) GetGitEvents(days int) ([]GitEvent, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	rows, err := db.conn.Query(`
		SELECT id, command_id, subcommand, branch, is_force, remote, message, created_at
		FROM git_events
		WHERE created_at >= ?
		ORDER BY created_at DESC`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []GitEvent
	for rows.Next() {
		var e GitEvent
		var force int
		if err := rows.Scan(&e.ID, &e.CommandID, &e.Subcommand, &e.Branch,
			&force, &e.Remote, &e.Message, &e.CreatedAt); err != nil {
			continue
		}
		e.IsForce = force == 1
		events = append(events, e)
	}
	return events, rows.Err()
}
