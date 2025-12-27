---
# beans-9w6d
title: Prettify JSON output in --json mode
status: completed
type: bug
created_at: 2025-12-27T21:36:52Z
updated_at: 2025-12-27T21:36:52Z
---

The `beans query --json` output was compact/raw JSON, but it should still be prettified (indented) - just without colors. This makes the output more readable when piping to tools like `jq` or saving to files, while still being valid for machine parsing.