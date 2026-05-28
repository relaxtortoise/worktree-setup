#!/bin/sh
set -e

REPO="relaxtortoise/worktree-setup"
BIN_NAME="wt"
INSTALL_DIR="${WT_INSTALL_DIR:-/usr/local/bin}"
VERSION="${WT_VERSION:-latest}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
esac

if [ "$VERSION" = "latest" ]; then
    URL="https://github.com/$REPO/releases/latest/download/${BIN_NAME}-${OS}-${ARCH}"
else
    URL="https://github.com/$REPO/releases/download/${VERSION}/${BIN_NAME}-${OS}-${ARCH}"
fi

echo "Downloading wt $VERSION for $OS/$ARCH..."
curl -fsSL "$URL" -o "/tmp/$BIN_NAME"
chmod +x "/tmp/$BIN_NAME"

if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR"
fi

mv "/tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
echo "wt installed to $INSTALL_DIR/$BIN_NAME"
echo ""
echo "Next steps:"
echo "  1. cd to your project and run: wt init"
echo "  2. Run: wt hooks"
echo "  3. Add shell integration to your .bashrc/.zshrc for 'wt switch'"
