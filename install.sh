#!/bin/sh
set -e

REPO="0xkowalskidev/gjq"
BINARY="gjq"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)      echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    *)              echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get latest release tag
TAG="$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)"
if [ -z "$TAG" ]; then
    echo "Failed to fetch latest release" >&2
    exit 1
fi
VERSION="${TAG#v}"

echo "Installing ${BINARY} ${TAG} (${OS}/${ARCH})..."

# Download archive and checksums
ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

curl -sSfL -o "${TMPDIR}/${ARCHIVE}" "$URL"
curl -sSfL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL"

# Verify checksum
cd "$TMPDIR"
if command -v sha256sum >/dev/null 2>&1; then
    grep "$ARCHIVE" checksums.txt | sha256sum -c --quiet
elif command -v shasum >/dev/null 2>&1; then
    grep "$ARCHIVE" checksums.txt | shasum -a 256 -c --quiet
else
    echo "Warning: no sha256sum or shasum found, skipping checksum verification" >&2
fi

# Extract binary
tar -xzf "$ARCHIVE" "$BINARY"

# Install
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
    echo "Installing to ${INSTALL_DIR} (no write access to /usr/local/bin)"
fi

mv "$BINARY" "${INSTALL_DIR}/${BINARY}"
echo "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"

case ":$PATH:" in
    *":${INSTALL_DIR}:"*)
        ;;
    *)
        SHELL_NAME="$(basename "$SHELL")"
        case "$SHELL_NAME" in
            zsh)  RC="${HOME}/.zshrc" ;;
            bash) RC="${HOME}/.bashrc" ;;
            *)    RC="${HOME}/.profile" ;;
        esac
        echo "export PATH=\"${INSTALL_DIR}:\$PATH\"" >> "$RC"
        echo "Added ${INSTALL_DIR} to PATH in ${RC} (restart your shell or run: source ${RC})"
        ;;
esac
