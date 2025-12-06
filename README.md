# Beans

An agentic-first issue tracker. Store and manage issues as markdown files in your project's `.beans/` directory.

## Installation

### Homebrew

```bash
brew install hmans/beans/beans
```

## Usage

```bash
beans init          # Initialize a .beans/ directory
beans list          # List all beans
beans show <id>     # Show a bean's contents
beans create "Title" # Create a new bean
beans status <id> <status>  # Change status (open, in-progress, done)
beans archive       # Delete all done beans
```

All commands support `--json` for machine-readable output.
