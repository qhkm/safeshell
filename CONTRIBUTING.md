# Contributing to SafeShell

Thank you for your interest in contributing to SafeShell! This guide will help you get started with development and understand our contribution process.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Architecture Overview](#architecture-overview)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Code Style](#code-style)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)

---

## Code of Conduct

We are committed to providing a welcoming and inclusive experience for everyone. Please:

- Be respectful and considerate
- Welcome newcomers and help them learn
- Focus on what's best for the community
- Show empathy towards others

---

## Getting Started

### Prerequisites

- **Go 1.20+** - [Install Go](https://golang.org/doc/install)
- **Git** - For version control
- **Make** - For build automation (optional but recommended)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/qhkm/safeshell.git
cd safeshell

# Install dependencies
go mod download

# Build
go build -o safeshell ./cmd/safeshell

# Run tests
go test ./...

# Install locally for testing
go install ./cmd/safeshell
```

---

## Development Setup

### 1. Fork and Clone

```bash
# Fork on GitHub, then:
git clone https://github.com/YOUR_USERNAME/safeshell.git
cd safeshell
git remote add upstream https://github.com/qhkm/safeshell.git
```

### 2. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
```

### 3. Set Up Your Environment

```bash
# Install dependencies
go mod download

# Verify everything works
go build ./...
go test ./...
```

### 4. Use a Test Configuration

During development, avoid affecting your real SafeShell installation:

```bash
# Use a test directory
export HOME=/tmp/safeshell-dev-test
mkdir -p $HOME

# Run your development build
./safeshell init
./safeshell status
```

---

## Project Structure

```
safeshell/
├── cmd/
│   └── safeshell/
│       └── main.go              # Entry point
├── internal/
│   ├── checkpoint/
│   │   ├── checkpoint.go        # Core checkpoint logic
│   │   ├── checkpoint_test.go   # Tests
│   │   ├── index.go             # Checkpoint index for fast lookups
│   │   ├── manifest.go          # Checkpoint metadata
│   │   └── storage.go           # File backup/restore operations
│   ├── cli/
│   │   ├── root.go              # Root command setup
│   │   ├── init.go              # `safeshell init`
│   │   ├── wrap.go              # `safeshell wrap <cmd>`
│   │   ├── list.go              # `safeshell list`
│   │   ├── rollback.go          # `safeshell rollback`
│   │   ├── diff.go              # `safeshell diff`
│   │   ├── search.go            # `safeshell search`
│   │   ├── tag.go               # `safeshell tag`
│   │   ├── status.go            # `safeshell status`
│   │   └── clean.go             # `safeshell clean`
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── mcp/
│   │   ├── server.go            # MCP server implementation
│   │   ├── tools.go             # MCP tool handlers
│   │   └── types.go             # MCP protocol types
│   ├── rollback/
│   │   ├── rollback.go          # Rollback logic
│   │   └── rollback_test.go     # Tests
│   └── wrapper/
│       ├── wrapper.go           # Command wrapping logic
│       ├── parser.go            # Argument parsing
│       ├── parser_test.go       # Tests
│       └── commands.go          # Supported command definitions
├── docs/
│   └── BEGINNERS_GUIDE.md       # User guide
├── scripts/
│   ├── install.sh               # Installation script
│   └── uninstall.sh             # Uninstallation script
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── CONTRIBUTING.md              # This file
└── LICENSE
```

---

## Architecture Overview

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                             │
│  (internal/cli/*.go - Cobra commands)                       │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                     Core Modules                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Wrapper    │  │  Checkpoint  │  │   Rollback   │       │
│  │   Module     │  │   Manager    │  │   Engine     │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                    Storage Layer                             │
│  (Hard links, file operations, index)                       │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow: Creating a Checkpoint

```
1. User runs: rm -rf ./build
              ↓
2. Shell alias redirects to: safeshell wrap rm -rf ./build
              ↓
3. Wrapper parses arguments, identifies targets: ["./build"]
              ↓
4. Checkpoint.Create() backs up files using hard links
              ↓
5. Original command executes via exec.Command("rm", "-rf", "./build")
              ↓
6. Index updated with new checkpoint metadata
```

### Data Flow: Rolling Back

```
1. User runs: safeshell rollback --last
              ↓
2. GetLatest() queries index for newest checkpoint
              ↓
3. Rollback.Execute() iterates manifest files
              ↓
4. For each file: copy from backup to original location
              ↓
5. Checkpoint marked as "rolled_back" in manifest
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Hard links for backup** | Zero extra disk space, instant backup |
| **Shell aliases** | No root required, works everywhere |
| **JSON manifests** | Human-readable, easy debugging |
| **Checkpoint index** | O(1) lookups instead of loading all manifests |
| **Session grouping** | Organize checkpoints by terminal session |

---

## Making Changes

### Adding a New CLI Command

1. Create `internal/cli/yourcommand.go`:

```go
package cli

import (
    "fmt"
    "github.com/spf13/cobra"
)

var yourCmd = &cobra.Command{
    Use:   "yourcommand [args]",
    Short: "Brief description",
    Long:  `Detailed description with examples.`,
    RunE:  runYourCommand,
}

func init() {
    rootCmd.AddCommand(yourCmd)
    // Add flags
    yourCmd.Flags().BoolVarP(&someFlag, "flag", "f", false, "Flag description")
}

func runYourCommand(cmd *cobra.Command, args []string) error {
    // Implementation
    return nil
}
```

2. The command is auto-registered via `init()`.

### Adding a New MCP Tool

1. Add the tool definition in `internal/mcp/server.go` in `handleListTools()`:

```go
{
    Name:        "checkpoint_yourtool",
    Description: "Description of what it does",
    InputSchema: InputSchema{
        Type: "object",
        Properties: map[string]Property{
            "param1": {
                Type:        "string",
                Description: "What this parameter does",
            },
        },
        Required: []string{"param1"},
    },
},
```

2. Add the handler in `internal/mcp/tools.go`:

```go
func (s *Server) registerTools() {
    // ... existing tools ...
    s.tools["checkpoint_yourtool"] = toolYourTool
}

func toolYourTool(args map[string]interface{}) (string, error) {
    // Implementation
    return "Success message", nil
}
```

### Modifying Checkpoint Logic

Key files:
- `internal/checkpoint/checkpoint.go` - Core operations (Create, List, Get, Delete)
- `internal/checkpoint/storage.go` - File operations (BackupFile, RestoreFile)
- `internal/checkpoint/manifest.go` - Metadata structure
- `internal/checkpoint/index.go` - Fast lookup index

### Adding Support for a New Command

1. Add command definition in `internal/wrapper/commands.go`:

```go
var supportedCommands = map[string]CommandDef{
    // ... existing commands ...
    "yourcmd": {
        Name:       "yourcmd",
        Parser:     parseYourCmdArgs,
        RiskLevel:  "medium",
    },
}
```

2. Add parser in `internal/wrapper/parser.go`:

```go
func parseYourCmdArgs(args []string) ParsedArgs {
    // Parse and return targets to backup
}
```

3. Add tests in `internal/wrapper/parser_test.go`.

---

## Testing

### Running All Tests

```bash
go test ./...
```

### Running Tests with Verbose Output

```bash
go test ./... -v
```

### Running Specific Package Tests

```bash
go test ./internal/checkpoint -v
go test ./internal/rollback -v
go test ./internal/wrapper -v
```

### Running a Specific Test

```bash
go test ./internal/checkpoint -v -run TestGetLatestCheckpoint
```

### Test Coverage

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out  # View in browser
```

### Writing Tests

Follow existing patterns:

```go
func TestYourFeature(t *testing.T) {
    // Setup test environment
    tmpDir, cleanup := setupTestEnv(t)
    defer cleanup()

    // Create test data
    testFile := filepath.Join(tmpDir, "testdata", "test.txt")
    os.WriteFile(testFile, []byte("content"), 0644)

    // Run the operation
    result, err := YourFunction(testFile)

    // Assert results
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Test Environment

Tests use isolated temp directories:

```go
func setupTestEnv(t *testing.T) (string, func()) {
    tmpDir, _ := os.MkdirTemp("", "safeshell-test-*")
    os.Setenv("HOME", tmpDir)
    config.Init()
    checkpoint.ResetIndex()  // Reset global state

    cleanup := func() {
        os.RemoveAll(tmpDir)
    }
    return tmpDir, cleanup
}
```

---

## Code Style

### Go Standards

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Use `go vet` for static analysis

### Formatting

```bash
# Format all code
gofmt -w .

# Check formatting without changing
gofmt -d .
```

### Linting (Optional but Recommended)

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Packages | lowercase | `checkpoint`, `rollback` |
| Public functions | PascalCase | `CreateCheckpoint()` |
| Private functions | camelCase | `parseArgs()` |
| Constants | PascalCase or ALL_CAPS | `MaxCheckpoints` |
| Variables | camelCase | `checkpointDir` |

### Error Handling

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create checkpoint: %w", err)
}

// Good: Return early on errors
result, err := doSomething()
if err != nil {
    return err
}
// Continue with result...
```

### Comments

```go
// CreateCheckpoint creates a new checkpoint for the given files.
// It returns the checkpoint and any error encountered.
func CreateCheckpoint(command string, paths []string) (*Checkpoint, error) {
    // ...
}
```

---

## Submitting Changes

### 1. Ensure Quality

```bash
# Format code
gofmt -w .

# Run tests
go test ./...

# Build successfully
go build ./...
```

### 2. Commit Messages

Follow conventional commits:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance

Examples:
```
feat(cli): add checkpoint search command

fix(rollback): handle permission errors gracefully

docs: add contributor guide

refactor(checkpoint): optimize GetLatest with index
```

### 3. Create Pull Request

```bash
# Push your branch
git push origin feature/your-feature

# Create PR on GitHub
```

### PR Description Template

```markdown
## Description
Brief description of changes.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass locally
- [ ] Added new tests for changes
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-reviewed the code
- [ ] Comments added for complex logic
- [ ] Documentation updated if needed
```

### 4. Code Review

- Address reviewer feedback
- Keep discussions constructive
- Update PR with requested changes

---

## Release Process

### Version Numbering

We use [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Creating a Release

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create git tag:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```
4. GoReleaser handles the rest (binaries, Homebrew formula)

---

## Getting Help

- **Questions**: Open a [Discussion](https://github.com/qhkm/safeshell/discussions)
- **Bugs**: Open an [Issue](https://github.com/qhkm/safeshell/issues)
- **Security**: Email security@example.com (do not open public issues)

---

## Recognition

Contributors are recognized in:
- GitHub contributors page
- Release notes for significant contributions

Thank you for contributing to SafeShell!
