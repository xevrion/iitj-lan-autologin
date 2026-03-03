#!/usr/bin/env bash

set -e

REPO_URL="https://raw.githubusercontent.com/xevrion/iitj-lan-autologin/main"
INSTALL_SCRIPT="install.sh"

echo "Downloading installer..."

curl -fsSL "$REPO_URL/$INSTALL_SCRIPT" -o "$INSTALL_SCRIPT"
chmod +x "$INSTALL_SCRIPT"

echo
echo "Installer downloaded."
echo
echo "Run the installer with:"
echo
echo "  ./$INSTALL_SCRIPT"
echo