# SafeShell Beginner's Guide

Welcome to SafeShell! This guide will help you understand how SafeShell protects your files and how to use it effectively.

## Table of Contents

- [What is SafeShell?](#what-is-safeshell)
- [Installation](#installation)
- [Your First Checkpoint](#your-first-checkpoint)
- [Basic Commands](#basic-commands)
- [Understanding Checkpoints](#understanding-checkpoints)
- [How It Works Under the Hood](#how-it-works-under-the-hood)
- [Common Scenarios](#common-scenarios)
- [Tips and Best Practices](#tips-and-best-practices)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

---

## What is SafeShell?

SafeShell is a safety net for your command line. It automatically creates backups (called "checkpoints") before you run potentially destructive commands like `rm`, `mv`, or `chmod`.

### The Problem It Solves

Have you ever accidentally deleted the wrong file?

```bash
rm -rf ./build    # Oops! I meant ./dist
```

With SafeShell, you can recover instantly:

```bash
safeshell rollback --last
# Your files are back!
```

### How It Works (Simple Explanation)

1. You type `rm important-file.txt`
2. SafeShell intercepts this command
3. SafeShell backs up `important-file.txt`
4. The original `rm` command runs
5. If you made a mistake, run `safeshell rollback --last`

**The backup uses "hard links"** - this means it doesn't use extra disk space!

---

## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/qhkm/safeshell/main/install.sh | bash
```

Then restart your terminal or run:
```bash
source ~/.zshrc   # for Zsh
source ~/.bashrc  # for Bash
```

### Verify Installation

```bash
safeshell status
```

You should see output like:
```
SafeShell Status
────────────────
Total checkpoints: 0
Storage used:      0 B
Config file:       ~/.safeshell/config.yaml
```

### What Gets Installed?

- The `safeshell` binary in `/usr/local/bin/`
- Shell aliases for `rm`, `mv`, `cp`, `chmod`, `chown`
- Configuration folder at `~/.safeshell/`

---

## Your First Checkpoint

Let's create a test file and see SafeShell in action:

### Step 1: Create a Test File

```bash
echo "Hello, World!" > test-file.txt
cat test-file.txt
# Output: Hello, World!
```

### Step 2: Delete It (SafeShell Will Protect It)

```bash
rm test-file.txt
# [safeshell] Checkpoint created: 2024-12-12T143022-a1b2c3
```

Notice the message? SafeShell automatically created a backup!

### Step 3: Verify the File is Gone

```bash
cat test-file.txt
# cat: test-file.txt: No such file or directory
```

### Step 4: Bring It Back!

```bash
safeshell rollback --last
# Successfully restored 1 files from checkpoint 2024-12-12T143022-a1b2c3
```

### Step 5: Verify It's Restored

```bash
cat test-file.txt
# Output: Hello, World!
```

Your file is back!

---

## Basic Commands

### List All Checkpoints

```bash
safeshell list
```

Output:
```
ID                          TIME              FILES  SIZE     COMMAND
─────────────────────────────────────────────────────────────────────
2024-12-12T143022-a1b2c3   5 minutes ago     3      1.2 KB   rm -rf ./build
2024-12-12T141511-d4e5f6   2 hours ago       1      256 B    rm config.json
```

### Rollback to the Last Checkpoint

```bash
safeshell rollback --last
```

### Rollback to a Specific Checkpoint

```bash
safeshell rollback 2024-12-12T143022-a1b2c3
```

### See What Would Be Restored (Diff)

Before rolling back, preview what will change:

```bash
safeshell diff --last
```

Output:
```
Checkpoint: 2024-12-12T143022-a1b2c3
Command:    rm -rf ./build
Time:       2024-12-12 14:30:22

Summary:
  • 3 file(s) deleted - will be restored
  • Total restore size: 1.2 KB

Files to restore:
  + build/index.js (512 B)
  + build/styles.css (384 B)
  + build/app.js (320 B)
```

### Check SafeShell Status

```bash
safeshell status
```

### Clean Up Old Checkpoints

```bash
safeshell clean                    # Remove checkpoints older than 7 days
safeshell clean --older-than 3d    # Remove checkpoints older than 3 days
safeshell clean --keep 10          # Keep only the last 10 checkpoints
```

### Disable/Enable SafeShell

```bash
safeshell disable   # Remove aliases, use normal binaries
safeshell enable    # Re-enable SafeShell protection
```

Your checkpoints remain intact when disabled.

---

## Understanding Checkpoints

### What is a Checkpoint?

A checkpoint is a snapshot of files at a specific moment in time. Each checkpoint contains:

- **ID**: Unique identifier (timestamp + random string)
- **Command**: The command that triggered the checkpoint
- **Files**: Backed up files and their original locations
- **Timestamp**: When the checkpoint was created

### Where Are Checkpoints Stored?

```
~/.safeshell/
├── config.yaml           # Your settings
└── checkpoints/
    ├── 2024-12-12T143022-a1b2c3/
    │   ├── manifest.json  # Metadata about the checkpoint
    │   └── files/         # Backed up files
    └── 2024-12-12T141511-d4e5f6/
        ├── manifest.json
        └── files/
```

### Checkpoint Lifecycle

```
Command executed → Checkpoint created → Files backed up
                                              ↓
                         Need to rollback? → safeshell rollback
                                              ↓
                                         Files restored!
```

---

## How It Works Under the Hood

### The Magic: Hard Links & Inodes

SafeShell uses **hard links** for backups, which means **zero extra disk space** for most backups.

#### What's an Inode?

Every file on disk has an **inode** (index node) - a unique ID that points to the actual data:

```
Filename        Inode       Data on disk
────────────────────────────────────────
report.txt  →   #12345  →   [Hello World...]
photo.jpg   →   #12346  →   [FFD8FFE0...]
```

#### Normal Copy vs Hard Link

**Normal copy** = new inode, duplicate data:
```
report.txt      →  inode #12345  →  [Hello World...]  ← 100KB
report_copy.txt →  inode #67890  →  [Hello World...]  ← 100KB (duplicate!)

Total: 200KB used
```

**Hard link** = same inode, same data:
```
report.txt  →  inode #12345  →  [Hello World...]  ← 100KB
backup.txt  →  inode #12345  ↗   (points to same!)

Total: 100KB used (zero extra!)
```

Both filenames point to the **same data on disk**. Not a copy - literally the same bytes.

### Step-by-Step: What Happens When You Delete a File

#### Step 1: You have a file
```
Filesystem:
┌─────────────────────────────────────────┐
│  report.txt  →  inode #12345  →  [data] │
│                 link count: 1           │
└─────────────────────────────────────────┘

Disk usage: 100KB
```

#### Step 2: You run `rm report.txt`

SafeShell intercepts and creates a hard link backup **before** deletion:
```
Filesystem:
┌─────────────────────────────────────────────────────────┐
│  report.txt                →  inode #12345  →  [data]  │
│  ~/.safeshell/.../backup   →  inode #12345  ↗          │
│                               link count: 2            │
└─────────────────────────────────────────────────────────┘

Disk usage: 100KB (still the same!)
```

#### Step 3: Real `rm` executes
```
Filesystem:
┌─────────────────────────────────────────────────────────┐
│  report.txt                   ❌ DELETED                │
│  ~/.safeshell/.../backup   →  inode #12345  →  [data]  │
│                               link count: 1            │
└─────────────────────────────────────────────────────────┘

Disk usage: 100KB (data survives because backup still points to it!)
```

#### Step 4: Rollback with `safeshell rollback --last`
```
SafeShell restores the file:

┌─────────────────────────────────────────────────────────┐
│  report.txt                →  inode #12345  →  [data]  │
│  ~/.safeshell/.../backup   →  inode #12345  ↗          │
│                               link count: 2            │
└─────────────────────────────────────────────────────────┘

✓ File restored to original location!
```

### The Complete Flow

```
You type:     rm report.txt
                   │
                   ▼
              ┌─────────────────────────────────┐
              │         SHELL ALIAS             │
              │  rm → safeshell wrap rm         │
              └─────────────────────────────────┘
                   │
                   ▼
              ┌─────────────────────────────────┐
              │        SAFESHELL WRAP           │
              │                                 │
              │  1. Parse command arguments     │
              │  2. Identify target files       │
              │  3. Create checkpoint:          │
              │     - Generate unique ID        │
              │     - Hard link files to backup │
              │     - Save manifest.json        │
              │  4. Execute real /bin/rm        │
              └─────────────────────────────────┘
                   │
                   ▼
              File deleted, but backup exists
                   │
            ┌──────┴──────┐
            │             │
         All good    Made a mistake?
            │             │
            ▼             ▼
          Done    safeshell rollback --last
                          │
                          ▼
                    ✓ Restored!
```

### Why Hard Links Are Brilliant

| Aspect | Regular Copy | Hard Link |
|--------|--------------|-----------|
| Disk space | 2x (duplicate) | 0 extra |
| Speed | Slow (copy bytes) | Instant |
| Data safety | Independent | Shared until modified |

**The catch:** Hard links only work on the same filesystem. If your file is on a different drive, SafeShell falls back to a regular copy.

### When Does Extra Space Get Used?

1. **Cross-filesystem backups** - Falls back to copying
2. **After rollback** - The restored file may use new space if original inode is gone
3. **Compressed checkpoints** - Uses space for the archive, but saves space long-term

---

## Common Scenarios

### Scenario 1: Accidentally Deleted a File

```bash
# Oops!
rm important-report.pdf

# Get it back
safeshell rollback --last
```

### Scenario 2: Deleted the Wrong Directory

```bash
# Meant to delete ./temp but deleted ./src
rm -rf ./src

# Recover
safeshell rollback --last
```

### Scenario 3: Moved a File to Wrong Location

```bash
# Moved to wrong place
mv config.json /some/wrong/path/

# Recover
safeshell rollback --last
```

### Scenario 4: Want to See What Happened Before Restoring

```bash
# First, check what the rollback will do
safeshell diff --last

# See content changes
safeshell diff --last --content

# If it looks right, rollback
safeshell rollback --last
```

### Scenario 5: Rollback a Specific File Only

```bash
# Only restore one file from the checkpoint
safeshell rollback --last --files "src/config.json"
```

### Scenario 6: Find an Old Checkpoint by File Name

```bash
# Search for checkpoints containing a specific file
safeshell search --file "important.txt"

# Search by the command that was run
safeshell search --command "rm -rf"

# Search by date
safeshell search --after 2024-12-01
```

### Scenario 7: Tag Important Checkpoints

```bash
# Add a tag to remember why this checkpoint matters
safeshell tag --last "before-refactor"

# Add a note
safeshell tag --last --note "Working state before major changes"

# Search by tag later
safeshell search --tag "before-refactor"
```

---

## Tips and Best Practices

### 1. Check Before Rolling Back

Always run `safeshell diff` before `safeshell rollback`:

```bash
safeshell diff --last          # See what will change
safeshell diff --last --content # See actual content differences
```

### 2. Use Tags for Important Checkpoints

Before making big changes, tag your checkpoint:

```bash
rm old-config.json  # Creates checkpoint
safeshell tag --last "before-config-update"
```

### 3. Regular Cleanup

Old checkpoints take disk space. Clean them periodically:

```bash
safeshell clean --older-than 7d
```

### 4. Selective Rollback

You don't have to restore everything:

```bash
# Only restore specific files
safeshell rollback --last --files "src/main.js,src/config.json"
```

### 5. Restore to Different Location

Want to compare before overwriting? Restore to a different path:

```bash
safeshell rollback --last --to ./recovered/
```

### 6. Preview with Dry-Run

See what SafeShell would backup without actually doing anything:

```bash
safeshell wrap --dry-run rm -rf ./build
```

---

## Troubleshooting

### "Command not found: safeshell"

Your shell hasn't loaded the new path. Run:
```bash
source ~/.zshrc   # for Zsh
source ~/.bashrc  # for Bash
```

Or restart your terminal.

### "No checkpoints found"

Either:
- No destructive commands have been run yet
- Checkpoints were cleaned up
- You're in a different shell session

Run `safeshell list` to see all checkpoints.

### Checkpoint Not Created for a Command

SafeShell only protects aliased commands. Check if aliases are active:
```bash
alias rm    # Should show: rm='safeshell wrap rm'
```

If not, run:
```bash
safeshell init
source ~/.zshrc
```

### Files Not Fully Restored

Check if files were excluded in config (`~/.safeshell/config.yaml`):
```yaml
exclude_paths:
  - "node_modules/*"
  - ".git/objects/*"
```

### "Permission denied" Errors

SafeShell can't backup files you don't have permission to read:
```bash
# This won't work if you can't read the file
rm /root/some-protected-file
```

---

## FAQ

### Q: Does SafeShell use extra disk space?

**A:** Minimal! SafeShell uses **hard links** when possible. A hard link is like a second name for the same file - it doesn't duplicate the data. Extra space is only used if:
- Files are on different filesystems (then it copies)
- The original file is modified after backup

### Q: Which commands are protected?

**A:** By default: `rm`, `mv`, `cp`, `chmod`, `chown`

### Q: Can I add more commands?

**A:** Yes! Add aliases manually:
```bash
echo "alias mycommand='safeshell wrap mycommand'" >> ~/.zshrc
```

### Q: Does it slow down my terminal?

**A:** Barely noticeable. Checkpoint creation adds ~10-50ms to each command.

### Q: Is it safe to use with git?

**A:** Yes! SafeShell doesn't interfere with git. It just backs up files before you delete/move them.

### Q: How do I completely disable SafeShell temporarily?

**A:** Use the `disable` command:
```bash
safeshell disable   # Removes aliases, reverts to normal binaries
source ~/.zshrc     # Apply changes

# Later, re-enable:
safeshell enable    # Adds aliases back
source ~/.zshrc
```

Or bypass aliases for a single command:
```bash
/bin/rm file.txt  # Bypasses SafeShell for this command only
```

### Q: Can I use SafeShell with Docker?

**A:** SafeShell works in your local shell. Inside Docker containers, you'd need to install it separately.

### Q: What happens if I run out of disk space?

**A:** SafeShell will warn you and may fail to create checkpoints. Run `safeshell clean` to free space.

---

## Next Steps

- Read the [full command reference](../README.md#commands)
- Set up [MCP integration](../README.md#mcp-integration-claude-code--others) for AI agents
- Customize your [configuration](../README.md#config)
- Want to contribute? See the [Contributing Guide](../CONTRIBUTING.md)

---

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/qhkm/safeshell/issues)
- **Discussions**: [GitHub Discussions](https://github.com/qhkm/safeshell/discussions)

Happy safe shelling!
