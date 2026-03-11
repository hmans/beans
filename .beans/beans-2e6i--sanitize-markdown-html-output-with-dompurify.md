---
# beans-2e6i
title: Sanitize markdown HTML output with DOMPurify
status: completed
type: task
priority: high
created_at: 2026-03-09T17:01:34Z
updated_at: 2026-03-11T14:40:15Z
order: zzV
parent: beans-oe8n
---

Both BeanDetail.svelte (line 256) and AgentChat.svelte (line 122) render user/agent-supplied markdown via {@html} after processing through Marked. Marked's GFM mode allows raw HTML in markdown input, so a malicious bean body like `<img onerror=alert(1)>` executes JavaScript. Fix: install DOMPurify, sanitize the HTML string returned by renderMarkdown() before passing it to {@html}. This is the highest-impact fix because XSS is exploitable even on localhost if someone opens a bean with a crafted body. Also audit markdown.ts line 145 where bean IDs are interpolated into HTML attributes — ensure the beanId value is escaped to prevent attribute breakout.

## Summary of Changes

- Added DOMPurify (v3.3.3) as a dependency
- Integrated DOMPurify.sanitize() into renderMarkdown() to sanitize all HTML output before it reaches {@html} directives
- Configured ADD_ATTR for 'data-bean-id' to preserve bean link functionality
- Sanitization only runs in browser context (consistent with Shiki highlighting pattern)
