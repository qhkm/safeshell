# SafeShell

**Let agents run freely. Everything is reversible.**

Safe shell operations with automatic checkpoints for AI agents.

## Install (One Command)

```bash
curl -fsSL https://raw.githubusercontent.com/qhkm/safeshell/main/install.sh | bash
```

Then restart your terminal. **That's it.**

## What It Does

AI agents (Claude Code, Cursor, etc.) run shell commands. Sometimes they make mistakes:

```bash
rm -rf ./src          # Oops, wrong directory
mv config.json old/   # Overwrote something important
```

**SafeShell catches these** by creating automatic backups before any destructive command:

```bash
rm important-file.txt
# [safeshell] Checkpoint created: 2024-12-12T143022-a1b2c3

# Oh no! Get it back:
safeshell rollback --last
# ✓ Rollback complete!
```

## Commands

```bash
# Core
safeshell list              # See all checkpoints
safeshell rollback --last   # Undo the last destructive command
safeshell rollback <id>     # Rollback to specific checkpoint
safeshell status            # Show stats

# Cleanup
safeshell clean             # Remove old checkpoints (based on retention_days)
safeshell clean --keep 10   # Keep only 10 most recent
safeshell clean --older-than 3d  # Remove checkpoints older than 3 days

# Configuration
safeshell config            # View all settings
safeshell config get <key>  # Get a setting
safeshell config set <key> <value>  # Change a setting

# Automatic cleanup
safeshell schedule          # View schedule status
safeshell schedule enable   # Enable daily auto-cleanup (midnight)
safeshell schedule enable --hourly --keep 20  # Hourly, keep 20
safeshell schedule disable  # Disable auto-cleanup

# Setup
safeshell disable           # Revert to normal binaries
safeshell enable            # Re-enable SafeShell protection
safeshell upgrade           # Upgrade to latest version
```

## Why This Approach?

We evaluated several approaches to make AI agents safer:

| Approach | How It Works | Pros | Cons |
|----------|--------------|------|------|
| **Shell Aliases** ✅ | Replace `rm`/`mv` with aliases that checkpoint first | Simple, no root needed, works everywhere | Only catches aliased commands |
| **Filesystem Sandbox** | Run agent in Docker/container with copy-on-write | Complete isolation | Heavy, complex setup, breaks some workflows |
| **FUSE Filesystem** | Virtual filesystem that intercepts all writes | Catches everything transparently | Requires kernel modules, significant overhead |
| **Git-based** | Auto-commit before every operation | Simple, familiar tooling | Only works for text files in git repos |
| **Btrfs/ZFS Snapshots** | OS-level filesystem snapshots | Very efficient, catches everything | Requires specific filesystem, root access |

**We chose Shell Aliases** because:
- ✅ Zero setup complexity (one curl command)
- ✅ No root/sudo required
- ✅ Works on any macOS/Linux system
- ✅ Minimal performance overhead
- ✅ Easy to understand and debug
- ✅ Hard links = zero extra disk space

The tradeoff is that only aliased commands are protected. But in practice, `rm`, `mv`, `cp`, `chmod`, and `chown` cover 95% of destructive operations that AI agents perform.

## How It Works

```
You run:  rm -rf ./build
              ↓
SafeShell:  Backup files → Execute command
              ↓
Mistake?    safeshell rollback --last
              ↓
Files:      Restored ✓
```

**Zero overhead**: Uses hard links (same inode, no extra disk space).

## Protected Commands

| Command | What's Saved |
|---------|--------------|
| `rm` | Files/dirs being deleted |
| `mv` | Source files before move |
| `cp` | Destination if overwriting |
| `chmod` | Original permissions |
| `chown` | Original ownership |

## For AI Agents

Add to your system prompt:

```
SafeShell is installed. If you make a destructive mistake, run:
safeshell rollback --last
```

## MCP Integration (Claude Code & Others)

SafeShell includes an MCP (Model Context Protocol) server that lets AI agents interact with checkpoints directly - no shell commands needed.

