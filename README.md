# SafeShell

**Let agents run freely. Everything is reversible.**

Safe shell operations with automatic checkpoints for AI agents.

## Install (One Command)

```bash
curl -fsSL https://raw.githubusercontent.com/safeshell/safeshell/main/install.sh | bash
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
safeshell list            # See all checkpoints
safeshell rollback --last # Undo the last destructive command
safeshell rollback <id>   # Rollback to specific checkpoint
safeshell status          # Show stats
safeshell clean           # Remove old checkpoints
```

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

## Alternative Install

### Homebrew (macOS/Linux)
```bash
brew install safeshell/tap/safeshell
```

### Go
```bash
go install github.com/safeshell/safeshell@latest
safeshell init
source ~/.zshrc
```

### Manual
```bash
git clone https://github.com/safeshell/safeshell
cd safeshell
make install
safeshell init
source ~/.zshrc
```

## Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/safeshell/safeshell/main/uninstall.sh | bash
```

## Config

Edit `~/.safeshell/config.yaml`:

```yaml
retention_days: 7      # Auto-delete old checkpoints
max_checkpoints: 100   # Keep last N checkpoints
exclude_paths:         # Don't backup these
  - "node_modules/*"
  - ".git/objects/*"
```

## License

MIT
