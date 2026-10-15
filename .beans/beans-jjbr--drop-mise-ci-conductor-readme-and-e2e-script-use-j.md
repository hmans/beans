---
# beans-jjbr
title: 'Drop mise: CI, conductor, README and e2e script use just'
status: completed
type: task
priority: normal
created_at: 2026-10-12T13:58:14Z
updated_at: 2026-10-12T13:58:14Z
---

The justfile is the only task file. mise.toml goes away; CI, conductor.json, README and frontend/e2e/run.sh switch to just.

## Todo

- [x] justfile header describes the justfile itself
- [x] CI workflows (test, release) install just and call its recipes
- [x] conductor.json, README, frontend/e2e/run.sh use just
- [x] Remove mise.toml
- [x] Verify: workflow YAML parses, recipes named in CI exist, git grep finds no mise outside history (CI itself runs on the next push)

## Summary of Changes

- `.github/workflows/test.yml`, `release.yml`: install just with `extractions/setup-just@v4` instead of mise; `just deps`, `just build`, `just test`, `just build-frontend`.
- `conductor.json`: `just setup`, `just dev` with the same `CONDUCTOR_PORT` wiring.
- `README.md`: just is the only build path.
- `frontend/e2e/run.sh`: calls just directly.
- `justfile` header describes the file itself.
- `mise.toml` removed.

Verified locally only: workflow YAML and `conductor.json` parse, `run.sh` passes `bash -n`, every recipe CI names exists. The workflows run on the next push. `pkg/config` keeps `"mise dev"` as an example string in a doc comment and its test data; that is not tool usage.
