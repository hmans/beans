---
# beans-kp3h
title: Wrap beans in an MCP server
status: completed
type: feature
priority: normal
tags:
    - idea
created_at: 2025-12-13T12:00:03Z
updated_at: 2025-12-13T12:46:50Z
---

Expose beans functionality through a Model Context Protocol (MCP) server, allowing AI assistants and other MCP-compatible clients to interact with beans programmatically.

## Overview

An MCP server would allow tools like Claude Code, Cursor, and other MCP-compatible clients to directly query and manage beans without shelling out to the CLI.

## Design: Single GraphQL Tool

Rather than exposing many granular tools, expose **one tool** that accepts raw GraphQL:

```
Tool: beans_graphql
Input: { "query": "...", "variables": {...} }
```

The tool description would include:
- The full GraphQL schema
- Usage instructions (similar to `beans prime` output)
- Example queries and mutations

**Benefits:**
- Minimal tool surface area (one tool vs many)
- Agents craft exactly the queries they need → lower token usage
- No need to maintain separate tool definitions for each operation
- GraphQL's flexibility handles all current and future operations

**Essentially `beans prime` but delivered via MCP instead of system prompt injection.**

## Relationship with beans-lbjp (beans web)

The `beans web` feature (beans-lbjp) plans to expose GraphQL over HTTP. These could share infrastructure, but MCP via stdio is simpler and doesn't require a running daemon.

## Implementation

- [x] Choose Go MCP library (or implement minimal stdio protocol)
- [x] Create `beans mcp` command that runs stdio MCP server
- [x] Single `beans_graphql` tool wrapping the existing GraphQL engine
- [x] Tool description generated from schema + usage docs
- [ ] Test with Claude Code MCP integration