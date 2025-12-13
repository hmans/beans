---
title: 'TUI: update updated_at when bean is edited via editor'
status: completed
type: feature
created_at: 2025-12-13T02:54:10Z
updated_at: 2025-12-13T02:54:10Z
---

When editing a bean in the TUI using 'e', track the file modification time before opening the editor. After the editor closes, if the file was modified, call Core.Update() to set updated_at.