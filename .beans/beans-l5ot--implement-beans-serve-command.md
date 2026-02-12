---
# beans-l5ot
title: Implement beans serve command
status: completed
type: task
priority: normal
created_at: 2025-12-18T16:44:42Z
updated_at: 2025-12-18T16:48:37Z
parent: beans-lbjp
---

Create the CLI command that starts the web server.

## Tasks

- [ ] Add cmd/serve.go with cobra command
- [ ] Implement HTTP server startup with graceful shutdown
- [ ] Add --port flag (default 8080)
- [ ] Add --dev flag to serve from filesystem instead of embedded assets
- [ ] Add --open flag to open browser automatically
- [ ] Serve embedded static assets at /
- [ ] Mount GraphQL handler at /graphql
- [ ] Print server URL on startup

## Notes

- Use context for graceful shutdown on SIGINT/SIGTERM
- In dev mode, serve from web/build/ directory