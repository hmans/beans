---
# beans-ms7d
title: 'Improve Integrate button prompt: safe merge without touching main'
status: completed
type: task
created_at: 2026-03-11T20:40:44Z
updated_at: 2026-03-11T20:40:44Z
---

Rewrite the Integrate action prompt to:
1. Update bean status before committing
2. Use 'git push . HEAD:main' instead of stashing main's working directory, avoiding conflicts with other agents working in main
3. Handle race conditions with atomic fast-forward-only push and retry logic
