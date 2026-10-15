---
# beans-jtas
title: beans update silently drops repeated --body-replace flags
status: todo
type: bug
created_at: 2026-10-06T12:50:48Z
updated_at: 2026-10-06T12:50:48Z
---

`beans update` accepts `--body-replace-old`/`--body-replace-new` more than once, but applies only the last pair. Earlier pairs are dropped without a warning, and the command still reports `success: true`.

Example: this call checks off nothing and only replaces the second text.

```
beans update <id> --body-replace-old "- [ ] A" --body-replace-new "- [x] A" \
                  --body-replace-old "old" --body-replace-new "new"
```

Cause: both flags are `StringVar` (`internal/commands/update.go:299-300`), so pflag keeps the last value.

## Requirements

- A repeated `--body-replace-old` or `--body-replace-new` ends the command with an error and a non-zero exit status, before any change is written. The error names the flag and points to `beans query` with `bodyMod.replace` for several replacements.
- The prime text (`internal/commands/prompt.tmpl`, section "Modifying Bean Body Content") states that the flags take one pair per call, and that several replacements go through `beans query`.

## Open question

`--status`, `--title`, `--parent`, `--body-append` and the other `StringVar` flags of `beans update` have the same last-wins behavior. Decide whether the error check covers them too, or only the two body-replace flags.

## Todo

- [ ] Reject a repeated `--body-replace-old`/`--body-replace-new` in `beans update`
- [ ] Table-driven test: one pair succeeds, a repeated flag fails without writing the bean
- [ ] Document the one-pair rule in `prompt.tmpl`
- [ ] `mise test`
