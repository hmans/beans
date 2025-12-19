---
# beans-6hbt
title: Generate embeddings for beans using OpenAI
status: draft
type: feature
tags:
    - idea
created_at: 2025-12-15T17:36:19Z
updated_at: 2025-12-15T17:36:19Z
---

## Summary

Add optional support for generating vector embeddings for beans using OpenAI's embedding API. This would enable semantic similarity search and relationship discovery between beans, which could be particularly valuable for larger projects with many beans.

## Key Points

- **Optional feature**: Only activated when the user provides an OpenAI API key (via environment variable or config)
- **Use case**: Find semantically related beans even when they don't share keywords
- **Storage**: Embeddings stored locally alongside bean files (or in a separate index)

## Potential Features

- [ ] Configuration for OpenAI API key
- [ ] Generate embeddings on bean creation/update
- [ ] `beans similar <bean-id>` command to find related beans
- [ ] Enhance GraphQL with a `similarTo` filter
- [ ] Cache embeddings to avoid redundant API calls
- [ ] Support for other embedding providers (e.g., local models, Anthropic when available)

## Considerations

- API costs: OpenAI embeddings are cheap but not free
- Privacy: Users should understand their bean content is sent to OpenAI
- Offline mode: Feature should degrade gracefully when API is unavailable
- Model selection: Start with `text-embedding-3-small` for cost efficiency

## Library Research

**Recommended: [chromem-go](https://github.com/philippgille/chromem-go)** (MPL-2.0)

An embeddable vector database for Go with zero third-party dependencies and no CGO requirement.

**Why it fits:**
- Pure Go, zero CGO - cross-compiles easily
- Built-in OpenAI embedding support (also Cohere, Mistral, Ollama, etc.)
- Optional file persistence with gob encoding
- Fast: ~0.3ms for 1,000 docs, ~40ms for 100,000 docs
- Chroma-like API, easy to use

**Tradeoffs:**
- Beta (pre-v1.0), possible breaking changes
- Exhaustive nearest-neighbor search only (no ANN/HNSW yet) - fine for typical beans scale
- In-memory with optional persistence

**Alternatives considered:**
- [kelindar/search](https://github.com/kelindar/search) - uses purego + llama.cpp, more complex
- [govector](https://pkg.go.dev/github.com/drewlanenga/govector) - basic vector math only, no DB features
- Milvus/Weaviate - full vector DBs, overkill for embedded use