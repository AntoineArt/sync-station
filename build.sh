#!/bin/bash

# Cross-platform build script for Config Sync Tool
# Creates executables for Windows, Linux, and macOS

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "================================================"
echo -e "${BLUE}Config Sync Tool - Cross-Platform Build Script${NC}"
echo "================================================"
echo

# Check if Go is installed
echo "Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}ERROR: Go is not installed or not in PATH${NC}"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

echo -e "${GREEN}Go found!${NC} $(go version)"
echo

# Update dependencies
echo "Updating dependencies..."
go mod tidy
echo

# Create build directory
mkdir -p build

# Track build results
declare -a SUCCESSFUL_BUILDS
declare -a FAILED_BUILDS

# Function to build for a platform
build_platform() {
    local goos=$1
    local goarch=$2
    local output=$3
    local description=$4
    
    echo -e "${BLUE}Building for $description...${NC}"
    
    if GOOS=$goos GOARCH=$goarch go build -o "build/$output" . 2>/dev/null; then
        size=$(stat -c%s "build/$output" 2>/dev/null || stat -f%z "build/$output" 2>/dev/null)
        echo -e "  ${GREEN}✓${NC} $output (${size} bytes)"
        SUCCESSFUL_BUILDS+=("$description")
    else
        echo -e "  ${RED}✗${NC} Failed to build $output"
        FAILED_BUILDS+=("$description")
    fi
}

# Build for all platforms
echo "Building for multiple platforms..."
echo

# Linux builds
build_platform "linux" "amd64" "config-sync-tool-linux-amd64" "Linux 64-bit"
build_platform "linux" "386" "config-sync-tool-linux-386" "Linux 32-bit"
build_platform "linux" "arm64" "config-sync-tool-linux-arm64" "Linux ARM64"

# Windows builds
build_platform "windows" "amd64" "config-sync-tool-windows-amd64.exe" "Windows 64-bit"
build_platform "windows" "386" "config-sync-tool-windows-386.exe" "Windows 32-bit"

# macOS builds
build_platform "darwin" "amd64" "config-sync-tool-macos-amd64" "macOS Intel"
build_platform "darwin" "arm64" "config-sync-tool-macos-arm64" "macOS Apple Silicon"

# Create macOS universal binary if lipo is available
if command -v lipo &> /dev/null && [ -f "build/config-sync-tool-macos-amd64" ] && [ -f "build/config-sync-tool-macos-arm64" ]; then
    echo -e "${BLUE}Creating macOS universal binary...${NC}"
    if lipo -create -output build/config-sync-tool-macos-universal build/config-sync-tool-macos-amd64 build/config-sync-tool-macos-arm64 2>/dev/null; then
        size=$(stat -c%s "build/config-sync-tool-macos-universal" 2>/dev/null || stat -f%z "build/config-sync-tool-macos-universal" 2>/dev/null)
        echo -e "  ${GREEN}✓${NC} config-sync-tool-macos-universal (${size} bytes)"
        SUCCESSFUL_BUILDS+=("macOS Universal")
    else
        echo -e "  ${YELLOW}Warning: Failed to create universal binary${NC}"
    fi
fi

echo
echo "================================================"
echo -e "${GREEN}Build Summary${NC}"
echo "================================================"

if [ ${#SUCCESSFUL_BUILDS[@]} -gt 0 ]; then
    echo -e "${GREEN}Successful builds (${#SUCCESSFUL_BUILDS[@]}):${NC}"
    for build in "${SUCCESSFUL_BUILDS[@]}"; do
        echo -e "  ${GREEN}✓${NC} $build"
    done
fi

if [ ${#FAILED_BUILDS[@]} -gt 0 ]; then
    echo
    echo -e "${RED}Failed builds (${#FAILED_BUILDS[@]}):${NC}"
    for build in "${FAILED_BUILDS[@]}"; do
        echo -e "  ${RED}✗${NC} $build"
    done
fi

echo
echo "Build directory contents:"
ls -la build/ 2>/dev/null || echo "No files in build directory"

echo
echo "================================================"
echo -e "${GREEN}Usage Instructions${NC}"
echo "================================================"
echo
echo "To run on different platforms:"
echo
echo -e "${BLUE}Linux:${NC}"
echo "  ./build/config-sync-tool-linux-amd64"
echo
echo -e "${BLUE}Windows:${NC}"
echo "  build\\config-sync-tool-windows-amd64.exe"
echo
echo -e "${BLUE}macOS:${NC}"
echo "  ./build/config-sync-tool-macos-amd64        (Intel Macs)"
echo "  ./build/config-sync-tool-macos-arm64        (Apple Silicon)"
if [ -f "build/config-sync-tool-macos-universal" ]; then
    echo "  ./build/config-sync-tool-macos-universal    (Both Intel & Apple Silicon)"
fi

echo
echo -e "${GREEN}Cross-platform build complete!${NC}"

# Exit with error code if any builds failed
if [ ${#FAILED_BUILDS[@]} -gt 0 ]; then
    exit 1
fi