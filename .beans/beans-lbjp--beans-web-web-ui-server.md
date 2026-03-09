---
# beans-lbjp
title: beans web - Web UI server
status: completed
type: feature
priority: normal
tags:
    - idea
created_at: 2025-12-08T17:11:36Z
updated_at: 2026-03-09T20:28:05Z
order: V1
parent: beans-f11p
---

Add a `beans serve` command that starts a webserver providing:

1. **Web UI** - A SvelteKit SPA for browsing and managing beans
2. **GraphQL API** - HTTP endpoint exposing the existing GraphQL schema
3. **Live Updates** - GraphQL subscriptions via WebSockets for real-time sync

## Architecture

- SvelteKit app built in SPA mode (`adapter-static`)
- Static assets embedded into the Go binary via `//go:embed`
- `beans serve` command starts an HTTP server
- GraphQL endpoint at `/graphql` (queries, mutations, subscriptions)
- Web UI served at `/`
- File watcher on `.beans/` directory triggers subscription events

## Development Workflow

- `--dev` flag to serve from filesystem instead of embedded assets (for hot reload)
- SvelteKit dev server proxies `/graphql` to the Go backend during development

## Summary of Changes

All subtasks completed: SvelteKit SPA with bean list, detail, and create/edit views; GraphQL HTTP endpoint with subscriptions; static assets embedded in Go binary; beans serve command with --dev mode. The web UI is fully functional.
