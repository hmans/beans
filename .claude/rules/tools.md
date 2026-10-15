# Tooling

All build/dev/test tasks use `just` (see `justfile`). Key commands:

- `just dev` — start the development environment (backend on :22880, frontend on :5173, with hot reload)
- `just build` — build the `./beans` executable (includes frontend embed)
- `just beans` — compile and run the beans CLI (use instead of `go run` or `./beans`)
- `just test` — run codegen, the e2e tests and all Go tests
- `just test-e2e` — run Playwright e2e tests
- `just codegen` — regenerate GraphQL code after schema changes
- `just setup` — install all dependencies and generate code (first-time setup)
- `just dev-kill` — kill lingering dev processes
