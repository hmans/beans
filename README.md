![beans](https://github.com/user-attachments/assets/776f094c-f2c4-4724-9a0b-5b87e88bc50d)

**Beans is an issue tracker for you, your team, and your coding agents.** Instead of tracking tasks in a separate application, Beans stores them right alongside your code. You can use the `beans` CLI to interact with your tasks, but more importantly, so can your favorite coding agent!

This gives your robot friends a juicy upgrade: now they get a complete view of your project, make suggestions for what to work on next, track their progress, create bug issues for problems they find, and more.

You've been programming all your life; now you get to be a product manager. Let's go! 🚀

> **Warning**: This project is in a very early state. There **will** be breaking changes. Use at your own risk!

## Features

- Track tasks, bugs, features, and more right alongside your code.
- Plain old Markdown files stored in a `.beans/` directory in your project. View and edit them directly if you want!
- Use the `beans` CLI to create, list, view, update, and archive beans; but more importantly, let your coding agent do it for you!
- Supercharge your robot friend with full context about your project and its open tasks.
- At your request, generates a Markdown roadmap for your project, structured by milestones and epics.

This project was inspired by Steve Yegge's great [Beads](https://github.com/steveyegge/beads). The main differences:

- Beans is significantly simpler and more lightweight. Its simplicity is a feature.
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

This will create a `.beans/` directory in your project and a `.beans.yml` configuration file at the project root. Everything is meant to be tracked in your version control system.

You can interact with your Beans through the `beans` CLI. To get a list of available commands:

```bash
beans help
```

But more importantly, you'll want to get your coding agent set up to use it. Let's dive in!

## Agent Configuration

We'll need to teach your coding agent that it should use Beans to track tasks, and how to do so. The exact steps will depend on which agent you're using.

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

## Usage

Assuming you have integrated Beans into your coding agent correctly, it will already know how to create and manage beans for you. You can use the usual assortment of natural language inquiries. If you've just
added Beans to an existing project, you could try asking your agent to identify potential tasks and create beans for them:

> "Are there any tasks we should be tracking for this project? If so, please create beans for them."

If you already have some beans available, you can ask your agent to recommend what to work on next:

> "What should we work on next?"

You can also specifically ask it to start working on a particular bean:

> "It's time to tackle myproj-123."

## GraphQL API

Beans includes a GraphQL API for flexible querying. Use `beans query` to run queries:

```bash
# List all beans with their status
beans query '{ beans { id title status } }'

# Get a specific bean with its relationships
beans query '{ bean(id: "abc") { title status parent { title } children { id title } } }'

# Filter beans
beans query '{ beans(filter: { status: ["todo", "in-progress"] }) { id title } }'
```

Queries can also be piped from stdin, which is useful for complex queries:

```bash
beans query <<'EOF'
{
  beans(filter: { excludeStatus: ["completed", "scrapped"] }) {
    id
    title
    status
    blockedBy { id title }
  }
}
EOF
```

Run `beans query --schema` to see the full GraphQL schema.

## Contributing

This project currently does not accept contributions -- it's just way too early for that!
But if you do have suggestions or feedback, please feel free to open an issue.
