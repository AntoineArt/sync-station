# Config Sync Tool

A lightweight, cross-platform GUI application for synchronizing configuration files between multiple computers using your own cloud storage.

## Features

- **Native GUI**: Built with Fyne for native look and feel on Windows, Linux, and macOS
- **Cloud Agnostic**: Works with any cloud storage provider (Dropbox, OneDrive, Google Drive, etc.)
- **Computer-Specific Paths**: Configure different paths for each computer you own
- **Diff Viewer**: Visual side-by-side comparison of files before syncing
- **Multiple Sync Modes**:
  - Smart Sync: Intelligent bidirectional sync based on file timestamps
  - Push: Local files → Cloud storage
  - Pull: Cloud storage → Local files
- **Conflict Resolution**: Automatic handling of file conflicts
- **Backup Support**: Creates backups before overwriting files

## How It Works

1. Place the executable in a folder that's synced by your cloud provider
2. Configure sync items with paths specific to each computer
3. The tool handles file comparison and synchronization
4. Your cloud provider handles the actual syncing between computers

## Usage

### First Time Setup

1. Download the executable for your platform
2. Place it in a folder that's synced by your cloud storage
3. Run the executable
4. Add sync items using the "Add Item" button
5. Configure paths for each computer you use

### Daily Usage

1. Run the tool when you want to sync configurations
2. Use "Smart Sync" for automatic bidirectional sync
3. Use "Push All" to upload local changes to cloud
4. Use "Pull All" to download cloud changes to local
5. Click on files in the tree to view differences

### Sync Modes

- **Smart Sync**: Compares file timestamps and syncs newer files automatically
- **Push All**: Uploads all local files to cloud, overwriting cloud versions
- **Pull All**: Downloads all cloud files to local, overwriting local versions

## Configuration

The tool automatically creates a `config.json` file with your sync settings:

```json
{
  "currentComputer": "my-laptop",
  "computers": {
    "my-laptop": {"id": "my-laptop", "name": "My Laptop"},
    "work-pc": {"id": "work-pc", "name": "Work PC"}
  },
  "syncItems": [
    {
      "name": "Claude Config",
      "paths": {
        "my-laptop": "/home/user/.claude/",
        "work-pc": "C:\\Users\\user\\.claude\\"
      },
      "enabled": true
    }
  ]
}
```

## File Structure

When you use the tool, it creates this structure in your cloud folder:

```
your-cloud-folder/
├── config-sync-tool(.exe)  # The executable
├── config.json             # Configuration file
├── configs/                 # Synced config files
│   ├── claude-config/
│   ├── cursor-settings/
│   └── other-configs/
└── backups/                 # Automatic backups
    └── [timestamp]/
```

## Building from Source

### Prerequisites

**All Platforms:**
- Go 1.18 or later
- Git (to clone the repository)

**Platform-Specific Dependencies:**

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install -y libgl1-mesa-dev xorg-dev
```

**Linux (CentOS/RHEL/Fedora):**
```bash
# CentOS/RHEL
sudo yum install -y mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel

# Fedora
sudo dnf install -y mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel
```

**macOS:**
- Xcode Command Line Tools: `xcode-select --install`
- No additional dependencies needed

**Windows:**
- No additional dependencies needed
- Optionally: TDM-GCC or MinGW for CGO (usually not required)

### Quick Build

**For current platform:**
```bash
git clone <repository-url>
cd config-sync-tool
go mod tidy
go build -o config-sync-tool
```

### Cross-Platform Compilation

**Build for all platforms (using provided script):**
```bash
chmod +x build.sh
./build.sh
```

**Manual cross-compilation:**

**Windows (from any platform):**
```bash
# Windows 64-bit
GOOS=windows GOARCH=amd64 go build -o config-sync-tool-windows.exe

# Windows 32-bit (if needed)
GOOS=windows GOARCH=386 go build -o config-sync-tool-windows-32.exe
```

**macOS (from any platform):**
```bash
# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o config-sync-tool-macos-intel

# macOS Apple Silicon (M1/M2)
GOOS=darwin GOARCH=arm64 go build -o config-sync-tool-macos-arm64

# Universal macOS binary (requires lipo tool on macOS)
GOOS=darwin GOARCH=amd64 go build -o config-sync-tool-macos-amd64
GOOS=darwin GOARCH=arm64 go build -o config-sync-tool-macos-arm64
lipo -create -output config-sync-tool-macos-universal config-sync-tool-macos-amd64 config-sync-tool-macos-arm64
```

**Linux (from any platform):**
```bash
# Linux 64-bit
GOOS=linux GOARCH=amd64 go build -o config-sync-tool-linux

# Linux 32-bit (if needed)
GOOS=linux GOARCH=386 go build -o config-sync-tool-linux-32

# Linux ARM64 (for Raspberry Pi 4, etc.)
GOOS=linux GOARCH=arm64 go build -o config-sync-tool-linux-arm64
```

### Platform-Specific Build Scripts

We provide dedicated build scripts for each platform with enhanced features:

**Cross-Platform (build.sh):**
```bash
chmod +x build.sh
./build.sh
```
Builds for all platforms with colorized output, error handling, and universal macOS binary.

**Linux (build-linux.sh):**
```bash
chmod +x build-linux.sh
./build-linux.sh
```
Features: dependency checking, multiple architectures, desktop entry creation.

**macOS (build-macos.sh):**
```bash
chmod +x build-macos.sh
./build-macos.sh
```
Features: universal binary creation, .app bundle generation, code signing support.

**Windows (build-windows.bat):**
```batch
build-windows.bat
```
Features: automatic dependency checking, 32-bit and 64-bit builds, size reporting.

### Troubleshooting Build Issues

**Linux: Missing GUI dependencies**
```bash
# Error: X11/Xlib.h: No such file or directory
sudo apt-get install xorg-dev

# Error: GL/gl.h: No such file or directory  
sudo apt-get install libgl1-mesa-dev
```

**Windows: CGO compilation issues**
```bash
# Use CGO_ENABLED=0 to disable CGO (creates larger binary but avoids C dependencies)
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o config-sync-tool-windows.exe
```

**macOS: Code signing (for distribution)**
```bash
# Sign the binary (requires Apple Developer account)
codesign --force --verify --verbose --sign "Developer ID Application: Your Name" config-sync-tool-macos

# Create a .app bundle (optional)
mkdir -p ConfigSyncTool.app/Contents/MacOS
cp config-sync-tool-macos ConfigSyncTool.app/Contents/MacOS/
```

### Build Output

After building, you'll get platform-specific executables:
- **Linux**: `config-sync-tool` (or `config-sync-tool-linux`)
- **Windows**: `config-sync-tool.exe` (or `config-sync-tool-windows.exe`)  
- **macOS**: `config-sync-tool` (or `config-sync-tool-macos`)

All executables are self-contained and don't require Go to be installed on the target machine.

## Supported Platforms

- Windows (amd64)
- Linux (amd64)
- macOS (amd64, arm64)

## License

MIT License - See LICENSE file for details