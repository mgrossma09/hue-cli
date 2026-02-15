# huectl

`huectl` is a Go CLI for controlling Philips Hue lights via Hue API v2.

Phase 1 in this repository focuses on project scaffolding (CLI wiring, config loading, CI, linting, and tests). Hue API command behavior is intentionally stubbed and is planned for Phase 2.

## Goals

- Simple CLI interface for common light operations.
- Configuration via environment variables or local config file.
- No bridge discovery (host is provided explicitly).
- Safe secret handling (do not print full tokens by default).

## Prerequisites

- Go 1.22+
- A reachable Philips Hue bridge
- A Hue API v2 application key/token

Hue token setup guidance is available in Philips Hue developer documentation for local API access.

## Configuration

Configuration sources:

- Environment variables (take precedence)
- Config file (fallback)

Environment variables:

- `HUE_BRIDGE_HOST`
- `HUE_API_TOKEN`

Config file path:

- Default: `$XDG_CONFIG_HOME/huectl/config.json` (or platform equivalent from `os.UserConfigDir`)

JSON schema:

```json
{
  "bridge_host": "192.168.1.2",
  "api_token": "your-hue-v2-token"
}
```

A sample file is provided in `config.example.json`.

Redaction behavior:

- API tokens must not be logged in full by default.
- When token display is needed for debugging, use a redacted format.

## CLI Commands (planned in Phase 2)

```bash
huectl lights list
huectl lights toggle --id <id>
huectl lights set --id <id> [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]
```

Resource IDs are treated as strings. For `lights set`, only explicitly provided flags will be sent in requests.

## Development

Common tasks:

```bash
make test
make lint
make build
make fmt
```

Binary output:

- `./bin/huectl`

### pre-commit

Install on macOS:

```bash
brew install pre-commit
pre-commit install
```

If local hook installation is unavailable in your environment, rely on CI lint/test/build checks.

## CI

GitHub Actions workflow at `.github/workflows/ci.yml` runs on:

- Pushes to `main`
- Pull requests

CI runs:

- `make test`
- `make lint`
- `make build`
