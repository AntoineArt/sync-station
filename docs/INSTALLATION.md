# Installation Guide

This guide covers all methods for installing Syncstation on your system.

## Quick Install (Recommended)

### Package Managers (needs testing)

**macOS (Homebrew):**
```bash
brew install syncstation
```

**Ubuntu/Debian:**
```bash
sudo apt install syncstation
```

**Arch Linux:**
```bash
yay -S syncstation
```

**Fedora/RHEL:**
```bash
sudo dnf install syncstation
```

*Note: Package manager installations are planned but not yet available. Use prebuilt binaries below.*

## Prebuilt Binaries

Download the latest release for your platform from the [releases page](https://github.com/AntoineArt/syncstation/releases).

### Linux
```bash
# Download and extract
wget https://github.com/AntoineArt/syncstation/releases/download/v1.0.0/syncstation-1.0.0-linux-amd64.tar.gz
tar -xzf syncstation-1.0.0-linux-amd64.tar.gz
cd syncstation

# Install system-wide
sudo ./install.sh

# Or install to user directory
mkdir -p ~/bin
cp syncstation-linux-amd64 ~/bin/syncstation
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### macOS
```bash
# Download and extract
curl -L https://github.com/AntoineArt/syncstation/releases/download/v1.0.0/syncstation-1.0.0-macos-universal.tar.gz | tar -xz
cd syncstation

# Install system-wide (requires sudo)
sudo ./install.sh

# Or install to user directory
mkdir -p ~/bin
cp syncstation-macos-universal ~/bin/syncstation
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Windows (needs testing)
1. Download `syncstation-1.0.0-windows-amd64.zip`
2. Extract to a folder (e.g., `C:\Program Files\syncstation\`)
3. Add the folder to your system PATH
4. Open Command Prompt or PowerShell and run `syncstation --help`

## Building from Source

See [BUILDING.md](BUILDING.md) for detailed instructions on building Syncstation from source code.

## Verification

After installation, verify it works:

```bash
syncstation --version
syncstation --help
```

You should see version information and help text.

## Platform Support

- **Linux**: Debian/Ubuntu, Arch, Fedora/RHEL (amd64, arm64)
- **macOS** (needs testing): Intel and Apple Silicon (universal binary available)
- **Windows** (needs testing): 64-bit (amd64, arm64)

## Troubleshooting

### Permission Denied
```bash
chmod +x syncstation
# or for system installation:
sudo chmod +x /usr/local/bin/syncstation
```

### Command Not Found
Ensure the installation directory is in your PATH:
```bash
# Check current PATH
echo $PATH

# Add to PATH (Linux/macOS)
export PATH="$PATH:/path/to/syncstation"

# Make permanent by adding to ~/.bashrc or ~/.zshrc
echo 'export PATH="$PATH:/path/to/syncstation"' >> ~/.bashrc
```

### macOS Security Warning
If you get a security warning on macOS:
1. Go to System Preferences → Security & Privacy
2. Click "Allow Anyway" next to the blocked app
3. Or run: `xattr -d com.apple.quarantine /path/to/syncstation`

## Uninstalling

### Package Manager Installation
```bash
# macOS
brew uninstall syncstation

# Ubuntu/Debian  
sudo apt remove syncstation

# Arch Linux
yay -R syncstation
```

### Manual Installation
```bash
# Remove binary
sudo rm /usr/local/bin/syncstation
# or for user installation:
rm ~/bin/syncstation

# Remove configuration (optional)
rm -rf ~/.config/syncstation
```

## Next Steps

After installation, see the [Quick Start Guide](../README.md#quick-start) to set up your first sync items.