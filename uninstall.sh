#!/bin/bash
#
# SafeShell Uninstaller
# Usage: curl -fsSL https://raw.githubusercontent.com/safeshell/safeshell/main/uninstall.sh | bash
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() {
    echo -e "${BLUE}→${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}!${NC} $1"
}

INSTALL_DIR="${SAFESHELL_INSTALL_DIR:-$HOME/.local/bin}"

echo ""
echo -e "${YELLOW}SafeShell Uninstaller${NC}"
echo ""

# Remove binary
if [[ -f "$INSTALL_DIR/safeshell" ]]; then
    info "Removing binary..."
    rm -f "$INSTALL_DIR/safeshell"
    success "Removed $INSTALL_DIR/safeshell"
else
    warn "Binary not found at $INSTALL_DIR/safeshell"
fi

# Remove aliases from shell config
remove_aliases() {
    local rc_file="$1"
    if [[ -f "$rc_file" ]] && grep -q "SafeShell" "$rc_file"; then
        info "Removing aliases from $rc_file..."
        # Create backup
        cp "$rc_file" "$rc_file.bak"
        # Remove SafeShell block
        sed -i.tmp '/# SafeShell/,/# End SafeShell/d' "$rc_file" 2>/dev/null || \
        sed '/# SafeShell/,/# End SafeShell/d' "$rc_file" > "$rc_file.new" && mv "$rc_file.new" "$rc_file"
        rm -f "$rc_file.tmp"
        success "Removed aliases from $rc_file"
    fi
}

remove_aliases "$HOME/.zshrc"
remove_aliases "$HOME/.bashrc"
remove_aliases "$HOME/.bash_profile"

# Ask about checkpoints
echo ""
if [[ -d "$HOME/.safeshell" ]]; then
    read -p "Remove checkpoint data (~/.safeshell)? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$HOME/.safeshell"
        success "Removed ~/.safeshell"
    else
        warn "Kept ~/.safeshell (your checkpoints are preserved)"
    fi
fi

echo ""
success "SafeShell has been uninstalled"
echo ""
echo "Please restart your terminal or run:"
echo "  source ~/.zshrc  # or ~/.bashrc"
