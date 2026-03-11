<script lang="ts">
  import { onMount, onDestroy, setContext } from 'svelte';
  import { changesStore } from '$lib/changes.svelte';
  import { ui } from '$lib/uiState.svelte';
  import type { ActionContext } from '$lib/actionContext';
  import PaneHeader from '$lib/components/PaneHeader.svelte';
  import ActionButton from '$lib/components/ActionButton.svelte';

  interface Props {
    path?: string;
    onAction?: (message: string) => void;
    agentBusy?: boolean;
  }

  let { path, onAction, agentBusy = false }: Props = $props();

  if (onAction) {
    setContext<ActionContext>('action', {
      onAction,
      get disabled() {
        return agentBusy;
      }
    });
  }

  const stagedChanges = $derived(changesStore.changes.filter((c) => c.staged));
  const unstagedChanges = $derived(changesStore.changes.filter((c) => !c.staged));
  const totalCount = $derived(changesStore.changes.length);

  onMount(() => {
    changesStore.startPolling(path);
  });

  onDestroy(() => {
    changesStore.stopPolling();
  });

  function statusColor(status: string): string {
    switch (status) {
      case 'added':
      case 'untracked':
        return 'text-success';
      case 'deleted':
        return 'text-danger';
      case 'renamed':
        return 'text-accent';
      default:
        return 'text-warning';
    }
  }

  function statusLabel(status: string): string {
    switch (status) {
      case 'modified':
        return 'M';
      case 'added':
        return 'A';
      case 'deleted':
        return 'D';
      case 'untracked':
        return '?';
      case 'renamed':
        return 'R';
      default:
        return '?';
    }
  }

  function fileName(path: string): string {
    return path.split('/').pop() ?? path;
  }

  function dirName(path: string): string {
    const parts = path.split('/');
    if (parts.length <= 1) return '';
    return parts.slice(0, -1).join('/') + '/';
  }
</script>

<div class="flex h-full flex-col border-l border-border bg-surface">
  <PaneHeader title="Status" onClose={() => ui.toggleChanges()}>
    {#snippet extra()}
      {#if totalCount > 0}
        <span class="ml-1 text-sm text-text-muted">({totalCount})</span>
      {/if}
    {/snippet}
  </PaneHeader>

  <div class="flex-1 overflow-auto">
    {#if totalCount === 0}
      <p class="px-3 py-4 text-center text-text-muted">No changes</p>
    {:else}
      {#if stagedChanges.length > 0}
        <div class="px-3 pt-2 pb-1 font-medium text-text-muted">Staged</div>
        {#each stagedChanges as change (change.path + ':staged')}
          <div class="flex items-center gap-1.5 px-3 py-0.5 hover:bg-surface-alt">
            <span class={['w-3 shrink-0 text-center font-mono font-bold', statusColor(change.status)]}>
              {statusLabel(change.status)}
            </span>
            <span class="min-w-0 flex-1 truncate" title={change.path}>
              <span class="text-text-muted">{dirName(change.path)}</span><span class="text-text">{fileName(change.path)}</span>
            </span>
            {#if change.additions > 0 || change.deletions > 0}
              <span class="shrink-0 font-mono">
                {#if change.additions > 0}<span class="text-success">+{change.additions}</span>{/if}
                {#if change.deletions > 0}<span class={[change.additions > 0 && 'ml-1', 'text-danger']}>-{change.deletions}</span>{/if}
              </span>
            {/if}
          </div>
        {/each}
      {/if}

      {#if unstagedChanges.length > 0}
        {#if stagedChanges.length > 0}
          <div class="px-3 pt-2 pb-1 font-medium text-text-muted">Unstaged</div>
        {/if}
        {#each unstagedChanges as change (change.path + ':unstaged')}
          <div class="flex items-center gap-1.5 px-3 py-0.5 hover:bg-surface-alt">
            <span class={['w-3 shrink-0 text-center font-mono font-bold', statusColor(change.status)]}>
              {statusLabel(change.status)}
            </span>
            <span class="min-w-0 flex-1 truncate" title={change.path}>
              <span class="text-text-muted">{dirName(change.path)}</span><span class="text-text">{fileName(change.path)}</span>
            </span>
            {#if change.additions > 0 || change.deletions > 0}
              <span class="shrink-0 font-mono">
                {#if change.additions > 0}<span class="text-success">+{change.additions}</span>{/if}
                {#if change.deletions > 0}<span class={[change.additions > 0 && 'ml-1', 'text-danger']}>-{change.deletions}</span>{/if}
              </span>
            {/if}
          </div>
        {/each}
      {/if}
    {/if}
  </div>

  {#if onAction}
    <div class="flex gap-2 border-t border-border px-3 py-2">
      <ActionButton
        prompt="Create a commit. If you have just implemented a change, make sure there is an associated bean, it is up to date, and possibly even marked as completed if you are done with the change. Then only commit changes related to that change. If you haven't, please examine the git diff and commit whatever changes you see."
      >
        Commit
      </ActionButton>

      <ActionButton prompt="Ask a subagent for a thorough code review.">
        Review
      </ActionButton>
    </div>
  {/if}
</div>
