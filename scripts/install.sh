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

# Fall back to ~/.local/bin if INSTALL_DIR is not writable
FALLBACK_DIR="$HOME/.local/bin"
NEEDS_PATH_REMINDER=false
if [ ! -w "$INSTALL_DIR" ]; then
    if [ ! -d "$INSTALL_DIR" ]; then
        # Directory doesn't exist — try to create it first
        if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            INSTALL_DIR="$FALLBACK_DIR"
            NEEDS_PATH_REMINDER=true
        fi
    else
        INSTALL_DIR="$FALLBACK_DIR"
        NEEDS_PATH_REMINDER=true
    fi
fi

mkdir -p "$INSTALL_DIR"
mv "/tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
echo "wt installed to $INSTALL_DIR/$BIN_NAME"

if $NEEDS_PATH_REMINDER; then
    echo ""
    echo "Note: $INSTALL_DIR is not on your PATH."
    echo "To use 'wt' from anywhere, add it to your shell config:"
    echo ""
    echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc  # for bash"
    echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc   # for zsh"
fi

echo ""
echo "Next steps:"
echo "  1. cd to your project and run: wt init"
echo "  2. Run: wt hooks"
echo "  3. Add shell integration to your .bashrc/.zshrc for 'wt switch'"
