package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNew_and_Migrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	ctx := context.Background()
	s, err := New(ctx, path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = s.Close() }()
	if s.db == nil {
		t.Fatal("db is nil")
	}
}

func TestEnsureProvider_and_EnsureDapp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	ctx := context.Background()
	s, err := New(ctx, path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = s.Close() }()

	id1, err := s.EnsureProvider(ctx, "aptoslabs", "https://fullnode.mainnet.aptoslabs.com/v1")
	if err != nil {
		t.Fatalf("EnsureProvider: %v", err)
	}
	if id1 <= 0 {
		t.Errorf("provider id = %d", id1)
	}
	id2, err := s.EnsureProvider(ctx, "aptoslabs", "https://other.url/v1")
	if err != nil {
		t.Fatalf("EnsureProvider again: %v", err)
	}
	if id2 != id1 {
		t.Errorf("same name should return same id: %d != %d", id2, id1)
	}

	dappID, err := s.EnsureDapp(ctx, "explorer", "https://explorer.aptoslabs.com")
	if err != nil {
		t.Fatalf("EnsureDapp: %v", err)
	}
	if dappID <= 0 {
		t.Errorf("dapp id = %d", dappID)
	}
}

func TestChecks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	ctx := context.Background()
	s, err := New(ctx, path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = s.Close() }()

	if err := s.InsertCheck(ctx, "rpc", "aptoslabs", true, int64Ptr(100), ""); err != nil {
		t.Fatalf("InsertCheck: %v", err)
	}
	if err := s.InsertCheck(ctx, "rpc", "aptoslabs", false, nil, "timeout"); err != nil {
		t.Fatalf("InsertCheck: %v", err)
	}
	list, err := s.RecentChecks(ctx, "rpc", "aptoslabs", 10)
	if err != nil {
		t.Fatalf("RecentChecks: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(RecentChecks) = %d", len(list))
	}
	var successCount int
	for _, c := range list {
		if c.Success {
			successCount++
		}
	}
	if successCount != 1 {
		t.Errorf("expected 1 success, got %d", successCount)
	}
}

func TestIncidents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	ctx := context.Background()
	s, err := New(ctx, path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = s.Close() }()

	ok, id, err := s.HasOpenIncident(ctx, "rpc", "aptoslabs")
	if err != nil {
		t.Fatalf("HasOpenIncident: %v", err)
	}
	if ok {
		t.Fatal("no open incident yet")
	}
	_ = id

	incID, err := s.OpenIncident(ctx, "rpc", "aptoslabs", "https://fullnode.mainnet.aptoslabs.com/v1", "CRIT", "RPC unreachable")
	if err != nil {
		t.Fatalf("OpenIncident: %v", err)
	}
	if incID <= 0 {
		t.Errorf("incident id = %d", incID)
	}

	ok, id, err = s.HasOpenIncident(ctx, "rpc", "aptoslabs")
	if err != nil {
		t.Fatalf("HasOpenIncident: %v", err)
	}
	if !ok || id != incID {
		t.Errorf("HasOpenIncident: ok=%v id=%d", ok, id)
	}

	inc, err := s.GetIncident(ctx, incID)
	if err != nil {
		t.Fatalf("GetIncident: %v", err)
	}
	if inc.State != IncidentStateOpen || inc.Severity != "CRIT" {
		t.Errorf("incident state=%q severity=%q", inc.State, inc.Severity)
	}

	if err := s.AddIncidentUpdate(ctx, incID, "Investigating"); err != nil {
		t.Fatalf("AddIncidentUpdate: %v", err)
	}
	updates, err := s.IncidentUpdates(ctx, incID)
	if err != nil {
		t.Fatalf("IncidentUpdates: %v", err)
	}
	if len(updates) != 1 {
		t.Fatalf("len(updates) = %d", len(updates))
	}

	if err := s.CloseIncident(ctx, incID, "Resolved"); err != nil {
		t.Fatalf("CloseIncident: %v", err)
	}
	inc, _ = s.GetIncident(ctx, incID)
	if inc.State != IncidentStateClosed {
		t.Errorf("state after close = %q", inc.State)
	}

	ok, _, _ = s.HasOpenIncident(ctx, "rpc", "aptoslabs")
	if ok {
		t.Fatal("incident should be closed")
	}

	list, err := s.ListIncidents(ctx, "closed", 10)
	if err != nil {
		t.Fatalf("ListIncidents: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListIncidents(closed) len = %d", len(list))
	}
}

func TestReports(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	ctx := context.Background()
	s, err := New(ctx, path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = s.Close() }()

	r := &Report{IssueType: "rpc_down", Description: "Cannot connect", Wallet: "0x1"}
	id, err := s.InsertReport(ctx, r)
	if err != nil {
		t.Fatalf("InsertReport: %v", err)
	}
	if id <= 0 {
		t.Errorf("report id = %d", id)
	}
	list, err := s.ListReports(ctx, 10)
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(ListReports) = %d", len(list))
	}
	if list[0].IssueType != "rpc_down" {
		t.Errorf("issue_type = %q", list[0].IssueType)
	}
}

func int64Ptr(n int64) *int64 { return &n }

func TestTrimChecks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	ctx := context.Background()
	s, err := New(ctx, path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = s.Close() }()
	for i := 0; i < 5; i++ {
		_ = s.InsertCheck(ctx, "rpc", "x", true, int64Ptr(50), "")
	}
	if err := s.TrimChecks(ctx, "rpc", "x", 2); err != nil {
		t.Fatalf("TrimChecks: %v", err)
	}
	list, _ := s.RecentChecks(ctx, "rpc", "x", 10)
	if len(list) != 2 {
		t.Errorf("after trim len = %d", len(list))
	}
}
