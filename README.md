# aptos-guardian

**Aptos Infra & Wallet Reliability Guardian** — Discord bot, incident API, and public status page for RPC and dApp reliability.

## Why this exists

Aptos ecosystem reliability directly affects support load and institutional adoption. This project reduces support burden by:

- Detecting RPC and dApp outages early
- Providing instant actionable fixes (e.g. switch RPC to a healthy provider)
- Surfacing a single status view for operators and users
- Correlating user reports with incidents

## Components

- **Monitoring agent**: Checks Aptos RPC providers and dApp endpoints; tracks latency, success rate, and recommends best RPC.
- **Discord bot**: Slash commands for status, RPC health, dApp status, fix macros, and guided reports; optional alert posting.
- **Public API**: Health, status, incidents, and report submission.
- **Status page**: Minimal web dashboard (HTML/JS) showing provider health and incidents.

## Quick start

```bash
docker compose up -d
```

Then open the status page and API at the configured host/port (default `http://localhost:8080`).

## Local run

1. Copy `configs/example.yaml` to `configs/local.yaml` and adjust if needed.
2. Build and run:

   ```bash
   go build -o aptos-guardian ./cmd/aptos-guardian
   ./aptos-guardian -config configs/local.yaml
   ```

## Config

See `configs/example.yaml` for a runnable config. Override with environment variables where supported.

## Repo layout

- `cmd/aptos-guardian` — main entrypoint
- `internal/` — config, monitor, incidents, store, api, metrics, discordbot, macros, util
- `web/` — status page assets
- `configs/` — YAML configs
- `deploy/` — Prometheus and deployment configs

## License

MIT. See [LICENSE](LICENSE).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Code of conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Security: [SECURITY.md](SECURITY.md).
