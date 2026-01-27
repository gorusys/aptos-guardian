package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	ErrorCategoryTimeout           = "timeout"
	ErrorCategoryDNS               = "dns"
	ErrorCategoryTLS               = "tls"
	ErrorCategoryHTTPStatus        = "http_status"
	ErrorCategoryJSONDecode        = "json_decode"
	ErrorCategoryUnexpectedPayload = "unexpected_payload"
)

type Result struct {
	Success       bool
	LatencyMs     int64
	ErrorCategory string
	ChainID       int
	LedgerVersion uint64
	BlockHeight   uint64
	Timestamp     string
}

type Checker struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewChecker(baseURL string, timeout time.Duration) *Checker {
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	return &Checker{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
			},
		},
	}
}

func (c *Checker) Check(ctx context.Context) Result {
	start := time.Now()
	res := Result{}

	// GET /v1
	url1 := c.BaseURL + "/v1"
	req1, err := http.NewRequestWithContext(ctx, http.MethodGet, url1, nil)
	if err != nil {
		res.ErrorCategory = ErrorCategoryUnexpectedPayload
		res.LatencyMs = time.Since(start).Milliseconds()
		return res
	}
	resp1, err := c.HTTPClient.Do(req1)
	if err != nil {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.ErrorCategory = categorizeErr(err)
		return res
	}
	body1, err := io.ReadAll(resp1.Body)
	_ = resp1.Body.Close()
	if err != nil {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.ErrorCategory = ErrorCategoryUnexpectedPayload
		return res
	}
	if resp1.StatusCode < 200 || resp1.StatusCode >= 400 {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.ErrorCategory = ErrorCategoryHTTPStatus
		return res
	}
	var v1 map[string]interface{}
	if err := json.Unmarshal(body1, &v1); err != nil {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.ErrorCategory = ErrorCategoryJSONDecode
		return res
	}
	if chainID, ok := getNumber(v1, "chain_id"); ok {
		res.ChainID = int(chainID)
	}

	// GET /v1/ledger_info
	url2 := c.BaseURL + "/v1/ledger_info"
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, url2, nil)
	if err != nil {
		res.ErrorCategory = ErrorCategoryUnexpectedPayload
		res.LatencyMs = time.Since(start).Milliseconds()
		return res
	}
	resp2, err := c.HTTPClient.Do(req2)
	if err != nil {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.ErrorCategory = categorizeErr(err)
		return res
	}
	body2, err := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if err != nil {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.ErrorCategory = ErrorCategoryUnexpectedPayload
		return res
	}
	if resp2.StatusCode < 200 || resp2.StatusCode >= 400 {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.ErrorCategory = ErrorCategoryHTTPStatus
		return res
	}
	var ledger map[string]interface{}
	if err := json.Unmarshal(body2, &ledger); err != nil {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.ErrorCategory = ErrorCategoryJSONDecode
		return res
	}
	if v, ok := getNumber(ledger, "ledger_version"); ok {
		res.LedgerVersion = uint64(v)
	}
	if blockHeight, ok := getNumber(ledger, "block_height"); ok {
		res.BlockHeight = uint64(blockHeight)
	}
	if info, ok := ledger["ledger_info"].(map[string]interface{}); ok {
		if timestamp, ok := info["timestamp"].(string); ok {
			res.Timestamp = timestamp
		}
	}

	res.Success = true
	res.LatencyMs = time.Since(start).Milliseconds()
	return res
}

func getNumber(m map[string]interface{}, key string) (int64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	default:
		return 0, false
	}
}

func categorizeErr(err error) string {
	if err == nil {
		return ""
	}
	var netErr *net.DNSError
	if errors.As(err, &netErr) {
		return ErrorCategoryDNS
	}
	var netOpErr *net.OpError
	if errors.As(err, &netOpErr) {
		if netOpErr.Op == "dial" && strings.Contains(err.Error(), "TLS") {
			return ErrorCategoryTLS
		}
	}
	if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
		return ErrorCategoryTimeout
	}
	if strings.Contains(err.Error(), "tls") || strings.Contains(err.Error(), "certificate") {
		return ErrorCategoryTLS
	}
	return ErrorCategoryUnexpectedPayload
}

func (r *Result) ErrorSummary() string {
	if r.Success {
		return ""
	}
	if r.ErrorCategory != "" {
		return r.ErrorCategory
	}
	return "unknown"
}
