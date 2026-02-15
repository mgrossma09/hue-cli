# Agent Rules (Codex) — huectl

This repo is developed with **jj** (Jujutsu). Follow these rules strictly.

## Version control
- Use `jj` for all VCS actions: `jj status`, `jj diff`, `jj log`, `jj commit`.
- Make **local commits** at meaningful milestones with clear messages.
- **Do NOT push** to `origin` unless the user explicitly asks.
- Do not rewrite history unless asked.

## Workflow / closing the loop
- After meaningful changes, run:
  - `go test ./...`
  - `make lint` (once it exists)
- Keep diffs small and incremental; avoid giant “mega commits”.
- Prefer short, informative commit messages like:
  - `chore: scaffold project structure`
  - `ci: add GitHub Actions for tests`
  - `docs: add README and config format`

## Project constraints
- Target: **Go 1.22+**.
- Hue API: **v2** (not v1).
- No bridge discovery (no mDNS/SSDP).
- Config must come from:
  - env vars OR a local config file (env wins)
- Never print secrets by default.
- Never commit secrets. Add config to `.gitignore`. If you need an example, use `config.example.json`.

## Tooling
- Add `golangci-lint` with a minimal, high-signal config.
- Add `pre-commit` config. If installing hooks is problematic in the environment, document it and continue—CI linting is the primary enforcement.

## Communication
- When you stop for review, provide:
  - summary of files changed/created
  - what’s stubbed vs implemented
  - how to run tests/lint/build
  - any assumptions or TODOs
