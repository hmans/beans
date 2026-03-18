<script lang="ts">
  import { fade } from 'svelte/transition';
  import { onDestroy } from 'svelte';
  import { worktreeStore, MAIN_WORKSPACE_ID, type WorktreeStatus } from '$lib/worktrees.svelte';
  import { MainChangesDocument, WorktreeStatusesDocument } from '$lib/graphql/generated';
  import { beansStore, type Bean } from '$lib/beans.svelte';
  import { agentStatusesStore } from '$lib/agentStatuses.svelte';
  import { configStore } from '$lib/config.svelte';
  import { client } from '$lib/graphqlClient';
  import { ui } from '$lib/uiState.svelte';
  import { typeBorders } from '$lib/styles';
  import { decryptText } from '$lib/actions/decryptText';
  import ConfirmModal from './ConfirmModal.svelte';

  interface PullRequestInfo {
    state: string;
    checkStatus: string;
    number: number;
    url: string;
  }

  interface WorkspaceItem {
    id: string;
    label: string;
    description: string | null;
    beans: Bean[];
    settingUp: boolean;
    pullRequest: PullRequestInfo | null;
  }

  /** Look up full Bean objects for a worktree's detected bean IDs. */
  function beansForWorktree(beanIds: string[]): Bean[] {
    return beanIds
      .map((id) => beansStore.get(id))
      .filter((b): b is Bean => b != null);
  }

  const mainWorkspace: WorkspaceItem = $derived({ id: MAIN_WORKSPACE_ID, label: configStore.mainBranch, description: null, beans: [], settingUp: false, pullRequest: null });

  const workspaceItems = $derived([
    mainWorkspace,
    ...worktreeStore.worktrees.map((wt): WorkspaceItem => ({
      id: wt.id,
      label: wt.name ?? wt.branch,
      description: wt.description ?? null,
      beans: beansForWorktree(wt.beanIds),
      settingUp: wt.setupStatus === 'RUNNING',
      pullRequest: wt.pullRequest ? {
        state: wt.pullRequest.state,
        checkStatus: wt.pullRequest.checkStatus,
        number: wt.pullRequest.number,
        url: wt.pullRequest.url,
      } : null,
    }))
  ]);

  // Track which items have been seen so only genuinely new items animate.
  // Items present when stores finish their initial load are seeded into the
  // set without animation; anything arriving afterwards gets the effect.
  let seenWorkspaceIds = new Set<string>();
  let seenBeanIds = new Set<string>();
  let workspacesSettled = $state(false);
  let beansSettled = $state(false);

  $effect(() => {
    if (!workspacesSettled && worktreeStore.initialized) {
      // Seed with whatever is in the store at the end of initial load
      seenWorkspaceIds = new Set(worktreeStore.worktrees.map((wt) => wt.id));
      workspacesSettled = true;
    }
  });

  $effect(() => {
    if (!beansSettled && !beansStore.loading) {
      seenBeanIds = new Set(beansStore.all.filter((b) => b.worktreeId).map((b) => b.id));
      beansSettled = true;
    }
  });

  function isWorkspaceSeen(id: string): boolean {
    if (!workspacesSettled) return true; // still loading — skip animation
    if (seenWorkspaceIds.has(id)) return true;
    seenWorkspaceIds.add(id);
    return false;
  }

  function isBeanSeen(id: string): boolean {
    if (!beansSettled) return true;
    if (seenBeanIds.has(id)) return true;
    seenBeanIds.add(id);
    return false;
  }

  // Poll for uncommitted changes in the main repo and worktree integration readiness
  let mainHasChanges = $state(false);
  let worktreeStatuses = $state(new Map<string, WorktreeStatus>());
  let readyWorktreeIds = $derived(new Set(
    [...worktreeStatuses.entries()]
      .filter(([, s]) => s.hasChanges || s.hasUnmergedCommits)
      .map(([id]) => id)
  ));

  async function fetchStatuses() {
    const [mainResult, wtResult] = await Promise.all([
      client.query(MainChangesDocument, {}).toPromise(),
      client.query(WorktreeStatusesDocument, {}).toPromise()
    ]);
    mainHasChanges = (mainResult.data?.fileChanges?.length ?? 0) > 0;
    const statuses = new Map<string, WorktreeStatus>();
    for (const wt of wtResult.data?.worktrees ?? []) {
      statuses.set(wt.id, { hasChanges: wt.hasChanges, hasUnmergedCommits: wt.hasUnmergedCommits });
    }
    worktreeStatuses = statuses;
  }

  fetchStatuses();
  const statusInterval = setInterval(fetchStatuses, 3000);
  onDestroy(() => clearInterval(statusInterval));

  let confirmingRemoveId = $state<string | null>(null);
  let confirmingStatus = $state<WorktreeStatus | null>(null);

  function promptDestroy(id: string) {
    // Show modal immediately with cached status, then refresh in background
    confirmingStatus = worktreeStatuses.get(id) ?? null;
    confirmingRemoveId = id;
    // Fetch fresh status so warnings update if cache was stale
    client.query(WorktreeStatusesDocument, {}).toPromise().then((result) => {
      const fresh = (result.data?.worktrees ?? []).find((wt) => wt.id === id);
      if (confirmingRemoveId === id) {
        confirmingStatus = fresh ? { hasChanges: fresh.hasChanges, hasUnmergedCommits: fresh.hasUnmergedCommits } : null;
      }
    });
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
  <div class="flex h-14 shrink-0 items-center border-b border-border p-3">
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
          transition:fade={{ duration: 150 }}
          class={[
            'rounded-md border transition-colors',
            ui.activeView === item.id
              ? 'border-accent/30 bg-surface'
              : 'border-border/50 bg-surface/50 hover:border-border hover:bg-surface'
          ]}
        >
          <div class="group flex items-start">
            <button
              onclick={() => ui.navigateTo(item.id)}
              class={[
                'flex min-w-0 flex-1 cursor-pointer items-center gap-2 px-3 py-2 text-left text-sm transition-colors',
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
                  <span class="block text-xs font-normal text-text-faint" use:decryptText={{ text: item.description, immediate: isWorkspaceSeen(item.id) }}></span>
                {/if}
              </div>
            </button>
            <div class="relative mt-2.5 mr-2 h-4 w-4 shrink-0">
              {#if agentStatusesStore.isRunning(item.id)}
                <div class="loader absolute inset-0" transition:fade={{ duration: 200 }}></div>
              {:else if item.id === MAIN_WORKSPACE_ID && mainHasChanges}
                <span class="icon-[uil--exclamation-triangle] absolute inset-0 block size-4 text-warning" title="Uncommitted changes"></span>
              {:else if item.id !== MAIN_WORKSPACE_ID && configStore.worktreeIntegrateMode === 'pr' && item.pullRequest}
                {@const pr = item.pullRequest}
                {@const [iconClass, colorClass, title] =
                  pr.state === 'merged'
                    ? ['icon-[uil--code-branch] rotate-180', 'text-purple-400', `PR #${pr.number} merged`]
                    : pr.checkStatus === 'fail'
                      ? ['icon-[uil--times-circle]', 'text-danger', `PR #${pr.number}: checks failed`]
                      : pr.checkStatus === 'pending'
                        ? ['icon-[uil--clock]', 'text-warning', `PR #${pr.number}: checks running`]
                        : ['icon-[uil--check-circle]', 'text-success', `PR #${pr.number}: checks passed`]
                }
                <a href={pr.url} target="_blank" rel="noopener noreferrer" class={["absolute inset-0 block size-4 group-hover:hidden", iconClass, colorClass]} title={title}></a>
                <button
                  onclick={() => promptDestroy(item.id)}
                  class="absolute inset-0 hidden cursor-pointer items-center justify-center rounded text-text-faint transition-opacity hover:text-danger group-hover:flex"
                  aria-label="Destroy worktree"
                >
                  <span class="icon-[uil--archive] block size-3.5"></span>
                </button>
              {:else if item.id !== MAIN_WORKSPACE_ID && readyWorktreeIds.has(item.id)}
                <span class="icon-[uil--check] absolute inset-0 block size-4 text-success group-hover:hidden" title="Ready to integrate"></span>
                <button
                  onclick={() => promptDestroy(item.id)}
                  class="absolute inset-0 hidden cursor-pointer items-center justify-center rounded text-text-faint transition-opacity hover:text-danger group-hover:flex"
                  aria-label="Destroy worktree"
                >
                  <span class="icon-[uil--archive] block size-3.5"></span>
                </button>
              {:else if item.id !== MAIN_WORKSPACE_ID}
                <button
                  onclick={() => promptDestroy(item.id)}
                  class="absolute inset-0 flex cursor-pointer items-center justify-center rounded text-text-faint opacity-0 transition-opacity hover:text-danger group-hover:opacity-100"
                  aria-label="Destroy worktree"
                >
                  <span class="icon-[uil--archive] block size-3.5"></span>
                </button>
              {/if}
            </div>
          </div>

          {#if item.beans.length > 0}
            <div class="flex flex-col gap-0.5 px-3 pb-2">
              {#each item.beans as wtBean (wtBean.id)}
                <button
                  onclick={() => {
                    ui.selectBeanForView(wtBean.id, item.id);
                    ui.navigateTo(item.id);
                  }}
                  class={[
                    'flex min-w-0 cursor-pointer items-baseline gap-1.5 rounded border-l-2 bg-surface-alt/50 px-2 py-1 text-left shadow-sm transition-colors hover:bg-surface-alt',
                    typeBorders[wtBean.type] ?? 'border-l-type-task-border'
                  ]}
                >
                  <span class="min-w-0 flex-1 text-xs text-text-muted" use:decryptText={{ text: wtBean.title, immediate: isBeanSeen(wtBean.id) }}></span>
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
