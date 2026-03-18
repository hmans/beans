<script lang="ts">
  import { changesStore, type FileChange } from '$lib/changes.svelte';
  import { client } from '$lib/graphqlClient';
  import { configStore } from '$lib/config.svelte';
  import { diffSelectionStore } from '$lib/diffSelection.svelte';
  import { ui } from '$lib/uiState.svelte';
  import { MAIN_WORKSPACE_ID } from '$lib/worktrees.svelte';
  import { SendAgentMessageDocument, DiscardFileChangeDocument } from '$lib/graphql/generated';

  import ConfirmModal from './ConfirmModal.svelte';

  interface Props {
    path?: string;
    worktreeId?: string;
    onAgentMessage?: () => void;
  }

  let { path, worktreeId, onAgentMessage }: Props = $props();

  const isWorktree = $derived(worktreeId != null && worktreeId !== MAIN_WORKSPACE_ID);
  const branchStatus = $derived(changesStore.branchStatus);

  let rebaseRequested = $state(false);

  async function requestRebase() {
    if (!worktreeId) return;
    rebaseRequested = true;
    onAgentMessage?.();
    await client
      .mutation(SendAgentMessageDocument, {
        beanId: worktreeId,
        message: `Please run \`git rebase ${configStore.worktreeBaseRef}\` and resolve any conflicts.`
      })
      .toPromise();
  }

  type Tab = 'unstaged' | 'all';
  let activeTab = $state<Tab>('all');

  const selectedFile = $derived(diffSelectionStore.selected);

  // Keep the store's context in sync with the current tab and worktree path
  $effect(() => {
    diffSelectionStore.setContext(activeTab, path);
  });

  function selectFile(change: FileChange) {
    diffSelectionStore.toggle(change.path, change.staged);
  }

  // Clear selection when the selected file disappears from the active changes list
  $effect(() => {
    if (selectedFile) {
      const list = activeTab === 'all' ? changesStore.allChanges : changesStore.changes;
      const stillExists = list.some(
        (c) => c.path === selectedFile!.path && (activeTab === 'all' || c.staged === selectedFile!.staged)
      );
      if (!stillExists) {
        diffSelectionStore.clear();
      }
    }
  });

  // When switching tabs, clear the diff selection
  let prevTab: Tab = 'all';
  $effect(() => {
    if (activeTab !== prevTab) {
      prevTab = activeTab;
      diffSelectionStore.clear();
    }
  });

  const stagedChanges = $derived(changesStore.changes.filter((c) => c.staged));
  const unstagedChanges = $derived(changesStore.changes.filter((c) => !c.staged));
  const hasUnstagedChanges = $derived(changesStore.changes.length > 0);
  const displayChanges = $derived(activeTab === 'all' ? changesStore.allChanges : changesStore.changes);
  const totalCount = $derived(displayChanges.length);

  let confirmingDiscard = $state<{ path: string; staged: boolean } | null>(null);

  async function discardChange(filePath: string, staged: boolean) {
    confirmingDiscard = null;
    await client.mutation(DiscardFileChangeDocument, { filePath, staged, path: path ?? null }).toPromise();
    // Clear selection if the discarded file was selected
    if (selectedFile?.path === filePath) {
      diffSelectionStore.clear();
    }
    changesStore.fetch(path);
  }

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

  function fileName(filePath: string): string {
    return filePath.split('/').pop() ?? filePath;
  }

  function dirName(filePath: string): string {
    const parts = filePath.split('/');
    if (parts.length <= 1) return '';
    return parts.slice(0, -1).join('/') + '/';
  }

  function isFileSelected(change: FileChange): boolean {
    if (activeTab === 'all') {
      return selectedFile?.path === change.path;
    }
    return selectedFile?.path === change.path && selectedFile?.staged === change.staged;
  }


</script>

{#snippet fileRow(change: FileChange)}
  <div
    class={[
      'group flex w-full cursor-pointer items-center gap-1.5 px-3 py-0.5 text-left hover:bg-surface-alt',
      isFileSelected(change) && 'bg-surface-alt'
    ]}
    role="button"
    tabindex="0"
    onclick={() => selectFile(change)}
    onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectFile(change); } }}
  >
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
    <button
      class="shrink-0 cursor-pointer rounded px-1 text-text-faint opacity-0 transition-opacity hover:text-text group-hover:opacity-100"
      title="Discard change"
      onclick={(e) => { e.stopPropagation(); confirmingDiscard = { path: change.path, staged: change.staged }; }}
    >
      <span class="icon-[eva--undo-fill] size-3.5"></span>
    </button>
  </div>
{/snippet}

