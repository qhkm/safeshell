# SafeShell

## What It Is

- Open-source CLI safety tool for AI agents running shell commands
- Core idea: don't restrict agents, make everything reversible
- Uses fast checkpoints (hard links) + rollback
- Zero setup, local-only, works today
- Stays free and open forever

## What It Is NOT

- Not a runtime
- Not a container platform
- Not a Daytona competitor
- Not cloud infrastructure

## Tech Stack

- Go
- Cobra CLI
- MCP (Model Context Protocol) server for AI agent integration
- Hard links for zero-disk-space checkpoints

## Commands

```bash
go build ./...           # Build
go test ./...            # Test
go test -bench=. ./...   # Benchmarks
safeshell status         # Check status
safeshell list           # List checkpoints
safeshell rollback --last # Rollback
```

## Architecture

```
internal/
├── checkpoint/   # Core checkpoint logic, index, storage
├── cli/          # Cobra commands
├── config/       # Configuration
├── mcp/          # MCP server for AI agents
├── rollback/     # Rollback logic
├── util/         # Shared utilities (formatBytes, formatTimeAgo)
└── wrapper/      # Command wrapping and parsing
```

## Strategic Direction

### Product Spectrum

```
SafeShell (OSS) ─── SafeShell Pro ─── Daytona
aliases             local sandbox      cloud VMs
90% coverage        ~100% coverage     100% coverage
zero setup          some setup         heavy infra
free                $9-29/one-time     $$$
```

### SafeShell Pro (Future)

Positioning: "Time Machine for AI agents"

Core features:
- Strong checkpoints (OverlayFS / filesystem snapshots)
- Human-readable diff (what changed, deleted, touched)
- Sessions (auto checkpoint → agent runs → diff → restore/commit)
- Soft guardrails (warn, never block)

Filesystem strategy (Linux-first):
1. Btrfs/ZFS snapshots (best when available)
2. OverlayFS (default Pro backend on Linux)
3. Hard-link checkpoints (fallback / OSS mode)

### Open Source + Pro Model

What stays open (forever):
- Core CLI
- Hard-link rollback
- MCP / agent integration
- Basic safety

What is Pro:
- OverlayFS / snapshot backends
- Sessions & lifecycle management
- Diff engine & heuristics
- Safety policies

Principle: Open the trust layer. Charge for the guarantee layer.

### What NOT to Build

- VMs
- Full process isolation
- Cloud orchestration
- Infra abstractions
- Daytona competitor

### The Niche

**You are not building infra. You are building psychological safety for developers using AI.**

That's the moat - Daytona sells to platform teams, SafeShell sells peace of mind to individual devs.

## Hard Links Limitation

SafeShell uses hard links which share the same inode. This means:
- ✅ Protects against: `rm`, `mv`, `cp`, `chmod`, `chown`
- ❌ Does NOT protect against in-place mutations: `echo >`, `truncate`, `sed -i`

This covers ~90% of real-world agent mistakes. Full coverage requires Pro (OverlayFS/snapshots).
