package store

import (
	"context"
	"database/sql"
	"time"
)

type CheckRow struct {
	ID            int64
	EntityType    string
	EntityName    string
	Success       bool
	LatencyMs     sql.NullInt64
	ErrorCategory sql.NullString
	CreatedAt     time.Time
}

func (s *Store) InsertCheck(ctx context.Context, entityType, entityName string, success bool, latencyMs *int64, errorCategory string) error {
	var lat sql.NullInt64
	if latencyMs != nil {
		lat = sql.NullInt64{Int64: *latencyMs, Valid: true}
	}
	var errCat sql.NullString
	if errorCategory != "" {
		errCat = sql.NullString{String: errorCategory, Valid: true}
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO checks (entity_type, entity_name, success, latency_ms, error_category) VALUES (?, ?, ?, ?, ?)`,
		entityType, entityName, success, lat, errCat)
	return err
}

func (s *Store) RecentChecks(ctx context.Context, entityType, entityName string, limit int) ([]CheckRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, entity_type, entity_name, success, latency_ms, error_category, created_at
		 FROM checks WHERE entity_type = ? AND entity_name = ? ORDER BY created_at DESC LIMIT ?`,
		entityType, entityName, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []CheckRow
	for rows.Next() {
		var c CheckRow
		var lat sql.NullInt64
		var errCat sql.NullString
		var createdAt string
		if err := rows.Scan(&c.ID, &c.EntityType, &c.EntityName, &c.Success, &lat, &errCat, &createdAt); err != nil {
			return nil, err
		}
		if lat.Valid {
			c.LatencyMs = lat
		}
		if errCat.Valid {
			c.ErrorCategory = errCat
		}
		if t, ok := parseTime(createdAt); ok {
			c.CreatedAt = t
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) TrimChecks(ctx context.Context, entityType, entityName string, keep int) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM checks WHERE entity_type = ? AND entity_name = ? AND id NOT IN (
			SELECT id FROM checks WHERE entity_type = ? AND entity_name = ? ORDER BY created_at DESC LIMIT ?
		)`,
		entityType, entityName, entityType, entityName, keep)
	return err
}
