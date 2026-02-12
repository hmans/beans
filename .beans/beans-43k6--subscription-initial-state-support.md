---
# beans-43k6
title: Subscription initial state support
status: completed
type: feature
priority: normal
created_at: 2025-12-20T10:28:17Z
updated_at: 2025-12-20T10:30:24Z
---

Add includeInitial argument to beanChanged subscription that emits all current beans on connect, with isInitial flag to distinguish from real changes.