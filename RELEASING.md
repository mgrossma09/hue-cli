# Releasing huectl

This project uses GitHub Actions + GoReleaser to publish release artifacts and update a Homebrew tap formula.

## Prerequisites

- Push access to `mgrossma09/hue-cli`
- A Homebrew tap repository at `mgrossma09/homebrew-tap`
- GitHub Actions secrets set in `mgrossma09/hue-cli`:
  - `GITHUB_TOKEN` (automatic in GitHub Actions)
  - `HOMEBREW_TAP_GITHUB_TOKEN` (token with write access to `mgrossma09/homebrew-tap`)

## One-time Homebrew tap setup

1. Create repo `mgrossma09/homebrew-tap` if it does not exist.
2. Ensure default branch is `main`.
3. Ensure `HOMEBREW_TAP_GITHUB_TOKEN` can push to that repo.

## Cut a release

1. Ensure `main` is green in CI.
2. Create and push a version tag:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

3. GitHub Actions workflow `.github/workflows/release.yml` will:
  - run `make test`
  - run `make lint`
  - run `goreleaser check`
  - run `goreleaser release --clean`

## What gets published

- GitHub release artifacts for:
  - darwin/amd64
  - darwin/arm64
  - linux/amd64
  - linux/arm64
- Archive naming pattern:
  - `huectl_<version>_<os>_<arch>.tar.gz`
- `checksums.txt`
- Homebrew formula updates in `mgrossma09/homebrew-tap`.

## Common failures

- `HOMEBREW_TAP_GITHUB_TOKEN` missing/invalid:
  - Formula publish step fails.
- Tap repo missing:
  - Homebrew publish fails until `mgrossma09/homebrew-tap` exists.
- Lint/test failures:
  - Release job stops before publishing.
