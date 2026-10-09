#!/bin/sh
#
# spring-box — installer for Linux and macOS.
#
# Downloads the prebuilt binary from GitHub Releases and puts it on your PATH.
#
# Usage (latest release):
#   curl -fsSL https://raw.githubusercontent.com/GzaJai/spring-box/main/install.sh | sh
#
# Usage (specific version):
#   curl -fsSL https://raw.githubusercontent.com/GzaJai/spring-box/main/install.sh | sh -s v0.1.0
#
# On Windows: download the .zip from Releases or use "go install".

set -eu

REPO="GzaJai/spring-box"
BINARY="spring-box"
BIN_DIR="${SPRINGBOX_BIN_DIR:-/usr/local/bin}"

info()  { printf '%s\n' "$*"; }
error() { printf '✗ %s\n' "$*" >&2; exit 1; }

# 1) Operating system
os="$(uname -s)"
case "$os" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *) error "Unsupported OS: $os. On Windows use the .zip from Releases or 'go install'." ;;
esac

# 2) Architecture
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64)  arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) error "Unsupported architecture: $arch." ;;
esac

# 3) Version: the one passed as argument, or the latest release
tag="${1:-}"
if [ -z "$tag" ]; then
  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')"
  [ -n "$tag" ] || error "Could not resolve the latest version."
fi
ver="${tag#v}"

# 4) Download the platform asset
asset="${BINARY}_${ver}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${tag}/${asset}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

info "Downloading spring-box ${tag} (${os}/${arch})..."
curl -fsSL "$url" -o "$tmp/$asset" || error "Could not download $url"
tar -xzf "$tmp/$asset" -C "$tmp"

# 5) Install (falls back to ~/.local/bin if /usr/local/bin is not writable)
if [ ! -w "$BIN_DIR" ]; then
  BIN_DIR="${HOME}/.local/bin"
  mkdir -p "$BIN_DIR"
fi
mv "$tmp/$BINARY" "$BIN_DIR/$BINARY"
chmod +x "$BIN_DIR/$BINARY"

info "✔ spring-box ${tag} installed to ${BIN_DIR}/${BINARY}"
case ":${PATH}:" in
  *":${BIN_DIR}:"*) ;;
  *)
    info "  Add ${BIN_DIR} to your PATH:"
    info "    export PATH=\"${BIN_DIR}:\$PATH\""
    ;;
esac
