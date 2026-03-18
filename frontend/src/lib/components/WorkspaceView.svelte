<script lang="ts">
  import { AgentChatStore } from '$lib/agentChat.svelte';
  import {
    StartRunDocument,
    StopRunDocument,
    IsRunningDocument,
    OpenInEditorDocument
  } from '$lib/graphql/generated';
  import { changesStore } from '$lib/changes.svelte';
  import { configStore } from '$lib/config.svelte';
  import { client } from '$lib/graphqlClient';
  import { ui } from '$lib/uiState.svelte';
  import { worktreeStore, MAIN_WORKSPACE_ID } from '$lib/worktrees.svelte';
  import { onDestroy } from 'svelte';
  import { diffSelectionStore } from '$lib/diffSelection.svelte';
  import SplitPane from './SplitPane.svelte';
  import AgentChat from './AgentChat.svelte';
  import BeanPane from './BeanPane.svelte';
  import ChangesPane from './ChangesPane.svelte';
  import DiffPane from './DiffPane.svelte';

  import TerminalPane from './TerminalPane.svelte';
  import ViewToolbar from './ViewToolbar.svelte';
  import AgentActions from './AgentActions.svelte';
  import ConfirmModal from './ConfirmModal.svelte';

  interface Props {
    worktreeId: string;
  }

  let { worktreeId }: Props = $props();

  // Run session state
  let isRunning = $state(false);
  let runPort = $state(0);

  const runSessionId = $derived(worktreeId + '__run');
  const hasRunCommand = $derived(!!configStore.worktreeRunCommand);

  async function handleRun() {
    const result = await client
      .mutation(StartRunDocument, { workspaceId: worktreeId })
      .toPromise();

    if (result.data) {
      isRunning = true;
      runPort = result.data.startRun;
    }
  }

  async function handleStop() {
    await client.mutation(StopRunDocument, { workspaceId: worktreeId }).toPromise();
    isRunning = false;
  }

  function handleRunSessionEnd() {
    isRunning = false;
  }

  function handleOpenApp() {
    if (runPort > 0) {
      window.open(`http://localhost:${runPort}/`, '_blank', 'noopener');
    }
  }

  async function handleOpenInEditor() {
    await client.mutation(OpenInEditorDocument, { workspaceId: worktreeId }).toPromise();
  }

  const agentStore = new AgentChatStore();

  $effect(() => {
    agentStore.subscribe(worktreeId);
  });

  $effect(() => {
    changesStore.startPolling(worktreePath);
    return () => changesStore.stopPolling();
  });

  // Clear diff selection when Changes pane is hidden
  $effect(() => {
    if (!ui.showChanges) {
      diffSelectionStore.clear();
    }
  });

  // Check initial run state on mount
  $effect(() => {
    const id = worktreeId;
    client.query(IsRunningDocument, { workspaceId: id }).toPromise().then((result) => {
      if (result.data?.isRunning) {
        isRunning = true;
      }
    });
  });

  onDestroy(() => {
    agentStore.unsubscribe();
  });

  const agentBusy = $derived(agentStore.session?.status === 'RUNNING');

  const hasNoChanges = $derived(changesStore.allChanges.length === 0);
  const isWorktree = $derived(worktreeId !== MAIN_WORKSPACE_ID);
  let confirmingDestroy = $state(false);

  async function handleDestroy() {
    confirmingDestroy = false;
    ui.navigateTo('planning');
    await worktreeStore.removeWorktree(worktreeId);
  }

  const destroying = $derived(worktreeStore.isDestroying(worktreeId));

  const worktree = $derived(
    worktreeId === MAIN_WORKSPACE_ID
      ? undefined
      : worktreeStore.worktrees.find((wt) => wt.id === worktreeId)
  );

  const worktreePath = $derived(worktree?.path);

  const setupRunning = $derived(worktree?.setupStatus === 'RUNNING');

  let scrollToBottomTrigger = $state(0);
</script>

