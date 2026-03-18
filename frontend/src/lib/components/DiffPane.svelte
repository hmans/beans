<script lang="ts">
  import { diffSelectionStore } from '$lib/diffSelection.svelte';

  const selected = $derived(diffSelectionStore.selected);
  const loading = $derived(diffSelectionStore.diffLoading);
  const lines = $derived(diffSelectionStore.parsedLines);
  const hasContent = $derived(diffSelectionStore.diffContent !== '');

  function diffLineClass(type: 'add' | 'del' | 'hunk' | 'context' | 'header'): string {
    if (type === 'add') return 'diff-add';
    if (type === 'del') return 'diff-del';
    if (type === 'hunk') return 'diff-hunk';
    return '';
  }
</script>

<div class="flex h-full flex-col bg-surface">
  <div class="pane-toolbar">
    <span class="min-w-0 truncate font-mono">
      {selected?.path}
    </span>
    <div class="flex-1"></div>
    <button
      class="btn-icon cursor-pointer"
      onclick={() => diffSelectionStore.clear()}
      aria-label="Close diff"
    >
      &#x2715;
    </button>
  </div>
  <div class="flex-1 overflow-auto bg-surface-alt">
    {#if loading}
      <p class="px-3 py-4 text-center text-text-muted">Loading...</p>
    {:else if !hasContent}
      <p class="px-3 py-4 text-center text-text-muted">No diff available</p>
    {:else}
      <table class="diff-table w-full font-mono">
        <tbody>
          {#each lines as line, i (i)}
            {#if line.type === 'hunk'}
              <tr class="diff-hunk">
                <td class="diff-gutter-hunk"></td>
                <td class="px-3 py-1">{line.content || '...'}</td>
              </tr>
            {:else}
              <tr class={diffLineClass(line.type)}>
                <td class="diff-gutter">{line.newNum ?? line.oldNum ?? ''}</td>
                <td class="whitespace-pre pr-3">{#if line.type === 'add'}<span class="diff-indicator">+</span>{:else if line.type === 'del'}<span class="diff-indicator">-</span>{:else}<span class="diff-indicator"> </span>{/if}{line.content}</td>
              </tr>
            {/if}
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>
