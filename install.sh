#!/bin/bash
set -euo pipefail

REPO="Nings-379/skillctl"
BINARY_NAME="skillctl"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux)  FILENAME="skillctl-linux-${ARCH}" ;;
    darwin) FILENAME="skillctl-darwin-${ARCH}" ;;
    *) echo "Error: Unsupported OS: $OS (use install.ps1 for Windows)"; exit 1 ;;
esac

echo "Installing skillctl..."
echo "  OS: ${OS}, Arch: ${ARCH}"

DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${FILENAME}"
echo "  Download: ${DOWNLOAD_URL}"
echo ""

TMPFILE=$(mktemp)
trap "rm -f $TMPFILE" EXIT

if command -v curl &>/dev/null; then
    curl -fSL -o "$TMPFILE" "$DOWNLOAD_URL" || { echo "Error: Download failed. Check if a release exists at: https://github.com/${REPO}/releases"; exit 1; }
elif command -v wget &>/dev/null; then
    wget -q -O "$TMPFILE" "$DOWNLOAD_URL" || { echo "Error: Download failed"; exit 1; }
else
    echo "Error: curl or wget is required"
    exit 1
fi

chmod +x "$TMPFILE"

if [ -w "$INSTALL_DIR" ]; then
    mv "$TMPFILE" "${INSTALL_DIR}/${BINARY_NAME}"
else
    echo "Need sudo to install to ${INSTALL_DIR}"
    sudo mv "$TMPFILE" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo ""
echo "skillctl installed successfully!"
echo "  Location: ${INSTALL_DIR}/${BINARY_NAME}"
echo ""
echo "Run 'skillctl --help' to get started."
