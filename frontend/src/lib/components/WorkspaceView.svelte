<script lang="ts">
  import { AgentChatStore } from '$lib/agentChat.svelte';
  import { WriteTerminalInputDocument, OpenInEditorDocument } from '$lib/graphql/generated';
  import { changesStore } from '$lib/changes.svelte';
  import { configStore } from '$lib/config.svelte';
  import { client } from '$lib/graphqlClient';
  import { ui } from '$lib/uiState.svelte';
  import { worktreeStore, MAIN_WORKSPACE_ID } from '$lib/worktrees.svelte';
  import { onDestroy } from 'svelte';
  import SplitPane from './SplitPane.svelte';
  import AgentChat from './AgentChat.svelte';
  import BeanPane from './BeanPane.svelte';
  import ChangesPane from './ChangesPane.svelte';

  import TerminalPane from './TerminalPane.svelte';
  import ViewToolbar from './ViewToolbar.svelte';
  import AgentActions from './AgentActions.svelte';
  import ConfirmModal from './ConfirmModal.svelte';

  async function handleRun() {
    // Show and initialize the terminal
    ui.terminalInitialized = true;
    ui.showTerminal = true;

    // Write the run command — the resolver creates the session on demand
    // if the terminal pane hasn't connected via WebSocket yet.
    await client
      .mutation(WriteTerminalInputDocument, {
        sessionId: worktreeId,
        data: configStore.worktreeRunCommand + '\n'
      })
      .toPromise();
  }

  async function handleOpenInEditor() {
    await client.mutation(OpenInEditorDocument, { workspaceId: worktreeId }).toPromise();
  }

  interface Props {
    worktreeId: string;
  }

  let { worktreeId }: Props = $props();

  const agentStore = new AgentChatStore();

  $effect(() => {
    agentStore.subscribe(worktreeId);
  });

  $effect(() => {
    changesStore.startPolling(worktreePath);
    return () => changesStore.stopPolling();
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

{#snippet terminalPanel()}
  {#if ui.terminalInitialized}
    <TerminalPane sessionId={worktreeId} />
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

{#snippet mainContent()}
  <SplitPane
    direction="horizontal"
    panels={[
      { content: agentChatPanel },
      { content: changesPanel, size: 420, collapsed: !ui.showChanges, persistKey: 'workspace-changes' },
      { content: beanDetailPanel, size: 480, collapsed: !ui.currentBean, persistKey: 'workspace-detail' }
    ]}
  />
{/snippet}

<div class="flex h-full flex-col">
  <ViewToolbar>
    {#if configStore.worktreeRunCommand}
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
      {#if worktree?.pullRequest}
        <a
          class="btn-toggle ml-1 cursor-pointer border-accent/30 bg-accent/10 text-accent hover:bg-accent/20"
          href={worktree.pullRequest.url}
          target="_blank"
          rel="noopener noreferrer"
          title={`PR #${worktree.pullRequest.number}: ${worktree.pullRequest.title}`}
        >
          <span class="icon-[uil--external-link-alt] size-4"></span>
          PR #{worktree.pullRequest.number}
        </a>
      {/if}
      {#if isWorktree}
        <button
          class={["btn-toggle ml-1 cursor-pointer border-border bg-transparent text-text-muted", agentBusy ? "opacity-50" : "hover:text-danger"]}
          title={agentBusy ? "Cannot destroy while agent is running" : "Destroy this worktree"}
          disabled={agentBusy}
          onclick={() => (confirmingDestroy = true)}
        >
          <span class="icon-[uil--archive] size-4"></span>
        </button>
      {/if}
    {/snippet}
  </ViewToolbar>

  <div class="flex min-h-0 flex-1 flex-col">
    <SplitPane
      direction="vertical"
      panels={[
        { content: mainContent },
        { content: terminalPanel, size: 300, collapsed: !ui.showTerminal, persistKey: 'workspace-terminal' }
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
