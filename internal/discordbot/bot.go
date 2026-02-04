package discordbot

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/gorusys/aptos-guardian/internal/store"
)

const (
	cmdStatus = "status"
	cmdRPC    = "rpc"
	cmdDapp   = "dapp"
	cmdFix    = "fix"
	cmdReport = "report"
)

type CommandContextBuilder func(ctx context.Context) (*CommandContext, error)

type Bot struct {
	session *discordgo.Session
	cfg     *BotConfig
	build   CommandContextBuilder
	log     *slog.Logger
}

type BotConfig struct {
	ApplicationID  string
	BotToken       string
	GuildID        string
	AlertChannelID string
	DMRefuseMsg    string
}

func NewBot(cfg *BotConfig, build CommandContextBuilder, log *slog.Logger) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}
	return NewBotWithSession(s, cfg, build, log), nil
}

func NewBotWithSession(s *discordgo.Session, cfg *BotConfig, build CommandContextBuilder, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	b := &Bot{session: s, cfg: cfg, build: build, log: log}
	s.AddHandler(b.handleInteraction)
	return b
}

func (b *Bot) Open() error {
	return b.session.Open()
}

func (b *Bot) Close() error {
	return b.session.Close()
}

func (b *Bot) RegisterCommands(ctx context.Context) error {
	appID := b.cfg.ApplicationID
	if appID == "" {
		appID = b.session.State.User.ID
	}
	guildID := b.cfg.GuildID
	cmds := []*discordgo.ApplicationCommand{
		{Name: cmdStatus, Description: "Overall status and recommended RPC"},
		{Name: cmdRPC, Description: "RPC health table and recommendation"},
		{Name: cmdDapp, Description: "dApp endpoint status", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "dApp name", Required: true},
		}},
		{Name: cmdFix, Description: "Quick fix macros", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "topic", Description: "gas, staking, switch_rpc, scam", Required: true},
		}},
		{Name: cmdReport, Description: "Submit a guided report (use in support channel)"},
	}
	for _, c := range cmds {
		_, err := b.session.ApplicationCommandCreate(appID, guildID, c)
		if err != nil {
			b.log.Warn("register command", "cmd", c.Name, "err", err)
		}
	}
	return nil
}

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	data := i.ApplicationCommandData()
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		b.respondErr(s, i, "Could not resolve channel.")
		return
	}
	if channel.Type == discordgo.ChannelTypeDM {
		msg := b.cfg.DMRefuseMsg
		if msg == "" {
			msg = "Please post in the support channel so the team can help."
		}
		b.respondEphemeral(s, i, msg)
		return
	}
	opts := map[string]string{}
	for _, o := range data.Options {
		opts[o.Name] = o.StringValue()
	}
	cmdCtx, err := b.build(context.Background())
	if err != nil {
		b.respondErr(s, i, "Failed to build context.")
		return
	}
	content, ephemeral := RunCommand(context.Background(), data.Name, opts, cmdCtx)
	if ephemeral {
		b.respondEphemeral(s, i, content)
	} else {
		b.respond(s, i, content)
	}
}

func (b *Bot) respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if len(content) > 2000 {
		content = content[:1997] + "..."
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content},
	})
}

func (b *Bot) respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if len(content) > 2000 {
		content = content[:1997] + "..."
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (b *Bot) respondErr(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	b.respondEphemeral(s, i, content)
}

func MaskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func BuildCommandContext(ctx context.Context, st *store.Store, engine interface {
	RecommendedRPCProvider(ctx context.Context, names []string, window int) string
}, rpcNames, dappNames []string) (*CommandContext, error) {
	cc := &CommandContext{RPCNames: rpcNames, DappNames: dappNames}
	if engine != nil && len(rpcNames) > 0 {
		cc.RecommendedRPC = engine.RecommendedRPCProvider(ctx, rpcNames, 50)
	}
	for _, name := range rpcNames {
		checks, _ := st.RecentChecks(ctx, "rpc", name, 1)
		ps := StatusProvider{Name: name}
		if len(checks) > 0 {
			c := checks[0]
			ps.Healthy = c.Success
			if c.LatencyMs.Valid {
				ps.LatencyMs = c.LatencyMs.Int64
			}
			if c.ErrorCategory.Valid {
				ps.LastError = c.ErrorCategory.String
			}
		}
		cc.RPCStatuses = append(cc.RPCStatuses, ps)
	}
	for _, name := range dappNames {
		checks, _ := st.RecentChecks(ctx, "dapp", name, 1)
		ds := DappStatus{Name: name}
		if len(checks) > 0 {
			c := checks[0]
			ds.Healthy = c.Success
			if c.LatencyMs.Valid {
				ds.LatencyMs = c.LatencyMs.Int64
			}
		}
		cc.DappStatuses = append(cc.DappStatuses, ds)
	}
	openList, _ := st.ListIncidents(ctx, store.IncidentStateOpen, 20)
	cc.OpenIncidents = openList
	return cc, nil
}

func ParseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) map[string]string {
	m := make(map[string]string)
	for _, o := range options {
		m[o.Name] = o.StringValue()
	}
	return m
}
