# huectl

`huectl` is a Go CLI for controlling Philips Hue lights via Hue API v2.

## Features

- `huectl lights list`
- `huectl lights toggle --id <id>`
- `huectl lights set --id <id> [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]`
- Configuration from environment variables or local config file (environment wins)
- No bridge discovery; bridge host is explicitly provided
- Token-safe behavior (never prints full token)

## Prerequisites

- Go 1.22+
- Reachable Philips Hue bridge
- Hue API v2 application key/token

## Configuration

Configuration is loaded from:

- Environment variables (highest priority)
- Config file (fallback)

Environment variables:

- `HUE_BRIDGE_HOST`
- `HUE_API_TOKEN`

Default config file path:

- `$XDG_CONFIG_HOME/huectl/config.json` (or platform equivalent from `os.UserConfigDir`)

Config schema:

```json
{
  "bridge_host": "192.168.1.2",
  "api_token": "your-hue-v2-token"
}
```

See `config.example.json`.

## Usage Examples

List lights:

```bash
huectl lights list
```

Toggle a light:

```bash
huectl lights toggle --id 01234567-89ab-cdef-0123-456789abcdef
```

Set a light on with brightness and color temperature:

```bash
huectl lights set --id 01234567-89ab-cdef-0123-456789abcdef --on --bri 60 --ct 250
```

Set XY color only:

```bash
huectl lights set --id 01234567-89ab-cdef-0123-456789abcdef --xy 0.21,0.32
```

Notes:

- Resource IDs are treated as strings.
- `lights set` sends only fields corresponding to provided flags.

## Development

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

If hook installation is unavailable locally, CI still enforces test/lint/build.

## CI

GitHub Actions workflow at `.github/workflows/ci.yml` runs on pushes to `main` and pull requests with:

- `make test`
- `make lint`
- `make build`
