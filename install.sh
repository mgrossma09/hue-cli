#!/usr/bin/env bash
set -euo pipefail

REPO_OWNER="${REPO_OWNER:-mgrossma09}"
REPO_NAME="${REPO_NAME:-hue-cli}"
BIN_NAME="huectl"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-}"

log() {
  printf '%s\n' "$*"
}

detect_os() {
  case "$(uname -s)" in
    Darwin)
      printf 'darwin'
      ;;
    Linux)
      printf 'linux'
      ;;
    *)
      log "Unsupported OS: $(uname -s)"
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      printf 'amd64'
      ;;
    arm64|aarch64)
      printf 'arm64'
      ;;
    *)
      log "Unsupported architecture: $(uname -m)"
      exit 1
      ;;
  esac
}

resolve_tag() {
  if [ "$VERSION" = "latest" ]; then
    curl -fsSL "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
      | head -n1
    return
  fi

  case "$VERSION" in
    v*)
      printf '%s' "$VERSION"
      ;;
    *)
      printf 'v%s' "$VERSION"
      ;;
  esac
}

checksum_tool() {
  if command -v shasum >/dev/null 2>&1; then
    printf 'shasum'
    return
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    printf 'sha256sum'
    return
  fi
  printf 'none'
}

verify_checksum() {
  archive_path="$1"
  checksums_path="$2"
  archive_name="$3"

  tool="$(checksum_tool)"
  if [ "$tool" = "none" ]; then
    log "No checksum tool found (shasum/sha256sum); skipping checksum verification"
    return
  fi

  expected="$(grep " ${archive_name}$" "$checksums_path" | awk '{print $1}')"
  if [ -z "$expected" ]; then
    log "Warning: no checksum entry found for ${archive_name}; skipping verification"
    return
  fi

  if [ "$tool" = "shasum" ]; then
    actual="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
  else
    actual="$(sha256sum "$archive_path" | awk '{print $1}')"
  fi

  if [ "$expected" != "$actual" ]; then
    log "Checksum verification failed for ${archive_name}"
    exit 1
  fi
}

resolve_install_dir() {
  if [ -n "$INSTALL_DIR" ]; then
    printf '%s' "$INSTALL_DIR"
    return
  fi

  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    printf '/usr/local/bin'
    return
  fi

  printf '%s/.local/bin' "$HOME"
}

main() {
  os="$(detect_os)"
  arch="$(detect_arch)"

  tag="$(resolve_tag)"
  if [ -z "$tag" ]; then
    log "Failed to resolve release tag"
    exit 1
  fi
  version_no_v="${tag#v}"

  archive_name="${BIN_NAME}_${version_no_v}_${os}_${arch}.tar.gz"
  checksums_name="checksums.txt"

  if [ "$VERSION" = "latest" ]; then
    archive_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download/${archive_name}"
    checksums_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download/${checksums_name}"
  else
    archive_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}/${archive_name}"
    checksums_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}/${checksums_name}"
  fi

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT

  archive_path="${tmp_dir}/${archive_name}"
  checksums_path="${tmp_dir}/${checksums_name}"

  log "Downloading ${archive_url}"
  curl -fL --retry 3 --retry-delay 1 -o "$archive_path" "$archive_url"

  log "Downloading ${checksums_url}"
  curl -fL --retry 3 --retry-delay 1 -o "$checksums_path" "$checksums_url"

  verify_checksum "$archive_path" "$checksums_path" "$archive_name"

  tar -xzf "$archive_path" -C "$tmp_dir"

  resolved_install_dir="$(resolve_install_dir)"
  mkdir -p "$resolved_install_dir"

  install -m 0755 "${tmp_dir}/${BIN_NAME}" "${resolved_install_dir}/${BIN_NAME}"

  log "Installed ${BIN_NAME} to ${resolved_install_dir}/${BIN_NAME}"
  log "Run '${BIN_NAME} --help' to get started."

  case ":$PATH:" in
    *":${resolved_install_dir}:"*) ;;
    *)
      log "Note: ${resolved_install_dir} is not in your PATH"
      ;;
  esac
}

main "$@"
