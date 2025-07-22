# Sync Station

A lightweight, cross-platform CLI/TUI application for synchronizing configuration files between multiple computers using your own cloud storage.

## Features

- **CLI Interface**: Fast, scriptable command-line interface for automation
- **Interactive TUI**: Beautiful terminal user interface with bulk operations
- **Cloud Agnostic**: Works with any cloud storage provider (Dropbox, OneDrive, Google Drive, etc.)
- **System Installation**: Install once, use anywhere on your system
- **Computer-Specific Paths**: Configure different paths for each computer you own
- **Hash-Based Sync**: Efficient file comparison using SHA256 hashes
- **Multiple Sync Modes**:
  - Smart Sync: Intelligent bidirectional sync based on file hashes and timestamps
  - Push: Local files → Cloud storage
  - Pull: Cloud storage → Local files
- **Safety Features**: Dry-run mode, conflict detection, hash-based verification
- **Cross-Platform**: Native support for Windows, Linux, and macOS
- **Package Manager Ready**: Homebrew, APT, RPM, and AUR packages available

## Quick Start

### Installation

**Download prebuilt binaries:**
- Download the latest release for your platform
- Move to a directory in your `$PATH` (e.g., `/usr/local/bin`)
- Make executable: `chmod +x syncstation`

**Package managers (coming soon):**
```bash
# macOS
brew install syncstation

# Ubuntu/Debian
sudo apt install syncstation

# Arch Linux
yay -S syncstation
```

### First Time Setup

1. **Initialize in your cloud folder:**
   ```bash
   cd ~/Dropbox  # or your cloud sync folder
   syncstation init
   ```

2. **Add your first sync item:**
   ```bash
   syncstation add "Neovim Config" ~/.config/nvim
   ```

3. **Check status:**
   ```bash
   syncstation status
   ```

4. **Start syncing:**
   ```bash
   syncstation sync
   ```

## Usage

### Command Line Interface

**Core Commands:**
```bash
syncstation init [--cloud-dir PATH]     # Initialize configuration
syncstation add NAME PATH               # Add sync item
syncstation sync [item-name]            # Smart sync (default)
syncstation push [item-name]            # Push local → cloud
syncstation pull [item-name]            # Pull cloud → local
syncstation status                      # Show sync status
syncstation list                        # List all sync items
syncstation tui                         # Launch interactive TUI
```

**Global Flags:**
```bash
--config-dir PATH     # Custom config directory
--force, -f          # Skip confirmation prompts
--dry-run            # Preview changes without executing
```

**Examples:**
```bash
# Initialize with custom cloud directory
syncstation init --cloud-dir ~/OneDrive/syncstation

# Add different types of sync items
syncstation add "VS Code Settings" ~/.config/Code/User/settings.json --type file
syncstation add "SSH Keys" ~/.ssh --exclude "*.pub,known_hosts"

# Sync specific items
syncstation sync "Neovim Config"
syncstation push "VS Code Settings"
syncstation pull "SSH Keys"

# Bulk operations
syncstation sync              # Sync all items
syncstation push --dry-run    # Preview all pushes
syncstation pull --force      # Pull all without prompts
```

### Interactive TUI

Launch the interactive terminal interface:
```bash
syncstation tui
```

**TUI Features:**
- Navigate with arrow keys or vim bindings (`j`/`k`)
- Select items with `Space`
- Bulk operations: `a` (select all), `n` (select none)
- Actions: `s` (sync), `p` (push), `l` (pull)
- Item management: `Enter` (details), `e` (edit), `d` (delete)
- Quit with `q` or `Ctrl+C`

**TUI Interface:**
```
┌─ Sync Station ─────────────────────────────────────────────────┐
│ Computer: work-laptop    Cloud: ~/Dropbox/syncstation        │
├────────────────────────────────────────────────────────────────┤
│ Sync Items                                    Status           │
├────────────────────────────────────────────────────────────────┤
│ [x] Neovim Config                             ✓ Synced         │
│     ~/.config/nvim/ ↔ configs/neovim/        23 files         │
│                                                                │
│ [ ] VS Code Settings                          ⚠ Local newer   │
│     ~/.config/Code/User/ ↔ configs/vscode/   8 files          │
│                                                                │
│ [x] SSH Keys                                  ⚠ Conflicts      │
│     ~/.ssh/ ↔ configs/ssh-keys/              5 files (2 exc.) │
├────────────────────────────────────────────────────────────────┤
│ [Space] select [A] all [S] sync [P] push [L] pull [Q] quit    │
└────────────────────────────────────────────────────────────────┘
```

