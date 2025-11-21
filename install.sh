#!/usr/bin/env bash
set -euo pipefail

OS="$(uname -s)"
ARCH="$(uname -m)"

# Get latest version from GitHub Releases
echo "[1/4] Fetching latest version..."
LATEST_VERSION=$(curl -s https://api.github.com/repos/alpdogan/kubetbe/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "${LATEST_VERSION}" ]; then
  echo "Error: Could not fetch latest version"
  exit 1
fi

echo "Latest version: ${LATEST_VERSION}"

# Check current version if kubetbe is installed
INSTALL_DIR="${HOME}/.local/bin"
CURRENT_VERSION=""
if [ -f "${INSTALL_DIR}/kubetbe" ]; then
  CURRENT_VERSION=$("${INSTALL_DIR}/kubetbe" --version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' || echo "")
  if [ -n "${CURRENT_VERSION}" ]; then
    echo "Current version: ${CURRENT_VERSION}"
    if [ "${CURRENT_VERSION}" = "${LATEST_VERSION}" ]; then
      echo "Already up to date! No update needed."
      exit 0
    else
      echo "Updating from ${CURRENT_VERSION} to ${LATEST_VERSION}..."
    fi
  else
    echo "kubetbe is already installed, but version could not be determined. Updating..."
  fi
else
  echo "Installing kubetbe..."
fi

case "${OS}_${ARCH}" in
  Darwin_arm64) BIN_NAME="kubetbe-darwin-arm64" ;;
  Darwin_x86_64) BIN_NAME="kubetbe-darwin-amd64" ;;
  Linux_x86_64)  BIN_NAME="kubetbe-linux-amd64" ;;
  *) echo "Unsupported platform: ${OS} ${ARCH}"; exit 1 ;;
esac

BIN_URL="https://github.com/alpdogan/kubetbe/releases/download/${LATEST_VERSION}/${BIN_NAME}"
TMP="$(mktemp)"

echo "[2/4] Downloading ${BIN_URL}"
curl -fsSL "${BIN_URL}" -o "${TMP}"

echo "[3/4] Installing to ${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}"
chmod +x "${TMP}"
mv "${TMP}" "${INSTALL_DIR}/kubetbe"

echo "[4/4] Done. Make sure ${INSTALL_DIR} is in your PATH."
if [ -n "${CURRENT_VERSION}" ]; then
  echo "Updated kubetbe from ${CURRENT_VERSION} to ${LATEST_VERSION}"
else
  echo "Installed kubetbe version: ${LATEST_VERSION}"
fi