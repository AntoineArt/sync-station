# Sync Station

A lightweight, cross-platform CLI/TUI application for synchronizing configuration files between multiple computers using your own cloud storage.

## Features

- **CLI & TUI Interface**: Fast command-line interface + beautiful interactive terminal UI
- **Cloud Agnostic**: Works with any cloud storage (Dropbox, OneDrive, Google Drive, etc.)
- **Git Mode** (needs testing): Use a git repository instead of cloud folders for version control
- **Smart Sync** (needs testing): Intelligent bidirectional sync using SHA256 hashes and timestamps
- **Multi-Platform** (needs testing): Native support for Windows, Linux, and macOS
- **Computer-Specific Paths** (needs testing): Different paths for each of your computers
- **Safety Features** (needs testing): Dry-run mode, conflict detection, hash-based verification

## Quick Start

### Installation

See [Installation Guide](docs/INSTALLATION.md) for all installation methods.

**Quick install:**
```bash
# Download latest release and install
curl -L https://github.com/AntoineArt/syncstation/releases/latest/download/syncstation-linux-amd64 -o syncstation
chmod +x syncstation
sudo mv syncstation /usr/local/bin/
```

### First Setup

```bash
# Initialize in your cloud folder (with interactive computer name prompt)
cd ~/Dropbox  # or your cloud sync folder
syncstation init
💻 Computer name (default: hostname-laptop): work-laptop

# Add your first sync item (auto-detects file vs folder)
syncstation add "Neovim Config" ~/.config/nvim

# Check status
syncstation status

# Start syncing
syncstation sync
```

## Usage

### Command Line Interface

**Core Commands:**
```bash
syncstation init [cloud-dir]           # Initialize configuration
syncstation add NAME PATH              # Add sync item
syncstation sync [item-name]           # Smart sync (default)
syncstation push/pull [item-name]      # One-way sync
syncstation status                     # Show sync status
syncstation list                       # List all sync items
syncstation tui                        # Launch interactive TUI
```

**Adding Sync Items:**
```bash
# Add a config file (auto-detects as file)
syncstation add "Vim Config" ~/.vimrc

# Add a config directory (auto-detects as folder)  
syncstation add "SSH Config" ~/.ssh

# Add with exclude patterns for folders
syncstation add "Project Config" ~/myproject --exclude "*.log,build/*,node_modules"

# Multi-computer workflow example:
# Computer A: 
syncstation add "Zsh Config" ~/.zshrc
# Computer B (different path):
syncstation add "Zsh Config" /Users/me/.zshrc  # Same name, different path
```

### Interactive TUI

Launch the beautiful terminal interface:
```bash
syncstation tui
```

Navigate with arrow keys, select with `Space`, and perform bulk operations. See [Usage Examples](docs/USAGE_EXAMPLES.md) for detailed workflows.

## How It Works

Syncstation keeps your configuration files synchronized across multiple computers using your existing cloud storage:

1. **Smart Sync Algorithm**: Uses SHA256 hashes and timestamps to detect changes and resolve conflicts intelligently
2. **Cross-Platform Paths**: Configure different paths for each computer (Linux, macOS, Windows)
3. **Cloud Storage**: Uses your existing Dropbox/OneDrive/Google Drive - no additional accounts needed
4. **Git Mode**: Optional version control with git repositories instead of cloud folders

For detailed architecture and configuration options, see [Configuration Guide](docs/CONFIGURATION.md).

## Documentation

- **[Installation Guide](docs/INSTALLATION.md)** - All installation methods including package managers and prebuilt binaries
- **[Building from Source](docs/BUILDING.md)** - Complete guide for building and development setup  
- **[Configuration Guide](docs/CONFIGURATION.md)** - Multi-computer setup, configuration files, and advanced options
- **[Usage Examples](docs/USAGE_EXAMPLES.md)** - Detailed workflows and real-world examples

## Supported Platforms

- **Linux**: Debian/Ubuntu, Arch, Fedora/RHEL (amd64, arm64)
- **macOS** (needs testing): Intel and Apple Silicon (universal binaries)
- **Windows** (needs testing): 64-bit (amd64, arm64)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

1. Fork the repository
2. Create a feature branch  
3. Run tests: `go test ./...`
4. Submit a pull request

## License

GPL v3 License - See [LICENSE](LICENSE) file for details.

## Author

Created by **Artan200** (AntoineArt)
