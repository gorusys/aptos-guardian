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

See `configs/example.yaml` for a runnable config. Key options:

- **interval** — How often to run RPC and dApp checks (e.g. `20s`).
- **server** — Host, port (default 8080), metrics path, optional pprof (off by default, localhost-only when on).
- **thresholds** — Latency warn/crit (ms), consecutive failures to open an incident, consecutive successes to close.
- **discord** — Set `enabled: true` and provide `application_id`, `bot_token`, `guild_id`, and optionally `alert_channel_id`, `mention`, `dm_refuse_msg`.
- **rpc_providers** / **dapps** — List of endpoints to monitor (name, url, timeout_ms, tags).

Override with env vars: `APTOS_GUARDIAN_SERVER_PORT`, `APTOS_GUARDIAN_DISCORD_BOT_TOKEN`, `APTOS_GUARDIAN_STORE_PATH`, etc.

## Commands

### Bot (Discord slash commands)

- **/status** — Overall summary, recommended RPC, provider and dApp status, open incidents.
- **/rpc** — RPC health table and recommendation.
- **/dapp &lt;name&gt;** — Endpoint status and last incident for that dApp.
- **/fix &lt;topic&gt;** — Quick fix macros: `gas`, `staking`, `switch_rpc`, `scam`.
- **/report** — Guided report (use in support channel); bot replies with an acknowledgment.

The bot refuses to handle DMs and directs users to the support channel (message is configurable).

### API

- **GET /healthz** — Liveness.
- **GET /v1/status** — Recommended RPC, provider and dApp status, open incidents.
- **GET /v1/incidents?state=open|closed&limit=50** — List incidents.
- **GET /v1/incidents/{id}** — Incident detail and updates.
- **POST /v1/report** — Submit a report (JSON: issue_type, wallet, device, region, description, url, tx_hash, user_agent).
- **GET /v1/reports?limit=50** — List reports (admin; sensitive fields redacted; see [SECURITY.md](SECURITY.md)).
- **GET /metrics** — Prometheus metrics.

## Incident model

- An **incident** is opened when an entity (RPC or dApp) reaches the configured consecutive failure count, or when RPC latency exceeds the warn/crit threshold.
- It is closed after the configured number of consecutive successful checks.
- Only one open incident per entity at a time (deduplication).
- Severity is CRIT for hard-down or latency above critical threshold, WARN for latency above warn threshold.
- The **recommended RPC** is derived from a rolling window of success rate and latency (best success rate, then lowest latency).

## Adding a monitored provider or dApp

- **RPC:** Add an entry under `rpc_providers` in your config with `name`, `url`, and optional `timeout_ms` (ms) and `tags`. Do not commit API keys; use env or a local file. Example with optional Alchemy: add a commented block and set the URL via env (e.g. `APTOS_GUARDIAN_ALCHEMY_RPC_URL`).
- **dApp:** Add an entry under `dapps` with `name`, `url`, and optional `timeout_ms` and `tags`.

## Deploy

- **Docker:** `docker compose up -d`. Data is stored in a volume; set `APTOS_GUARDIAN_STORE_PATH` if needed. Optional Prometheus: `docker compose --profile monitoring up -d`.
- **Binary:** Build with `go build -o aptos-guardian ./cmd/aptos-guardian`, then run with `-config /path/to/config.yaml`. Ensure the process can write to the store path and that the web root (e.g. `web/`) is next to the binary or set via config if supported.

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
