---
title: Fix prime command to search upward for .beans.yml
status: completed
type: bug
created_at: 2025-12-13T10:30:56Z
updated_at: 2025-12-13T10:30:56Z
---

The `beans prime` command was not properly detecting beans projects. It used `LoadFromDirectory` which never fails (returns default config if no file found), then checked if the beans directory exists at cwd.

Fixed to use `FindConfig` directly, which properly searches upward through parent directories for `.beans.yml`. If no config file is found, it silently exits. This aligns with how other commands discover the project root.