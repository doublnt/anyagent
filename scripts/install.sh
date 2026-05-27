#!/bin/bash
set -e

# AnyAgent CLI Installer
# Usage: curl -fsSL https://anyagent.dev/install.sh | sh

REPO="anyagent/anyagent"
BINARY="agentx"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        CYGWIN*|MINGW*|MSYS*) os="windows" ;;
        *)
            echo "Unsupported OS: $(uname -s)"
            exit 1
            ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            echo "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Get latest release version
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
    if [ -z "$version" ]; then
        echo "Failed to get latest version"
        exit 1
    fi
    echo "$version"
}

# Download and install
main() {
    local platform version download_url temp_dir

    platform=$(detect_platform)
    version=$(get_latest_version)

    echo "Installing AnyAgent CLI v${version} for ${platform}..."

    # Construct download URL
    if [[ "$platform" == *"windows"* ]]; then
        download_url="https://github.com/${REPO}/releases/download/v${version}/${BINARY}-${platform}.exe"
    else
        download_url="https://github.com/${REPO}/releases/download/v${version}/${BINARY}-${platform}"
    fi

    # Create temp directory
    temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" EXIT

    # Download binary
    echo "Downloading ${download_url}..."
    if ! curl -fsSL "$download_url" -o "${temp_dir}/${BINARY}"; then
        echo "Failed to download. Trying alternate URL..."
        # Try without extension
        download_url="https://github.com/${REPO}/releases/download/v${version}/${BINARY}-${platform}"
        curl -fsSL "$download_url" -o "${temp_dir}/${BINARY}"
    fi

    # Make executable
    chmod +x "${temp_dir}/${BINARY}"

    # Install
    echo "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "${temp_dir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        sudo mv "${temp_dir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    fi

    # Verify installation
    if command -v $BINARY &> /dev/null; then
        echo ""
        echo "✓ AnyAgent CLI installed successfully!"
        echo ""
        echo "Run 'agentx --help' to get started."
        echo ""
        echo "Quick start:"
        echo "  agentx init          # Initialize project"
        echo "  agentx login         # Authenticate"
        echo "  agentx install code-reviewer  # Install an agent"
        echo "  agentx mcp           # Start MCP server"
    else
        echo ""
        echo "⚠️  Installation complete, but 'agentx' is not in PATH."
        echo "Add ${INSTALL_DIR} to your PATH or run:"
        echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
}

main "$@"
