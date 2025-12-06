---
title: Sort list output by status order from config
status: done
created_at: 2025-12-06T14:48:15Z
updated_at: 2025-12-06T15:00:48Z
---



The `list` command should sort beans by their status, using the order defined in the `statuses.available` array from the configuration file (beans.toml).

For example, with the default config `["open", "in-progress", "done"]`, beans with status 'open' should appear first, followed by 'in-progress', then 'done'.

This makes the list output more useful by grouping beans by their workflow stage.