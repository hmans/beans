---
paths:
  - "internal/agent/**"
---

# Claude CLI Process Spawning

This codebase spawns `claude` CLI processes programmatically (see `internal/agent/`). When constructing CLI arguments:

- Use `--model <name>` to specify a model, **not** `-m`. The `-m` shorthand does not exist.
- Use `--print` (or `-p`) for non-interactive single-shot calls.
- Run `claude --help` to verify flags before using them — don't assume standard shorthand conventions apply.
