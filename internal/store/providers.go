package store

import (
	"context"
	"database/sql"
)

func (s *Store) EnsureProvider(ctx context.Context, name, url string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM providers WHERE name = ?`, name).Scan(&id)
	if err == nil {
		_, err = s.db.ExecContext(ctx, `UPDATE providers SET url = ? WHERE id = ?`, url, id)
		return id, err
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO providers (name, url) VALUES (?, ?)`, name, url)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) EnsureDapp(ctx context.Context, name, url string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM dapps WHERE name = ?`, name).Scan(&id)
	if err == nil {
		_, err = s.db.ExecContext(ctx, `UPDATE dapps SET url = ? WHERE id = ?`, url, id)
		return id, err
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO dapps (name, url) VALUES (?, ?)`, name, url)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