## How It Works

### Architecture

1. **Local Configuration**: Stored in platform-specific directories
   - Linux/Unix: `~/.config/syncstation/`
   - macOS: `~/Library/Application Support/syncstation/`
   - Windows: `%APPDATA%/syncstation/`

2. **Cloud Sync Data**: Stored in your existing cloud folder
   - `sync-items.json` - Item definitions shared across computers
   - `file-metadata.json` - File hashes and sync state
   - `configs/` - Actual synced configuration files

3. **Separation of Concerns**: CLI tool installed system-wide, data lives in cloud

### Sync Algorithm

1. **Hash Comparison**: Uses SHA256 to detect real changes
2. **Timestamp Check**: Considers modification times for conflict detection
3. **Smart Decisions**: Automatically syncs newer files, warns about conflicts
4. **Safety First**: Detects conflicts and prevents data loss

### Multi-Computer Workflow

1. **Computer A**: Add sync items and push to cloud
2. **Computer B**: Initialize with same cloud directory
3. **Auto-Discovery**: Sync items appear automatically
4. **Path Mapping**: Configure local paths for each computer
5. **Continuous Sync**: Use smart sync to keep everything in sync

## Configuration

### Local Config (`~/.config/syncstation/config.json`)
```json
{
  "cloudSyncDir": "/home/user/Dropbox/syncstation",
  "currentComputer": "work-laptop"
}
```

### Cloud Sync Items (`~/Dropbox/syncstation/sync-items.json`)
```json
{
  "syncItems": [
    {
      "name": "Neovim Config",
      "type": "folder",
      "paths": {
        "work-laptop": "~/.config/nvim/",
        "home-desktop": "/home/user/.config/nvim/",
        "macbook": "/Users/user/.config/nvim/"
      },
      "excludePatterns": ["*.log", ".cache/", "lazy-lock.json"]
    }
  ]
}
```

## File Structure

```
~/Dropbox/syncstation/          # Your cloud folder
├── sync-items.json              # Sync item definitions
├── file-metadata.json           # File hashes and sync state
└── configs/                     # Synced configuration files
    ├── neovim-config/
    ├── vscode-settings/
    └── ssh-keys/

~/.config/syncstation/          # Local configuration
├── config.json                 # Local computer config
└── file-states.json            # Local file state cache
```

## Building and Installing from Source

### Prerequisites