{#snippet fileList()}
  <div class="flex-1 overflow-auto">
    {#if totalCount === 0}
      <p class="px-3 py-4 text-center text-text-muted">No changes</p>
    {:else if activeTab === 'all'}
      {#each changesStore.allChanges as change (change.path)}
        {@render fileRow(change)}
      {/each}
    {:else}
      {#if stagedChanges.length > 0}
        <div class="px-3 pt-2 pb-1 font-medium text-text-muted">Staged</div>
        {#each stagedChanges as change (change.path + ':staged')}
          {@render fileRow(change)}
        {/each}
      {/if}

      {#if unstagedChanges.length > 0}
        {#if stagedChanges.length > 0}
          <div class="px-3 pt-2 pb-1 font-medium text-text-muted">Unstaged</div>
        {/if}
        {#each unstagedChanges as change (change.path + ':unstaged')}
          {@render fileRow(change)}
        {/each}
      {/if}
    {/if}
  </div>
{/snippet}

{#snippet tabSwitcher()}
  <div class="flex p-3">
    <div class="flex w-full">
      <button
        class={[
          'btn-tab-sm flex-1 rounded-l-md',
          activeTab === 'all' ? 'btn-tab-active' : 'btn-tab-inactive'
        ]}
        onclick={() => { activeTab = 'all'; }}
      >
        All Changes
        {#if changesStore.allChanges.length > 0}
          <span class="ml-1 opacity-60">({changesStore.allChanges.length})</span>
        {/if}
      </button>
      <button
        class={[
          'btn-tab-sm flex-1 rounded-r-md border-l-0',
          activeTab === 'unstaged'
            ? 'btn-tab-active'
            : hasUnstagedChanges
              ? 'btn-tab-inactive'
              : 'btn-tab-inactive opacity-50 cursor-not-allowed'
        ]}
        disabled={!hasUnstagedChanges}
        onclick={() => { activeTab = 'unstaged'; }}
      >
        Unstaged
        {#if changesStore.changes.length > 0}
          <span class="ml-1 opacity-60">({changesStore.changes.length})</span>
        {/if}
      </button>
    </div>
  </div>
{/snippet}

{#snippet branchStatusBar()}
  <div class="flex items-center gap-2 border-b border-border p-3">
    <span class="min-w-0 truncate text-text">
      {branchStatus.commitsBehind} commit{branchStatus.commitsBehind === 1 ? '' : 's'} behind
      {#if branchStatus.hasConflicts}
        <span class="text-text-muted">(merge conflicts)</span>
      {/if}
    </span>
    <button
      class="btn-toggle btn-toggle-inactive ml-auto shrink-0 cursor-pointer"
      onclick={requestRebase}
      disabled={rebaseRequested}
      title="Ask the agent to rebase this branch against main"
    >
      {#if rebaseRequested}
        Rebase requested
      {:else if branchStatus.hasConflicts}
        Resolve Conflicts
      {:else}
        Rebase
      {/if}
    </button>
  </div>
{/snippet}

<div class="flex h-full flex-col bg-surface">
  <div class="pane-toolbar">
    <span>Changes</span>
    <div class="flex-1"></div>
    <button onclick={() => ui.toggleChanges()} class="btn-icon" title="Close"> &#x2715; </button>
  </div>
  {#if isWorktree && branchStatus.commitsBehind > 0}
    {@render branchStatusBar()}
  {/if}
  {@render tabSwitcher()}

  {@render fileList()}
</div>

{#if confirmingDiscard}
  <ConfirmModal
    title="Discard Change"
    message={`Are you sure you want to discard changes to "${confirmingDiscard.path}"? This cannot be undone.`}
    confirmLabel="Discard"
    danger
    onConfirm={() => discardChange(confirmingDiscard!.path, confirmingDiscard!.staged)}
    onCancel={() => { confirmingDiscard = null; }}
  />
{/if}
