#!/usr/bin/env bash

set -e

REPO_URL="https://raw.githubusercontent.com/xevrion/iitj-lan-autologin/main"
INSTALL_SCRIPT="install.sh"

echo "Downloading installer..."

curl -fsSL "$REPO_URL/$INSTALL_SCRIPT" -o "$INSTALL_SCRIPT"

chmod +x "$INSTALL_SCRIPT"

echo "Launching installer..."
echo

exec bash "$INSTALL_SCRIPT"