#!/usr/bin/env bash
# IITJ LAN Auto Login — Linux/macOS bootstrap
# Usage: curl -fsSL <url>/bootstrap.sh | bash

set -e

REPO="https://github.com/xevrion/iitj-lan-autologin"
BINARY="iitj-login"
INSTALL_DIR="$HOME/.local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  armv7l)  ARCH="arm" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS — use bootstrap.ps1 on Windows"; exit 1 ;;
esac

echo "Detected: $OS/$ARCH"
mkdir -p "$INSTALL_DIR"

# Build from source if Go is available.
if command -v go >/dev/null 2>&1; then
  echo "Go found — building from source..."
  TMP=$(mktemp -d)
  trap 'rm -rf "$TMP"' EXIT
  git clone --depth 1 "$REPO" "$TMP/src"
  (cd "$TMP/src" && go build -o "$TMP/$BINARY" .)
  mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
  chmod +x "$INSTALL_DIR/$BINARY"
  echo "Installed to $INSTALL_DIR/$BINARY"
else
  # Download pre-built binary from GitHub Releases.
  TAG=$(curl -fsSL "https://api.github.com/repos/xevrion/iitj-lan-autologin/releases/latest" \
    2>/dev/null | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  if [ -z "$TAG" ]; then
    echo "No releases found. Install Go and re-run, or build manually:"
    echo "  git clone $REPO && cd iitj-lan-autologin && go build -o iitj-login ."
    exit 1
  fi
  URL="$REPO/releases/download/$TAG/iitj-login-$OS-$ARCH"
  echo "Downloading $URL..."
  curl -fsSL "$URL" -o "$INSTALL_DIR/$BINARY"
  chmod +x "$INSTALL_DIR/$BINARY"
  echo "Installed to $INSTALL_DIR/$BINARY"
fi

echo ""
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo "Add $INSTALL_DIR to your PATH:"
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
  echo ""
  echo "Then run: iitj-login install"
  echo "Or run directly: $INSTALL_DIR/$BINARY install"
else
  "$INSTALL_DIR/$BINARY" install
fi