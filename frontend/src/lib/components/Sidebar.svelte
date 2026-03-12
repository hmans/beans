<script lang="ts">
  import { worktreeStore } from '$lib/worktrees.svelte';
  import { agentStatusesStore } from '$lib/agentStatuses.svelte';
  import { configStore } from '$lib/config.svelte';
  import { ui } from '$lib/uiState.svelte';
  import ConfirmModal from './ConfirmModal.svelte';

  interface WorkspaceItem {
    id: string;
    label: string;
  }

  const workspaceItems = $derived(
    worktreeStore.worktrees.map((wt): WorkspaceItem => {
      return {
        id: wt.id,
        label: wt.name ?? wt.branch
      };
    })
  );

  let confirmingRemoveId = $state<string | null>(null);

  async function handleCreateWorktree() {
    const name = window.prompt('Worktree name:');
    if (!name) return;

    const wt = await worktreeStore.createWorktree(name);
    if (wt) {
      ui.navigateTo(wt.id);
    }
  }

  async function handleRemoveWorktree(id: string) {
    confirmingRemoveId = null;
    const ok = await worktreeStore.removeWorktree(id);
    if (ok && ui.activeView === id) {
      ui.navigateTo('planning');
    }
  }
</script>

<nav class="flex h-full flex-col bg-surface-alt">
  <div class="flex h-14 shrink-0 items-center border-b border-border px-3">
    <span class="text-sm font-semibold text-text">{configStore.projectName || 'beans'}</span>
  </div>

  <div class="flex-1 overflow-y-auto p-2">
    <!-- Planning item -->
    <button
      onclick={() => ui.navigateTo('planning')}
      class={[
        'flex w-full cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors',
        ui.isPlanning
          ? 'bg-surface font-medium text-text'
          : 'text-text-muted hover:bg-surface hover:text-text'
      ]}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        class="h-4 w-4 shrink-0"
      >
        <path
          fill-rule="evenodd"
          d="M6 4.75A.75.75 0 016.75 4h10.5a.75.75 0 010 1.5H6.75A.75.75 0 016 4.75zM6 10a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H6.75A.75.75 0 016 10zm0 5.25a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H6.75a.75.75 0 01-.75-.75zM1.99 4.75a1 1 0 011-1h.01a1 1 0 010 2h-.01a1 1 0 01-1-1zm0 5.25a1 1 0 011-1h.01a1 1 0 010 2h-.01a1 1 0 01-1-1zm0 5.25a1 1 0 011-1h.01a1 1 0 010 2h-.01a1 1 0 01-1-1z"
          clip-rule="evenodd"
        />
      </svg>
      Planning
      {#if configStore.agentEnabled && agentStatusesStore.isRunning('__central__')}
        <span class="ml-auto h-2 w-2 shrink-0 animate-pulse rounded-full bg-success"></span>
      {/if}
    </button>

    {#if configStore.agentEnabled}
      <!-- Workspaces section -->
      <div class="mt-4 mb-1 flex items-center justify-between px-3">
        <span class="text-xs font-medium tracking-wider text-text-faint uppercase">
          Workspaces
        </span>
        <button
          onclick={handleCreateWorktree}
          class="cursor-pointer rounded p-0.5 text-text-faint transition-colors hover:bg-surface hover:text-text"
          aria-label="Create worktree"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 16 16"
            fill="currentColor"
            class="h-3.5 w-3.5"
          >
            <path d="M8.75 3.75a.75.75 0 0 0-1.5 0v3.5h-3.5a.75.75 0 0 0 0 1.5h3.5v3.5a.75.75 0 0 0 1.5 0v-3.5h3.5a.75.75 0 0 0 0-1.5h-3.5v-3.5Z" />
          </svg>
        </button>
      </div>

      {#each workspaceItems as item (item.id)}
        <div class="group flex items-center">
          <button
            onclick={() => ui.navigateTo(item.id)}
            class={[
              'flex min-w-0 flex-1 cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors',
              ui.activeView === item.id
                ? 'bg-surface font-medium text-text'
                : 'text-text-muted hover:bg-surface hover:text-text'
            ]}
          >
            <span class="min-w-0 flex-1 truncate">{item.label}</span>
            {#if agentStatusesStore.isRunning(item.id)}
              <span class="h-2 w-2 shrink-0 animate-pulse rounded-full bg-success"></span>
            {/if}
          </button>
          <button
            onclick={(e) => {
              e.stopPropagation();
              confirmingRemoveId = item.id;
            }}
            class="mr-1 cursor-pointer rounded p-1 text-text-faint opacity-0 transition-opacity hover:bg-surface hover:text-danger group-hover:opacity-100"
            aria-label="Destroy worktree"
          >
            <span class="icon-[uil--archive] block size-3.5"></span>
          </button>
        </div>
      {/each}
    {/if}
  </div>

  {#if confirmingRemoveId}
    {@const label = workspaceItems.find((w) => w.id === confirmingRemoveId)?.label ?? 'this worktree'}
    <ConfirmModal
      title="Destroy Worktree"
      message={`Are you sure you want to destroy the worktree for "${label}"? This cannot be undone.`}
      confirmLabel="Destroy"
      danger
      onConfirm={() => handleRemoveWorktree(confirmingRemoveId!)}
      onCancel={() => (confirmingRemoveId = null)}
    />
  {/if}
</nav>
