package monitor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gorusys/aptos-guardian/internal/config"
	"github.com/gorusys/aptos-guardian/internal/monitor/httpcheck"
	"github.com/gorusys/aptos-guardian/internal/monitor/rpc"
	"github.com/gorusys/aptos-guardian/internal/store"
)

type IncidentProcessor interface {
	ProcessRPCResult(ctx context.Context, name, url string, success bool, latencyMs int64) (opened, closed bool, err error)
	ProcessDappResult(ctx context.Context, name, url string, success bool) (opened, closed bool, err error)
}

type Runner struct {
	cfg      *config.Config
	store    *store.Store
	engine   IncidentProcessor
	log      *slog.Logger
}

func NewRunner(cfg *config.Config, st *store.Store, log *slog.Logger) *Runner {
	if log == nil {
		log = slog.Default()
	}
	return &Runner{cfg: cfg, store: st, log: log}
}

func (r *Runner) SetIncidentEngine(engine IncidentProcessor) {
	r.engine = engine
}

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()
	r.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Runner) runOnce(ctx context.Context) {
	var wg sync.WaitGroup
	for _, p := range r.cfg.RPCProviders {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.checkRPC(ctx, &p)
		}()
	}
	for _, d := range r.cfg.Dapps {
		d := d
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.checkDapp(ctx, &d)
		}()
	}
	wg.Wait()
}

func (r *Runner) checkRPC(ctx context.Context, p *config.RPCProvider) {
	_, _ = r.store.EnsureProvider(ctx, p.Name, p.URL)
	checker := rpc.NewChecker(p.URL, p.Timeout.Duration())
	res := checker.Check(ctx)
	latPtr := (*int64)(nil)
	if res.Success {
		latPtr = &res.LatencyMs
	}
	errCat := res.ErrorCategory
	if err := r.store.InsertCheck(ctx, "rpc", p.Name, res.Success, latPtr, errCat); err != nil {
		r.log.Error("insert rpc check", "provider", p.Name, "err", err)
		return
	}
	if r.engine != nil {
		if _, _, err := r.engine.ProcessRPCResult(ctx, p.Name, p.URL, res.Success, res.LatencyMs); err != nil {
			r.log.Error("process rpc incident", "provider", p.Name, "err", err)
		}
	}
	r.log.Debug("rpc check", "provider", p.Name, "success", res.Success, "latency_ms", res.LatencyMs, "error", errCat)
}

func (r *Runner) checkDapp(ctx context.Context, d *config.DappEndpoint) {
	_, _ = r.store.EnsureDapp(ctx, d.Name, d.URL)
	checker := httpcheck.NewChecker(d.URL, d.Timeout.Duration())
	res := checker.Check(ctx)
	latPtr := (*int64)(nil)
	if res.Success {
		latPtr = &res.LatencyMs
	}
	errCat := ""
	if !res.Success && res.Status > 0 {
		errCat = "http_status"
	}
	if err := r.store.InsertCheck(ctx, "dapp", d.Name, res.Success, latPtr, errCat); err != nil {
		r.log.Error("insert dapp check", "dapp", d.Name, "err", err)
		return
	}
	if r.engine != nil {
		if _, _, err := r.engine.ProcessDappResult(ctx, d.Name, d.URL, res.Success); err != nil {
			r.log.Error("process dapp incident", "dapp", d.Name, "err", err)
		}
	}
	r.log.Debug("dapp check", "dapp", d.Name, "success", res.Success, "latency_ms", res.LatencyMs)
}
