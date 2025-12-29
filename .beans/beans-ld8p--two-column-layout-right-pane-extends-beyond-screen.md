---
# beans-ld8p
title: 'Two-column layout: right pane extends beyond screen width'
status: scrapped
type: bug
priority: normal
created_at: 2025-12-28T19:20:10Z
updated_at: 2025-12-28T19:22:00Z
parent: t0tv
---

Same root cause as beans-m3mq - the footer in list.View() is not width-constrained, causing lipgloss.JoinHorizontal to miscalculate widths. See beans-m3mq for details.