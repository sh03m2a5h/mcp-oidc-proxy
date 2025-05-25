#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default installation directory
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# GitHub repository
REPO="sh03m2a5h/mcp-oidc-proxy"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

case $OS in
    linux|darwin) ;;
    *) echo -e "${RED}Unsupported OS: $OS${NC}"; exit 1 ;;
esac

BINARY_NAME="mcp-oidc-proxy-${OS}-${ARCH}"

# Get latest release
echo -e "${YELLOW}Fetching latest release...${NC}"
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo -e "${RED}Failed to fetch latest release${NC}"
    exit 1
fi

echo -e "${GREEN}Latest version: ${LATEST_RELEASE}${NC}"

# Download URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_RELEASE}/${BINARY_NAME}"

# Download binary
echo -e "${YELLOW}Downloading ${BINARY_NAME}...${NC}"
curl -L -o "/tmp/${BINARY_NAME}" "$DOWNLOAD_URL"

# Make executable
chmod +x "/tmp/${BINARY_NAME}"

# Install
echo -e "${YELLOW}Installing to ${INSTALL_DIR}...${NC}"
sudo mv "/tmp/${BINARY_NAME}" "${INSTALL_DIR}/mcp-oidc-proxy"

echo -e "${GREEN}✓ Installation complete!${NC}"
echo -e "${GREEN}Run 'mcp-oidc-proxy --help' to get started${NC}"