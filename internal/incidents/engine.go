package incidents

import (
	"context"
	"log/slog"

	"github.com/gorusys/aptos-guardian/internal/config"
	"github.com/gorusys/aptos-guardian/internal/store"
)

type Engine struct {
	store          *store.Store
	cfg            *config.Config
	log            *slog.Logger
	OnIncidentOpen   func(ctx context.Context, inc *store.Incident)
	OnIncidentClosed func(ctx context.Context, inc *store.Incident)
}

func NewEngine(st *store.Store, cfg *config.Config, log *slog.Logger) *Engine {
	if log == nil {
		log = slog.Default()
	}
	return &Engine{store: st, cfg: cfg, log: log}
}

func (e *Engine) ProcessRPCResult(ctx context.Context, name, url string, success bool, latencyMs int64) (opened, closed bool, err error) {
	checks, err := e.store.RecentChecks(ctx, "rpc", name, e.cfg.Thresholds.ConsecutiveFailuresForIncident+e.cfg.Thresholds.RecoveriesForClose+2)
	if err != nil {
		return false, false, err
	}
	openThreshold := e.cfg.Thresholds.ConsecutiveFailuresForIncident
	closeThreshold := e.cfg.Thresholds.RecoveriesForClose
	latWarn := e.cfg.Thresholds.LatencyWarnMS
	latCrit := e.cfg.Thresholds.LatencyCritMS

	hasOpen, openID, err := e.store.HasOpenIncident(ctx, "rpc", name)
	if err != nil {
		return false, false, err
	}

	if hasOpen {
		if success {
			consecutiveSuccess := countConsecutiveSuccess(checks, true)
			if consecutiveSuccess >= closeThreshold {
				summary := "RPC recovered after consecutive successes."
				if closeErr := e.store.CloseIncident(ctx, openID, summary); closeErr != nil {
					return false, false, closeErr
				}
				_ = e.store.AddIncidentUpdate(ctx, openID, summary)
				e.alertClosed(ctx, openID)
				e.log.Info("incident closed", "entity_type", "rpc", "entity_name", name, "incident_id", openID)
				return false, true, nil
			}
		}
		return false, false, nil
	}

	if !success {
		consecutiveFail := countConsecutiveSuccess(checks, false)
		if consecutiveFail >= openThreshold {
			summary := "RPC unreachable or failing (consecutive failures)."
			id, openErr := e.store.OpenIncident(ctx, "rpc", name, url, store.SeverityCrit, summary)
			if openErr != nil {
				return false, false, openErr
			}
			_ = e.store.AddIncidentUpdate(ctx, id, summary)
			e.alertOpen(ctx, id)
			e.log.Info("incident opened", "entity_type", "rpc", "entity_name", name, "incident_id", id, "severity", store.SeverityCrit)
			return true, false, nil
		}
		return false, false, nil
	}

	if latencyMs >= int64(latCrit) {
		summary := "RPC latency critical (above threshold)."
		id, openErr := e.store.OpenIncident(ctx, "rpc", name, url, store.SeverityCrit, summary)
		if openErr != nil {
			return false, false, openErr
		}
		_ = e.store.AddIncidentUpdate(ctx, id, summary)
		e.alertOpen(ctx, id)
		e.log.Info("incident opened", "entity_type", "rpc", "entity_name", name, "incident_id", id, "severity", store.SeverityCrit)
		return true, false, nil
	}
	if latencyMs >= int64(latWarn) {
		summary := "RPC latency elevated (warning)."
		id, openErr := e.store.OpenIncident(ctx, "rpc", name, url, store.SeverityWarn, summary)
		if openErr != nil {
			return false, false, openErr
		}
		_ = e.store.AddIncidentUpdate(ctx, id, summary)
		e.alertOpen(ctx, id)
		e.log.Info("incident opened", "entity_type", "rpc", "entity_name", name, "incident_id", id, "severity", store.SeverityWarn)
		return true, false, nil
	}
	return false, false, nil
}

