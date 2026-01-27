package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChecker_Check_Success(t *testing.T) {
	v1Resp := map[string]interface{}{"chain_id": float64(1)}
	ledgerResp := map[string]interface{}{
		"ledger_version": float64(12345),
		"block_height":   float64(100),
		"ledger_info":    map[string]interface{}{"timestamp": "1234567890"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1" {
			_ = json.NewEncoder(w).Encode(v1Resp)
			return
		}
		if r.URL.Path == "/v1/ledger_info" {
			_ = json.NewEncoder(w).Encode(ledgerResp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker(server.URL, 0)
	ctx := context.Background()
	res := checker.Check(ctx)
	if !res.Success {
		t.Fatalf("expected success: %+v", res)
	}
	if res.LatencyMs < 0 {
		t.Errorf("latency_ms = %d", res.LatencyMs)
	}
	if res.ChainID != 1 {
		t.Errorf("chain_id = %d", res.ChainID)
	}
	if res.LedgerVersion != 12345 {
		t.Errorf("ledger_version = %d", res.LedgerVersion)
	}
	if res.BlockHeight != 100 {
		t.Errorf("block_height = %d", res.BlockHeight)
	}
}

func TestChecker_Check_HTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	checker := NewChecker(server.URL, 0)
	res := checker.Check(context.Background())
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.ErrorCategory != ErrorCategoryHTTPStatus {
		t.Errorf("error_category = %q", res.ErrorCategory)
	}
}

func TestChecker_Check_JSONDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1" {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()
	checker := NewChecker(server.URL, 0)
	res := checker.Check(context.Background())
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.ErrorCategory != ErrorCategoryJSONDecode {
		t.Errorf("error_category = %q", res.ErrorCategory)
	}
}
