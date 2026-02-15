# PROMPT_PHASE4.md — Release & Installation (GitHub Releases + Homebrew Tap + install.sh)

You are an autonomous coding agent working in this repo. Implement Phase 4: make `huectl` installable via:
1) GitHub Releases with prebuilt binaries (macOS + Linux, amd64 + arm64)
2) Homebrew tap formula (generated/updated on release)
3) A simple `install.sh` that downloads the correct release asset

Follow repo rules in `AGENTS.md` and `.ai/AGENTS.md`:
- Use `jj` for VCS.
- Make local commits for meaningful milestones.
- Do NOT push to origin unless I explicitly ask.

## Assumptions / placeholders
- GitHub repo: `REPO_OWNER/REPO_NAME` (use the current repo’s remote URL to infer; if missing, use placeholders and document where to edit).
- Homebrew tap repo: `REPO_OWNER/homebrew-tap` (same owner; if it doesn't exist yet, still configure GoReleaser to target it and document that I must create it).

## Requirements

### A) GoReleaser config (GitHub release assets)
- Add `.goreleaser.yaml` (GoReleaser v2 syntax) to build and package `huectl`.
- Targets:
  - OS: `darwin`, `linux`
  - Arch: `amd64`, `arm64`
  - `CGO_ENABLED=0`
- Artifacts:
  - tar.gz archives containing `huectl` (+ README and LICENSE if present)
  - `checksums.txt`
- Ensure name templates are consistent and stable, e.g.:
  - `huectl_<version>_<os>_<arch>.tar.gz`
- Ensure `goreleaser check` would pass (don’t run release locally, but keep config valid).

### B) GitHub Actions: release pipeline
- Add `.github/workflows/release.yml` that:
  - triggers on pushing tags like `v*` (e.g. `v0.1.0`)
  - runs tests (`make test`) and lint (`make lint`) before release
  - runs GoReleaser with `--clean`
  - uses `GITHUB_TOKEN`
- Keep existing CI workflow intact (PR/push to main).

### C) Homebrew tap (formula automation)
- Configure GoReleaser `brews:` to publish a formula to a tap repo:
  - Tap: `REPO_OWNER/homebrew-tap`
  - Formula name: `huectl`
  - Install stanza installs the `huectl` binary
- If tap repo doesn’t exist, do NOT attempt to create it automatically.
  - Instead, document steps in README (“create tap repo, add permissions”) and in a short `RELEASING.md`.

### D) install.sh (curl-based installer)
- Add `install.sh` in repo root (or `scripts/install.sh`), plus docs.
- Behavior:
  - Detect OS and arch (macOS/Linux; amd64/arm64)
  - Download the correct asset from GitHub Releases:
    - Prefer `releases/latest/download/...` by default
    - Allow overriding version via env var, e.g. `VERSION=0.1.0`
  - Verify checksum against `checksums.txt` from the same release if feasible
    - If checksum verification isn’t feasible on macOS due to tooling differences, implement best-effort and document requirements.
  - Install destination:
    - default: `/usr/local/bin` if writable
    - fallback: `~/.local/bin`
    - allow override `INSTALL_DIR=...`
  - Print final “Installed huectl to …” and `huectl --help` hint.
- Add `.shellcheck` compliance where easy (avoid bashisms that break macOS’ bash 3.2 if possible; or require `bash` explicitly).

### E) Docs
Update `README.md` to include an “Install” section with:
- Homebrew:
  - `brew tap REPO_OWNER/tap`
  - `brew install huectl`
- GitHub Releases manual install: download tarball, extract, move binary
- install.sh:
  - Safe two-step (download script then run)
  - Optional one-liner variant
- Add `RELEASING.md`:
  - How to cut a release: `git tag vX.Y.Z` + push tag
  - What the workflow does
  - Homebrew tap prerequisites and common failures

### F) Keep quality gates
- Ensure after changes:
  - `make test` passes
  - `make lint` passes
  - `make build` passes

## Deliverables / Stop condition
- Implement all changes above.
- Make 2–4 logical `jj` commits (e.g., “chore: add GoReleaser”, “ci: add release workflow”, “docs: add install/releasing docs”, “chore: add install script”).
- Do not push to origin.

Start now.
