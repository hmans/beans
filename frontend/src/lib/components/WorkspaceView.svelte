<script lang="ts">
  import { AgentChatStore } from '$lib/agentChat.svelte';
  import { AgentActionsStore } from '$lib/agentActions.svelte';
  import { ui } from '$lib/uiState.svelte';
  import { worktreeStore } from '$lib/worktrees.svelte';
  import { onDestroy } from 'svelte';
  import SplitPane from './SplitPane.svelte';
  import AgentChat from './AgentChat.svelte';
  import ChangesPane from './ChangesPane.svelte';

  import TerminalPane from './TerminalPane.svelte';
  import ViewToolbar from './ViewToolbar.svelte';

  interface Props {
    worktreeId: string;
  }

  let { worktreeId }: Props = $props();

  const agentStore = new AgentChatStore();
  const actionsStore = new AgentActionsStore();

  $effect(() => {
    agentStore.subscribe(worktreeId);
  });

  onDestroy(() => {
    agentStore.unsubscribe();
  });

  const agentBusy = $derived(agentStore.session?.status === 'RUNNING');

  // Fetch agent actions on mount and when agent finishes
  $effect(() => {
    actionsStore.fetch(worktreeId);
  });

  $effect(() => {
    actionsStore.notifyAgentStatus(worktreeId, agentBusy);
  });

  const worktreePath = $derived(
    worktreeStore.worktrees.find((wt) => wt.id === worktreeId)?.path
  );
</script>

{#snippet changesPanel()}
  <ChangesPane path={worktreePath} />
{/snippet}

{#snippet agentChatPanel()}
  <AgentChat beanId={worktreeId} store={agentStore} />
{/snippet}

{#snippet terminalPanel()}
  {#if ui.terminalInitialized}
    <TerminalPane sessionId={worktreeId} />
  {/if}
{/snippet}

{#snippet mainContent()}
  <SplitPane
    direction="horizontal"
    panels={[
      { content: agentChatPanel },
      { content: changesPanel, size: 420, collapsed: !ui.showChanges, persistKey: 'workspace-changes' }
    ]}
  />
{/snippet}

<div class="flex h-full flex-col">
  <ViewToolbar>
    {#snippet right()}
      {#each actionsStore.actions as action (action.id)}
        <button
          class={[
            'btn-toggle btn-toggle-inactive ml-1'
          ]}
          disabled={agentBusy || !!actionsStore.executingAction}
          title={action.description ?? undefined}
          onclick={() => actionsStore.execute(worktreeId, action.id, agentBusy)}
        >
          {action.label}
        </button>
      {/each}
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
