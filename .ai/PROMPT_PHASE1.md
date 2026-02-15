You are an autonomous coding agent working in this repo. Build a Go CLI tool called `huectl` to control Philips Hue using **Hue API v2**.

Follow the repo rules in `AGENTS.md` and `.ai/AGENTS.md`:
- Use **jj** for version control.
- Make local commits at meaningful milestones.
- **Do not push** to origin unless I explicitly ask.

IMPORTANT: Work in TWO PHASES.
- Phase 1 (now): create the scaffold + CI + README + linting setup + test scaffolding, then STOP for my review/approval.
- Phase 2 (later, only after I approve): implement Hue v2 client + commands + tests until `go test ./...` is green.

## Requirements (Phase 1 scope: scaffold only)
### Language / build
- Go 1.22+ (go.mod).
- Provide a Makefile:
  - `make test` -> `go test ./...`
  - `make build` -> build `./cmd/huectl` to `./bin/huectl` (or similar)
  - `make fmt` -> gofmt (and goimports if included)
  - `make lint` -> golangci-lint run ./...

### CLI scope (MVP commands to be implemented in Phase 2, but scaffold now)
- `huectl lights list`
- `huectl lights toggle --id <id>`
- `huectl lights set --id <id> [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]`
Notes:
- Treat resource IDs as **strings**.
- Flags optional; only send fields provided.

### Config / secrets
- No bridge discovery.
- Bridge host/IP and Hue v2 token come from either:
  - env vars (preferred), OR
  - a local config file.
- Env vars must take precedence.
- Provide a `config.example.json`.
- Ensure real config is ignored via `.gitignore`.
- Never print the token by default; redact if shown.

Suggested env var names:
- `HUE_BRIDGE_HOST`
- `HUE_API_TOKEN`
(Also document config file path in README.)

### Repo structure (scaffold in Phase 1)
- `cmd/huectl/main.go`
- `internal/config` (load/validate config)
- `internal/hue` (Hue v2 API client/types; stubs only in Phase 1)
- `internal/cli` (command wiring; stubs only in Phase 1)
Keep it simple and idiomatic; avoid overengineering.

### Testing (“closing the loop”)
- Add tests that do not require a real Hue bridge or internet.
- Use `httptest.Server` for mocked Hue API interactions in Phase 2.
- In Phase 1:
  - Create placeholder tests and/or small unit tests for config parsing.
  - Ensure `go test ./...` passes (no failing tests).

### Linting / pre-commit
- Add `.golangci.yml` (minimal, high-signal linters).
- Add `.pre-commit-config.yaml` with Go formatting hooks and (optionally) golangci-lint.
- If pre-commit hook installation is problematic, document it, but do not block CI.

### GitHub Actions CI
Add `.github/workflows/ci.yml`:
- Trigger on:
  - push to `main`
  - any pull request
- Jobs run:
  - `make test`
  - `make lint`
  - `make build`
- Use setup-go (1.22+) and caching.

### Documentation
Create `README.md` with:
- Overview + goals
- Prereqs (Hue bridge reachable, token creation guidance)
- Configuration:
  - env vars
  - config file path + JSON schema
  - redaction behavior
- Examples for each command
- Dev workflow:
  - `make test`, `make lint`, `make build`
  - pre-commit install instructions (macOS brew + `pre-commit install`)
- CI summary

### Planning
Create `PLAN.md` describing Phase 2 implementation steps including:
- Hue API v2 endpoints you intend to use for list/get/update lights
- authentication approach (token header)
- how IDs are obtained and used
- test strategy with httptest

## Phase 1 STOP condition (must comply)
In Phase 1 you MUST:
- Create scaffolding files/folders.
- Add Makefile.
- Add GitHub Actions workflow.
- Add README.md.
- Add golangci-lint + pre-commit configs.
- Add config example + .gitignore rules.
- Add tests that pass (`go test ./...` must succeed).
- Add PLAN.md.

Then STOP and print a short summary:
- files created/changed
- what’s implemented vs stubbed
- any assumptions/decisions

Do NOT implement the Hue client or command logic beyond stubs/types in Phase 1.
Wait for my approval before proceeding to Phase 2.
Start now.
