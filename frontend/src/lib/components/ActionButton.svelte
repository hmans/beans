<script lang="ts">
  import type { Snippet } from 'svelte';
  import { getContext } from 'svelte';
  import type { ActionContext } from '$lib/actionContext';

  interface Props {
    prompt: string;
    children: Snippet;
  }

  let { prompt, children }: Props = $props();

  const ctx = getContext<ActionContext>('action');
</script>

<button
  class={[
    'flex-1 rounded border border-border px-3 py-1.5 text-sm font-medium transition-colors',
    ctx.disabled
      ? 'cursor-not-allowed text-text-faint'
      : 'cursor-pointer text-text-muted hover:bg-surface-alt hover:text-text'
  ]}
  disabled={ctx.disabled}
  onclick={() => ctx.onAction(prompt)}
>
  {@render children()}
</button>
