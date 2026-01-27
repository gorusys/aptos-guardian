package httpcheck

import (
	"context"
	"io"
	"net/http"
	"time"
)

type Result struct {
	Success   bool
	LatencyMs int64
	Status    int
}

type Checker struct {
	URL        string
	HTTPClient *http.Client
}

func NewChecker(url string, timeout time.Duration) *Checker {
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	return &Checker{
		URL: url,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Checker) Check(ctx context.Context) Result {
	start := time.Now()
	res := Result{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		return res
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		res.LatencyMs = time.Since(start).Milliseconds()
		return res
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	res.LatencyMs = time.Since(start).Milliseconds()
	res.Status = resp.StatusCode
	res.Success = resp.StatusCode >= 200 && resp.StatusCode <= 399
	return res
}
