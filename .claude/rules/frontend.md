---
paths:
  - "frontend/**"
---

# Frontend

- Use `pnpm` for package management and running scripts. NEVER `npm`.
- We're using SvelteKit with `adapter-static` for a fully static **SPA**. There are no server load functions, form actions, or remote functions — all data fetching happens client-side via GraphQL.

## Architecture

The frontend is a single-page app with two main views:

- **Planning view** (`/planning`, `/planning/board`) — backlog list and kanban board for managing beans
- **Workspace view** (`/workspace/[worktreeId]`) — agent chat, file changes, terminal, and bean detail for a specific worktree

### Routing

Routes live in `src/routes/`. The root `+page.ts` redirects to `/planning`. Workspace routes use `{#key worktreeId}` to fully remount `WorkspaceView` when switching workspaces.

### State Management

State is managed through **singleton store classes** using Svelte 5 runes, instantiated as module-level exports:

- `beansStore` (`beans.svelte.ts`) — all beans, fed by `beanChanged` subscription with `includeInitial: true`
- `worktreeStore` (`worktrees.svelte.ts`) — active worktrees, fed by `worktreesChanged` subscription
- `agentStatusesStore` (`agentStatuses.svelte.ts`) — which agents are running, fed by `activeAgentStatuses` subscription
- `configStore` (`config.svelte.ts`) — project config, loaded once via query
- `ui` (`uiState.svelte.ts`) — UI state (active view, selected bean, panel visibility), synced with URL and localStorage
- `changesStore` (`changes.svelte.ts`) — file changes for the active worktree, polled

Per-workspace stores like `AgentChatStore` are instantiated per component (one per `WorkspaceView`), not as singletons.

**Store pattern**: stores use `$state` for reactive fields, `$derived` for computed properties, `#private` fields for internal state (subscription handles, etc.), and expose `subscribe()`/`unsubscribe()` methods. GraphQL subscriptions use `urql` + `wonka` pipes.

### GraphQL Client

`graphqlClient.ts` exports a singleton `urql` `Client` with two exchanges:
- `fetchExchange` for queries and mutations
- `subscriptionExchange` backed by `graphql-ws` WebSocket client with infinite retry and exponential backoff

In dev, Vite proxies `/api` to the backend. In production, the Go server serves everything.

### Optimistic Updates

Mutations that affect UI responsiveness (e.g., `worktreeStore.removeWorktree`, `agentChatStore.sendMessage`) apply optimistic local state updates before the mutation completes, and roll back on failure. The subscription eventually delivers the authoritative state.

## Svelte

- Use **Svelte 5** with runes (`$state`, `$derived`, `$props`, `$effect`, etc.). Do not use legacy Svelte 4 patterns (`export let`, `$:`, stores via `writable`/`readable`).
- **Prefer `$derived` over `$effect`** for computing values from reactive state. Only use `$effect` for true side effects (DOM manipulation, external state, subscriptions). If a value can be expressed as a derivation, use `$derived` or `$derived.by`.
- Use `SvelteMap` from `svelte/reactivity` instead of plain `Map` when the map is reactive state (e.g., `beansStore.beans`).
- Use Svelte 5 snippet blocks (`{#snippet name()}...{/snippet}`) for passing content to components like `SplitPane`, not slot-based patterns.

## SvelteKit SSR/Prerender Pitfalls

- `localStorage`, `window`, and other browser APIs are **not available** during SSR or prerendering. Never access them in module scope, `$state` initializers, or universal load functions without a `browser` guard.
- To initialize client-side state from `localStorage` without a flash of incorrect content, use a **load function** in `+layout.ts` / `+page.ts` with `export const ssr = false`. The load function runs client-side before the component renders, so the component gets the correct initial values. Do **not** try to read localStorage in `onMount` — that fires after the first paint, causing a visible flash.
- This app uses `ssr = false` in the root `+layout.ts`, so all load functions run client-side only.

## Accessibility

- **Never suppress a11y warnings with `svelte-ignore` comments.** Fix the underlying issue instead — use semantic HTML elements, add proper ARIA roles, or restructure the code to avoid the warning.
- Use `<button>` for clickable elements, not `<span role="button">` or `<div onclick>`.

