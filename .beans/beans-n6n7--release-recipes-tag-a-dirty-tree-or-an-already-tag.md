---
# beans-n6n7
title: Release recipes tag a dirty tree or an already tagged HEAD
status: todo
type: task
priority: deferred
created_at: 2026-10-12T13:59:03Z
updated_at: 2026-10-12T13:59:03Z
---

`just release-patch`, `release-minor` and `release-major` (`justfile`, recipe `_release`) tag HEAD with the next svu version without checks:

- an uncommitted change does not stop it, so the tag can name a commit that differs from the tested tree;
- an already tagged HEAD gets a second tag.

## Todo

- [ ] `_release` refuses a dirty tree (`git diff --quiet HEAD`) and a tagged HEAD (`git describe --exact-match --tags HEAD`), each with a message
- [ ] Verify in a throwaway clone: clean untagged HEAD tags; dirty tree and tagged HEAD refuse
