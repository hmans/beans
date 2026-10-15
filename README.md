
[![License](https://img.shields.io/github/license/plasmobit/beans?style=for-the-badge)](LICENSE)

> **Note:** This repository is a fork of [Beans](https://github.com/hmans/beans) by [Hendrik Mans](https://github.com/hmans). The original has seen no development since April 2026 (last release: v0.4.2), so this fork continues it. See [Credits](#credits).

**Beans is an issue tracker for you, your team, and your coding agents.** Instead of tracking tasks in a separate application, Beans stores them right alongside your code. You can use the `beans` CLI to interact with your tasks, but more importantly, so can your favorite coding agent!

This gives your robot friends a juicy upgrade: now they get a complete view of your project, make suggestions for what to work on next, track their progress, create bug issues for problems they find, and more.

You've been programming all your life; now you get to be a product manager. Let's go! 🚀

## Stability Warning ⚠️

Beans is still under heavy development, and its features and APIs may still change significantly. If you decide to use it now, please follow the commit history closely; this fork publishes no releases yet.

Since Beans emits its own prompt instructions for your coding agent, most changes will "just work"; but sometimes, we modify the schema of the underlying data files, which may require some manual migration steps. If you get caught by one of these changes, your agent will often be able to migrate your data for you:

```
The Beans data format has changed. Please migrate this project's beans to the new format.
```

## Features

- **Track tasks, bugs, features**, and more right alongside your code.
- **Plain old Markdown files** stored in a `.beans` directory in your project. Easy to version control, readable and editable by humans and machines alike!
- Use the `beans` CLI to create, list, view, update, and archive beans; but more importantly, **let your coding agent do it for you**!
- **Supercharge your robot friend** with full context about your project and its open tasks. A built-in **GraphQL query engine** allows your agent to get exactly the information it needs, keeping token use to a minimum.
- **Project memory**: `beans archive` moves completed and scrapped beans to `.beans/archive/`, where they serve as project memory that your coding agent can query for context about past work.
- A beautiful **TUI** (`beans-tui`) for browsing and managing your beans from the terminal.
- A **web UI** (`beans-serve`) with a backlog board, per-bean agent chat (Claude Code), git worktree management, file change diffs, and embedded terminals.
- Generates a **Markdown roadmap document** for your project from your data.

## Installation

We'll need to do three things:

1. Install the `beans` CLI tool.
2. Configure your project to use it.
3. Configure your coding agent to interact with it.

This fork publishes no releases yet, so build it from source. The build needs Go, Node.js with corepack (pnpm runs through it), and [just](https://just.systems/):

```bash
git clone https://github.com/plasmobit/beans.git
cd beans

just setup && just install
```

This builds three binaries and copies them to `~/.local/bin/`:

- `beans`: the CLI
- `beans-tui`: the terminal UI
- `beans-serve`: the web UI

The releases and the Homebrew tap of the original project install upstream v0.4.2, which lacks this fork's changes and still ships the TUI as `beans tui`.

## Configure Your Project

Inside the root directory of your project, run:

```bash
beans init
```

This will create a `.beans/` directory and a `.beans.yml` configuration file at the project root. All of it is meant to be tracked in your version control system, except the agent chat logs in `.beans/.conversations/`, which `beans init` lists in `.beans/.gitignore`.

From this point onward, you can interact with your Beans through the `beans` CLI. To get a list of available commands:

```bash
beans help
```

But more importantly, you'll want to get your coding agent set up to use it. Let's dive in!

## Agent Configuration

The most basic way to teach your agent about Beans is to simply add the following instruction to your `AGENTS.md`, `CLAUDE.md`, or equivalent file:

```
**IMPORTANT**: before you do anything else, run the `beans prime` command and heed its output.
```

Some agents provide mechanisms to automate this step:

### Claude Code

An official Beans plugin for Claude is in the works, but for the time being, please manually add the following hooks to your project's `.claude/settings.json` file:

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [{ "type": "command", "command": "beans prime" }] }
    ],
    "PreCompact": [
      { "hooks": [{ "type": "command", "command": "beans prime" }] }
    ]
  }
}
```

### OpenCode

Beans integrates with OpenCode via a plugin that injects task context into your sessions. To set it up, **copy the plugin** from [`.opencode/plugin/beans-prime.ts`](.opencode/plugin/beans-prime.ts) to your project's `.opencode/plugins/` directory (or `~/.config/opencode/plugins/` for global availability across all projects).

## Usage Hints

As a human, you can get an overview of the CLI's functionalities by running:

```bash
beans help
```

You might specifically be interested in the interactive TUI:

```bash
beans-tui
```

Or in the web UI, served on the port that `server.port` in `.beans.yml` sets (8080 by default; `--port` overrides it):

```bash
beans-serve
```

### Example Workflows

**But the real power of Beans** comes from letting your coding agent manage your tasks for you.

Assuming you have integrated Beans into your coding agent correctly, it will already know how to create and manage beans for you. You can use the usual assortment of natural language inquiries. If you've just
added Beans to an existing project, you could try asking your agent to identify potential tasks and create beans for them:

```
Are there any tasks we should be tracking for this project? If so, please create beans for them.
```

If you already have some beans available, you can ask your agent to recommend what to work on next:

```
What should we work on next?
```

You can also specifically ask it to start working on a particular bean:

```
It's time to tackle myproj-123.
```

Consider that your agent will be just as capable to deal with beans as it is with code, so how about using it to quickly restructure your tasks?

```
Please inspect this project's beans and reorganize them into epics. Also please create 2-3 milestones to group these epics in a meaningful way.
```

You can also add Beans-specific instructions to your `AGENTS.md`, `CLAUDE.md` or equivalent file, for example:

```
When making a commit, include the relevant bean IDs in the commit message
```

## Credits

Beans was created by [Hendrik Mans](https://github.com/hmans). This fork builds on the original project, [hmans/beans](https://github.com/hmans/beans); all credit for the original design and implementation goes to him and its contributors.

## License

This project is licensed under the Apache-2.0 License. See the [LICENSE](LICENSE) file for details.
