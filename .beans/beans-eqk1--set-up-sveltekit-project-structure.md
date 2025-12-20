---
# beans-eqk1
title: Set up SvelteKit project structure
status: completed
type: task
priority: normal
created_at: 2025-12-18T16:44:18Z
updated_at: 2025-12-20T12:40:08Z
parent: beans-lbjp
---

Initialize the SvelteKit project for the web UI.

## Tasks

- [ ] Create `web/` directory for the SvelteKit project
- [ ] Initialize SvelteKit with `adapter-static` for SPA mode
- [ ] Configure TypeScript
- [ ] Set up basic project structure (routes, components, lib)
- [ ] Configure Vite proxy for `/graphql` to Go backend during development
- [ ] Add build script to `mise.toml` for building the web assets

## Notes

- Use `pnpm` as package manager (consistent with modern SvelteKit projects)
- Target directory: `web/build/` for static output