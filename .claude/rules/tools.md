# Tooling

All build/dev/test tasks use `mise`. Key commands:

- `mise dev` — start the development environment (backend on :22880, frontend on :5173, with hot reload)
- `mise build` — build the `./beans` executable (includes frontend embed)
- `mise beans` — compile and run the beans CLI (use instead of `go run` or `./beans`)
- `mise test` — run all Go tests
- `mise test:e2e` — run Playwright e2e tests
- `mise codegen` — regenerate GraphQL code after schema changes
- `mise setup` — install all dependencies and generate code (first-time setup)
