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

  $effect(() => {
    store.fetch(beanId);
  });

  $effect(() => {
    store.notifyAgentStatus(beanId, agentBusy);
  });
</script>

{#if agentBusy}
  <div class="loader mr-2" transition:fade={{ duration: 200 }}></div>
{/if}
{#each store.actions as action (action.id)}
  <button
    class={[
      'btn-toggle ml-1',
      action.id === 'integrate' || (action.id === 'create-pr' && action.label === 'Merge PR')
        ? 'border-success/30 bg-success/10 text-success hover:bg-success/20'
        : action.id === 'create-pr'
          ? 'border-accent/30 bg-accent/10 text-accent hover:bg-accent/20'
          : 'btn-toggle-inactive'
    ]}
    disabled={agentBusy || !!store.executingAction || action.disabled}
    title={action.disabled ? (action.disabledReason ?? undefined) : (action.description ?? undefined)}
    onclick={() => { store.execute(beanId, action.id); onExecute?.(); }}
  >
    {#if action.id === 'integrate'}
      <span class="icon-[uil--check] size-4"></span>
    {:else if action.id === 'create-pr' && action.label === 'Merge PR'}
      <span class="icon-[uil--check-circle] size-4"></span>
    {:else if action.id === 'create-pr'}
      <span class="icon-[uil--code-branch] size-4"></span>
    {/if}
    {action.label}
  </button>
{/each}
