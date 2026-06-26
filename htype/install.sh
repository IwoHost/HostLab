#!/usr/bin/env bash
set -euo pipefail

BASE="https://iwohost.github.io/HostLab/htype/dist"
BINARY="htype"

# ── detect OS ──
OS="$(uname -s)"
case "$OS" in
  Linux*)  OS_KEY="linux"  ;;
  Darwin*) OS_KEY="macos"  ;;
  *)
    echo "htype: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# ── detect arch ──
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)          ARCH_KEY="amd64" ;;
  arm64 | aarch64) ARCH_KEY="arm64" ;;
  *)
    echo "htype: unsupported arch: $ARCH" >&2
    exit 1
    ;;
esac

URL="${BASE}/${BINARY}-${OS_KEY}-${ARCH_KEY}"

# ── pick install dir ──
if   [ -w /usr/local/bin ]; then INSTALL_DIR=/usr/local/bin
elif [ -w /usr/bin        ]; then INSTALL_DIR=/usr/bin
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi
DEST="${INSTALL_DIR}/${BINARY}"

echo "→ downloading htype (${OS_KEY}/${ARCH_KEY})…"

if command -v curl &>/dev/null; then
  curl -fsSL "$URL" -o "$DEST"
elif command -v wget &>/dev/null; then
  wget -qO "$DEST" "$URL"
else
  echo "htype: curl or wget required" >&2
  exit 1
fi

chmod +x "$DEST"
echo "✓ installed to $DEST"

if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
  echo ""
  echo "  add to your shell profile:"
  echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
fi

echo ""
echo "  htype             # 25 words"
echo "  htype 50          # 50 words"
echo "  htype -t 60       # 60-second timed test"
echo "  htype -m code     # code identifiers"
echo "  htype -m quote    # famous quotes"
echo ""
echo "  Tab mode · T time · +/- words · ESC quit"
