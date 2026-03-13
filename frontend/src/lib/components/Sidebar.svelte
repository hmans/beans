<script lang="ts">
  import { fade } from 'svelte/transition';
  import {
    worktreeStore,
    MAIN_WORKSPACE_ID,
    type WorktreeBean
  } from '$lib/worktrees.svelte';
  import { agentStatusesStore } from '$lib/agentStatuses.svelte';
  import { configStore } from '$lib/config.svelte';
  import { ui } from '$lib/uiState.svelte';
  import { generateWorkspaceName } from '$lib/nameGenerator';
  import { typeBorders } from '$lib/styles';
  import ConfirmModal from './ConfirmModal.svelte';
  import greenbean from '$lib/assets/greenbean.png';

  interface WorkspaceItem {
    id: string;
    label: string;
    beans: WorktreeBean[];
  }

  const mainWorkspace: WorkspaceItem = { id: MAIN_WORKSPACE_ID, label: 'main', beans: [] };

  const workspaceItems = $derived([
    mainWorkspace,
    ...worktreeStore.worktrees.map((wt): WorkspaceItem => ({
      id: wt.id,
      label: wt.name ?? wt.branch,
      beans: wt.beans ?? []
    }))
  ]);

  let confirmingRemoveId = $state<string | null>(null);

  async function handleCreateWorktree() {
    const name = generateWorkspaceName();
    const wt = await worktreeStore.createWorktree(name);
    if (wt) {
      ui.navigateTo(wt.id);
    }
  }

  async function handleRemoveWorktree(id: string) {
    confirmingRemoveId = null;
    // Navigate away immediately since the store optimistically removes the item
    if (ui.activeView === id) {
      ui.navigateTo('planning');
    }
    await worktreeStore.removeWorktree(id);
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
        <div class="loader ml-auto shrink-0" transition:fade={{ duration: 200 }}></div>
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
        <button
          onclick={() => ui.navigateTo(item.id)}
          class={[
            'group flex w-full min-w-0 cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors',
            ui.activeView === item.id
              ? 'bg-surface font-medium text-text'
              : 'text-text-muted hover:bg-surface hover:text-text'
          ]}
        >
          <span class="min-w-0 flex-1 truncate">{item.label}</span>
          <div class="relative ml-auto h-4 w-4 shrink-0">
            {#if agentStatusesStore.isRunning(item.id)}
              <div class="loader absolute inset-0" transition:fade={{ duration: 200 }}></div>
            {:else if item.id !== MAIN_WORKSPACE_ID}
              <span
                role="button"
                tabindex="-1"
                onclick={(e) => {
                  e.stopPropagation();
                  confirmingRemoveId = item.id;
                }}
                class="absolute inset-0 flex cursor-pointer items-center justify-center rounded text-text-faint opacity-0 transition-opacity hover:text-danger group-hover:opacity-100"
                aria-label="Destroy worktree"
              >
                <span class="icon-[uil--archive] block size-3.5"></span>
              </span>
            {/if}
          </div>
        </button>

        {#if item.beans.length > 0}
          <div class="mt-0.5 mb-1 ml-5 mr-1 flex flex-col gap-0.5">
            {#each item.beans as wtBean (wtBean.id)}
              <button
                onclick={() => {
                  ui.navigateTo('planning');
                  ui.selectBeanById(wtBean.id);
                }}
                class={[
                  'flex min-w-0 cursor-pointer items-center gap-1.5 rounded-xs border-l-2 bg-surface px-2 py-1 text-left transition-colors hover:bg-surface-alt',
                  typeBorders[wtBean.type] ?? 'border-l-type-task-border'
                ]}
              >
                <code class="shrink-0 text-[9px] text-text-faint">{wtBean.id.slice(-4)}</code>
                <span class="min-w-0 flex-1 truncate text-xs text-text-muted">{wtBean.title}</span>
              </button>
            {/each}
          </div>
        {/if}
      {/each}
    {/if}
  </div>

  <div class="shrink-0 overflow-hidden">
    <img src={greenbean} alt="" class="relative -bottom-8 -left-4 h-auto w-52 opacity-40" />
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
