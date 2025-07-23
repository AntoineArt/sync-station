# Building from Source

This guide covers building Syncstation from source code for development or custom installations.

## Prerequisites

**Required:**
- **Go 1.18 or later** - [Download from golang.org](https://golang.org/dl/)
- **Git** - For cloning the repository

**Check your Go version:**
```bash
go version
# Should show go1.18 or later
```

**No GUI dependencies required** - this is a CLI/TUI application.

## Quick Build

```bash
# Clone repository
git clone https://github.com/AntoineArt/syncstation.git
cd syncstation

# Download dependencies
go mod tidy

# Build for current platform
go build -v -o syncstation ./cmd/syncstation

# Verify build
./syncstation --version
```

## Cross-Platform Build (needs testing)

Use the included build script to create binaries for all platforms:

```bash
chmod +x scripts/build-cross-platform.sh
./scripts/build-cross-platform.sh
```

This creates binaries in the `build/` directory and distribution packages in `dist/`:

```
build/
├── syncstation-linux-amd64
├── syncstation-linux-arm64
├── syncstation-windows-amd64.exe
├── syncstation-windows-arm64.exe
├── syncstation-macos-amd64
├── syncstation-macos-arm64
└── syncstation-macos-universal    # macOS only (needs testing)

dist/
├── syncstation-1.0.0-linux-amd64.tar.gz
├── syncstation-1.0.0-windows-amd64.zip  # (needs testing)
├── syncstation-1.0.0-macos-universal.tar.gz  # (needs testing)
└── syncstation-1.0.0-checksums.sha256
```

## Manual Cross-Compilation

Build for specific platforms manually:

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

## Development Build

For development with additional debugging:

```bash
# Build with race detection
go build -race -v -o syncstation ./cmd/syncstation

# Build with debug symbols
go build -gcflags="all=-N -l" -v -o syncstation ./cmd/syncstation
```

## Installation

### System-Wide Installation

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
# Build Windows executable
GOOS=windows GOARCH=amd64 go build -v -o syncstation.exe ./cmd/syncstation

# Copy syncstation.exe to a directory in your PATH
# Or add the directory containing the executable to your PATH
```

### Go Install Method

If the repository is public, install directly with Go:

```bash
# Install latest version
go install -v github.com/AntoineArt/syncstation/cmd/syncstation@latest

# Verify installation
which syncstation
syncstation --version
```

This installs to `$GOPATH/bin` or `$HOME/go/bin`.

## Build Options

### Optimized Release Build
```bash
go build -ldflags "-s -w" -o syncstation ./cmd/syncstation
```

### With Version Information
```bash
VERSION="1.0.0"
go build -ldflags "-s -w -X main.Version=$VERSION" -o syncstation ./cmd/syncstation
```

### Static Binary (Linux)
```bash
CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -o syncstation ./cmd/syncstation
```

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

## Code Quality

### Formatting and Linting
```bash
# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Run static analysis (if staticcheck is installed)
staticcheck ./...
```

## Troubleshooting

### "no Go files" Error
```bash
# Wrong - this won't work
go build -o syncstation

# Correct - specify the cmd directory
go build -v -o syncstation ./cmd/syncstation
```

### Permission Denied After Installation
```bash
chmod +x syncstation
# or
sudo chmod +x /usr/local/bin/syncstation
```

### Go Version Too Old
Update Go to version 1.18 or later from [golang.org](https://golang.org/dl/).

### Missing Dependencies
```bash
# Download and update dependencies
go mod download
go mod tidy
```

### Build Fails on Windows
Ensure you're using a recent version of Go and try:
```bash
set CGO_ENABLED=0
go build -v -o syncstation.exe ./cmd/syncstation
```

## Uninstalling

```bash
# If installed to /usr/local/bin
sudo rm /usr/local/bin/syncstation

# If installed to ~/bin
rm ~/bin/syncstation

# If installed via go install
rm $GOPATH/bin/syncstation  # or $HOME/go/bin/syncstation
```

## Contributing

After building successfully, see [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on contributing to the project.