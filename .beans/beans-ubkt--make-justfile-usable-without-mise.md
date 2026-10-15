---
# beans-ubkt
title: Make justfile usable without mise
status: in-progress
type: task
priority: normal
created_at: 2026-10-02T08:45:25Z
updated_at: 2026-10-02T08:48:12Z
---

Follow-up to beans-xwvt. Review found three defects on machines without mise.

- [x] e2e/run.sh calls mise; call go directly
- [x] just beans/beans-tui lose argument quoting; use positional-arguments
- [x] corepack prerequisite and download prompt; document and disable prompt
- [x] install: create ~/.local/bin

- [ ] Verify e2e suite once Playwright browsers are installed (`npx playwright install chromium`)
