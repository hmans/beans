---
# beans-4vu3
title: Agent tests spawn the real Claude CLI instead of a mock
status: completed
type: bug
priority: critical
created_at: 2026-10-09T15:34:32Z
updated_at: 2026-10-09T16:54:31Z
blocking:
    - beans-eudm
---

The `internal/agent` and `internal/graph` tests start the real Claude Code CLI whenever a `claude` binary is on `PATH`. The tests assume none is installed ("SendMessage will try to spawn a process, which will fail since there's no claude binary in test", `describe_test.go:70`, `manager_test.go:464`). On a developer machine with Claude Code installed, that assumption is false.

The root cause is in production code: all three spawn sites hardcode the string `"claude"`, so no caller, test or otherwise, can choose which agent binary runs.

## Observed

One run of `go test ./internal/agent/` started 10 `claude` processes:

- 6 × `claude --print --model haiku ...` with the working directory `internal/agent` inside this repository. They ran `git status`, `git log`, `git config user.name` there.
- 1 × an agent session with `--dangerously-skip-permissions` in `/tmp`.
- 3 × agent sessions in plan mode in `/tmp`.

The processes inherit the full environment (`buildClaudeEnv` removes only `CLAUDECODE`). Under `git rebase --exec` that includes `GIT_DIR` pointing at the real repository. They receive test prompts such as "Fix the auth bug" and "proceed" and may make real API calls.

## Spawn sites

- `internal/agent/claude.go:118`: `spawnAndRun`, reached from `Manager.SendMessage` (`manager.go:239`) and `claude.go:792`
- `internal/agent/quickreplies.go:49`: `generateQuickReplies`, reached from the output parser on turn completion (`claude.go:561`); sets no `cmd.Dir`
- `internal/agent/describe.go:53`: `GenerateDescription`, a package-level function called from `internal/commands/serve.go:249`

## Affected tests

- `internal/agent`: the `SendMessage` tests in `describe_test.go`, `manager_test.go`, `quickreplies_test.go`; the `readOutput` tests in `claude_test.go`, whose `{"type":"result"}` lines trigger `generateQuickReplies`.
- `internal/graph`: the `ExecuteAgentAction` tests in `schema.resolvers_test.go` ("commit action", "review action", ...) call `SendMessage` on a real `agent.NewManager("", nil)`, so they start an agent in act mode with a commit or review prompt in a temp dir.

## Design (decided)

The agent command becomes an explicit dependency of `Manager`, not a name resolved from `PATH`.

- `Manager` gets a field `agentCommand string`, set only by `SetAgentCommand`. `NewManager` leaves it empty; `serve.go` sets `"claude"`.
- Every spawn site goes through `newAgentCmd`, which returns `errNoAgentCommand` for an empty command and never falls back to `"claude"`. Every manager built by tests, via `NewManager` or as a struct literal, therefore fails before starting any process; no test call site needs a change.
- `GenerateQuickReplies` and `GenerateDescription` take the command as a parameter.
- `PATH` stubbing is rejected: it keeps the hardcoded name and protects only packages that remember to install it.
- `AgentType: "claude"` in session data is a display label, starts nothing, and stays unchanged.

## Tasks

- [x] Decide the design
- [x] Add `agentCommand` to `Manager`, empty by default; empty value refuses to spawn
- [x] Route `spawnAndRun`, `GenerateQuickReplies` and `GenerateDescription` through it; set `"claude"` in `serve.go`
- [x] Unit test: without a command, each spawn site fails and starts nothing; with a stub command, the generators run it
- [x] Guard test: `NewManager` configures no agent command
- [x] Update the test comments that assume a missing binary
- [x] Verify: `go test ./internal/agent/ ./internal/graph/` starts no `claude` process

## Summary of Changes

- `internal/agent`: new `Manager.SetAgentCommand`; `newAgentCmd` builds every agent process and returns `errNoAgentCommand` when no command is set. `GenerateQuickReplies(command, ...)` and `GenerateDescription(command, ...)` take the command explicitly.
- `internal/commands/serve.go`: sets the agent command `"claude"` and passes it to `GenerateDescription`.
- `internal/agent/agentcommand_test.go`: `NewManager` has no command; spawns without a command fail with `errNoAgentCommand`; the generators run a configured stub executable.
- Updated the two test comments that assumed no `claude` binary is installed.

Verification: a logging `claude` placed first on `PATH` recorded 10 calls from `go test ./internal/agent/ ./internal/graph/` before the change (6 quick-reply `haiku` calls in `internal/agent`, 4 agent sessions in `/tmp`; the `internal/graph` `ExecuteAgentAction` tests spawn one only intermittently, because the test ends before the spawn goroutine runs) and 0 calls after it, also over 30 repeated runs and over `go test ./...`.

`go test ./...` still fails in `internal/gitutil` (10 tests, `push.pushOption` from the global git config); that is beans-eudm.
