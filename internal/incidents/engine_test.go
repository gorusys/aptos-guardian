package incidents

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gorusys/aptos-guardian/internal/config"
	"github.com/gorusys/aptos-guardian/internal/store"
)

func mustLoadConfig(t *testing.T) *config.Config {
	t.Helper()
	path := filepath.Join("..", "..", "configs", "example.yaml")
	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return c
}

func TestEngine_ProcessRPCResult_OpenOnConsecutiveFailures(t *testing.T) {
	ctx := context.Background()
	st, err := store.New(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = st.Close() }()
	cfg := mustLoadConfig(t)
	cfg.Thresholds.ConsecutiveFailuresForIncident = 3
	cfg.Thresholds.RecoveriesForClose = 2
	eng := NewEngine(st, cfg, nil)

	_, _ = st.EnsureProvider(ctx, "aptoslabs", "https://fullnode.mainnet.aptoslabs.com/v1")
	for i := 0; i < 3; i++ {
		_ = st.InsertCheck(ctx, "rpc", "aptoslabs", false, nil, "timeout")
	}
	opened, closed, err := eng.ProcessRPCResult(ctx, "aptoslabs", "https://fullnode.mainnet.aptoslabs.com/v1", false, 0)
	if err != nil {
		t.Fatalf("ProcessRPCResult: %v", err)
	}
	if !opened {
		t.Error("expected incident to open")
	}
	if closed {
		t.Error("expected not closed")
	}
	hasOpen, id, _ := st.HasOpenIncident(ctx, "rpc", "aptoslabs")
	if !hasOpen || id <= 0 {
		t.Errorf("expected open incident: hasOpen=%v id=%d", hasOpen, id)
	}
}

func TestEngine_ProcessRPCResult_DedupeNoDoubleOpen(t *testing.T) {
	ctx := context.Background()
	st, err := store.New(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = st.Close() }()
	cfg := mustLoadConfig(t)
	cfg.Thresholds.ConsecutiveFailuresForIncident = 2
	eng := NewEngine(st, cfg, nil)

	_, _ = st.EnsureProvider(ctx, "x", "https://x.com")
	_ = st.InsertCheck(ctx, "rpc", "x", false, nil, "timeout")
	_ = st.InsertCheck(ctx, "rpc", "x", false, nil, "timeout")
	opened1, _, _ := eng.ProcessRPCResult(ctx, "x", "https://x.com", false, 0)
	if !opened1 {
		t.Fatal("expected first open")
	}
	_ = st.InsertCheck(ctx, "rpc", "x", false, nil, "timeout")
	opened2, _, _ := eng.ProcessRPCResult(ctx, "x", "https://x.com", false, 0)
	if opened2 {
		t.Error("should not open second incident (dedupe)")
	}
	list, _ := st.ListIncidents(ctx, "open", 10)
	if len(list) != 1 {
		t.Errorf("expected 1 open incident, got %d", len(list))
	}
}

func TestEngine_ProcessRPCResult_CloseOnRecoveries(t *testing.T) {
	ctx := context.Background()
	st, err := store.New(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = st.Close() }()
	cfg := mustLoadConfig(t)
	cfg.Thresholds.ConsecutiveFailuresForIncident = 2
	cfg.Thresholds.RecoveriesForClose = 2
	eng := NewEngine(st, cfg, nil)

	_, _ = st.EnsureProvider(ctx, "y", "https://y.com")
	_ = st.InsertCheck(ctx, "rpc", "y", false, nil, "timeout")
	_ = st.InsertCheck(ctx, "rpc", "y", false, nil, "timeout")
	opened, _, _ := eng.ProcessRPCResult(ctx, "y", "https://y.com", false, 0)
	if !opened {
		t.Fatal("expected open")
	}
	_ = st.InsertCheck(ctx, "rpc", "y", true, int64Ptr(50), "")
	_, closed1, _ := eng.ProcessRPCResult(ctx, "y", "https://y.com", true, 50)
	if closed1 {
		t.Error("one success should not close yet")
	}
	_ = st.InsertCheck(ctx, "rpc", "y", true, int64Ptr(50), "")
	_, closed2, _ := eng.ProcessRPCResult(ctx, "y", "https://y.com", true, 50)
	if !closed2 {
		t.Error("two consecutive successes should close")
	}
	hasOpen, _, _ := st.HasOpenIncident(ctx, "rpc", "y")
	if hasOpen {
		t.Error("incident should be closed")
	}
}

func TestEngine_RecommendedRPCProvider(t *testing.T) {
	ctx := context.Background()
	st, err := store.New(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = st.Close() }()
	cfg := mustLoadConfig(t)
	eng := NewEngine(st, cfg, nil)

	names := []string{"a", "b", "c"}
	_, _ = st.EnsureProvider(ctx, "a", "https://a.com")
	_, _ = st.EnsureProvider(ctx, "b", "https://b.com")
	_, _ = st.EnsureProvider(ctx, "c", "https://c.com")
	for i := 0; i < 5; i++ {
		_ = st.InsertCheck(ctx, "rpc", "a", true, int64Ptr(100), "")
	}
	for i := 0; i < 5; i++ {
		_ = st.InsertCheck(ctx, "rpc", "b", true, int64Ptr(50), "")
	}
	for i := 0; i < 5; i++ {
		_ = st.InsertCheck(ctx, "rpc", "c", false, nil, "timeout")
	}
	best := eng.RecommendedRPCProvider(ctx, names, 10)
	if best != "b" {
		t.Errorf("recommended = %q, want b (best success + lowest latency)", best)
	}
}

func int64Ptr(n int64) *int64 { return &n }
