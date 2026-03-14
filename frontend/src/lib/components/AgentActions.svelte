<script lang="ts">
  import { fade } from 'svelte/transition';
  import { AgentActionsStore } from '$lib/agentActions.svelte';

  interface Props {
    beanId: string;
    agentBusy: boolean;
  }

  let { beanId, agentBusy }: Props = $props();

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
      action.id === 'integrate'
        ? 'border-success/30 bg-success/10 text-success hover:bg-success/20'
        : 'btn-toggle-inactive'
    ]}
    disabled={agentBusy || !!store.executingAction || action.disabled}
    title={action.disabled ? (action.disabledReason ?? undefined) : (action.description ?? undefined)}
    onclick={() => store.execute(beanId, action.id)}
  >
    {#if action.id === 'integrate'}
      <span class="icon-[uil--check] size-4"></span>
    {/if}
    {action.label}
  </button>
{/each}
