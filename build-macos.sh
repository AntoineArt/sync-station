#!/bin/bash

echo "========================================"
echo "Config Sync Tool - macOS Build Script"
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
    exit 1
fi

echo -e "${GREEN}Go found!${NC} $(go version)"
echo

# Check if we're running on macOS for lipo command
LIPO_AVAILABLE=false
if command -v lipo &> /dev/null; then
    LIPO_AVAILABLE=true
    echo -e "${GREEN}lipo command available - can create universal binary${NC}"
else
    echo -e "${YELLOW}lipo command not available - will create separate binaries${NC}"
fi
echo

echo "Updating dependencies..."
go mod tidy

echo
echo "Building for macOS..."

# Create build directory
mkdir -p build

# Build for current architecture
echo -e "${BLUE}Building for current architecture...${NC}"
go build -o build/config-sync-tool-macos .
if [ $? -ne 0 ]; then
    echo -e "${RED}ERROR: Build failed for current architecture${NC}"
    exit 1
fi

# Build for Intel (amd64)
echo -e "${BLUE}Building for Intel Macs (amd64)...${NC}"
GOOS=darwin GOARCH=amd64 go build -o build/config-sync-tool-macos-intel .
if [ $? -ne 0 ]; then
    echo -e "${RED}ERROR: Build failed for Intel Macs${NC}"
    exit 1
fi

# Build for Apple Silicon (arm64)
echo -e "${BLUE}Building for Apple Silicon (arm64)...${NC}"
GOOS=darwin GOARCH=arm64 go build -o build/config-sync-tool-macos-arm64 .
if [ $? -ne 0 ]; then
    echo -e "${RED}ERROR: Build failed for Apple Silicon${NC}"
    exit 1
fi

# Create universal binary if lipo is available
if [ "$LIPO_AVAILABLE" = true ]; then
    echo -e "${BLUE}Creating universal binary...${NC}"
    lipo -create -output build/config-sync-tool-macos-universal build/config-sync-tool-macos-intel build/config-sync-tool-macos-arm64
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Universal binary created successfully!${NC}"
    else
        echo -e "${YELLOW}Warning: Failed to create universal binary${NC}"
    fi
fi

echo
echo "========================================"
echo -e "${GREEN}Build Complete!${NC}"
echo "========================================"
echo
echo "Files created in build/ directory:"

if [ -f "build/config-sync-tool-macos" ]; then
    size=$(stat -f%z "build/config-sync-tool-macos" 2>/dev/null || stat -c%s "build/config-sync-tool-macos" 2>/dev/null)
    echo -e "  ${GREEN}✓${NC} config-sync-tool-macos (current arch) - ${size} bytes"
fi

if [ -f "build/config-sync-tool-macos-intel" ]; then
    size=$(stat -f%z "build/config-sync-tool-macos-intel" 2>/dev/null || stat -c%s "build/config-sync-tool-macos-intel" 2>/dev/null)
    echo -e "  ${GREEN}✓${NC} config-sync-tool-macos-intel - ${size} bytes"
fi

if [ -f "build/config-sync-tool-macos-arm64" ]; then
    size=$(stat -f%z "build/config-sync-tool-macos-arm64" 2>/dev/null || stat -c%s "build/config-sync-tool-macos-arm64" 2>/dev/null)
    echo -e "  ${GREEN}✓${NC} config-sync-tool-macos-arm64 - ${size} bytes"
fi

if [ -f "build/config-sync-tool-macos-universal" ]; then
    size=$(stat -f%z "build/config-sync-tool-macos-universal" 2>/dev/null || stat -c%s "build/config-sync-tool-macos-universal" 2>/dev/null)
    echo -e "  ${GREEN}✓${NC} config-sync-tool-macos-universal - ${size} bytes"
fi

echo
echo "To run the application:"
echo "  ./build/config-sync-tool-macos"
echo
echo "Or for specific architecture:"
echo "  ./build/config-sync-tool-macos-intel    (Intel Macs)"
echo "  ./build/config-sync-tool-macos-arm64    (Apple Silicon)"
if [ -f "build/config-sync-tool-macos-universal" ]; then
    echo "  ./build/config-sync-tool-macos-universal (Both)"
fi
echo

# Optional: Create .app bundle
read -p "Create .app bundle? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}Creating .app bundle...${NC}"
    
    APP_NAME="Config Sync Tool.app"
    mkdir -p "build/$APP_NAME/Contents/MacOS"
    mkdir -p "build/$APP_NAME/Contents/Resources"
    
    # Copy binary (prefer universal if available)
    if [ -f "build/config-sync-tool-macos-universal" ]; then
        cp "build/config-sync-tool-macos-universal" "build/$APP_NAME/Contents/MacOS/Config Sync Tool"
    else
        cp "build/config-sync-tool-macos" "build/$APP_NAME/Contents/MacOS/Config Sync Tool"
    fi
    
    # Create Info.plist
    cat > "build/$APP_NAME/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>Config Sync Tool</string>
    <key>CFBundleIdentifier</key>
    <string>com.example.config-sync-tool</string>
    <key>CFBundleName</key>
    <string>Config Sync Tool</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF
    
    echo -e "${GREEN}✓${NC} $APP_NAME created successfully!"
    echo "You can now drag it to your Applications folder."
fi

echo
echo -e "${GREEN}Done!${NC}"