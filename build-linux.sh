#!/bin/bash

echo "========================================"
echo "Config Sync Tool - Linux Build Script"
echo "========================================"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Go is installed
echo "Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}ERROR: Go is not installed or not in PATH${NC}"
    echo "Please install Go from https://golang.org/dl/"
    echo "Or use your package manager:"
    echo "  Ubuntu/Debian: sudo apt install golang-go"
    echo "  CentOS/RHEL:   sudo yum install golang"
    echo "  Fedora:        sudo dnf install golang"
    echo "  Arch:          sudo pacman -S go"
    exit 1
fi

echo -e "${GREEN}Go found!${NC} $(go version)"
echo

# Check for required system dependencies
echo "Checking system dependencies..."
missing_deps=()

# Check for required headers/libraries
if ! pkg-config --exists gl 2>/dev/null; then
    missing_deps+=("OpenGL development libraries")
fi

if ! pkg-config --exists x11 2>/dev/null; then
    missing_deps+=("X11 development libraries")
fi

if [ ${#missing_deps[@]} -gt 0 ]; then
    echo -e "${YELLOW}Warning: Some dependencies might be missing:${NC}"
    for dep in "${missing_deps[@]}"; do
        echo "  - $dep"
    done
    echo
    echo "To install dependencies:"
    echo -e "${BLUE}Ubuntu/Debian:${NC}"
    echo "  sudo apt-get install libgl1-mesa-dev xorg-dev"
    echo -e "${BLUE}CentOS/RHEL:${NC}"
    echo "  sudo yum install mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel"
    echo -e "${BLUE}Fedora:${NC}"
    echo "  sudo dnf install mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel"
    echo
    echo "Continuing with build anyway..."
fi

echo "Updating dependencies..."
go mod tidy

echo
echo "Building for Linux..."

# Create build directory
mkdir -p build

# Build for current architecture
echo -e "${BLUE}Building for current architecture...${NC}"
go build -o build/config-sync-tool-linux .
if [ $? -ne 0 ]; then
    echo -e "${RED}ERROR: Build failed for current architecture${NC}"
    echo
    echo "Common solutions:"
    echo "1. Install missing system dependencies (see above)"
    echo "2. Try building without CGO: CGO_ENABLED=0 go build -o build/config-sync-tool-linux ."
    exit 1
fi

# Build for amd64 (64-bit)
echo -e "${BLUE}Building for Linux amd64 (64-bit)...${NC}"
GOOS=linux GOARCH=amd64 go build -o build/config-sync-tool-linux-amd64 .
if [ $? -ne 0 ]; then
    echo -e "${YELLOW}Warning: Build failed for amd64${NC}"
fi

# Build for 386 (32-bit)
echo -e "${BLUE}Building for Linux 386 (32-bit)...${NC}"
GOOS=linux GOARCH=386 go build -o build/config-sync-tool-linux-386 .
if [ $? -ne 0 ]; then
    echo -e "${YELLOW}Warning: Build failed for 386${NC}"
fi

# Build for ARM64
echo -e "${BLUE}Building for Linux ARM64...${NC}"
GOOS=linux GOARCH=arm64 go build -o build/config-sync-tool-linux-arm64 .
if [ $? -ne 0 ]; then
    echo -e "${YELLOW}Warning: Build failed for ARM64${NC}"
fi

echo
echo "========================================"
echo -e "${GREEN}Build Complete!${NC}"
echo "========================================"
echo
echo "Files created in build/ directory:"

for file in build/config-sync-tool-linux*; do
    if [ -f "$file" ]; then
        size=$(stat -c%s "$file")
        basename_file=$(basename "$file")
        echo -e "  ${GREEN}✓${NC} $basename_file - ${size} bytes"
    fi
done

echo
echo "To run the application:"
echo "  ./build/config-sync-tool-linux"
echo
echo "Or copy to system location:"
echo "  sudo cp build/config-sync-tool-linux /usr/local/bin/config-sync-tool"
echo "  config-sync-tool"
echo

# Check if we can create a .desktop file
if command -v desktop-file-validate &> /dev/null; then
    read -p "Create desktop entry for application menu? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}Creating desktop entry...${NC}"
        
        desktop_file="$HOME/.local/share/applications/config-sync-tool.desktop"
        mkdir -p "$(dirname "$desktop_file")"
        
        cat > "$desktop_file" << EOF
[Desktop Entry]
Name=Config Sync Tool
Comment=Synchronize configuration files between computers
Exec=$(pwd)/build/config-sync-tool-linux
Icon=applications-utilities
Terminal=false
Type=Application
Categories=Utility;System;
Keywords=config;sync;backup;settings;
EOF
        
        if desktop-file-validate "$desktop_file" 2>/dev/null; then
            echo -e "${GREEN}✓${NC} Desktop entry created: $desktop_file"
            echo "The application should now appear in your application menu."
        else
            echo -e "${YELLOW}Warning: Desktop entry validation failed${NC}"
        fi
    fi
fi

echo
echo -e "${GREEN}Done!${NC}"