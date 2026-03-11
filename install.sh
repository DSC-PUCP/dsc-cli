#!/bin/sh
set -e

REPO="DSC-PUCP/dsc-cli"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Error: could not fetch latest version"
  exit 1
fi

URL="https://github.com/$REPO/releases/download/v${LATEST}/dsc_${OS}_${ARCH}.tar.gz"

echo "Installing dsc v${LATEST} (${OS}/${ARCH})..."

TMP=$(mktemp -d)
curl -fsSL "$URL" -o "$TMP/dsc.tar.gz"
tar -xzf "$TMP/dsc.tar.gz" -C "$TMP"
install -m 755 "$TMP/dsc" "$INSTALL_DIR/dsc"
rm -rf "$TMP"

echo "dsc v${LATEST} installed to $INSTALL_DIR/dsc"
