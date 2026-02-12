# beans-prime

A Claude Code plugin that integrates [Beans](https://github.com/hmans/beans) into your coding sessions. It runs `beans prime` automatically at session start and before context compaction, giving Claude full context about your project's tasks.

## Installation

```
/plugin marketplace add hmans/beans
/plugin install beans-prime@beans
```

## What it does

This plugin registers hooks for the following Claude Code events:

- **SessionStart**: Runs `beans prime` when a new session begins, providing Claude with your project's task context and instructions.
- **PreCompact**: Runs `beans prime` before context compaction, ensuring task context is preserved when the conversation is compressed.
