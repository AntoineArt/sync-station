# Configuration Guide

This guide covers how to configure Syncstation for your specific setup, including multi-computer workflows and advanced options.

## Configuration Files

Syncstation uses two types of configuration:

1. **Local Configuration** - Stored on each computer individually
2. **Cloud Configuration** - Shared across all your computers via cloud storage

### Local Configuration

Stored in platform-specific directories:
- **Linux/Unix**: `~/.config/syncstation/config.json`
- **macOS**: `~/Library/Application Support/syncstation/config.json`
- **Windows**: `%APPDATA%/syncstation/config.json`

### Cloud Configuration

Stored in your cloud sync directory:
- `sync-items.json` - Sync item definitions (shared)
- `file-metadata.json` - File hashes and sync state (shared)
- `configs/` - Actual synced configuration files

## Local Configuration Format

```json
{
  "cloudSyncDir": "/home/user/Dropbox/syncstation",
  "currentComputer": "work-laptop",
  "lastSyncTimes": {
    "Neovim Config": "2024-01-15T10:30:00Z",
    "VS Code Settings": "2024-01-15T09:15:00Z"
  },
  "gitMode": false,
  "gitRepoRoot": ""
}
```

### Configuration Fields

| Field | Description | Example |
|-------|-------------|---------|
| `cloudSyncDir` | Path to your cloud sync directory | `"/home/user/Dropbox/syncstation"` |
| `currentComputer` | Unique identifier for this computer | `"work-laptop"` |
| `lastSyncTimes` | Timestamps of last sync per item | Auto-managed |
| `gitMode` | Use git repository instead of cloud folder | `true` or `false` |
| `gitRepoRoot` | Root of git repository (if gitMode is true) | `"/home/user/dotfiles"` |

## Cloud Sync Items Configuration

The `sync-items.json` file defines what gets synced:

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
    },
    {
      "name": "SSH Keys",
      "type": "folder", 
      "paths": {
        "work-laptop": "~/.ssh/",
        "home-desktop": "~/.ssh/",
        "macbook": "~/.ssh/"
      },
      "excludePatterns": ["*.pub", "known_hosts", "authorized_keys"]
    },
    {
      "name": "VS Code Settings",
      "type": "file",
      "paths": {
        "work-laptop": "~/.config/Code/User/settings.json",
        "home-desktop": "~/.config/Code/User/settings.json", 
        "macbook": "~/Library/Application Support/Code/User/settings.json"
      },
      "excludePatterns": []
    }
  ]
}
```

### Sync Item Fields

| Field | Description | Required |
|-------|-------------|----------|
| `name` | Display name for the sync item | Yes |
| `type` | `"file"` or `"folder"` | Yes |
| `paths` | Computer ID → local path mapping | Yes |
| `excludePatterns` | Patterns to exclude during sync | No |

## Multi-Computer Setup

### Step 1: Initialize on First Computer

```bash
cd ~/Dropbox  # or your cloud sync folder
syncstation init
```

### Step 2: Add Sync Items

```bash
# Add different types of configurations
syncstation add "Neovim Config" ~/.config/nvim
syncstation add "VS Code Settings" ~/.config/Code/User/settings.json
syncstation add "Shell Config" ~/.bashrc
```

### Step 3: Set Up Additional Computers

On each additional computer:

```bash
# Initialize with the same cloud directory
syncstation init --cloud-dir ~/Dropbox/syncstation

# Items will automatically appear, but configure paths for this computer
syncstation list  # Shows items with no local paths configured
```

### Step 4: Configure Computer-Specific Paths

Edit the `sync-items.json` file to add paths for each computer, or use the TUI:

```bash
syncstation tui
```

## Advanced Configuration

### Exclude Patterns

Use glob patterns to exclude files/folders:

```json
"excludePatterns": [
  "*.log",           // All log files
  "*.tmp",           // Temporary files  
  ".cache/",         // Cache directories
  "node_modules/",   // Node.js dependencies
  ".git/",           // Git directories
  "*.lock"           // Lock files
]
```

### Git Mode

For version-controlled syncing:

```json
{
  "cloudSyncDir": "/home/user/dotfiles",
  "gitMode": true,
  "gitRepoRoot": "/home/user/dotfiles"
}
```

In git mode:
- Metadata is stored in git notes instead of files
- Automatic git operations can be configured
- Supports branch-based workflows

### Path Expansion

Syncstation expands paths automatically:

| Pattern | Expands To |
|---------|------------|
| `~/` | User home directory |
| `$HOME/` | User home directory |
| `$XDG_CONFIG_HOME/` | XDG config directory |

## Configuration Examples

### Developer Workstation

```json
{
  "syncItems": [
    {
      "name": "Neovim Config",
      "type": "folder",
      "paths": {
        "work-laptop": "~/.config/nvim/",
        "home-pc": "~/.config/nvim/"
      },
      "excludePatterns": [".lazy/", "lazy-lock.json", "*.log"]
    },
    {
      "name": "Git Config",
      "type": "file",
      "paths": {
        "work-laptop": "~/.gitconfig",
        "home-pc": "~/.gitconfig"
      }
    },
    {
      "name": "Shell Aliases",
      "type": "file", 
      "paths": {
        "work-laptop": "~/.bash_aliases",
        "home-pc": "~/.bash_aliases"
      }
    }
  ]
}
```

### Cross-Platform Setup

```json
{
  "syncItems": [
    {
      "name": "VS Code Settings",
      "type": "folder",
      "paths": {
        "linux-desktop": "~/.config/Code/User/",
        "macos-laptop": "~/Library/Application Support/Code/User/",
        "windows-pc": "%APPDATA%/Code/User/"
      },
      "excludePatterns": ["workspaceStorage/", "logs/"]
    }
  ]
}
```

## File Structure Layout

Your cloud directory will look like:

```
~/Dropbox/syncstation/          # Your cloud folder
├── sync-items.json              # Sync item definitions (shared)
├── file-metadata.json           # File hashes and sync state (shared)
└── configs/                     # Synced configuration files
    ├── Neovim-Config/          # Folder sync item
    │   ├── init.lua
    │   └── lua/
    ├── VS-Code-Settings/        # File sync item
    │   └── settings.json
    └── SSH-Keys/                # Folder with excludes
        ├── id_rsa
        └── config
```

## Troubleshooting Configuration

### Invalid JSON

If configuration files become corrupted:

```bash
# Validate JSON syntax
cat ~/.config/syncstation/config.json | jq .

# Reset local config (will prompt for setup)
rm ~/.config/syncstation/config.json
syncstation init
```

### Path Issues

```bash
# Test path expansion
syncstation status  # Shows resolved paths

# Check file/folder existence
ls -la ~/.config/nvim/  # Verify local path exists
```

### Permission Problems

```bash
# Fix config directory permissions
chmod 755 ~/.config/syncstation/
chmod 644 ~/.config/syncstation/*.json
```

### Cloud Directory Issues

```bash
# Verify cloud directory is accessible
ls -la ~/Dropbox/syncstation/

# Re-initialize if needed
syncstation init --cloud-dir ~/Dropbox/syncstation
```