package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type durationMs time.Duration

func (d *durationMs) UnmarshalYAML(value *yaml.Node) error {
	var ms int
	if err := value.Decode(&ms); err != nil {
		return err
	}
	*d = durationMs(ms) * durationMs(time.Millisecond)
	return nil
}

func (d durationMs) Duration() time.Duration { return time.Duration(d) }

type Config struct {
	Interval     time.Duration  `yaml:"interval"`
	Server       ServerConfig   `yaml:"server"`
	Thresholds   Thresholds     `yaml:"thresholds"`
	Discord      DiscordConfig  `yaml:"discord"`
	RPCProviders []RPCProvider  `yaml:"rpc_providers"`
	Dapps        []DappEndpoint `yaml:"dapps"`
	StorePath    string         `yaml:"store_path"`
}

type ServerConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	MetricsPath string `yaml:"metrics_path"`
	EnablePprof bool   `yaml:"enable_pprof"`
	PprofBind   string `yaml:"pprof_bind"`
}

type Thresholds struct {
	LatencyWarnMS                  int `yaml:"latency_warn_ms"`
	LatencyCritMS                  int `yaml:"latency_crit_ms"`
	ConsecutiveFailuresForIncident int `yaml:"consecutive_failures_for_incident"`
	RecoveriesForClose             int `yaml:"recoveries_for_close"`
}

type DiscordConfig struct {
	Enabled        bool   `yaml:"enabled"`
	ApplicationID  string `yaml:"application_id"`
	BotToken       string `yaml:"bot_token"`
	GuildID        string `yaml:"guild_id"`
	AlertChannelID string `yaml:"alert_channel_id"`
	Mention        string `yaml:"mention"`
	DMRefuseMsg    string `yaml:"dm_refuse_msg"`
}

type RPCProvider struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Timeout durationMs        `yaml:"timeout_ms"`
	Tags    map[string]string `yaml:"tags"`
}

type DappEndpoint struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Timeout durationMs        `yaml:"timeout_ms"`
	Tags    map[string]string `yaml:"tags"`
}

func (r *RPCProvider) TimeoutMS() int {
	return int(r.Timeout.Duration() / time.Millisecond)
}

func (d *DappEndpoint) TimeoutMS() int {
	return int(d.Timeout.Duration() / time.Millisecond)
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	applyEnvOverrides(&c)
	if err := Validate(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func applyEnvOverrides(c *Config) {
	if v := os.Getenv("APTOS_GUARDIAN_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Interval = d
		}
	}
	if v := os.Getenv("APTOS_GUARDIAN_SERVER_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("APTOS_GUARDIAN_SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Server.Port = p
		}
	}
	if v := os.Getenv("APTOS_GUARDIAN_DISCORD_ENABLED"); v != "" {
		c.Discord.Enabled = v == "1" || v == "true" || v == "yes"
	}
	if v := os.Getenv("APTOS_GUARDIAN_DISCORD_BOT_TOKEN"); v != "" {
		c.Discord.BotToken = v
	}
	if v := os.Getenv("APTOS_GUARDIAN_DISCORD_APPLICATION_ID"); v != "" {
		c.Discord.ApplicationID = v
	}
	if v := os.Getenv("APTOS_GUARDIAN_DISCORD_GUILD_ID"); v != "" {
		c.Discord.GuildID = v
	}
	if v := os.Getenv("APTOS_GUARDIAN_DISCORD_ALERT_CHANNEL_ID"); v != "" {
		c.Discord.AlertChannelID = v
	}
	if v := os.Getenv("APTOS_GUARDIAN_STORE_PATH"); v != "" {
		c.StorePath = v
	}
}

func Validate(c *Config) error {
	if c.Interval <= 0 {
		c.Interval = 20 * time.Second
	}
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port <= 0 {
		c.Server.Port = 8080
	}
	if c.Server.MetricsPath == "" {
		c.Server.MetricsPath = "/metrics"
	}
	if c.Server.PprofBind == "" {
		c.Server.PprofBind = "127.0.0.1:6060"
	}
	if c.Thresholds.LatencyWarnMS <= 0 {
		c.Thresholds.LatencyWarnMS = 600
	}
	if c.Thresholds.LatencyCritMS <= 0 {
		c.Thresholds.LatencyCritMS = 1500
	}
	if c.Thresholds.ConsecutiveFailuresForIncident <= 0 {
		c.Thresholds.ConsecutiveFailuresForIncident = 3
	}
	if c.Thresholds.RecoveriesForClose <= 0 {
		c.Thresholds.RecoveriesForClose = 2
	}
	if c.Discord.DMRefuseMsg == "" {
		c.Discord.DMRefuseMsg = "Please post in the support channel so the team can help. Mods never DM first."
	}
	if c.StorePath == "" {
		c.StorePath = "data/guardian.db"
	}
	for i := range c.RPCProviders {
		r := &c.RPCProviders[i]
		if r.Name == "" {
			return fmt.Errorf("rpc_providers[%d]: name required", i)
		}
		if r.URL == "" {
			return fmt.Errorf("rpc_providers[%d]: url required", i)
		}
		if r.Timeout.Duration() <= 0 {
			r.Timeout = durationMs(4000) * durationMs(time.Millisecond)
		}
		if r.Tags == nil {
			r.Tags = make(map[string]string)
		}
	}
	for i := range c.Dapps {
		d := &c.Dapps[i]
		if d.Name == "" {
			return fmt.Errorf("dapps[%d]: name required", i)
		}
		if d.URL == "" {
			return fmt.Errorf("dapps[%d]: url required", i)
		}
		if d.Timeout.Duration() <= 0 {
			d.Timeout = durationMs(4000) * durationMs(time.Millisecond)
		}
		if d.Tags == nil {
			d.Tags = make(map[string]string)
		}
	}
	if c.Discord.Enabled {
		if c.Discord.BotToken == "" {
			return fmt.Errorf("discord.enabled is true but bot_token is empty")
		}
		if c.Discord.ApplicationID == "" {
			return fmt.Errorf("discord.enabled is true but application_id is empty")
		}
		if c.Discord.GuildID == "" {
			return fmt.Errorf("discord.enabled is true but guild_id is empty")
		}
	}
	return nil
}
