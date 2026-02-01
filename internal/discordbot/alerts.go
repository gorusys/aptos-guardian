package discordbot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/gorusys/aptos-guardian/internal/store"
)

type Alerter struct {
	session       *discordgo.Session
	alertChannelID string
	mention        string
	log            *slog.Logger
}

func NewAlerter(session *discordgo.Session, alertChannelID, mention string, log *slog.Logger) *Alerter {
	if log == nil {
		log = slog.Default()
	}
	return &Alerter{session: session, alertChannelID: alertChannelID, mention: mention, log: log}
}

func (a *Alerter) PostIncidentOpen(ctx context.Context, inc *store.Incident) error {
	if a.alertChannelID == "" {
		return nil
	}
	prefix := ""
	if a.mention != "" {
		prefix = a.mention + " "
	}
	msg := prefix + fmt.Sprintf("**🚨 Incident opened**\n**%s** / %s\nSeverity: %s\n%s\nStarted: %s",
		inc.EntityType, inc.EntityName, inc.Severity, inc.Summary, inc.StartedAt.Format("2006-01-02 15:04:05 UTC"))
	_, err := a.session.ChannelMessageSend(a.alertChannelID, msg)
	if err != nil {
		a.log.Warn("alert post open", "err", err, "incident_id", inc.ID)
		return err
	}
	a.log.Info("alert posted", "type", "open", "entity", inc.EntityName)
	return nil
}

func (a *Alerter) PostIncidentClosed(ctx context.Context, inc *store.Incident) error {
	if a.alertChannelID == "" {
		return nil
	}
	ended := ""
	if inc.EndedAt != nil {
		ended = inc.EndedAt.Format("2006-01-02 15:04:05 UTC")
	}
	msg := fmt.Sprintf("**✅ Incident closed**\n**%s** / %s\n%s\nEnded: %s",
		inc.EntityType, inc.EntityName, inc.Summary, ended)
	_, err := a.session.ChannelMessageSend(a.alertChannelID, msg)
	if err != nil {
		a.log.Warn("alert post closed", "err", err, "incident_id", inc.ID)
		return err
	}
	a.log.Info("alert posted", "type", "closed", "entity", inc.EntityName)
	return nil
}

func (a *Alerter) PostRPCTransition(ctx context.Context, name, url string, healthy bool, latencyMs int64, errCat string) error {
	if a.alertChannelID == "" {
		return nil
	}
	status := "✅ Healthy"
	if !healthy {
		status = "❌ Unhealthy"
		if errCat != "" {
			status += " (" + errCat + ")"
		}
	} else {
		status += fmt.Sprintf(" (%d ms)", latencyMs)
	}
	msg := fmt.Sprintf("**RPC status change:** %s → %s\nURL: %s", name, status, url)
	_, err := a.session.ChannelMessageSend(a.alertChannelID, msg)
	if err != nil {
		a.log.Warn("alert post rpc transition", "err", err)
		return err
	}
	return nil
}

func (a *Alerter) PostDappTransition(ctx context.Context, name, url string, healthy bool) error {
	if a.alertChannelID == "" {
		return nil
	}
	status := "✅ Reachable"
	if !healthy {
		status = "❌ Unreachable"
	}
	msg := fmt.Sprintf("**dApp status change:** %s → %s\nURL: %s", name, status, url)
	_, err := a.session.ChannelMessageSend(a.alertChannelID, msg)
	if err != nil {
		a.log.Warn("alert post dapp transition", "err", err)
		return err
	}
	return nil
}
