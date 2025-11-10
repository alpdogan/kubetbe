OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}_${ARCH}" in
  Darwin_arm64) BIN_URL="https://github.com/alpdogan/kubetbe/releases/download/v0.1.0/kubetbe-darwin-arm64" ;;
  Darwin_x86_64) BIN_URL="https://github.com/alpdogan/kubetbe/releases/download/v0.1.0/kubetbe-darwin-amd64" ;;
  Linux_x86_64) BIN_URL="https://github.com/alpdogan/kubetbe/releases/download/v0.1.0/kubetbe-linux-amd64" ;;
  *) echo "Unsupported platform: ${OS} ${ARCH}"; exit 1 ;;
esac