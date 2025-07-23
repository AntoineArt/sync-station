# Release Notes

## Version 1.0.0 - Initial Open Source Release (2025-01-22)

🎉 **First public release of Syncstation!**

Syncstation is a lightweight CLI/TUI application for synchronizing configuration files between multiple computers using your own cloud storage.

### ✨ Key Features

- **CLI & TUI Interface**: Fast command-line interface + beautiful interactive terminal UI
- **Cloud Agnostic**: Works with any cloud storage (Dropbox, OneDrive, Google Drive, etc.)
- **Git Mode** (needs testing): Use a git repository instead of cloud folders for version control
- **Smart Sync** (needs testing): Intelligent bidirectional sync using SHA256 hashes and timestamps
- **Multi-Platform** (needs testing): Native support for Windows, Linux, and macOS
- **Computer-Specific Paths** (needs testing): Different paths for each of your computers
- **Safety Features** (needs testing): Dry-run mode, conflict detection, hash-based verification

### 🔧 Installation

**Build from Source:**
```bash
git clone https://github.com/AntoineArt/syncstation.git
cd syncstation
go build -o syncstation ./cmd/syncstation
```

**Package Managers** (needs testing):
- Homebrew, APT, RPM, and AUR configurations included

### 📖 Quick Start

```bash
# Initialize in your cloud folder
cd ~/Dropbox
syncstation init

# Add configuration files
syncstation add "Neovim Config" ~/.config/nvim

# Launch interactive TUI or use CLI
syncstation tui
syncstation sync
```

### 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

### 📜 License

GPL v3 License - See [LICENSE](LICENSE) for details.

### 👨‍💻 Author

Created by **Artan200** (AntoineArt)