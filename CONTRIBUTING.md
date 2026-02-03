# Contributing to aptos-guardian

Thank you for your interest in contributing.

## Development setup

1. Clone the repo and enter the directory.
2. Ensure Go 1.22+ (or the version in `go.mod`) is installed.
3. Run `go mod download` and `go build ./cmd/aptos-guardian`.
4. Copy `configs/example.yaml` to `configs/local.yaml` and adjust as needed.

## Before submitting

- Run `gofmt -w .`
- Run `go test ./...`
- Run `go vet ./...`
- Run `golangci-lint run ./...`
- Run `go build ./...`

## Pull requests

- Use conventional commits (e.g. `feat:`, `fix:`, `docs:`).
- Keep changes focused; prefer several small PRs over one large one.
- Ensure new code has tests where practical.

## Adding a monitored provider or dApp

- **RPC provider:** Add an entry under `rpc_providers` in the config with `name`, `url`, and optional `timeout_ms` and `tags`. Do not commit API keys; use env overrides or commented examples.
- **dApp:** Add an entry under `dapps` with `name`, `url`, and optional `timeout_ms` and `tags`.

## Code style

- Follow standard Go style and the existing patterns in the repo.
- Use `slog` for logging; avoid printing secrets (e.g. bot tokens).
