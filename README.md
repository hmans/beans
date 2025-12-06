# Beans

Beans is an issue tracker for you, your team, and your coding agents. Instead of tracking tasks in a separate application, Beans stores them right alongside your code. You can use the `beans` CLI to interact with your tasks, but more importantly, so can your favorite coding agent!

This gives your robot friends a juicy upgrade: now they get a complete view of your project, make suggestions for what to work on next, track their progress, create bug issues for problems they find, and more.

You've been programming all your life; now you get to be a product manager. Let's go! 🚀

## Features

- Beans are just Markdown files stored in a `.beans/` directory in your project. View and edit them directly if you want!
- Use the `beans` CLI to create, list, view, update, and archive beans; or let your coding agent do it for you!
- All CLI commands have optional `--json` output for accurate machine readability. Your agent will love it.

This project was inspired by Steve Yegge's great [Beads](https://github.com/steveyegge/beads). The main differences:

- Beans is significantly simpler and more lightweight.
- Most importantly, your data is just Markdown files, readable and editable by humans and machines alike. No separate databases to sync.
- It is much more customizable, allowing you to define your own bean types, statuses, and workflows.

## Installation

Either download Beans from the [Releases section](https://github.com/hmans/beans/releases), or install it via Homebrew:

```bash
brew install hmans/beans/beans
```

Now initialize Beans in your project:

```bash
beans init
```

Get a list of available commands:

```bash
beans help
```

But more importantly, get your coding agent set up to use Beans!

## Agent Configuration

### Claude Code

Beans integrates with [Claude Code](https://claude.ai/code) via hooks. Add this to your `.claude/settings.json`:

```json
{
  // ... other settings ...
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [{ "type": "command", "command": "beans prompt" }]
      }
    ],
    "PreCompact": [
      {
        "matcher": "",
        "hooks": [{ "type": "command", "command": "beans prompt" }]
      }
    ]
  }
}
```

This runs `beans prompt` at session start and before context compaction, injecting instructions that teach Claude to use Beans for task tracking instead of its built-in TodoWrite tool.

### Other Agents

You can use Beans with other coding agents by configuring them to run `beans prompt` to get the prompt instructions for task management. We'll add specific integrations for popular agents over time.

## Contributing

This project currently does not accept contributions -- it's just way too early for that!
But if you do have suggestions or feedback, please feel free to open an issue.
