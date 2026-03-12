<script lang="ts">
  import { AgentChatStore } from '$lib/agentChat.svelte';
  import { ui } from '$lib/uiState.svelte';
  import { worktreeStore } from '$lib/worktrees.svelte';
  import { onDestroy } from 'svelte';
  import SplitPane from './SplitPane.svelte';
  import AgentChat from './AgentChat.svelte';
  import ChangesPane from './ChangesPane.svelte';
  import PaneHeader from './PaneHeader.svelte';
  import TerminalPane from './TerminalPane.svelte';

  interface Props {
    worktreeId: string;
  }

  let { worktreeId }: Props = $props();

  const agentStore = new AgentChatStore();

  $effect(() => {
    agentStore.subscribe(worktreeId);
  });

  onDestroy(() => {
    agentStore.unsubscribe();
  });

  const agentBusy = $derived(agentStore.session?.status === 'RUNNING');

  const worktreePath = $derived(
    worktreeStore.worktrees.find((wt) => wt.id === worktreeId)?.path
  );
</script>

{#snippet agentToolbar()}
  <PaneHeader title="Agent">
    {#snippet actions()}
      <button
        onclick={() => ui.toggleChanges()}
        class={['btn-toggle-icon', ui.showChanges ? 'btn-toggle-active' : 'btn-toggle-inactive']}
        title={ui.showChanges ? 'Hide status' : 'Show status'}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="currentColor"
          class="h-4 w-4"
        >
          <path
            d="M18 2H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h10c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm-1 9h-3v3h-2v-3H9V9h3V6h2v3h3v2zM4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6zm12 9H10v-2h6v2z"
          />
        </svg>
      </button>
      <button
        onclick={() => ui.toggleTerminal()}
        class={['btn-toggle-icon', ui.showTerminal ? 'btn-toggle-active' : 'btn-toggle-inactive']}
        title={ui.showTerminal ? 'Hide terminal' : 'Show terminal'}
      >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
          <path fill-rule="evenodd" d="M3.25 3A2.25 2.25 0 001 5.25v9.5A2.25 2.25 0 003.25 17h13.5A2.25 2.25 0 0019 14.75v-9.5A2.25 2.25 0 0016.75 3H3.25zm.943 8.752a.75.75 0 01.055-1.06L6.128 9l-1.88-1.693a.75.75 0 111.004-1.114l2.5 2.25a.75.75 0 010 1.114l-2.5 2.25a.75.75 0 01-1.06-.055zM9.75 10.25a.75.75 0 000 1.5h2.5a.75.75 0 000-1.5h-2.5z" clip-rule="evenodd" />
        </svg>
      </button>
    {/snippet}
  </PaneHeader>
{/snippet}

{#snippet agentChatPanel()}
  <div class="flex h-full flex-col border-l border-border bg-surface">
    {@render agentToolbar()}
    <div class="min-h-0 flex-1">
      <AgentChat beanId={worktreeId} store={agentStore} />
    </div>
  </div>
{/snippet}

{#snippet changesChatSplit()}
  {#if ui.showChanges}
    <SplitPane
      direction="horizontal"
      side="end"
      persistKey="workspace-changes-chat-split"
      initialSize={480}
    >
      {#snippet children()}
        <ChangesPane path={worktreePath} beanId={worktreeId} {agentBusy} />
      {/snippet}
      {#snippet aside()}
        {@render agentChatPanel()}
      {/snippet}
    </SplitPane>
  {:else}
    {@render agentChatPanel()}
  {/if}
{/snippet}

<SplitPane direction="vertical" side="end" persistKey="workspace-terminal" initialSize={300} collapsed={!ui.showTerminal}>
  {#snippet children()}
    {@render changesChatSplit()}
  {/snippet}
  {#snippet aside()}
    {#if ui.terminalInitialized}
      <TerminalPane sessionId={worktreeId} onClose={() => ui.toggleTerminal()} />
    {/if}
  {/snippet}
</SplitPane>
