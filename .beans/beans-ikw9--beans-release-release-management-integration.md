---
# beans-ikw9
title: beans release - Release management integration
status: draft
type: feature
priority: normal
tags:
    - idea
created_at: 2025-12-08T17:11:36Z
updated_at: 2025-12-13T14:44:14Z
parent: beans-f11p
---

Add a `beans release` command that uses Beans for release management and changelog updating.

This would integrate beans into the release workflow by:
- Consuming beans marked as completed since the last release
- Automatically updating changelogs based on completed beans
- Tagging releases with appropriate version numbers
- Generating release notes from bean descriptions

## Potential Features
- Auto-detect version bump type (major/minor/patch) based on bean types
- Integration with existing changelog tools (changie, etc.)
- Support for different changelog formats
- Dry-run mode to preview changes
- Filter which beans to include (by type, label, etc.)

## Open Questions
- How to determine version bump? (bean types, labels, manual?)
- What changelog format(s) to support?
- Should it integrate with git tags directly?
- How to handle beans that span multiple releases?