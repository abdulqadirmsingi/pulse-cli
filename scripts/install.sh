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
    *) echo -e "${RED}  unsupported arch: $ARCH${R}"; exit 1 ;;
esac

# ── Detect shell config file ───────────────────────────────────────────
SHELL_RC=""
case "$SHELL" in
    */zsh)  SHELL_RC="$HOME/.zshrc" ;;
    */bash) SHELL_RC="$HOME/.bashrc" ;;
    */fish) SHELL_RC="$HOME/.config/fish/config.fish" ;;
    *)      SHELL_RC="$HOME/.bashrc" ;;
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
echo -e "  ${GREEN}✓${R}  binary installed to ${INSTALL_DIR}/${BINARY}"

# ── Add ~/.local/bin to PATH in shell config if missing ───────────────
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    mkdir -p "$(dirname "$SHELL_RC")"
    echo "" >> "$SHELL_RC"
    if [[ "$SHELL_RC" == *"config.fish" ]]; then
        echo "fish_add_path \$HOME/.local/bin" >> "$SHELL_RC"
    else
        echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$SHELL_RC"
    fi
    echo -e "  ${GREEN}✓${R}  added ~/.local/bin to PATH in ${SHELL_RC}"
    export PATH="$INSTALL_DIR:$PATH"
fi

# ── Run pulse init automatically ──────────────────────────────────────
echo ""
"${INSTALL_DIR}/${BINARY}" init

# ── Done ──────────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}${BOLD}  almost done!${R}"
echo ""
echo -e "  any ${BOLD}new terminal${R} you open will track automatically."
echo ""
echo -e "  to activate in ${BOLD}this${R} terminal right now, run:"
echo -e "  ${CYAN}source ${SHELL_RC}${R}"
echo ""
echo -e "  then try: ${CYAN}pulse stats${R} 📊"
echo ""
