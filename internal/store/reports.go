package store

import (
	"context"
	"database/sql"
	"time"
)

type Report struct {
	ID          int64
	IssueType   string
	Wallet      string
	Device      string
	Region      string
	Description string
	URL         string
	TxHash      string
	UserAgent   string
	IncidentID  sql.NullInt64
	CreatedAt   time.Time
}

const (
	MaxReportIssueType   = 64
	MaxReportWallet      = 128
	MaxReportDevice      = 64
	MaxReportRegion      = 64
	MaxReportDescription = 2048
	MaxReportURL         = 512
	MaxReportTxHash      = 128
	MaxReportUserAgent   = 512
)

func (s *Store) InsertReport(ctx context.Context, r *Report) (int64, error) {
	var incidentID sql.NullInt64
	if r.IncidentID.Valid {
		incidentID = r.IncidentID
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO reports (issue_type, wallet, device, region, description, url, tx_hash, user_agent, incident_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		trunc(r.IssueType, MaxReportIssueType),
		trunc(r.Wallet, MaxReportWallet),
		trunc(r.Device, MaxReportDevice),
		trunc(r.Region, MaxReportRegion),
		trunc(r.Description, MaxReportDescription),
		trunc(r.URL, MaxReportURL),
		trunc(r.TxHash, MaxReportTxHash),
		trunc(r.UserAgent, MaxReportUserAgent),
		incidentID)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return id, err
}

func (s *Store) ListReports(ctx context.Context, limit int) ([]Report, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, issue_type, wallet, device, region, description, url, tx_hash, user_agent, incident_id, created_at
		 FROM reports ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Report
	for rows.Next() {
		var r Report
		var createdAt string
		if err := rows.Scan(&r.ID, &r.IssueType, &r.Wallet, &r.Device, &r.Region, &r.Description, &r.URL, &r.TxHash, &r.UserAgent, &r.IncidentID, &createdAt); err != nil {
			return nil, err
		}
		if t, ok := parseTime(createdAt); ok {
			r.CreatedAt = t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
