#!/usr/bin/env bash
set -euo pipefail

OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}_${ARCH}" in
  Darwin_arm64) BIN_URL="https://github.com/alpdogan/kubetbe/releases/download/v0.1.0/kubetbe-darwin-arm64" ;;
  Darwin_x86_64) BIN_URL="https://github.com/alpdogan/kubetbe/releases/download/v0.1.0/kubetbe-darwin-amd64" ;;
  Linux_x86_64)  BIN_URL="https://github.com/alpdogan/kubetbe/releases/download/v0.1.0/kubetbe-linux-amd64" ;;
  *) echo "Unsupported platform: ${OS} ${ARCH}"; exit 1 ;;
esac

INSTALL_DIR="${HOME}/.local/bin"
TMP="$(mktemp)"

echo "[1/3] Downloading ${BIN_URL}"
curl -fsSL "${BIN_URL}" -o "${TMP}"

echo "[2/3] Installing to ${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}"
chmod +x "${TMP}"
mv "${TMP}" "${INSTALL_DIR}/kubetbe"

echo "[3/3] Done. Make sure ${INSTALL_DIR} is in your PATH."