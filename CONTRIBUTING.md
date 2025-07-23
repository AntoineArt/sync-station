# Contributing to Syncstation

Thank you for your interest in contributing to Syncstation! This document provides guidelines for contributing to the project.

## 🚀 Quick Start

1. **Fork the repository**
2. **Clone your fork**
   ```bash
   git clone https://github.com/YourUsername/syncstation.git
   cd syncstation
   ```
3. **Install dependencies**
   ```bash
   go mod tidy
   ```
4. **Build and test**
   ```bash
   go build -v ./cmd/syncstation
   go test ./...
   ```

## 🔧 Development Setup

### Prerequisites

- **Go 1.18+** - [Download here](https://golang.org/dl/)
- **Git** - For version control
- **Your favorite editor** - VS Code, GoLand, Vim, etc.

### Project Structure

```
syncstation/
├── cmd/syncstation/       # Main application entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── sync/             # Sync operations and logic
│   ├── tui/              # Terminal UI components
│   └── diff/             # File comparison utilities
├── docs/                 # Documentation
├── packaging/            # Package manager configurations
└── scripts/              # Build and utility scripts
```

## 🎯 How to Contribute

### 🐛 Bug Reports

Use our [bug report template](.github/ISSUE_TEMPLATE/bug_report.yml) and include:

- **OS and version** (Windows 11, macOS 14, Ubuntu 22.04, etc.)
- **Syncstation version** (`syncstation --version`)
- **Steps to reproduce** the issue
- **Expected vs actual behavior**
- **Relevant log output** or error messages

### ✨ Feature Requests

Use our [feature request template](.github/ISSUE_TEMPLATE/feature_request.yml) and include:

- **Clear description** of the proposed feature
- **Use cases** - how would this help users?
- **Implementation ideas** (if you have any)
- **Alternatives considered**

### 🔀 Pull Requests

1. **Check existing issues** - see if your idea is already being discussed
2. **Create an issue first** for significant changes to discuss the approach
3. **Follow the code style** - use `gofmt` and follow Go conventions
4. **Write tests** - ensure your changes don't break existing functionality
5. **Update documentation** - including README and code comments
6. **Test across platforms** - at minimum test on your development platform

#### PR Guidelines

- **One feature per PR** - keep changes focused and reviewable
- **Clear commit messages** - use conventional commit format when possible
- **Reference issues** - link to related issue(s) in the PR description
- **Update CHANGELOG.md** if applicable

## 🧪 Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run specific package tests
go test ./internal/sync/
```

### Manual Testing

```bash
# Build the binary
go build -o syncstation ./cmd/syncstation

# Test basic functionality
./syncstation --version
./syncstation --help

# Test with a temporary cloud directory
mkdir -p /tmp/test-syncstation
cd /tmp/test-syncstation
../syncstation init
../syncstation add "Test Config" ~/.bashrc 
../syncstation list
```

## 📝 Code Style

### Go Standards

- Follow standard Go formatting with `gofmt`
- Use `golint` and `go vet` for code quality
- Write idiomatic Go code
- Add comments for exported functions and types
- Keep functions focused and reasonably sized

### Naming Conventions

- **Files**: `snake_case.go`
- **Types**: `PascalCase`
- **Functions/Variables**: `camelCase`
- **Constants**: `PascalCase` or `SCREAMING_SNAKE_CASE` for package-level

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Use meaningful error messages
if !fileExists(path) {
    return fmt.Errorf("config file not found at %s", path)
}
```

## 🗺️ Development Roadmap

See our [issues](https://github.com/AntoineArt/syncstation/issues) and [project boards](https://github.com/AntoineArt/syncstation/projects) for current priorities.

### High Priority Areas

- **Cross-platform compatibility** improvements
- **Performance optimization** for large file sets
- **Git integration** enhancements
- **TUI/CLI feature parity**
- **Package manager** distribution

## 🤝 Community

- **Discussions**: Use [GitHub Discussions](https://github.com/AntoineArt/syncstation/discussions) for questions
- **Issues**: Use GitHub Issues for bugs and feature requests
- **Be respectful**: Follow our [Code of Conduct](CODE_OF_CONDUCT.md)

## 🏷️ Release Process

1. **Version updates** follow [semantic versioning](https://semver.org/)
2. **Changelog updates** for each release
3. **Cross-platform builds** automatically generated
4. **Package manager updates** for major releases

## 📚 Resources

- **Go Documentation**: https://golang.org/doc/
- **Go by Example**: https://gobyexample.com/
- **Effective Go**: https://golang.org/doc/effective_go.html

## ❓ Questions?

- Check the [FAQ](https://github.com/AntoineArt/syncstation/discussions/categories/q-a)
- Start a [discussion](https://github.com/AntoineArt/syncstation/discussions)
- Look at existing [issues](https://github.com/AntoineArt/syncstation/issues)

---

**Thank you for contributing to Syncstation!** 🎉