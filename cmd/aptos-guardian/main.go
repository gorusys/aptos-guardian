package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gorusys/aptos-guardian/internal/api"
	"github.com/gorusys/aptos-guardian/internal/config"
	"github.com/gorusys/aptos-guardian/internal/discordbot"
	"github.com/gorusys/aptos-guardian/internal/incidents"
	"github.com/gorusys/aptos-guardian/internal/metrics"
	"github.com/gorusys/aptos-guardian/internal/monitor"
	"github.com/gorusys/aptos-guardian/internal/store"
	"github.com/gorusys/aptos-guardian/internal/util/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configPath := flag.String("config", "configs/example.yaml", "Path to YAML config")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		for k, v := range version.Info() {
			fmt.Printf("%s: %s\n", k, v)
		}
		os.Exit(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := run(ctx, *configPath); err != nil && ctx.Err() == nil {
		slog.Error("run failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if os.Getenv("LOG_LEVEL") == "debug" {
		slog.SetDefault(slog.Default().With("level", "debug"))
	}
	slog.Info("starting", "config", configPath, "interval", cfg.Interval.String())

	st, err := store.New(ctx, cfg.StorePath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	engine := incidents.NewEngine(st, cfg, nil)
	rpcNames := make([]string, 0, len(cfg.RPCProviders))
	rpcURLs := make(map[string]string)
	for _, p := range cfg.RPCProviders {
		rpcNames = append(rpcNames, p.Name)
		rpcURLs[p.Name] = p.URL
	}
	dappNames := make([]string, 0, len(cfg.Dapps))
	dappURLs := make(map[string]string)
	for _, d := range cfg.Dapps {
		dappNames = append(dappNames, d.Name)
		dappURLs[d.Name] = d.URL
	}

	var discordSession *discordgo.Session
	if cfg.Discord.Enabled && cfg.Discord.BotToken != "" {
		slog.Info("discord enabled", "app_id", cfg.Discord.ApplicationID, "token", discordbot.MaskToken(cfg.Discord.BotToken))
		s, err := discordgo.New("Bot " + cfg.Discord.BotToken)
		if err != nil {
			return err
		}
		discordSession = s
		alerter := discordbot.NewAlerter(s, cfg.Discord.AlertChannelID, cfg.Discord.Mention, nil)
		engine.OnIncidentOpen = func(ctx context.Context, inc *store.Incident) {
			_ = alerter.PostIncidentOpen(ctx, inc)
		}
		engine.OnIncidentClosed = func(ctx context.Context, inc *store.Incident) {
			_ = alerter.PostIncidentClosed(ctx, inc)
		}
	}

	runner := monitor.NewRunner(cfg, st, nil)
	runner.SetIncidentEngine(engine)
	go runner.Run(ctx)

	metrics.SetBuildInfo(version.Version, version.Commit, version.BuildDate)
	openList, _ := st.ListIncidents(ctx, store.IncidentStateOpen, 100)
	metrics.SetIncidentsOpen(float64(len(openList)))
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				list, _ := st.ListIncidents(ctx, store.IncidentStateOpen, 100)
				metrics.SetIncidentsOpen(float64(len(list)))
			}
		}
	}()

	handlers := &api.Handlers{
		Store:     st,
		Engine:    engine,
		RPCNames:  rpcNames,
		DappNames: dappNames,
		RPCURLs:   rpcURLs,
		DappURLs:  dappURLs,
	}
	webRoot := api.DefaultWebRoot()
	mux := api.Router(handlers, cfg.Server.MetricsPath, promhttp.Handler(), webRoot)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	slog.Info("listening", "addr", addr)

	if discordSession != nil {
		botCfg := &discordbot.BotConfig{
			ApplicationID:  cfg.Discord.ApplicationID,
			BotToken:       cfg.Discord.BotToken,
			GuildID:        cfg.Discord.GuildID,
			AlertChannelID: cfg.Discord.AlertChannelID,
			DMRefuseMsg:    cfg.Discord.DMRefuseMsg,
		}
		buildCtx := func(ctx context.Context) (*discordbot.CommandContext, error) {
			return discordbot.BuildCommandContext(ctx, st, engine, rpcNames, dappNames)
		}
		bot := discordbot.NewBotWithSession(discordSession, botCfg, buildCtx, nil)
		if err := bot.Open(); err != nil {
			slog.Warn("discord open", "err", err)
		} else {
			defer func() { _ = bot.Close() }()
			_ = bot.RegisterCommands(ctx)
		}
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