func (e *Engine) ProcessDappResult(ctx context.Context, name, url string, success bool) (opened, closed bool, err error) {
	checks, err := e.store.RecentChecks(ctx, "dapp", name, e.cfg.Thresholds.ConsecutiveFailuresForIncident+e.cfg.Thresholds.RecoveriesForClose+2)
	if err != nil {
		return false, false, err
	}
	openThreshold := e.cfg.Thresholds.ConsecutiveFailuresForIncident
	closeThreshold := e.cfg.Thresholds.RecoveriesForClose

	hasOpen, openID, err := e.store.HasOpenIncident(ctx, "dapp", name)
	if err != nil {
		return false, false, err
	}

	if hasOpen {
		if success {
			consecutiveSuccess := countConsecutiveSuccess(checks, true)
			if consecutiveSuccess >= closeThreshold {
				summary := "Endpoint recovered."
				if closeErr := e.store.CloseIncident(ctx, openID, summary); closeErr != nil {
					return false, false, closeErr
				}
				_ = e.store.AddIncidentUpdate(ctx, openID, summary)
				e.alertClosed(ctx, openID)
				e.log.Info("incident closed", "entity_type", "dapp", "entity_name", name, "incident_id", openID)
				return false, true, nil
			}
		}
		return false, false, nil
	}

	if !success {
		consecutiveFail := countConsecutiveSuccess(checks, false)
		if consecutiveFail >= openThreshold {
			summary := "Endpoint unreachable or failing."
			id, openErr := e.store.OpenIncident(ctx, "dapp", name, url, store.SeverityCrit, summary)
			if openErr != nil {
				return false, false, openErr
			}
			_ = e.store.AddIncidentUpdate(ctx, id, summary)
			e.alertOpen(ctx, id)
			e.log.Info("incident opened", "entity_type", "dapp", "entity_name", name, "incident_id", id)
			return true, false, nil
		}
	}
	return false, false, nil
}

func (e *Engine) alertOpen(ctx context.Context, id int64) {
	if e.OnIncidentOpen == nil {
		return
	}
	inc, err := e.store.GetIncident(ctx, id)
	if err != nil {
		return
	}
	e.OnIncidentOpen(ctx, inc)
}

func (e *Engine) alertClosed(ctx context.Context, id int64) {
	if e.OnIncidentClosed == nil {
		return
	}
	inc, err := e.store.GetIncident(ctx, id)
	if err != nil {
		return
	}
	e.OnIncidentClosed(ctx, inc)
}

func countConsecutiveSuccess(checks []store.CheckRow, success bool) int {
	n := 0
	for i := range checks {
		if checks[i].Success == success {
			n++
		} else {
			break
		}
	}
	return n
}

func (e *Engine) RecommendedRPCProvider(ctx context.Context, providerNames []string, window int) string {
	if len(providerNames) == 0 {
		return ""
	}
	if window <= 0 {
		window = 50
	}
	type score struct {
		name         string
		successRate  float64
		avgLatencyMs float64
		successCount int
		totalCount   int
	}
	var scores []score
	for _, name := range providerNames {
		checks, err := e.store.RecentChecks(ctx, "rpc", name, window)
		if err != nil || len(checks) == 0 {
			scores = append(scores, score{name: name, successRate: 0, avgLatencyMs: 1e9})
			continue
		}
		var sumLat int64
		var successCount int
		for _, c := range checks {
			if c.Success {
				successCount++
				if c.LatencyMs.Valid {
					sumLat += c.LatencyMs.Int64
				}
			}
		}
		sr := float64(successCount) / float64(len(checks))
		avgLat := float64(1e9)
		if successCount > 0 {
			avgLat = float64(sumLat) / float64(successCount)
		}
		scores = append(scores, score{name: name, successRate: sr, avgLatencyMs: avgLat, successCount: successCount, totalCount: len(checks)})
	}
	best := ""
	bestScore := -1.0
	for _, s := range scores {
		combined := s.successRate * 1e6 / (1 + s.avgLatencyMs)
		if combined > bestScore {
			bestScore = combined
			best = s.name
		}
	}
	return best
}
