package store

import (
	"context"
	"database/sql"
	"time"
)

const (
	IncidentStateOpen   = "open"
	IncidentStateClosed = "closed"
	SeverityWarn        = "WARN"
	SeverityCrit        = "CRIT"
)

type Incident struct {
	ID         int64
	EntityType string
	EntityName string
	EntityURL  string
	State      string
	Severity   string
	Summary    string
	StartedAt  time.Time
	EndedAt    *time.Time
	CreatedAt  time.Time
}

type IncidentUpdate struct {
	ID         int64
	IncidentID int64
	Message    string
	CreatedAt  time.Time
}

func (s *Store) OpenIncident(ctx context.Context, entityType, entityName, entityURL, severity, summary string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO incidents (entity_type, entity_name, entity_url, state, severity, summary, started_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entityType, entityName, entityURL, IncidentStateOpen, severity, summary, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) CloseIncident(ctx context.Context, id int64, summary string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `UPDATE incidents SET state = ?, ended_at = ?, summary = ? WHERE id = ?`, IncidentStateClosed, now, summary, id)
	return err
}

func (s *Store) HasOpenIncident(ctx context.Context, entityType, entityName string) (bool, int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM incidents WHERE entity_type = ? AND entity_name = ? AND state = ?`,
		entityType, entityName, IncidentStateOpen).Scan(&id)
	if err == sql.ErrNoRows {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, id, nil
}

func (s *Store) GetIncident(ctx context.Context, id int64) (*Incident, error) {
	var i Incident
	var startedAt, endedAt, createdAt sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, entity_type, entity_name, entity_url, state, severity, summary, started_at, ended_at, created_at
		 FROM incidents WHERE id = ?`, id).Scan(
		&i.ID, &i.EntityType, &i.EntityName, &i.EntityURL, &i.State, &i.Severity, &i.Summary,
		&startedAt, &endedAt, &createdAt)
	if err != nil {
		return nil, err
	}
	if startedAt.Valid {
		if t, ok := parseTime(startedAt.String); ok {
			i.StartedAt = t
		}
	}
	if endedAt.Valid {
		if t, ok := parseTime(endedAt.String); ok {
			i.EndedAt = &t
		}
	}
	if createdAt.Valid {
		if t, ok := parseTime(createdAt.String); ok {
			i.CreatedAt = t
		}
	}
	return &i, nil
}

func (s *Store) ListIncidents(ctx context.Context, state string, limit int) ([]Incident, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, entity_type, entity_name, entity_url, state, severity, summary, started_at, ended_at, created_at FROM incidents`
	args := []interface{}{}
	if state != "" {
		query += ` WHERE state = ?`
		args = append(args, state)
	}
	query += ` ORDER BY started_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Incident
	for rows.Next() {
		var i Incident
		var startedAt, endedAt, createdAt sql.NullString
		err := rows.Scan(&i.ID, &i.EntityType, &i.EntityName, &i.EntityURL, &i.State, &i.Severity, &i.Summary,
			&startedAt, &endedAt, &createdAt)
		if err != nil {
			return nil, err
		}
		if startedAt.Valid {
			if t, ok := parseTime(startedAt.String); ok {
				i.StartedAt = t
			}
		}
		if endedAt.Valid {
			if t, ok := parseTime(endedAt.String); ok {
				i.EndedAt = &t
			}
		}
		if createdAt.Valid {
			if t, ok := parseTime(createdAt.String); ok {
				i.CreatedAt = t
			}
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) AddIncidentUpdate(ctx context.Context, incidentID int64, message string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO incident_updates (incident_id, message) VALUES (?, ?)`, incidentID, message)
	return err
}

func (s *Store) IncidentUpdates(ctx context.Context, incidentID int64) ([]IncidentUpdate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, incident_id, message, created_at FROM incident_updates WHERE incident_id = ? ORDER BY created_at ASC`,
		incidentID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []IncidentUpdate
	for rows.Next() {
		var u IncidentUpdate
		var createdAt string
		if err := rows.Scan(&u.ID, &u.IncidentID, &u.Message, &createdAt); err != nil {
			return nil, err
		}
		if t, ok := parseTime(createdAt); ok {
			u.CreatedAt = t
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
