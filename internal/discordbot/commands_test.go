package discordbot

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gorusys/aptos-guardian/internal/store"
)

func TestBuildStatusResponse(t *testing.T) {
	cc := &CommandContext{
		RecommendedRPC: "aptoslabs",
		RPCStatuses: []StatusProvider{
			{Name: "aptoslabs", Healthy: true, LatencyMs: 80},
			{Name: "other", Healthy: false, LastError: "timeout"},
		},
		DappStatuses: []DappStatus{
			{Name: "explorer", Healthy: true, LatencyMs: 120},
		},
		OpenIncidents: []store.Incident{
			{EntityName: "other", Severity: "CRIT", Summary: "RPC down"},
		},
	}
	ctx := context.Background()
	out := cc.BuildStatusResponse(ctx)
	if !strings.Contains(out, "aptoslabs") {
		t.Error("expected recommended RPC")
	}
	if !strings.Contains(out, "80") {
		t.Error("expected latency")
	}
	if !strings.Contains(out, "timeout") {
		t.Error("expected error")
	}
	if !strings.Contains(out, "Open incidents") {
		t.Error("expected incidents section")
	}
}

func TestBuildRPCResponse(t *testing.T) {
	cc := &CommandContext{
		RecommendedRPC: "aptoslabs",
		RPCStatuses: []StatusProvider{
			{Name: "aptoslabs", Healthy: true, LatencyMs: 50},
		},
	}
	out := cc.BuildRPCResponse(context.Background())
	if !strings.Contains(out, "Recommended") {
		t.Error("expected recommended")
	}
	if !strings.Contains(out, "50") {
		t.Error("expected latency")
	}
}

func TestBuildDappResponse(t *testing.T) {
	cc := &CommandContext{
		DappStatuses: []DappStatus{
			{Name: "aptos-explorer", Healthy: true, LatencyMs: 100},
		},
		DappNames: []string{"aptos-explorer"},
	}
	out := cc.BuildDappResponse(context.Background(), "aptos-explorer")
	if !strings.Contains(out, "aptos-explorer") {
		t.Error("expected dapp name")
	}
	if !strings.Contains(out, "100") {
		t.Error("expected latency")
	}
	outEmpty := cc.BuildDappResponse(context.Background(), "")
	if !strings.Contains(outEmpty, "Usage") {
		t.Error("expected usage for empty name")
	}
	outUnknown := cc.BuildDappResponse(context.Background(), "unknown")
	if !strings.Contains(outUnknown, "Unknown") {
		t.Error("expected unknown message")
	}
}

func TestBuildFixResponse(t *testing.T) {
	cc := &CommandContext{}
	out := cc.BuildFixResponse("gas")
	if !strings.Contains(out, "APT") {
		t.Error("gas fix should mention APT")
	}
	outScam := cc.BuildFixResponse("scam")
	if !strings.Contains(outScam, "never DM") {
		t.Error("scam fix should mention never DM")
	}
	outUnknown := cc.BuildFixResponse("xyz")
	if !strings.Contains(outUnknown, "Unknown") {
		t.Error("expected unknown topic message")
	}
}

func TestRunCommand(t *testing.T) {
	cc := &CommandContext{
		RPCStatuses:  []StatusProvider{{Name: "a", Healthy: true, LatencyMs: 10}},
		DappStatuses: []DappStatus{{Name: "d", Healthy: true}},
		RPCNames:     []string{"a"},
		DappNames:    []string{"d"},
	}
	ctx := context.Background()
	content, ep := RunCommand(ctx, "status", nil, cc)
	if content == "" {
		t.Error("status should return content")
	}
	if ep {
		t.Error("status should not be ephemeral")
	}
	content, ep = RunCommand(ctx, "fix", map[string]string{"topic": "scam"}, cc)
	if !strings.Contains(content, "Mods") {
		t.Error("fix scam should mention mods")
	}
	if ep {
		t.Error("fix should not be ephemeral")
	}
	_, ep = RunCommand(ctx, "report", nil, cc)
	if !ep {
		t.Error("report should be ephemeral")
	}
	content, _ = RunCommand(ctx, "unknown_cmd", nil, cc)
	if !strings.Contains(content, "Unknown") {
		t.Error("unknown command should return message")
	}
	content, _ = RunCommand(ctx, "status", nil, nil)
	if content != "Configuration error." {
		t.Error("nil context should return error")
	}
	_ = ep
	_ = content
}

func TestRunCommand_DappOption(t *testing.T) {
	cc := &CommandContext{
		DappStatuses: []DappStatus{{Name: "aptos-explorer", Healthy: true, LatencyMs: 50}},
		DappNames:    []string{"aptos-explorer"},
	}
	ctx := context.Background()
	content, _ := RunCommand(ctx, "dapp", map[string]string{"name": "aptos-explorer"}, cc)
	if !strings.Contains(content, "aptos-explorer") {
		t.Errorf("dapp response: %s", content)
	}
}

func TestOpenIncidentInDappResponse(t *testing.T) {
	cc := &CommandContext{
		DappStatuses: []DappStatus{{Name: "explorer", Healthy: false, LatencyMs: 0}},
		DappNames:    []string{"explorer"},
		OpenIncidents: []store.Incident{
			{EntityType: "dapp", EntityName: "explorer", Summary: "Outage", StartedAt: time.Now()},
		},
	}
	out := cc.BuildDappResponse(context.Background(), "explorer")
	if !strings.Contains(out, "Incident") {
		t.Error("expected incident in dapp response")
	}
	if !strings.Contains(out, "Outage") {
		t.Error("expected incident summary")
	}
}
