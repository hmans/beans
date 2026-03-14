<script lang="ts">
  import { fade } from 'svelte/transition';
  import { onDestroy } from 'svelte';
  import { gql } from 'urql';
  import { worktreeStore, MAIN_WORKSPACE_ID, type WorktreeStatus } from '$lib/worktrees.svelte';
  import { beansStore, type Bean } from '$lib/beans.svelte';
  import { agentStatusesStore } from '$lib/agentStatuses.svelte';
  import { configStore } from '$lib/config.svelte';
  import { client } from '$lib/graphqlClient';
  import { ui } from '$lib/uiState.svelte';
  import { typeBorders } from '$lib/styles';
  import ConfirmModal from './ConfirmModal.svelte';

  interface WorkspaceItem {
    id: string;
    label: string;
    description: string | null;
    beans: Bean[];
    settingUp: boolean;
  }

  /** Beans linked to a worktree via worktreeId, derived from the bean store. */
  function beansForWorktree(worktreeId: string): Bean[] {
    return beansStore.all.filter((b) => b.worktreeId === worktreeId);
  }

  const mainWorkspace: WorkspaceItem = $derived({ id: MAIN_WORKSPACE_ID, label: configStore.mainBranch, description: null, beans: [], settingUp: false });

  const workspaceItems = $derived([
    mainWorkspace,
    ...worktreeStore.worktrees.map((wt): WorkspaceItem => ({
      id: wt.id,
      label: wt.name ?? wt.branch,
      description: wt.description,
      beans: beansForWorktree(wt.id),
      settingUp: wt.setupStatus === 'RUNNING'
    }))
  ]);

  // Poll for uncommitted changes in the main repo and worktree integration readiness
  const MAIN_CHANGES_QUERY = gql`query { fileChanges { path } }`;
  const WORKTREE_STATUS_QUERY = gql`query { worktrees { id hasChanges hasUnmergedCommits } }`;
  let mainHasChanges = $state(false);
  let readyWorktreeIds = $state(new Set<string>());

  async function fetchStatuses() {
    const [mainResult, wtResult] = await Promise.all([
      client.query(MAIN_CHANGES_QUERY, {}).toPromise(),
      client.query(WORKTREE_STATUS_QUERY, {}).toPromise()
    ]);
    mainHasChanges = (mainResult.data?.fileChanges?.length ?? 0) > 0;
    const ready = new Set<string>();
    for (const wt of wtResult.data?.worktrees ?? []) {
      if (wt.hasChanges || wt.hasUnmergedCommits) ready.add(wt.id);
    }
    readyWorktreeIds = ready;
  }

  fetchStatuses();
  const statusInterval = setInterval(fetchStatuses, 3000);
  onDestroy(() => clearInterval(statusInterval));

  let confirmingRemoveId = $state<string | null>(null);
  let confirmingStatus = $state<WorktreeStatus | null>(null);

  async function promptDestroy(id: string) {
    const status = await worktreeStore.getWorktreeStatus(id);
    confirmingStatus = status;
    confirmingRemoveId = id;
  }

  async function handleCreateWorktree() {
    const wt = await worktreeStore.createWorktree();
    if (wt) {
      ui.navigateTo(wt.id);
    }
  }

  async function handleRemoveWorktree(id: string) {
    confirmingRemoveId = null;
    confirmingStatus = null;
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
      {#if configStore.agentEnabled && agentStatusesStore.isRunning(MAIN_WORKSPACE_ID)}
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

      <div class="flex flex-col gap-1">
      {#each workspaceItems as item (item.id)}
        <div
          class={[
            'rounded-md border transition-colors',
            ui.activeView === item.id
              ? 'border-accent/30 bg-surface'
              : 'border-border/50 bg-surface/50 hover:border-border hover:bg-surface'
          ]}
        >
          <button
            onclick={() => ui.navigateTo(item.id)}
            class={[
              'group flex w-full min-w-0 cursor-pointer items-center gap-2 px-3 py-2 text-left text-sm transition-colors',
              ui.activeView === item.id
                ? 'font-medium text-text'
                : 'text-text-muted hover:text-text'
            ]}
          >
            <div class="min-w-0 flex-1">
              <span class="block truncate">{item.label}</span>
              {#if item.settingUp}
                <span class="block text-xs font-normal text-text-faint animate-pulse">Setting up...</span>
              {:else if item.description}
                <span class="block text-xs font-normal text-text-faint">{item.description}</span>
              {/if}
            </div>
            <div class="relative ml-auto h-4 w-4 shrink-0 self-start mt-0.5">
              {#if agentStatusesStore.isRunning(item.id)}
                <div class="loader absolute inset-0" transition:fade={{ duration: 200 }}></div>
              {:else if item.id === MAIN_WORKSPACE_ID && mainHasChanges}
                <span class="icon-[uil--exclamation-triangle] absolute inset-0 block size-4 text-warning" title="Uncommitted changes"></span>
              {:else if item.id !== MAIN_WORKSPACE_ID && readyWorktreeIds.has(item.id)}
                <span class="icon-[uil--check] absolute inset-0 block size-4 text-success" title="Ready to integrate"></span>
              {:else if item.id !== MAIN_WORKSPACE_ID}
                <span
                  role="button"
                  tabindex="-1"
                  onclick={(e) => {
                    e.stopPropagation();
                    promptDestroy(item.id);
                  }}
                  onkeydown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      e.stopPropagation();
                      promptDestroy(item.id);
                    }
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
            <div class="flex flex-col gap-0.5 px-3 pb-2">
              {#each item.beans as wtBean (wtBean.id)}
                <button
                  onclick={() => {
                    ui.selectBeanForView(wtBean.id, item.id);
                    ui.navigateTo(item.id);
                  }}
                  class={[
                    'flex min-w-0 cursor-pointer items-start gap-1.5 rounded-xs border-l-2 px-2 py-1 text-left transition-colors hover:bg-surface-alt',
                    typeBorders[wtBean.type] ?? 'border-l-type-task-border'
                  ]}
                >
                  <code class="mt-0.5 shrink-0 text-[9px] leading-none text-text-faint">{wtBean.id.slice(-4)}</code>
                  <span class="min-w-0 flex-1 text-xs text-text-muted">{wtBean.title}</span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
      </div>
    {/if}
  </div>

  {#if confirmingRemoveId}
    {@const label = workspaceItems.find((w) => w.id === confirmingRemoveId)?.label ?? 'this worktree'}
    {@const warnings = [
      confirmingStatus?.hasChanges && 'uncommitted changes',
      confirmingStatus?.hasUnmergedCommits && 'unmerged commits'
    ].filter(Boolean)}
    <ConfirmModal
      title="Destroy Worktree"
      message={warnings.length > 0
        ? `This worktree has ${warnings.join(' and ')}. Are you sure you want to destroy the worktree for "${label}"? This cannot be undone.`
        : `Are you sure you want to destroy the worktree for "${label}"? This cannot be undone.`}
      confirmLabel="Destroy"
      danger
      onConfirm={() => handleRemoveWorktree(confirmingRemoveId!)}
      onCancel={() => { confirmingRemoveId = null; confirmingStatus = null; }}
    />
  {/if}
</nav>
