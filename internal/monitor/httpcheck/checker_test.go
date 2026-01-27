package httpcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChecker_Check_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	checker := NewChecker(server.URL, 0)
	res := checker.Check(context.Background())
	if !res.Success {
		t.Fatalf("expected success: %+v", res)
	}
	if res.Status != 200 {
		t.Errorf("status = %d", res.Status)
	}
	if res.LatencyMs < 0 {
		t.Errorf("latency_ms = %d", res.LatencyMs)
	}
}

func TestChecker_Check_3xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer server.Close()
	checker := NewChecker(server.URL, 0)
	res := checker.Check(context.Background())
	if !res.Success {
		t.Errorf("3xx should be success: %+v", res)
	}
}

func TestChecker_Check_4xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	checker := NewChecker(server.URL, 0)
	res := checker.Check(context.Background())
	if res.Success {
		t.Fatal("4xx should be failure")
	}
	if res.Status != 404 {
		t.Errorf("status = %d", res.Status)
	}
}
