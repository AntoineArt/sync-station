#!/bin/bash

# Sync Station Cross-Platform Build Script
# Builds executables for Windows, macOS, and Linux

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build information
APP_NAME="syncstation"
VERSION="1.0.0"
BUILD_DIR="$(pwd)/build"
DIST_DIR="$(pwd)/dist"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_build() {
    echo -e "${BLUE}[BUILD]${NC} $1"
}

# Clean previous builds
print_status "Cleaning previous builds..."
rm -rf "$BUILD_DIR" "$DIST_DIR"
mkdir -p "$BUILD_DIR" "$DIST_DIR"

# Check Go installation
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

print_status "Go version: $(go version)"

# Build function
build_binary() {
    local goos=$1
    local goarch=$2
    local output_name=$3
    local output_path="$BUILD_DIR/$output_name"
    
    print_build "Building $goos/$goarch -> $output_name"
    
    CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
        -ldflags "-s -w -X main.Version=$VERSION" \
        -o "$output_path" \
        ./cmd/syncstation
    
    if [ $? -eq 0 ]; then
        local size=$(du -h "$output_path" | cut -f1)
        print_status "✓ Built $output_name ($size)"
    else
        print_error "✗ Failed to build $output_name"
        return 1
    fi
}

# Build for all platforms
print_status "Building cross-platform binaries..."

# Linux builds
build_binary "linux" "amd64" "${APP_NAME}-linux-amd64"
build_binary "linux" "arm64" "${APP_NAME}-linux-arm64"

# Windows builds  
build_binary "windows" "amd64" "${APP_NAME}-windows-amd64.exe"
build_binary "windows" "arm64" "${APP_NAME}-windows-arm64.exe"

# macOS builds
build_binary "darwin" "amd64" "${APP_NAME}-macos-amd64"
build_binary "darwin" "arm64" "${APP_NAME}-macos-arm64"

# Create universal macOS binary (if on macOS)
if [[ "$OSTYPE" == "darwin"* ]] && command -v lipo &> /dev/null; then
    print_build "Creating universal macOS binary..."
    lipo -create -output "$BUILD_DIR/${APP_NAME}-macos-universal" \
        "$BUILD_DIR/${APP_NAME}-macos-amd64" \
        "$BUILD_DIR/${APP_NAME}-macos-arm64"
    
    if [ $? -eq 0 ]; then
        local size=$(du -h "$BUILD_DIR/${APP_NAME}-macos-universal" | cut -f1)
        print_status "✓ Created universal binary ($size)"
    else
        print_warning "Failed to create universal binary (continuing...)"
    fi
fi

# Create distribution packages
print_status "Creating distribution packages..."

# Function to create archive
create_archive() {
    local platform=$1
    local binary=$2
    local archive_name=$3
    
    local temp_dir=$(mktemp -d)
    local package_dir="$temp_dir/$APP_NAME"
    
    mkdir -p "$package_dir"
    
    # Copy binary
    cp "$BUILD_DIR/$binary" "$package_dir/"
    
    # Copy documentation
    cp README.md "$package_dir/" 2>/dev/null || print_warning "README.md not found"
    cp docs/CONVERSION_DISCUSSION.md "$package_dir/" 2>/dev/null || print_warning "docs/CONVERSION_DISCUSSION.md not found"
    cp CLAUDE.md "$package_dir/" 2>/dev/null || print_warning "CLAUDE.md not found"
    
    # Create simple install script for Unix platforms
    if [[ "$platform" != "windows" ]]; then
        cat > "$package_dir/install.sh" << 'EOF'
#!/bin/bash
# Simple install script for Sync Station

set -e

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="syncstation"

# Detect binary name
if [ -f "./syncstation-linux-amd64" ]; then
    BINARY_FILE="./syncstation-linux-amd64"
elif [ -f "./syncstation-macos-amd64" ]; then
    BINARY_FILE="./syncstation-macos-amd64"
elif [ -f "./syncstation-macos-arm64" ]; then
    BINARY_FILE="./syncstation-macos-arm64"
elif [ -f "./syncstation-macos-universal" ]; then
    BINARY_FILE="./syncstation-macos-universal"
else
    echo "Error: No compatible binary found"
    exit 1
fi

echo "Installing Sync Station to $INSTALL_DIR..."

# Check if we need sudo
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Note: sudo required for installation to $INSTALL_DIR"
    sudo cp "$BINARY_FILE" "$INSTALL_DIR/$BINARY_NAME"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    cp "$BINARY_FILE" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

echo "✓ Sync Station installed successfully!"
echo "Run 'syncstation --help' to get started."
EOF
        chmod +x "$package_dir/install.sh"
    fi
    
    # Create archive
    if [[ "$platform" == "windows" ]]; then
        cd "$temp_dir" && zip -r "$archive_name" "$APP_NAME"
    else
        cd "$temp_dir" && tar -czf "$archive_name" "$APP_NAME"
    fi
    
    mv "$temp_dir/$archive_name" "$DIST_DIR/"
    rm -rf "$temp_dir"
    
    local size=$(du -h "$DIST_DIR/$archive_name" | cut -f1)
    print_status "✓ Created $archive_name ($size)"
}

# Create distribution archives
create_archive "linux" "${APP_NAME}-linux-amd64" "${APP_NAME}-${VERSION}-linux-amd64.tar.gz"
create_archive "linux" "${APP_NAME}-linux-arm64" "${APP_NAME}-${VERSION}-linux-arm64.tar.gz"
create_archive "windows" "${APP_NAME}-windows-amd64.exe" "${APP_NAME}-${VERSION}-windows-amd64.zip"
create_archive "windows" "${APP_NAME}-windows-arm64.exe" "${APP_NAME}-${VERSION}-windows-arm64.zip"
create_archive "macos" "${APP_NAME}-macos-amd64" "${APP_NAME}-${VERSION}-macos-amd64.tar.gz"
create_archive "macos" "${APP_NAME}-macos-arm64" "${APP_NAME}-${VERSION}-macos-arm64.tar.gz"

# Create universal macOS archive if binary exists
if [ -f "$BUILD_DIR/${APP_NAME}-macos-universal" ]; then
    create_archive "macos" "${APP_NAME}-macos-universal" "${APP_NAME}-${VERSION}-macos-universal.tar.gz"
fi

# Generate checksums
print_status "Generating checksums..."
cd "$DIST_DIR"
sha256sum *.tar.gz *.zip > "${APP_NAME}-${VERSION}-checksums.sha256" 2>/dev/null || \
shasum -a 256 *.tar.gz *.zip > "${APP_NAME}-${VERSION}-checksums.sha256" 2>/dev/null || \
print_warning "Could not generate checksums (sha256sum/shasum not available)"

# Summary
print_status "Build completed successfully!"
echo ""
echo "Built binaries:"
ls -lah "$BUILD_DIR"
echo ""
echo "Distribution packages:"
ls -lah "$DIST_DIR"
echo ""
print_status "Ready for distribution!"