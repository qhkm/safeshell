#!/bin/bash
#
# SafeShell Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/qhkm/safeshell/main/install.sh | bash
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Configuration
REPO="qhkm/safeshell"
INSTALL_DIR="${SAFESHELL_INSTALL_DIR:-$HOME/.local/bin}"
GITHUB_URL="https://github.com/$REPO"

print_banner() {
    echo -e "${CYAN}"
    echo '  ____         __      ____  _          _ _ '
    echo ' / ___|  __ _ / _| ___/ ___|| |__   ___| | |'
    echo ' \___ \ / _` | |_ / _ \___ \|  _ \ / _ \ | |'
    echo '  ___) | (_| |  _|  __/___) | | | |  __/ | |'
    echo ' |____/ \__,_|_|  \___|____/|_| |_|\___|_|_|'
    echo -e "${NC}"
    echo -e "${BOLD}Let agents run freely. Everything is reversible.${NC}"
    echo ""
}

info() {
    echo -e "${BLUE}→${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}!${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
    exit 1
}

detect_os() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux*)  OS="linux" ;;
        Darwin*) OS="darwin" ;;
        *)       error "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64)  ARCH="amd64" ;;
        amd64)   ARCH="amd64" ;;
        arm64)   ARCH="arm64" ;;
        aarch64) ARCH="arm64" ;;
        *)       error "Unsupported architecture: $ARCH" ;;
    esac

    echo "${OS}_${ARCH}"
}

get_latest_version() {
    curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | \
        grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "v0.1.2"
}

detect_shell() {
    CURRENT_SHELL="$(basename "$SHELL")"
    case "$CURRENT_SHELL" in
        zsh)
            if [[ -f "$HOME/.zshrc" ]]; then
                echo "$HOME/.zshrc"
            else
                echo "$HOME/.zshrc"
            fi
            ;;
        bash)
            if [[ -f "$HOME/.bash_profile" ]]; then
                echo "$HOME/.bash_profile"
            elif [[ -f "$HOME/.bashrc" ]]; then
                echo "$HOME/.bashrc"
            else
                echo "$HOME/.bashrc"
            fi
            ;;
        *)
            echo "$HOME/.profile"
            ;;
    esac
}

add_to_path() {
    local shell_rc="$1"
    local path_line="export PATH=\"\$HOME/.local/bin:\$PATH\""

    if ! grep -q '.local/bin' "$shell_rc" 2>/dev/null; then
        echo "" >> "$shell_rc"
        echo "# Added by SafeShell installer" >> "$shell_rc"
        echo "$path_line" >> "$shell_rc"
        return 0
    fi
    return 1
}

add_aliases() {
    local shell_rc="$1"
    local alias_block='
# SafeShell - Automatic filesystem checkpoints
# Run `safeshell init` to regenerate these aliases
alias rm='\''safeshell wrap rm'\''
alias mv='\''safeshell wrap mv'\''
alias cp='\''safeshell wrap cp'\''
alias chmod='\''safeshell wrap chmod'\''
alias chown='\''safeshell wrap chown'\''
# End SafeShell'

    if grep -q "SafeShell" "$shell_rc" 2>/dev/null; then
        return 1
    fi

    echo "$alias_block" >> "$shell_rc"
    return 0
}

install_binary() {
    local platform="$1"
    local version="$2"

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Download pre-built binary from releases
    local filename="safeshell_${platform}.tar.gz"
    local download_url="$GITHUB_URL/releases/download/$version/$filename"

    info "Downloading SafeShell $version for $platform..."

    local tmp_dir=$(mktemp -d)
    cd "$tmp_dir"

    if curl -fsSL "$download_url" -o safeshell.tar.gz 2>/dev/null; then
        tar -xzf safeshell.tar.gz
        # The archive contains just the binary named safeshell_platform
        mv safeshell_* "$INSTALL_DIR/safeshell" 2>/dev/null || mv safeshell "$INSTALL_DIR/safeshell"
        chmod +x "$INSTALL_DIR/safeshell"
        cd - > /dev/null
        rm -rf "$tmp_dir"
        return 0
    fi

    cd - > /dev/null
    rm -rf "$tmp_dir"

    # Fallback: build from source if Go is available
    if command -v go &> /dev/null; then
        info "Download failed, building from source..."
        local tmp_dir=$(mktemp -d)
        cd "$tmp_dir"

        if git clone --depth 1 "$GITHUB_URL.git" safeshell 2>/dev/null; then
            cd safeshell
            go build -o "$INSTALL_DIR/safeshell" ./cmd/safeshell
            cd - > /dev/null
            rm -rf "$tmp_dir"
            return 0
        fi

        cd - > /dev/null
        rm -rf "$tmp_dir"
    fi

    error "Could not install SafeShell. Please check your internet connection or install Go."
}

main() {
    print_banner

    # Detect platform
    info "Detecting platform..."
    PLATFORM=$(detect_os)
    success "Platform: $PLATFORM"

    # Get version
    info "Checking latest version..."
    VERSION=$(get_latest_version)
    success "Version: $VERSION"

    # Install binary
    info "Installing SafeShell to $INSTALL_DIR..."
    install_binary "$PLATFORM" "$VERSION"
    success "Binary installed"

    # Detect shell config
    SHELL_RC=$(detect_shell)
    info "Detected shell config: $SHELL_RC"

    # Add to PATH if needed
    if add_to_path "$SHELL_RC"; then
        success "Added $INSTALL_DIR to PATH"
    fi

    # Add aliases
    if add_aliases "$SHELL_RC"; then
        success "Added SafeShell aliases"
    else
        warn "SafeShell aliases already exist"
    fi

    # Create safeshell directory
    mkdir -p "$HOME/.safeshell/checkpoints"
    success "Created ~/.safeshell directory"

    echo ""
    echo -e "${GREEN}${BOLD}Installation complete!${NC}"
    echo ""
    echo -e "To activate SafeShell, run:"
    echo -e "  ${CYAN}source $SHELL_RC${NC}"
    echo ""
    echo -e "Or simply ${BOLD}restart your terminal${NC}."
    echo ""
    echo -e "Once activated, these commands will auto-checkpoint:"
    echo -e "  ${YELLOW}rm${NC}, ${YELLOW}mv${NC}, ${YELLOW}cp${NC}, ${YELLOW}chmod${NC}, ${YELLOW}chown${NC}"
    echo ""
    echo -e "Useful commands:"
    echo -e "  ${CYAN}safeshell list${NC}          - View all checkpoints"
    echo -e "  ${CYAN}safeshell rollback --last${NC} - Undo last destructive command"
    echo -e "  ${CYAN}safeshell status${NC}        - Show statistics"
    echo ""
    echo -e "${BOLD}Let agents run freely. Everything is reversible.${NC}"
}

main "$@"