**Required:**
- **Go 1.18 or later** - [Download from golang.org](https://golang.org/dl/)
- **Git** - For cloning the repository

**No GUI dependencies required** - this is a CLI/TUI application.

**Check your Go version:**
```bash
go version
# Should show go1.18 or later
```

### Step 1: Clone the Repository

```bash
git clone https://github.com/AntoineArt/syncstation.git
cd syncstation
```

### Step 2: Download Dependencies

```bash
go mod tidy
```

### Step 3: Build the Application

**Option A: Build for your current platform**
```bash
# Build with verbose logging
go build -v -o syncstation ./cmd/syncstation

# Verify the build
./syncstation --version
```

**Option B: Build all platforms using the build script**
```bash
chmod +x scripts/build-cross-platform.sh
./scripts/build-cross-platform.sh
```

This creates binaries in the `build/` directory for Linux, Windows, and macOS.

### Step 4: Install System-Wide

**Linux/macOS:**
```bash
# Option 1: Install to /usr/local/bin (requires sudo)
sudo cp syncstation /usr/local/bin/
sudo chmod +x /usr/local/bin/syncstation

# Option 2: Install to ~/bin (user-only)
mkdir -p ~/bin
cp syncstation ~/bin/
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc  # or ~/.zshrc
source ~/.bashrc  # or restart terminal
```

**Windows:**
```bash
# Build Windows executable with verbose logging
GOOS=windows GOARCH=amd64 go build -v -o syncstation.exe ./cmd/syncstation

# Copy syncstation.exe to a directory in your PATH
# Or add the directory containing the executable to your PATH
```

### Step 5: Verify Installation

```bash
syncstation --version
syncstation --help
```

### Alternative: Use Go Install

If the repository is public, you can install directly:
```bash
# Install with verbose logging
go install -v github.com/AntoineArt/syncstation/cmd/syncstation@latest

# Verify installation
which syncstation
syncstation --version
```

This automatically downloads, builds, and installs to `$GOPATH/bin` or `$HOME/go/bin`.

### Manual Cross-Platform Compilation

If you need to build for a specific platform:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -v -o syncstation-linux ./cmd/syncstation

# Windows
GOOS=windows GOARCH=amd64 go build -v -o syncstation-windows.exe ./cmd/syncstation

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -v -o syncstation-macos-intel ./cmd/syncstation

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -v -o syncstation-macos-arm64 ./cmd/syncstation
```

### Troubleshooting Build Issues

**"no Go files" error:**
```bash
# Wrong - this won't work
go build -o syncstation

# Correct - specify the cmd directory with verbose logging
go build -v -o syncstation ./cmd/syncstation
```

**Permission denied after installation:**
```bash
chmod +x syncstation
# or
sudo chmod +x /usr/local/bin/syncstation
```

**Go version too old:**
Update Go to version 1.18 or later from [golang.org](https://golang.org/dl/)

### Uninstalling

```bash
# If installed to /usr/local/bin
sudo rm /usr/local/bin/syncstation

# If installed to ~/bin
rm ~/bin/syncstation

# If installed via go install
rm $GOPATH/bin/syncstation  # or $HOME/go/bin/syncstation
# Note: binary name is 'syncstation' regardless of the cmd/syncstation path
```

## Advanced Usage

### Automation and Scripting

```bash
# Automated sync script
#!/bin/bash
syncstation sync --dry-run
if [ $? -eq 0 ]; then
    syncstation sync --force
    echo "Sync completed successfully"
fi

# Cron job for regular sync (every 30 minutes)
# */30 * * * * /usr/local/bin/syncstation sync --force > /dev/null 2>&1
```

### Integration with Dotfiles

```bash
# Add dotfiles to sync
syncstation add "Zsh Config" ~/.zshrc --type file
syncstation add "Git Config" ~/.gitconfig --type file
syncstation add "Tmux Config" ~/.tmux.conf --type file

# Sync entire .config directory (with exclusions)
syncstation add "User Config" ~/.config \
  --exclude "*/cache/*,*/logs/*,*/tmp/*,*/node_modules/*"
```

### Conflict Resolution

When conflicts occur, Sync Station will:
1. **Warn** about conflicting files
2. **Stop** the sync operation to prevent data loss
3. **Prompt** for user decision (unless `--force` is used)
4. **Log** all operations for review

## Troubleshooting

### Common Issues

**"Not initialized" error:**
```bash
syncstation init  # Run from your cloud sync directory
```

**Path not found:**
```bash
syncstation status  # Check current paths
syncstation list    # Verify sync items
```

**Sync conflicts:**
```bash
syncstation sync --dry-run  # Preview changes
syncstation status          # Check file states
```

### Debug Information

```bash
# Verbose output
syncstation --config-dir ~/.config/syncstation status

# Check configuration
cat ~/.config/syncstation/config.json
cat ~/Dropbox/syncstation/sync-items.json
```

## Supported Platforms

- **Linux** (amd64, arm64)
- **Windows** (amd64)
- **macOS** (amd64, arm64)

## License

GPL v3 License - See LICENSE file for details

## Author

Created by **Artan200** (AntoineArt)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Build: `go build -v ./cmd/syncstation`
6. Submit a pull request

## Migration from GUI Version

If you're upgrading from the old GUI version:

1. **Export** your existing config from the GUI version
2. **Install** the CLI version system-wide
3. **Initialize** with your existing cloud directory:
   ```bash
   syncstation init --cloud-dir /path/to/existing/cloud/folder
   ```
4. Your sync items will be automatically detected and imported