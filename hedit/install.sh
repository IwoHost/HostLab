#!/usr/bin/env bash
set -e

BASE="https://iwohost.github.io/HostLab/hedit/dist"

# colors
G='\033[0;32m'; Y='\033[0;33m'; R='\033[0;31m'; B='\033[1m'; N='\033[0m'

echo ""
echo -e "${B}  hedit installer${N}"
echo -e "  ${G}nano, but it looks good${N}"
echo ""

# detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)  OS_KEY="linux"  ;;
  Darwin) OS_KEY="macos"  ;;
  *)
    echo -e "${R}  ✗ unsupported OS: $OS${N}"
    echo    "    try a manual download at https://iwohost.github.io/HostLab/pages/terminal.html"
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64)        ARCH_KEY="amd64" ;;
  arm64|aarch64) ARCH_KEY="arm64" ;;
  *)
    echo -e "${R}  ✗ unsupported arch: $ARCH${N}"
    exit 1
    ;;
esac

BINARY="hedit-${OS_KEY}-${ARCH_KEY}"
URL="${BASE}/${BINARY}"
TMP="$(mktemp)"

echo -e "  ${Y}→${N} detected ${OS_KEY}/${ARCH_KEY}"
echo -e "  ${Y}→${N} downloading ${BINARY}..."

if command -v curl &>/dev/null; then
  curl -fsSL "$URL" -o "$TMP"
elif command -v wget &>/dev/null; then
  wget -qO "$TMP" "$URL"
else
  echo -e "${R}  ✗ curl or wget required${N}"
  exit 1
fi

chmod +x "$TMP"

# pick install dir — prefer /usr/local/bin, fall back to ~/.local/bin
if [ -w /usr/local/bin ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -w /usr/bin ]; then
  INSTALL_DIR="/usr/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

mv "$TMP" "${INSTALL_DIR}/hedit"

echo -e "  ${G}✓${N} installed → ${INSTALL_DIR}/hedit"
echo ""

# PATH hint if using ~/.local/bin
if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
  echo -e "  ${Y}!${N} add this to your shell profile if hedit isn't found:"
  echo -e "      export PATH=\"\$HOME/.local/bin:\$PATH\""
  echo ""
fi

echo -e "  ${B}usage:${N}  hedit <file>"
echo -e "          hedit notes.txt"
echo -e "          hedit ~/.bashrc"
echo ""
echo -e "  ${G}^S${N} save  ${G}^Q${N} quit  ${G}^F${N} find  ${G}^T${N} themes"
echo ""
