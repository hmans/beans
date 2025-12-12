---
title: Add tag editing UI to TUI
status: backlog
type: feature
created_at: 2025-12-12T22:58:14Z
updated_at: 2025-12-12T22:58:14Z
---

Add the ability to edit tags on beans from within the TUI.

## Context
The TUI currently supports editing various bean properties (status, parent, etc.) but lacks UI for managing tags. Users should be able to add and remove tags directly from the TUI without needing to use the CLI.

## Requirements
- Add a tag picker/editor modal accessible via a keyboard shortcut (suggest 't')
- Display current tags on the bean
- Allow adding new tags (with autocomplete from existing tags in the project)
- Allow removing existing tags
- Follow the existing modal patterns (see parentpicker.go, statuspicker.go)

## Implementation Notes
- Create a new tagpicker.go similar to existing picker modals
- Use the existing tag list from the bean store for autocomplete suggestions
- Consider allowing free-form tag entry (not just selection from existing tags)