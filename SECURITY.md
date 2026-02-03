# Security

## Reporting a vulnerability

Please do not open a public issue for security vulnerabilities. Instead, report them to the maintainers (e.g. via the email in the repository or a private disclosure process). Include a clear description and steps to reproduce if possible.

## Security practices in this project

- **Secrets:** Bot tokens and API keys are never committed. Use environment variables or local config files (e.g. `configs/local.yaml`) that are listed in `.gitignore`.
- **Logs:** Bot tokens are masked in log output.
- **Input:** Report and API inputs are sanitized and length-limited.
- **Network:** The server binds to configurable hosts; pprof is disabled by default and, when enabled, bound to localhost only.

## GET /v1/reports

The reports listing endpoint is intended for admin use. It redacts sensitive fields (e.g. wallet, URL, tx_hash, user_agent) in the response. In a production deployment you should add authentication or restrict access to this endpoint.
