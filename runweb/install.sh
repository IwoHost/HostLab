#!/usr/bin/env bash
set -e

URL="https://iwohost.github.io/HostLab/runweb/runweb.py"

# colors
G='\033[0;32m'; Y='\033[0;33m'; R='\033[0;31m'; B='\033[1m'; N='\033[0m'

echo ""
echo -e "${B}  runweb installer${N}"
echo -e "  ${G}serve any page on your LAN${N}"
echo ""

# runweb is a python script — needs python3 on PATH
if command -v python3 &>/dev/null; then
  PY="python3"
elif command -v python &>/dev/null && python -c 'import sys; sys.exit(0 if sys.version_info[0] == 3 else 1)'; then
  PY="python"
else
  echo -e "${R}  ✗ python 3 required but not found${N}"
  echo    "    install it from https://www.python.org/downloads/ and run this again"
  exit 1
fi

PYVER="$($PY -c 'import sys; print("%d.%d" % sys.version_info[:2])')"
echo -e "  ${Y}→${N} found python ${PYVER} ($(command -v $PY))"
echo -e "  ${Y}→${N} downloading runweb..."

TMP="$(mktemp)"
if command -v curl &>/dev/null; then
  curl -fsSL "$URL" -o "$TMP"
elif command -v wget &>/dev/null; then
  wget -qO "$TMP" "$URL"
else
  echo -e "${R}  ✗ curl or wget required${N}"
  rm -f "$TMP"
  exit 1
fi

# sanity check: did we actually get the script, not a 404 page?
if ! head -1 "$TMP" | grep -q python; then
  echo -e "${R}  ✗ download looks wrong — got something that isn't runweb${N}"
  rm -f "$TMP"
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

mv "$TMP" "${INSTALL_DIR}/runweb"

echo -e "  ${G}✓${N} installed → ${INSTALL_DIR}/runweb"
echo ""

# PATH hint if using ~/.local/bin
if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
  echo -e "  ${Y}!${N} add this to your shell profile if runweb isn't found:"
  echo -e "      export PATH=\"\$HOME/.local/bin:\$PATH\""
  echo ""
fi

echo -e "  ${B}ready — cd into a folder and type:${N}"
echo ""
echo -e "    ${G}runweb index.html -o${N}"
echo ""
echo -e "  ${G}-p${N} port  ${G}-o${N} open browser  ${G}-q${N} QR code  ${G}-s${N} https"
echo ""
