# huectl

`huectl` is a Go CLI for controlling Philips Hue lights via Hue API v2.

## Install

### Homebrew

```bash
brew tap mgrossma09/tap
brew install huectl
```

Note: the tap repository is `mgrossma09/homebrew-tap`.

### GitHub Releases (manual)

1. Download the archive for your platform from Releases.
2. Extract and move the binary into your PATH.

Example for macOS arm64 (`v0.1.0`):

```bash
curl -fL -o huectl.tar.gz https://github.com/mgrossma09/hue-cli/releases/download/v0.1.0/huectl_0.1.0_darwin_arm64.tar.gz
tar -xzf huectl.tar.gz
mv huectl /usr/local/bin/huectl
```

### install.sh

Safe two-step:

```bash
curl -fL -o install.sh https://raw.githubusercontent.com/mgrossma09/hue-cli/main/install.sh
bash install.sh
```

One-liner:

```bash
curl -fsSL https://raw.githubusercontent.com/mgrossma09/hue-cli/main/install.sh | bash
```

Optional installer environment variables:

- `VERSION=0.1.0` or `VERSION=v0.1.0` to install a specific release (default: latest)
- `INSTALL_DIR=/custom/bin` to override install destination
- `REPO_OWNER` / `REPO_NAME` to override GitHub source

Installer behavior:

- Detects macOS/Linux and amd64/arm64
- Downloads matching tarball + `checksums.txt`
- Verifies SHA256 using `shasum -a 256` or `sha256sum` when available
- Installs into `/usr/local/bin` if writable, otherwise `~/.local/bin`

## Features

- `huectl lights list [--json|--csv] [--with-group] [--with-state] [--wide] [--group <name>]`
- `huectl lights toggle (--id <id> | --group <group> [--name <name>])`
- `huectl lights set (--id <id> | --group <group> [--name <name>]) [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]`
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
- `HUE_INSECURE_TLS` (`true`/`false`, optional, defaults to `false`)

Default config file path:

- `$XDG_CONFIG_HOME/huectl/config.json` (or platform equivalent from `os.UserConfigDir`)

Config schema:

```json
{
  "bridge_host": "192.168.1.2",
  "api_token": "your-hue-v2-token",
  "insecure_tls": false
}
```

See `config.example.json`.

For Hue bridges using certificates that fail local verification, you can set:

```bash
export HUE_INSECURE_TLS=true
```

Use this only on trusted local networks.

## Usage Examples

List lights:

```bash
huectl lights list
```

List lights with group metadata (room/zone):

```bash
huectl lights list --with-group
```

List lights with additional state:

```bash
huectl lights list --with-state
```

List lights in wide mode (`--with-group --with-state`):

```bash
huectl lights list --wide
```

List lights as CSV with wide columns:

```bash
huectl lights list --csv --wide
```

List lights for a specific room/zone:

```bash
huectl lights list --group Kitchen
```

Toggle a light:

```bash
huectl lights toggle --id 01234567-89ab-cdef-0123-456789abcdef
```

Toggle all lights in a group:

```bash
huectl lights toggle --group Kitchen
```

Toggle one light by group/name:

```bash
huectl lights toggle --group Kitchen --name Island
```

Set a light on with brightness and color temperature:

```bash
huectl lights set --id 01234567-89ab-cdef-0123-456789abcdef --on --bri 60 --ct 250
```

Set all lights in a group:

```bash
huectl lights set --group Kitchen --off
```

Set one light by group/name:

```bash
huectl lights set --group Kitchen --name Island --on --bri 50
```

Set XY color only:

```bash
huectl lights set --id 01234567-89ab-cdef-0123-456789abcdef --xy 0.21,0.32
```

Notes:

- Resource IDs are treated as strings.
- `lights set` sends only fields corresponding to provided flags.
- For `lights toggle` and `lights set`, target selection is:
  - `--id <id>` OR
  - `--group <group> [--name <name>]`
- `--id` is mutually exclusive with `--group`/`--name`.
- If only `--group` is provided for toggle/set, the command applies to all lights in that group.
- `lights list --csv` column order:
  - default: `id,name,on`
  - `--with-group`: `id,name,group,group_type,on`
  - `--with-state`: `id,name,on,reachable,bri,color`
  - `--wide`: `id,name,group,group_type,on,reachable,bri,color`
- `color` is one of `yes`, `no`, or `unknown`.
- `--group <name>` filters list output by group/room name (case-insensitive exact match) and implies `--with-group`.
- If a light appears in multiple groups, `huectl` picks the first group deterministically (sorted by group type then ID).

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

GitHub Actions workflows:

- `.github/workflows/ci.yml` for push/PR checks
- `.github/workflows/release.yml` for tag (`v*`) releases

Release workflow runs:

- `make test`
- `make lint`
- `goreleaser check`
- `goreleaser release --clean`

See `RELEASING.md` for release instructions.
