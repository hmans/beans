---
# beans-sli4
title: Embed web assets into Go binary
status: completed
type: task
priority: normal
created_at: 2025-12-18T16:44:50Z
updated_at: 2025-12-18T17:26:56Z
parent: beans-lbjp
---

Set up go:embed to include the built SvelteKit assets in the binary.

## Tasks

- [ ] Create internal/web/embed.go with //go:embed directive
- [ ] Embed web/build/* directory
- [ ] Create http.FileSystem wrapper for serving embedded assets
- [ ] Handle SPA routing (serve index.html for unknown paths)
- [ ] Add build step to mise.toml: build web assets before go build

## Notes

- SPA routing is important: all routes should fall back to index.html
- Consider gzip/brotli pre-compression for production builds