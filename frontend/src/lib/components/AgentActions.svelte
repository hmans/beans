<script lang="ts">
  import { fade } from 'svelte/transition';
  import { AgentActionsStore } from '$lib/agentActions.svelte';

  interface Props {
    beanId: string;
    agentBusy: boolean;
    onExecute?: () => void;
  }

  let { beanId, agentBusy, onExecute }: Props = $props();

  const store = new AgentActionsStore();

  const integrationActionIds = new Set(['integrate', 'create-pr']);

  let standardActions = $derived(store.actions.filter((a) => !integrationActionIds.has(a.id)));
  let integrationActions = $derived(store.actions.filter((a) => integrationActionIds.has(a.id)));

  $effect(() => {
    // Fast initial fetch (skip forge/PR lookup) so local actions render instantly,
    // then immediately follow up with a full fetch to get PR state without waiting
    // for the first poll interval.
    store.fetch(beanId, true).then(() => store.fetch(beanId));
    store.startPolling(beanId);
    return () => store.stopPolling();
  });

  $effect(() => {
    store.notifyAgentStatus(beanId, agentBusy);
  });

  function prActionStyle(label: string): string {
    switch (label) {
      case 'Merge PR':
        return 'border-success/30 bg-success/10 text-success hover:bg-success/20';
      case 'Checks Running':
        return 'border-warning/30 bg-warning/10 text-warning';
      case 'Fix Tests':
        return 'border-danger/30 bg-danger/10 text-danger hover:bg-danger/20';
      case 'Loading...':
        return 'border-border bg-transparent text-text-muted';
      default:
        return 'border-accent/30 bg-accent/10 text-accent hover:bg-accent/20';
    }
  }
</script>

<div class="flex items-center gap-1">
  {#if agentBusy}
    <div class="loader mr-1" transition:fade={{ duration: 200 }}></div>
  {/if}
  {#each standardActions as action (action.id)}
    <button
      class={['btn-toggle btn-toggle-inactive']}
      disabled={agentBusy || !!store.executingAction || action.disabled}
      title={action.disabled ? (action.disabledReason ?? undefined) : (action.description ?? undefined)}
      onclick={() => { store.execute(beanId, action.id); onExecute?.(); }}
    >
      {action.label}
    </button>
  {/each}
</div>

{#if integrationActions.length > 0}
  <div class="ml-2 flex items-center gap-1">
    {#each integrationActions as action (action.id)}
      <button
        class={[
          'btn-toggle',
          action.id === 'integrate'
            ? 'border-success/30 bg-success/10 text-success hover:bg-success/20'
            : prActionStyle(action.label)
        ]}
        disabled={agentBusy || !!store.executingAction || action.disabled}
        title={action.disabled ? (action.disabledReason ?? undefined) : (action.description ?? undefined)}
        onclick={() => { store.execute(beanId, action.id); onExecute?.(); }}
      >
        {#if action.id === 'integrate'}
          <span class="icon-[uil--check] size-4"></span>
        {:else if action.id === 'create-pr' && action.label === 'Loading...'}
          <div class="loader size-4"></div>
        {:else if action.id === 'create-pr' && action.label === 'Merge PR'}
          <span class="icon-[uil--check-circle] size-4"></span>
        {:else if action.id === 'create-pr' && action.label === 'Checks Running'}
          <span class="icon-[uil--clock] size-4"></span>
        {:else if action.id === 'create-pr' && action.label === 'Fix Tests'}
          <span class="icon-[uil--exclamation-triangle] size-4"></span>
        {:else if action.id === 'create-pr'}
          <span class="icon-[uil--code-branch] size-4"></span>
        {/if}
        {action.label}
      </button>
    {/each}
  </div>
{/if}