{#snippet changesPanel()}
  <ChangesPane path={worktreePath} {worktreeId} onAgentMessage={() => scrollToBottomTrigger++} />
{/snippet}

{#snippet agentChatPanel()}
  <AgentChat beanId={worktreeId} store={agentStore} {setupRunning} {scrollToBottomTrigger} />
{/snippet}

{#snippet shellPane()}
  <div class="flex h-full min-h-0 flex-col bg-surface">
    <div class="pane-toolbar">
      <span>Terminal</span>
      <div class="flex-1"></div>
      <button onclick={() => ui.toggleTerminal()} class="btn-icon cursor-pointer" title="Close">&#x2715;</button>
    </div>
    <TerminalPane sessionId={worktreeId} hideToolbar />
  </div>
{/snippet}

{#snippet runPane()}
  <div class="flex h-full min-h-0 flex-col bg-surface">
    <div class="pane-toolbar">
      <span>Run</span>
    </div>
    {#key runSessionId}
      <TerminalPane sessionId={runSessionId} hideToolbar onSessionEnd={handleRunSessionEnd} />
    {/key}
  </div>
{/snippet}

{#snippet terminalPanel()}
  {#if ui.showTerminal && isRunning}
    <SplitPane
      direction="horizontal"
      panels={[
        { content: shellPane },
        { content: runPane, size: 500, persistKey: 'workspace-shell-run' }
      ]}
    />
  {:else if ui.showTerminal && ui.terminalInitialized}
    {@render shellPane()}
  {:else if isRunning}
    {@render runPane()}
  {/if}
{/snippet}

{#snippet beanDetailPanel()}
  {#if ui.currentBean}
    <BeanPane
      bean={ui.currentBean}
      onSelect={(b) => ui.selectBean(b)}
      onEdit={(b) => ui.openEditForm(b)}
      onClose={() => ui.clearSelection()}
    />
  {/if}
{/snippet}

{#snippet diffPanel()}
  <DiffPane />
{/snippet}

{#snippet mainContent()}
  <SplitPane
    direction="horizontal"
    panels={[
      { content: agentChatPanel },
      { content: diffPanel, size: 600, collapsed: !diffSelectionStore.selected, persistKey: 'workspace-diff' },
      { content: changesPanel, size: 420, collapsed: !ui.showChanges, persistKey: 'workspace-changes' },
      { content: beanDetailPanel, size: 480, collapsed: !ui.currentBean, persistKey: 'workspace-detail' }
    ]}
  />
{/snippet}

<div class="flex h-full flex-col">
  <ViewToolbar>
    {#if hasRunCommand}
      {#if isRunning}
        <button
          class="btn-toggle ml-1 cursor-pointer border-danger/30 bg-danger/10 text-danger hover:bg-danger/20"
          title="Stop the running process"
          onclick={handleStop}
        >
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
            <rect x="4" y="4" width="12" height="12" rx="1" />
          </svg>
          Stop
        </button>
        {#if runPort > 0}
          <button
            class="btn-toggle ml-1 cursor-pointer border-accent/30 bg-accent/10 text-accent hover:bg-accent/20"
            title={`Open http://localhost:${runPort}/`}
            onclick={handleOpenApp}
          >
            <span class="icon-[uil--external-link-alt] size-4"></span>
            Open
          </button>
        {/if}
      {:else}
        <button
          class="btn-toggle btn-toggle-inactive ml-1 cursor-pointer"
          title={`Run: ${configStore.worktreeRunCommand}`}
          onclick={handleRun}
        >
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
            <path d="M6.3 2.84A1.5 1.5 0 004 4.11v11.78a1.5 1.5 0 002.3 1.27l9.344-5.891a1.5 1.5 0 000-2.538L6.3 2.84z" />
          </svg>
          Run
        </button>
      {/if}
    {/if}
    <button
      class="btn-toggle btn-toggle-inactive ml-1 cursor-pointer"
      title="Open in VS Code"
      onclick={handleOpenInEditor}
    >
      <span class="icon-[simple-icons--visualstudiocode] size-4"></span>
      VS Code
    </button>
    {#snippet right()}
      <AgentActions beanId={worktreeId} {agentBusy} onExecute={() => scrollToBottomTrigger++} />
      {#if worktree?.pullRequest && configStore.worktreeIntegrateMode === 'pr'}
        {@const isMerged = worktree.pullRequest.state === 'merged'}
        <a
          class={[
            "btn-toggle ml-1 cursor-pointer",
            isMerged
              ? "border-purple-500/30 bg-purple-500/10 text-purple-400 hover:bg-purple-500/20"
              : "border-accent/30 bg-accent/10 text-accent hover:bg-accent/20"
          ]}
          href={worktree.pullRequest.url}
          target="_blank"
          rel="noopener noreferrer"
          title={`PR #${worktree.pullRequest.number}: ${worktree.pullRequest.title}`}
        >
          <span class={["size-4", isMerged ? "icon-[uil--code-branch] rotate-180" : "icon-[uil--external-link-alt]"]}></span>
          PR #{worktree.pullRequest.number}
        </a>
      {/if}
      {#if isWorktree}
        <button
          class={["btn-toggle ml-1 cursor-pointer border-accent/30 bg-accent/10 text-accent", (agentBusy || destroying) ? "opacity-50" : "hover:bg-accent/20"]}
          title={destroying ? "Destroying..." : agentBusy ? "Cannot destroy while agent is running" : "Close this workspace"}
          disabled={agentBusy || destroying}
          onclick={() => (confirmingDestroy = true)}
        >
          <span class="icon-[uil--archive] size-4"></span>
          Close Workspace
        </button>
      {/if}
    {/snippet}
  </ViewToolbar>

  <div class="flex min-h-0 flex-1 flex-col">
    <SplitPane
      direction="vertical"
      panels={[
        { content: mainContent },
        { content: terminalPanel, size: 300, collapsed: !(ui.showTerminal || isRunning), persistKey: 'workspace-terminal' }
      ]}
    />
  </div>
</div>

{#if confirmingDestroy}
  {@const label = worktree?.name ?? worktreeId}
  {@const warning = hasNoChanges
    ? `Are you sure you want to destroy the worktree for "${label}"? This cannot be undone.`
    : `The worktree "${label}" has uncommitted changes that will be lost. Are you sure you want to destroy it? This cannot be undone.`}
  <ConfirmModal
    title="Destroy Worktree"
    message={warning}
    confirmLabel="Destroy"
    danger
    onConfirm={handleDestroy}
    onCancel={() => (confirmingDestroy = false)}
  />
{/if}
