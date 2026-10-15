---
# beans-rren
title: Avoid disk I/O while holding write lock in Core.Update()
status: scrapped
type: task
priority: normal
created_at: 2026-03-13T17:49:05Z
updated_at: 2026-10-13T09:30:03Z
parent: beans-oyic
---

In core.go Update(), the ETag calculation reads from disk while holding the write lock. If the filesystem is slow, this blocks all other goroutines. Consider reading the file before acquiring the lock, or caching the on-disk ETag.

## Reasons for Scrapping

The premise no longer holds. Since beans-o1if, `Core.Update` compares the ETag in memory and reads no file. The only disk I/O under the lock is the write, which keeps memory and file in step on purpose.

The remaining concern moved to beans-qypq: a process that holds `Core.mu` while it waits for the cross-process file lock stalls every goroutine of `beans serve`, readers included.
