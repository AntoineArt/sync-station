# Sync Station Usage Examples

This document provides comprehensive examples and real-world workflows for using Syncstation across multiple computers.

## Table of Contents

- [Single Computer Setup](#single-computer-setup)
- [Multi-Computer Workflow](#multi-computer-workflow)
- [Common Usage Patterns](#common-usage-patterns)
- [Advanced Scenarios](#advanced-scenarios)
- [Troubleshooting Examples](#troubleshooting-examples)

## Single Computer Setup

### Initial Setup and Configuration

```bash
# 1. Navigate to your cloud sync directory
cd ~/Dropbox/syncstation  # or OneDrive, Google Drive, etc.

# 2. Initialize Sync Station
syncstation init
# Output:
# Initializing sync station...
# Cloud directory: /home/user/Dropbox/syncstation
# Computer name: work-laptop
# Config directory: /home/user/.config/syncstation
# ✓ Local config saved to: /home/user/.config/syncstation/config.json
# ✓ Cloud sync directory: /home/user/Dropbox/syncstation
# ✓ Configs directory: /home/user/Dropbox/syncstation/configs
# Setup complete! You can now add sync items with: syncstation add

# 3. Add your first configuration files
syncstation add "Neovim Config" ~/.config/nvim
# ✓ Added sync item: Neovim Config
#   Type: folder
#   Path: /home/user/.config/nvim

syncstation add "Git Config" ~/.gitconfig
# ✓ Added sync item: Git Config
#   Type: file
#   Path: /home/user/.gitconfig

syncstation add "SSH Keys" ~/.ssh --exclude "*.pub,known_hosts"
# ✓ Added sync item: SSH Keys
#   Type: folder
#   Path: /home/user/.ssh
#   Excludes: *.pub, known_hosts

# 4. Check what's configured
syncstation list
# Configured Sync Items:
# 
# 1. Neovim Config (folder)
#    work-laptop: /home/user/.config/nvim (current)
# 
# 2. Git Config (file)
#    work-laptop: /home/user/.gitconfig (current)
# 
# 3. SSH Keys (folder)
#    work-laptop: /home/user/.ssh (current)
#    Excludes: *.pub, known_hosts

# 5. Push configurations to cloud
syncstation push
# Pushing all 3 items...
# ✓ Pushed: Neovim Config (23 files)
# ✓ Pushed: Git Config (1 file)
# ✓ Pushed: SSH Keys (3 files, 2 excluded)
# 
# Push completed successfully!
```

## Multi-Computer Workflow

### Setting Up Second Computer

```bash
# On your second computer (home-desktop)

# 1. Navigate to the SAME cloud directory
cd ~/Dropbox/syncstation  # Same folder as first computer

# 2. Initialize on this computer
syncstation init
# Initializing sync station...
# Cloud directory: /home/user/Dropbox/syncstation
# Computer name: home-desktop
# Config directory: /home/user/.config/syncstation
# ✓ Local config saved to: /home/user/.config/syncstation/config.json
# ✓ Cloud sync directory: /home/user/Dropbox/syncstation
# Setup complete! You can now add sync items with: syncstation add

# 3. Check what sync items are available
syncstation list
# Found 3 sync item(s) from other computers but not configured for this computer:
# Computer: home-desktop
# Cloud Directory: /home/user/Dropbox/syncstation
# 
# Found 3 sync item(s) from other computers:
# 
# 1. Neovim Config (folder)
#    Existing paths:
#      work-laptop: /home/user/.config/nvim
#    This computer: ⚠ Not configured
# 
# 2. Git Config (file)
#    Existing paths:
#      work-laptop: /home/user/.gitconfig
#    This computer: ⚠ Not configured
# 
# 3. SSH Keys (folder)
#    Existing paths:
#      work-laptop: /home/user/.ssh
#    This computer: ⚠ Not configured
#    Excludes: *.pub, known_hosts
# 
# ⚠ Found 3 item(s) that need configuration on this computer.
# 
# Would you like to configure these items now? [Y/n]: y
# 
# 🔧 Configuring: Neovim Config (folder)
# Suggested path: /home/user/.config/nvim
# Enter local path for this computer [/home/user/.config/nvim]: 
# ✓ Configured Neovim Config -> /home/user/.config/nvim
# 
# 🔧 Configuring: Git Config (file)
# Suggested path: /home/user/.gitconfig
# Enter local path for this computer [/home/user/.gitconfig]: 
# ✓ Configured Git Config -> /home/user/.gitconfig
# 
# 🔧 Configuring: SSH Keys (folder)
# Suggested path: /home/user/.ssh
# Enter local path for this computer [/home/user/.ssh]: 
# ✓ Configured SSH Keys -> /home/user/.ssh
# 
# ✅ Setup complete! You can now run:
#   syncstation status  # Check sync status
#   syncstation pull    # Download configs from cloud
#   syncstation sync    # Smart sync all items

# 4. Pull configurations from cloud
syncstation pull
# Pulling all 3 items...
# ✓ Pulled: Neovim Config (23 files)
# ✓ Pulled: Git Config (1 file)
# ✓ Pulled: SSH Keys (3 files, 2 excluded)
# 
# Pull completed successfully!

# 5. Verify everything is working
syncstation status
# Sync Station Status
# Computer: home-desktop
# Cloud Directory: /home/user/Dropbox/syncstation
# Config Directory: /home/user/.config/syncstation
# 
# Sync Items (3):
# 
# 1. Neovim Config (folder)
#    Path: /home/user/.config/nvim
#    Status: ✓ Local exists
#    Cloud: ✓ Exists
# 
# 2. Git Config (file)
#    Path: /home/user/.gitconfig
#    Status: ✓ Local exists
#    Cloud: ✓ Exists
# 
# 3. SSH Keys (folder)
#    Path: /home/user/.ssh
#    Status: ✓ Local exists
#    Excludes: *.pub, known_hosts
#    Cloud: ✓ Exists
```

### Cross-Platform Setup (Windows)

```bash
# On Windows computer

# 1. Navigate to cloud directory
cd "C:\Users\user\OneDrive\syncstation"

# 2. Initialize
syncstation init
# Computer name: windows-pc

# 3. Setup with different paths
syncstation setup
# 🔧 Configuring: Neovim Config (folder)
# Suggested path: %USERPROFILE%\.config\nvim
# Enter local path for this computer [%USERPROFILE%\.config\nvim]: C:\Users\user\AppData\Local\nvim
# ✓ Configured Neovim Config -> C:\Users\user\AppData\Local\nvim
# 
# 🔧 Configuring: Git Config (file)
# Suggested path: %USERPROFILE%\.gitconfig
# Enter local path for this computer [%USERPROFILE%\.gitconfig]: 
# ✓ Configured Git Config -> C:\Users\user\.gitconfig
# 
# 🔧 Configuring: SSH Keys (folder)
# Suggested path: %USERPROFILE%\.ssh
# Enter local path for this computer [%USERPROFILE%\.ssh]: 
# ✓ Configured SSH Keys -> C:\Users\user\.ssh
```

## Common Usage Patterns

### Daily Workflow

```bash
# Before starting work - sync latest configs
syncstation sync
# Smart syncing all 3 items...
# ✓ Neovim Config: No changes
# ✓ Git Config: Updated from cloud (newer)
# ✓ SSH Keys: No changes

# After making config changes - push to cloud
syncstation push
# Pushing all 3 items...
# ✓ Neovim Config: Pushed 2 changed files
# ✓ Git Config: No changes
# ✓ SSH Keys: No changes
```

### Adding New Configurations

```bash
# Add a new config on any computer
syncstation add "VS Code Settings" ~/.config/Code/User/settings.json

# Push the new item definition and files
syncstation push

# On other computers, setup will detect the new item
syncstation setup
# Found 1 new item: VS Code Settings
```

### Selective Syncing

```bash
# Sync only specific items
syncstation sync "Neovim Config"
syncstation push "Git Config"
syncstation pull "SSH Keys"

# Use dry-run to preview changes
syncstation sync --dry-run
# [DRY RUN] Would sync:
# - Neovim Config: 2 files would be pushed to cloud
# - Git Config: No changes
# - SSH Keys: 1 file would be pulled from cloud
```

### Status Checking

```bash
# Check overall status
syncstation status

# List all configured items
syncstation list

# Use the interactive TUI for visual management
syncstation tui
```

## Advanced Scenarios

### Handling Conflicts

```bash
# When files conflict (both local and cloud modified)
syncstation sync
# Smart syncing all 3 items...
# ✓ Neovim Config: No changes
# ⚠ Git Config: Conflict detected
#   Local file: modified 2025-07-19 10:30
#   Cloud file: modified 2025-07-19 11:15
#   Recommendation: Pull from cloud (newer)
# 
# Would you like to:
# [1] Pull from cloud (overwrites local)
# [2] Push to cloud (overwrites cloud)
# [3] Skip this item
# [4] Show differences
# Choose [1-4]: 4

# View differences before deciding
# [Shows diff output]
# Choose [1-4]: 1
# ✓ Git Config: Pulled from cloud (local backed up)
```

### Excluding Files and Patterns

```bash
# Add exclusions to existing items
syncstation add "Development Config" ~/.config/dev --exclude "*.log,*.tmp,cache/*,node_modules/*"

# Complex exclusions for large directories
syncstation add "VS Code Extensions" ~/.vscode/extensions --exclude "*/logs/*,*/node_modules/*,*/out/*,*.vsix"
```

### Different Paths Per Computer

```bash
# View how paths differ across computers
syncstation list
# 1. Neovim Config (folder)
#    work-laptop: /home/user/.config/nvim (current)
#    home-desktop: /home/user/.config/nvim
#    windows-pc: C:\Users\user\AppData\Local\nvim
#    macbook: /Users/user/.config/nvim

# Setup with custom paths during configuration
syncstation setup
# 🔧 Configuring: Neovim Config (folder)
# Suggested path: ~/.config/nvim
# Enter local path for this computer [~/.config/nvim]: ~/my-custom-nvim-path
```

### Conflict Resolution

```bash
# Sync Station detects conflicts and stops to prevent data loss
# Manual resolution required for conflicts
syncstation sync
# Conflict detected for Git Config - both files modified
# Error: Both local and cloud files have been modified since last sync

# Option 1: Keep local version (push to cloud)
syncstation push "Git Config"

# Option 2: Keep cloud version (pull from cloud)  
syncstation pull "Git Config"

# Option 3: Check file differences manually first
diff ~/.gitconfig ~/Dropbox/syncstation/configs/git-config/.gitconfig
```

## Troubleshooting Examples

### Setup Issues

```bash
# Problem: "not initialized" error
syncstation setup
# Error: not initialized. Run 'syncstation init' first

# Solution: Initialize first
syncstation init

# Problem: No sync items found
syncstation setup
# No sync items found in cloud directory.
# Use 'syncstation add' to create sync items on your first computer.

# Solution: Either add items on this computer or ensure cloud directory has existing data
```

### Path Issues

```bash
# Problem: Path doesn't exist
syncstation setup
# 🔧 Configuring: Neovim Config (folder)
# Enter local path for this computer: ~/.config/nvim-new
# ⚠ Path does not exist: /home/user/.config/nvim-new
# Create directory? [y/N]: y
# ✓ Created directory: /home/user/.config/nvim-new
```

### Sync Conflicts

```bash
# Check for potential conflicts before syncing
syncstation status
# Shows current state of all items

# Use smart sync to automatically handle most cases
syncstation sync
# Handles conflicts intelligently based on timestamps and content
```

### Recovery Scenarios

```bash
# If cloud directory is accidentally deleted
# 1. Recreate from any computer that still has local configs
syncstation init --cloud-dir ~/Dropbox/syncstation-new
syncstation push  # Recreates cloud from local

# 2. On other computers, update cloud directory
syncstation init --cloud-dir ~/Dropbox/syncstation-new --force
```

## Tips and Best Practices

### Workflow Recommendations

1. **Daily routine:**
   ```bash
   syncstation sync  # Start of day
   # ... work ...
   syncstation push  # End of day
   ```

2. **Before major config changes:**
   ```bash
   syncstation push  # Backup current state
   # ... make changes ...
   syncstation push  # Save changes
   ```

3. **Setting up new computer:**
   ```bash
   cd ~/cloud-folder/syncstation
   syncstation init
   syncstation setup  # Interactive wizard
   syncstation pull   # Get all configs
   ```

### Organization Tips

- Use descriptive names for sync items
- Group related configs (e.g., "Development Tools", "Shell Config")
- Use exclusion patterns liberally for logs and temporary files
- Keep cloud directory organized with consistent naming

### Performance Tips

- Use `--dry-run` for large operations first
- Exclude large files and directories that don't need syncing
- Run `syncstation status` periodically to catch issues early

This comprehensive guide should help you effectively use Sync Station across multiple computers with various cloud storage providers and operating systems.