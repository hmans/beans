---
# beans-nvd2
title: Fix beans version command to work without .beans directory
status: completed
type: bug
priority: normal
created_at: 2025-12-24T22:20:21Z
updated_at: 2025-12-24T22:21:58Z
---

The version command fails when run in a directory without .beans/ because PersistentPreRunE validates the directory exists. Add version to the exemption list in cmd/root.go.