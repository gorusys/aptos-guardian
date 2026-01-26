package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Example(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "example.yaml")
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Interval.String() != "20s" {
		t.Errorf("interval = %v, want 20s", c.Interval)
	}
	if c.Server.Port != 8080 {
		t.Errorf("server.port = %d, want 8080", c.Server.Port)
	}
	if c.Server.MetricsPath != "/metrics" {
		t.Errorf("metrics_path = %q", c.Server.MetricsPath)
	}
	if c.Thresholds.LatencyWarnMS != 600 {
		t.Errorf("latency_warn_ms = %d", c.Thresholds.LatencyWarnMS)
	}
	if c.Thresholds.ConsecutiveFailuresForIncident != 3 {
		t.Errorf("consecutive_failures = %d", c.Thresholds.ConsecutiveFailuresForIncident)
	}
	if len(c.RPCProviders) < 1 {
		t.Fatal("expected at least one rpc provider")
	}
	if c.RPCProviders[0].Name != "aptoslabs" {
		t.Errorf("first rpc name = %q", c.RPCProviders[0].Name)
	}
	if c.RPCProviders[0].TimeoutMS() != 4000 {
		t.Errorf("timeout_ms = %d", c.RPCProviders[0].TimeoutMS())
	}
	if len(c.Dapps) < 2 {
		t.Fatal("expected at least two dapps")
	}
	if c.Discord.Enabled {
		t.Error("discord should be disabled in example")
	}
	if c.StorePath == "" {
		t.Error("store_path should be set")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValidate_Defaults(t *testing.T) {
	c := &Config{}
	if err := Validate(c); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if c.Interval.String() != "20s" {
		t.Errorf("default interval = %v", c.Interval)
	}
	if c.Server.Host != "0.0.0.0" {
		t.Errorf("default host = %q", c.Server.Host)
	}
	if c.Server.Port != 8080 {
		t.Errorf("default port = %d", c.Server.Port)
	}
	if c.Thresholds.LatencyWarnMS != 600 {
		t.Errorf("default latency_warn = %d", c.Thresholds.LatencyWarnMS)
	}
	if c.Discord.DMRefuseMsg == "" {
		t.Error("dm_refuse_msg should be set")
	}
}

func TestValidate_EmptyRPCName(t *testing.T) {
	c := &Config{
		RPCProviders: []RPCProvider{{Name: "", URL: "https://x.com"}},
	}
	if err := Validate(c); err == nil {
		t.Fatal("expected error for empty rpc name")
	}
}

func TestValidate_EmptyRPCURL(t *testing.T) {
	c := &Config{
		RPCProviders: []RPCProvider{{Name: "x", URL: ""}},
	}
	if err := Validate(c); err == nil {
		t.Fatal("expected error for empty rpc url")
	}
}

func TestValidate_DiscordEnabledNoToken(t *testing.T) {
	c := &Config{Discord: DiscordConfig{Enabled: true, ApplicationID: "1", GuildID: "2"}}
	if err := Validate(c); err == nil {
		t.Fatal("expected error when discord enabled but no token")
	}
}

func TestEnvOverrides(t *testing.T) {
	_ = os.Setenv("APTOS_GUARDIAN_SERVER_PORT", "9090")
	defer func() { _ = os.Unsetenv("APTOS_GUARDIAN_SERVER_PORT") }()
	_ = os.Setenv("APTOS_GUARDIAN_STORE_PATH", "/tmp/guardian.db")
	defer func() { _ = os.Unsetenv("APTOS_GUARDIAN_STORE_PATH") }()

	path := filepath.Join("..", "..", "configs", "example.yaml")
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Server.Port != 9090 {
		t.Errorf("port after env = %d, want 9090", c.Server.Port)
	}
	if c.StorePath != "/tmp/guardian.db" {
		t.Errorf("store_path after env = %q", c.StorePath)
	}
}
