---
globs: ["frontend/**"]
---

# Frontend

- Use `pnpm` for package management and running scripts. NEVER `npm`.
- We're using SvelteKit with `adapter-static` for a fully static **SPA**. There are no server load functions, form actions, or remote functions — all data fetching happens client-side via GraphQL.

## Svelte

- Use **Svelte 5** with runes (`$state`, `$derived`, `$props`, `$effect`, etc.). Do not use legacy Svelte 4 patterns (`export let`, `$:`, stores via `writable`/`readable`).
- **Prefer `$derived` over `$effect`** for computing values from reactive state. Only use `$effect` for true side effects (DOM manipulation, external state, subscriptions). If a value can be expressed as a derivation, use `$derived` or `$derived.by`.

## SvelteKit SSR/Prerender Pitfalls

- `localStorage`, `window`, and other browser APIs are **not available** during SSR or prerendering. Never access them in module scope, `$state` initializers, or universal load functions without a `browser` guard.
- To initialize client-side state from `localStorage` without a flash of incorrect content, use a **load function** in `+layout.ts` / `+page.ts` with `export const ssr = false`. The load function runs client-side before the component renders, so the component gets the correct initial values. Do **not** try to read localStorage in `onMount` — that fires after the first paint, causing a visible flash.
- This app uses `ssr = false` in the root `+layout.ts`, so all load functions run client-side only.

## Accessibility

- **Never suppress a11y warnings with `svelte-ignore` comments.** Fix the underlying issue instead — use semantic HTML elements, add proper ARIA roles, or restructure the code to avoid the warning.

## Styling

- Use **Tailwind CSS v4** utility classes. **Never write raw CSS properties** — always use Tailwind utilities, either inline or via `@apply` in custom classes.
- Define custom utility classes with `@apply` in `layout.css` when styling dynamically rendered HTML (e.g. markdown output) or when a pattern repeats across components.
- When writing `@apply` classes or `<style>` blocks, compose exclusively from Tailwind utilities — no raw CSS properties.
- **All** interactive elements (`<button>`, `<a>`, clickable `<div>`s, etc.) must have `cursor-pointer`.
- **Always use Svelte 5's array-based `class` syntax** for conditional classes instead of string interpolation. Falsy values are automatically filtered out:
  ```svelte
  <!-- DO: array syntax -->
  <div class={["base-class", condition && "active", isOpen ? "open" : "closed"]} />

  <!-- DON'T: string interpolation -->
  <div class="base-class {condition ? 'active' : ''} {isOpen ? 'open' : 'closed'}" />
  ```

## E2E Testing

- Write or update Playwright e2e tests (`frontend/e2e/`) for any web UI changes.
- Use the page object model (see `e2e/pages/`).
- Tests run in parallel with per-test server isolation — see `e2e/fixtures.ts`.
- Run e2e tests: `mise test:e2e` (or `bash frontend/e2e/run.sh`).

## Agent Chat E2E Testing

- **Never spawn Claude Code in e2e tests.** To test agent chat functionality, seed conversation data by writing JSONL files directly to `<beansPath>/.conversations/<beanId>.jsonl`. The agent manager lazily loads from disk when a subscription connects with no in-memory session.
- The central agent chat uses beanId `__central__`.
- See `e2e/agent-chat.spec.ts` for the pattern.

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
