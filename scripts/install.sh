#!/usr/bin/env bash
# Pulse one-line installer
# Usage: curl -fsSL https://raw.githubusercontent.com/abdulqadirmsingi/pulse-cli/main/scripts/install.sh | bash
set -euo pipefail

REPO="abdulqadirmsingi/pulse-cli"
BINARY="pulse"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

CYAN='\033[0;36m'; GREEN='\033[0;32m'; RED='\033[0;31m'; BOLD='\033[1m'; R='\033[0m'

echo -e "${CYAN}${BOLD}"
cat <<'EOF'
  ██████╗ ██╗   ██╗██╗     ███████╗███████╗
  ██╔══██╗██║   ██║██║     ██╔════╝██╔════╝
  ██████╔╝██║   ██║██║     ███████╗█████╗
  ██╔═══╝ ██║   ██║██║     ╚════██║██╔══╝
  ██║     ╚██████╔╝███████╗███████║███████╗
  ╚═╝      ╚═════╝ ╚══════╝╚══════╝╚══════╝
EOF
echo -e "${R}"

# ── Detect OS + arch ──────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo -e "${RED}unsupported arch: $ARCH${R}"; exit 1 ;;
esac

# ── Fetch latest release tag ───────────────────────────────────────────
echo -e "  checking latest version..."
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo -e "${RED}  could not fetch latest version — check your connection${R}"
    exit 1
fi

# ── Download + extract ─────────────────────────────────────────────────
ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo -e "  downloading ${CYAN}pulse ${VERSION}${R} for ${OS}/${ARCH}..."
curl -fsSL "$URL" -o "/tmp/${ARCHIVE}"
tar -xzf "/tmp/${ARCHIVE}" -C /tmp "${BINARY}"
chmod +x "/tmp/${BINARY}"
rm -f "/tmp/${ARCHIVE}"

# ── Install to ~/.local/bin (no sudo needed) ───────────────────────────
mkdir -p "$INSTALL_DIR"
mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"

# ── Add to PATH if needed ──────────────────────────────────────────────
SHELL_RC=""
case "$SHELL" in
    */zsh)  SHELL_RC="$HOME/.zshrc" ;;
    */bash) SHELL_RC="$HOME/.bashrc" ;;
esac

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]] && [ -n "$SHELL_RC" ]; then
    echo "" >> "$SHELL_RC"
    echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$SHELL_RC"
    echo -e "  added ${CYAN}~/.local/bin${R} to PATH in ${SHELL_RC}"
fi

echo ""
echo -e "  ${GREEN}✓ pulse ${VERSION} installed!${R}"
echo ""
echo -e "  next steps:"
echo -e "    ${CYAN}source ${SHELL_RC:-~/.zshrc}${R}   # reload ur shell"
echo -e "    ${CYAN}pulse init${R}               # set up shell tracking"
echo -e "    ${CYAN}pulse stats${R}              # check ur dev pulse 📊"
echo ""
