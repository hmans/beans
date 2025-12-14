![beans](https://github.com/user-attachments/assets/776f094c-f2c4-4724-9a0b-5b87e88bc50d)

[![License](https://img.shields.io/github/license/hmans/beans?style=for-the-badge)](LICENSE)
[![Release](https://img.shields.io/github/v/release/hmans/beans?style=for-the-badge)](https://github.com/hmans/beans/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/hmans/beans/test.yml?branch=main&label=tests&style=for-the-badge)](https://github.com/hmans/beans/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/hmans/beans?style=for-the-badge)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/hmans/beans?style=for-the-badge)](https://goreportcard.com/report/github.com/hmans/beans)

**Beans is an issue tracker for you, your team, and your coding agents.** Instead of tracking tasks in a separate application, Beans stores them right alongside your code. You can use the `beans` CLI to interact with your tasks, but more importantly, so can your favorite coding agent!

This gives your robot friends a juicy upgrade: now they get a complete view of your project, make suggestions for what to work on next, track their progress, create bug issues for problems they find, and more.

You've been programming all your life; now you get to be a product manager. Let's go! 🚀

## Announcement Trailer ✨

https://github.com/user-attachments/assets/dbe45408-d3ed-4681-a436-a5e3046163da

## Stability Warning ⚠️

Beans is still under heavy development, and its features and APIs may still change significantly. If you decide to use it now, please follow the release notes closely.

Since Beans emits its own prompt instructions for your coding agent, most changes will "just work"; but sometimes, we modify the schema of the underlying data files, which may require some manual migration steps. If you get caught by one of these changes, your agent will often be able to migrate your data for you:

> "The Beans data format has changed. Please migrate this project's beans to the new format."

## Features

- **Track tasks, bugs, features**, and more right alongside your code.
- **Plain old Markdown files** stored in a `.beans` directory in your project. Easy to version control, readable and editable by humans and machines alike!
- Use the `beans` CLI to create, list, view, update, and archive beans; but more importantly, **let your coding agent do it for you**!
- **Supercharge your robot friend** with full context about your project and its open tasks. A built-in **GraphQL query engine** allows your agent to get exactly the information it needs, keeping token use to a minimum.
- A beautiful **built-in** TUI for browsing and managing your beans from the terminal.
- Generates a **Markdown roadmap document** for your project from your data.

## Installation

We'll need to do three things:

1. Install the `beans` CLI tool.
2. Configure your project to use it.
3. Configure your coding agent to interact with it.

Either download Beans from the [Releases section](https://github.com/hmans/beans/releases), or install it via Homebrew:

```bash
brew install hmans/beans/beans
```

Alternatively, install directly via Go:

```bash
go install github.com/hmans/beans@latest
```

## Configure Your Project

Inside the root directory of your project, run:

```bash
beans init
```

This will create a `.beans/` directory and a `.beans.yml` configuration file at the project root. All of it is meant to be tracked in your version control system.

From this point onward, you can interact with your Beans through the `beans` CLI. To get a list of available commands:

```bash
beans help
```

But more importantly, you'll want to get your coding agent set up to use it. Let's dive in!

## Agent Configuration

We'll need to teach your coding agent that it should use Beans to track tasks, and how to do so. The exact steps will depend on which agent you're using.

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

### Other Agents

You can use Beans with other coding agents by configuring them to run `beans prime` to get the prompt instructions for task management. We'll add specific integrations for popular agents over time.

## Usage Hints

As a human, you can get an overview of the CLI's functionalities by running:

```bash
beans help
```

You might specifically be interested in the interactive TUI:

```bash
beans tui
```

**But the real power of Beans** comes from letting your coding agent manage your tasks for you.

Assuming you have integrated Beans into your coding agent correctly, it will already know how to create and manage beans for you. You can use the usual assortment of natural language inquiries. If you've just
added Beans to an existing project, you could try asking your agent to identify potential tasks and create beans for them:

> "Are there any tasks we should be tracking for this project? If so, please create beans for them."

If you already have some beans available, you can ask your agent to recommend what to work on next:

> "What should we work on next?"

You can also specifically ask it to start working on a particular bean:

> "It's time to tackle myproj-123."

## Launchers

You can configure external tools to launch from the TUI. Press `!` when viewing or selecting a bean to open the launcher picker.

**First-time setup:** If no launchers are configured, you'll be prompted to select from default options (opencode, claude, crush). Only installed tools will be pre-selected. This provides a quick start with sensible defaults.

Configure launchers in `.beans.yml`:

```yaml
launchers:
  # Simple single-line launcher
  - name: opencode
    exec: opencode -p "Work on task $BEANS_ID"
    description: "Open task in OpenCode"
  
  # Git worktree + tmux launcher
  - name: worktree-tmux
    description: "Create git worktree and open in new tmux pane"
    exec: |
      #!/bin/bash
      set -euo pipefail
      
      WORKTREE_DIR=".worktrees/$BEANS_ID"
      
      # Create worktree if it doesn't exist
      if [ ! -d "$WORKTREE_DIR" ]; then
        git worktree add "$WORKTREE_DIR" -b "task/$BEANS_ID"
      fi
      
      # Create new tmux pane with task ID as title
      tmux split-window -h -c "$BEANS_ROOT/$WORKTREE_DIR" \
        "printf '\033]2;%s\033\\' '$BEANS_ID'; opencode -p 'Work on task $BEANS_ID'; exec bash"
  
  # Kitty terminal window launcher
  - name: worktree-kitty
    description: "Create git worktree and open in new kitty window"
    exec: |
      #!/bin/bash
      set -euo pipefail
      
      WORKTREE_DIR=".worktrees/$BEANS_ID"
      
      # Create worktree if it doesn't exist
      if [ ! -d "$WORKTREE_DIR" ]; then
        git worktree add "$WORKTREE_DIR" -b "task/$BEANS_ID"
      fi
      
      # Launch kitty window with task ID as title
      kitty @ launch --type=window --title="$BEANS_ID" \
        --cwd="$BEANS_ROOT/$WORKTREE_DIR" \
        bash -c "opencode -p 'Work on task $BEANS_ID'; exec bash"
```

### How It Works

- **Single-line exec**: Runs via `sh -c`, just like a shell command. Use for simple commands.
- **Multi-line exec**: Creates a temporary executable script file. **Must start with a shebang** (`#!/bin/bash`, `#!/usr/bin/env python3`, etc.). The shebang determines which interpreter executes your script.
- **Environment variables**: All launchers receive `$BEANS_ID`, `$BEANS_ROOT`, `$BEANS_TASK`
- **Working directory**: Set to project root (`$BEANS_ROOT`)
- **Unix/Linux/macOS only**: Shebang mechanism is Unix-specific

### Examples

```yaml
# Simple tool invocation
- name: cursor
  exec: cursor "$BEANS_TASK"

# Shell command with pipes
- name: show-bean
  exec: cat "$BEANS_TASK" | less

# Multi-step bash script
- name: test-and-open
  exec: |
    #!/bin/bash
    set -e
    cd "$BEANS_ROOT"
    npm test -- "$BEANS_ID" || true
    code "$BEANS_TASK"

# Ruby script
- name: ruby-tool
  exec: |
    #!/usr/bin/env ruby
    bean_id = ENV['BEANS_ID']
    puts "Processing #{bean_id}"
```

### Environment Variables

Launchers receive these environment variables:

- `BEANS_ROOT`: Project root directory
- `BEANS_ID`: Bean ID (e.g., `abc123`)
- `BEANS_TASK`: Full path to bean file (e.g., `/path/to/project/.beans/beans-abc123.md`)

## Contributing

This project currently does not accept contributions -- it's just way too early for that!
But if you do have suggestions or feedback, please feel free to open an issue.

## License

This project is licensed under the Apache-2.0 License. See the [LICENSE](LICENSE) file for details.

## Getting in Touch

If you have any questions, suggestions, or just want to say hi, feel free to reach out to me [on Bluesky](https://bsky.app/profile/hmans.dev), or [open an issue](https://github.com/hmans/beans/issues) in this repository.