### Setup for Claude Code

Add to your Claude Code MCP settings (`~/.claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "safeshell": {
      "command": "safeshell",
      "args": ["mcp"]
    }
  }
}
```

### MCP Tools Available

| Tool | Description |
|------|-------------|
| `checkpoint_create` | Create a checkpoint BEFORE risky operations |
| `checkpoint_list` | List all available checkpoints |
| `checkpoint_rollback` | Rollback to a checkpoint (use `id: "latest"` for most recent) |
| `checkpoint_status` | Get SafeShell status and statistics |
| `checkpoint_delete` | Delete a specific checkpoint |

### Example Agent Workflow

```
Agent: "I need to delete the build folder. Let me create a checkpoint first."
→ Uses checkpoint_create(paths: ["./build"], reason: "before cleanup")

Agent: "Now deleting..."
→ Runs rm -rf ./build

Agent: "Oops, that was the wrong folder!"
→ Uses checkpoint_rollback(id: "latest")

Agent: "Files restored. Let me try again with the correct path."
```

### Why MCP?

- **Proactive safety**: Agent creates checkpoint BEFORE destructive operations
- **Direct integration**: No shell parsing needed
- **Rich context**: Include reasons for checkpoints
- **Better control**: Query what's protected, selective rollback

## Alternative Install

### Homebrew (macOS/Linux)
```bash
brew install safeshell/tap/safeshell
```

### Go
```bash
go install github.com/qhkm/safeshell@latest
safeshell init
source ~/.zshrc
```

### Manual
```bash
git clone https://github.com/qhkm/safeshell
cd safeshell
make install
safeshell init
source ~/.zshrc
```

## Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/qhkm/safeshell/main/uninstall.sh | bash
```

## Config

View and modify settings with `safeshell config`:

```bash
safeshell config                        # View all
safeshell config set retention_days 3   # Change cleanup threshold
safeshell config set max_storage_mb 2000  # Limit to 2GB
```

Or edit `~/.safeshell/config.yaml` directly:

```yaml
# Storage limits
max_storage_mb: 5000       # Total storage limit (default: 5GB)
max_file_size_mb: 100      # Skip files larger than this (default: 100MB)
max_checkpoints: 100       # Maximum checkpoints to keep

# Cleanup
retention_days: 7          # 'safeshell clean' removes older than this

# Security
warn_sensitive_files: true # Warn when backing up .env, *.pem, etc.
sensitive_patterns:        # Patterns that trigger warnings
  - ".env"
  - "*.pem"
  - "*.key"
  - "id_rsa"

# Exclusions (never backed up)
exclude_paths:
  - "node_modules/*"
  - ".git/objects/*"
  - "*.tmp"

# Commands that trigger automatic checkpoints
wrapped_commands:
  - rm
  - mv
  - cp
  - chmod
  - chown
```

## Documentation

- **[Beginner's Guide](docs/BEGINNERS_GUIDE.md)** - Complete guide for new users
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to SafeShell

## The Story Behind the Name

The name **SafeShell** is inspired by [SafeTensors](https://github.com/huggingface/safetensors) from Hugging Face.

SafeTensors solved a critical problem in ML: pickle files could execute arbitrary code when loaded, making model sharing dangerous. Their solution? A simple, safe format that just stores tensors - nothing executable, nothing dangerous.

We saw a parallel problem emerging with AI agents. As agents like Claude Code and Cursor gain the ability to execute shell commands, they can accidentally cause real damage - deleted files, overwritten configs, broken systems. The traditional approach of "ask permission for everything" breaks the flow and defeats the purpose of autonomous agents.

**SafeShell applies the same philosophy**: instead of restricting what agents can do, make everything safely reversible. Let them run freely, experiment boldly, and fix mistakes instantly.

Just as SafeTensors made ML model sharing safe by design, SafeShell makes AI agent operations safe by design.

*"The best safety is invisible safety."*

## License

Apache 2.0
