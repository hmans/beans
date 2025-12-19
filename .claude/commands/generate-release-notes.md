# Generate Release Notes

Generate release notes from commits since the last version tag and write them to `.docs.local/RELEASE.md`.

## Instructions

1. **Find the latest version tag:**
   ```bash
   git describe --tags --abbrev=0
   ```

2. **Get commits since that tag:**
   ```bash
   git log <tag>..HEAD --oneline --no-merges
   ```

3. **Write conversational release notes** to `.docs.local/RELEASE.md` following this style:

### Writing Style

**DO NOT** just list commits. Instead, write release notes that tell a story:

- **Lead with what matters most** - Start with a brief intro paragraph highlighting the 2-3 most impactful changes
- **Group related changes** - Combine multiple commits that work toward the same goal into a single, well-explained entry
- **Explain the "why"** - Help users understand how changes benefit them, not just what changed
- **Use natural language** - Write like you're telling a colleague about the release over coffee
- **Highlight breaking changes prominently** - These need extra attention and migration guidance

### Structure

```markdown
# Release Notes for vX.Y.Z

<Opening paragraph: 1-3 sentences summarizing the release theme and most exciting changes>

## Breaking Changes

<If any: explain what changed, why, and how to migrate. Be helpful, not just factual.>

## Highlights

<The 2-4 most significant improvements, with context about why they matter>

## Other Improvements

<Grouped by area (e.g., "TUI", "CLI", "Performance") with brief descriptions>

## Bug Fixes

<Notable fixes, especially ones users might have encountered>
```

### Linking to Commits and PRs

Include links to commits or PRs so readers can dig deeper:

1. **Get the GitHub repo URL:**
   ```bash
   git remote get-url origin
   ```

2. **Link to commits** using the short hash:
   ```markdown
   Fixed TUI crash on narrow terminals ([9e32a11](https://github.com/owner/repo/commit/9e32a11))
   ```

3. **Link to PRs** when commits reference them (look for `(#123)` in commit messages):
   ```markdown
   Simplified bean linking ([#17](https://github.com/owner/repo/pull/17))
   ```

When grouping multiple commits into one entry, link to the most significant commit or the PR that introduced the feature.

### Example Tone

Instead of:
> - feat(tui): add status picker modal with 's' shortcut

Write:
> **Quick status changes** — Press `s` in the TUI to instantly change a bean's status without leaving the list view. No more navigating to edit mode for simple updates. ([ad3382e](https://github.com/owner/repo/commit/ad3382e))

### Version Bumping

Suggest the next version based on:
- **Major bump**: Breaking changes (but stay in 0.x.y for pre-1.0 projects)
- **Minor bump**: New features
- **Patch bump**: Only bug fixes

## Output

**Overwrite** `.docs.local/RELEASE.md` with freshly generated release notes (create the directory if needed). Do not read or incorporate existing contents - always generate from scratch based on the git history.
