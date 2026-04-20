package db

import "time"

type FavoriteRow struct {
	ID        int64
	Command   string
	Alias     string
	CreatedAt time.Time
}

func (db *DB) AddFavorite(command, alias string) (int64, error) {
	res, err := db.conn.Exec(
		`INSERT INTO favorites (command, alias) VALUES (?, ?)`,
		command, alias,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) ListFavorites() ([]FavoriteRow, error) {
	rows, err := db.conn.Query(
		`SELECT id, command, alias, created_at FROM favorites ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FavoriteRow
	for rows.Next() {
		var f FavoriteRow
		if err := rows.Scan(&f.ID, &f.Command, &f.Alias, &f.CreatedAt); err != nil {
			continue
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// RemoveFavorite deletes a favorite by ID. Returns false if the ID didn't exist.
func (db *DB) RemoveFavorite(id int64) (bool, error) {
	res, err := db.conn.Exec(`DELETE FROM favorites WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