## Styling

- Use **Tailwind CSS v4** utility classes. **Never write raw CSS properties** — always use Tailwind utilities, either inline or via `@apply` in custom classes.
- Define custom utility classes with `@apply` in `layout.css` when styling dynamically rendered HTML (e.g. markdown output) or when a pattern repeats across components.
- When writing `@apply` classes or `<style>` blocks, compose exclusively from Tailwind utilities — no raw CSS properties.

### Theme System

The app uses a two-layer theming approach:

1. **CSS custom properties** (`--th-*`) in `layout.css` define light/dark palettes using `var(--color-<palette>-<shade>)` references to Tailwind's built-in colors. Never use hardcoded hex values.
2. **Tailwind theme tokens** (`@theme inline`) map `--th-*` vars to semantic Tailwind color names (`surface`, `text`, `accent`, `danger`, `status-*`, `type-*`).

Use the semantic names in components: `bg-surface`, `text-text-muted`, `border-border`, `bg-status-todo-bg`, `text-type-bug-text`, etc. Status/type color mappings live in `styles.ts` as class string records.

### Icons

Icons use `@iconify/tailwind4` — reference icons as `icon-[set--name]` Tailwind classes (e.g., `icon-[uil--archive]`) or `iconify set--name` classes.

### Reusable UI Classes

`layout.css` defines shared button/badge classes: `btn-primary`, `btn-icon`, `btn-tab`, `btn-tab-sm`, `btn-toggle`, `badge`, `badge-sm`, `bean-link`. Use these instead of duplicating button styles.

### Other Styling Rules

- **All** interactive elements (`<button>`, `<a>`, clickable `<div>`s, etc.) must have `cursor-pointer`. The global CSS already applies `cursor: pointer` to `a` and `[role='button']`, but explicit `<button>` elements need the class.
- **Always use Svelte 5's array-based `class` syntax** for conditional classes instead of string interpolation. Falsy values are automatically filtered out:
  ```svelte
  <!-- DO: array syntax -->
  <div class={["base-class", condition && "active", isOpen ? "open" : "closed"]} />

  <!-- DON'T: string interpolation -->
  <div class="base-class {condition ? 'active' : ''} {isOpen ? 'open' : 'closed'}" />
  ```

## Markdown Rendering

`markdown.ts` provides `renderMarkdown()` using Marked + Shiki + DOMPurify. Key conventions:

- Shiki is initialized lazily with `preloadHighlighter()` called at app start
- Only explicitly imported language grammars are supported — see `BUNDLED_LANGS` in `markdown.ts`
- Bean IDs (e.g., `beans-s1m0`) are auto-linked via a custom Marked extension and rendered with the `.bean-link` CSS class
- All rendered HTML is sanitized through DOMPurify with `data-bean-id` whitelisted
- Rendered markdown goes inside a `<div class="prose">` container, styled in `layout.css`

## E2E Testing

- Write or update Playwright e2e tests (`frontend/e2e/`) for any web UI changes.
- Use the page object model (see `e2e/pages/`).
- Tests run in parallel with per-test server isolation — see `e2e/fixtures.ts`.
- Run e2e tests: `mise test:e2e` (or `bash frontend/e2e/run.sh`).

### Agent Chat E2E Testing

- **Never spawn Claude Code in e2e tests.** To test agent chat functionality, seed conversation data by writing JSONL files directly to `<beansPath>/.conversations/<beanId>.jsonl`. The agent manager lazily loads from disk when a subscription connects with no in-memory session.
- The central agent chat uses beanId `__central__`.
- See `e2e/agent-chat.spec.ts` for the pattern.

## Build Warnings

Before finishing any frontend work, run `pnpm build` (or `mise build`) and check for **Svelte compiler warnings**. All warnings must be resolved before the work is considered complete.

## Bundle Size

The frontend is embedded into the Go binary via `//go:embed`, which stores files **uncompressed**. Keep bundle size minimal:

- Avoid large dependencies when possible
- Use subpath imports to enable tree-shaking (e.g., `shiki/core` instead of `shiki`)
- When adding new Shiki language grammars, import from `shiki/langs/*.mjs` and add to `BUNDLED_LANGS` in `markdown.ts`
