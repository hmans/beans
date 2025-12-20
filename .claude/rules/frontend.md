---
globs: ["frontend/**"]
---

# Frontend

- Use `pnpm` for package management and running scripts. NEVER `npm`.
- We're using SvelteKit with the `adapter-static` for a fully static frontend.

## Bundle Size

The frontend is embedded into the Go binary via `//go:embed`, which stores files **uncompressed**. Keep bundle size minimal:

- Avoid large dependencies when possible
- Use subpath imports to enable tree-shaking (e.g., `shiki/core` instead of `shiki`)

## Shiki Syntax Highlighting

Shiki bundles ~300 language grammars (~9MB). To keep the bundle small:

1. **Use `shiki/core`** instead of `shiki` - this gives you just the highlighter core
2. **Import specific languages** from `shiki/langs/*.mjs` (e.g., `shiki/langs/javascript.mjs`)
3. **Import themes** from `shiki/themes/*.mjs` (e.g., `shiki/themes/github-dark.mjs`)
4. **Use `createHighlighterCore`** instead of `createHighlighter`

Example:

```typescript
import { createHighlighterCore } from "shiki/core";
import { createOnigurumaEngine } from "shiki/engine/oniguruma";
import githubDark from "shiki/themes/github-dark.mjs";
import langGo from "shiki/langs/go.mjs";

const highlighter = await createHighlighterCore({
  engine: createOnigurumaEngine(import("shiki/wasm")),
  themes: [githubDark],
  langs: [langGo],
});
```

**Build-time Note**: Shiki requires browser APIs (like `URL.createObjectURL`). Since SvelteKit runs code during the static build, check `browser` from `$app/environment` to skip highlighting at build time.
