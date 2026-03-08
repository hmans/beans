---
globs: ["frontend/**"]
---

# Frontend

- Use `pnpm` for package management and running scripts. NEVER `npm`.
- We're using SvelteKit with `adapter-static` for a fully static **SPA**. There are no server load functions, form actions, or remote functions — all data fetching happens client-side via GraphQL.

## Svelte

- Use **Svelte 5** with runes (`$state`, `$derived`, `$props`, `$effect`, etc.). Do not use legacy Svelte 4 patterns (`export let`, `$:`, stores via `writable`/`readable`).

## Styling

- Use **Tailwind CSS v4** utility classes. Avoid plain CSS or `<style>` blocks when Tailwind utilities suffice.
- Define custom utility classes in the Tailwind theme (`@theme`) when a pattern repeats across components.

## E2E Testing

- Write or update Playwright e2e tests (`frontend/e2e/`) for any web UI changes.
- Use the page object model (see `e2e/pages/`).
- Tests run in parallel with per-test server isolation — see `e2e/fixtures.ts`.
- Run e2e tests: `mise test:e2e` (or `bash frontend/e2e/run.sh`).

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
