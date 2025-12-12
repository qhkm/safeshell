# SafeShell - Community Post

*Copy-paste ready for Reddit, Discord, dev.to, etc.*

---

**Hey everyone!** So here's a more technical post. Built this small tool to solve a problem that I saw happening quite frequently.

---

**SafeShell - Let AI agents run freely. Everything is reversible.**

You know that moment when your AI agent confidently runs `rm -rf ./src` and you watch your codebase vanish? Yeah, happened to me one too many times.

The common solutions didn't work for me:
- **"Confirm every action"** → Kills the whole point of autonomous agents
- **Docker sandbox** → Heavy, breaks workflows
- **Just use git** → Doesn't protect untracked files, node_modules, configs

So I built SafeShell. Dead simple idea: **checkpoint before, rollback after.**

```bash
rm -rf ./build             # agent deleted wrong folder
safeshell rollback --last  # everything back in 2 seconds
```

It wraps `rm`, `mv`, `cp`, `chmod`, `chown` with automatic backups. Zero friction - you don't even notice it's there until you need it.

**The name?** Inspired by Hugging Face's SafeTensors. They solved "pickle files can execute arbitrary code" with a safe-by-design format. Same philosophy here - don't restrict agents, just make everything reversible.

**Technical bits:**
- Shell aliases (no root, no kernel modules)
- Hard links = zero extra disk space until files change
- tar.gz compression for old checkpoints (60-80% savings)
- MCP server so agents can checkpoint proactively
- Written in Go, single binary, works on macOS/Linux

**One-liner install:**
```bash
curl -fsSL https://raw.githubusercontent.com/qhkm/safeshell/main/install.sh | bash
```

GitHub: https://github.com/qhkm/safeshell

Still early days - would love feedback, especially from folks running Claude Code, Cursor, or other AI agents locally. What other destructive patterns should I catch?

---

## Short Version (Twitter/X)

Built SafeShell - automatic checkpoints for AI agent shell commands.

Your agent runs `rm -rf ./src` by mistake? One command rollback.

- Zero config install
- Hard links (no extra disk space)
- MCP server for AI integration

Inspired by SafeTensors: don't restrict agents, make everything reversible.

github.com/qhkm/safeshell

---

## One-liner (for comments/replies)

SafeShell: automatic checkpoints before rm/mv/cp. AI agent makes a mistake → `safeshell rollback --last` → fixed. github.com/qhkm/safeshell
