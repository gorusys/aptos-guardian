package discordbot

import (
	"context"
	"fmt"
	"strings"

	"github.com/gorusys/aptos-guardian/internal/incidents"
	"github.com/gorusys/aptos-guardian/internal/macros"
	"github.com/gorusys/aptos-guardian/internal/store"
)

type StatusProvider struct {
	Name      string
	Healthy   bool
	LatencyMs int64
	LastError string
}

type DappStatus struct {
	Name      string
	Healthy   bool
	LatencyMs int64
}

type CommandContext struct {
	Store          *store.Store
	Engine         *incidents.Engine
	RPCNames       []string
	DappNames      []string
	RecommendedRPC string
	RPCStatuses    []StatusProvider
	DappStatuses   []DappStatus
	OpenIncidents  []store.Incident
}

func (c *CommandContext) BuildStatusResponse(ctx context.Context) string {
	var b strings.Builder
	b.WriteString("**Aptos Guardian – Status**\n\n")
	if c.RecommendedRPC != "" {
		b.WriteString("**Recommended RPC:** " + c.RecommendedRPC + "\n\n")
	}
	b.WriteString("**RPC providers:**\n")
	for _, p := range c.RPCStatuses {
		status := "❌ Down"
		if p.Healthy {
			status = fmt.Sprintf("✅ %d ms", p.LatencyMs)
		} else if p.LastError != "" {
			status = "❌ " + p.LastError
		}
		b.WriteString(fmt.Sprintf("- %s: %s\n", p.Name, status))
	}
	b.WriteString("\n**dApps:**\n")
	for _, d := range c.DappStatuses {
		status := "❌ Down"
		if d.Healthy {
			status = fmt.Sprintf("✅ %d ms", d.LatencyMs)
		}
		b.WriteString(fmt.Sprintf("- %s: %s\n", d.Name, status))
	}
	if len(c.OpenIncidents) > 0 {
		b.WriteString("\n**Open incidents:**\n")
		for _, i := range c.OpenIncidents {
			b.WriteString(fmt.Sprintf("- [%s] %s: %s\n", i.Severity, i.EntityName, i.Summary))
		}
	}
	return b.String()
}

func (c *CommandContext) BuildRPCResponse(ctx context.Context) string {
	var b strings.Builder
	b.WriteString("**RPC health**\n\n")
	if c.RecommendedRPC != "" {
		b.WriteString("**Recommended:** " + c.RecommendedRPC + "\n\n")
	}
	for _, p := range c.RPCStatuses {
		status := "❌"
		if p.Healthy {
			status = fmt.Sprintf("✅ %d ms", p.LatencyMs)
		} else if p.LastError != "" {
			status = "❌ " + p.LastError
		}
		b.WriteString(fmt.Sprintf("- **%s:** %s\n", p.Name, status))
	}
	return b.String()
}

func (c *CommandContext) BuildDappResponse(ctx context.Context, dappName string) string {
	dappName = strings.TrimSpace(strings.ToLower(dappName))
	if dappName == "" {
		return "Usage: `/dapp <name>`. Example: `/dapp aptos-explorer`."
	}
	for _, d := range c.DappStatuses {
		if strings.ToLower(d.Name) == dappName {
			status := "❌ Down"
			if d.Healthy {
				status = fmt.Sprintf("✅ Up (%d ms)", d.LatencyMs)
			}
			msg := fmt.Sprintf("**%s:** %s\n", d.Name, status)
			for _, i := range c.OpenIncidents {
				if i.EntityType == "dapp" && strings.EqualFold(i.EntityName, d.Name) {
					msg += fmt.Sprintf("\n**Incident:** %s", i.Summary)
					break
				}
			}
			return msg
		}
	}
	return fmt.Sprintf("Unknown dApp: `%s`. Known: %s.", dappName, strings.Join(c.DappNames, ", "))
}

func (c *CommandContext) BuildFixResponse(topic string) string {
	return macros.FixContent(topic)
}

func (c *CommandContext) BuildReportAck() string {
	return "Thanks for your report. The team will look into it. For urgent issues, post in the support channel."
}

func RunCommand(ctx context.Context, cmd string, options map[string]string, cc *CommandContext) (content string, ephemeral bool) {
	if cc == nil {
		return "Configuration error.", true
	}
	switch cmd {
	case "status":
		return cc.BuildStatusResponse(ctx), false
	case "rpc":
		return cc.BuildRPCResponse(ctx), false
	case "dapp":
		return cc.BuildDappResponse(ctx, options["name"]), false
	case "fix":
		return cc.BuildFixResponse(options["topic"]), false
	case "report":
		return cc.BuildReportAck(), true
	default:
		return "Unknown command.", true
	}
}

func RecommendedRPCFromEngine(ctx context.Context, engine *incidents.Engine, names []string) string {
	if engine == nil || len(names) == 0 {
		return ""
	}
	return engine.RecommendedRPCProvider(ctx, names, 50)
}
