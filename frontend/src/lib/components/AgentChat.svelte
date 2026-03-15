<script lang="ts">
  import { AgentChatStore } from '$lib/agentChat.svelte';
  import { pickThinkingPhrase } from '$lib/thinkingPhrases';
  import { onDestroy } from 'svelte';
  import AgentMessages from './AgentMessages.svelte';
  import PendingInteraction from './PendingInteraction.svelte';
  import AgentComposer from './AgentComposer.svelte';

  interface Props {
    beanId: string;
    store?: AgentChatStore;
    setupRunning?: boolean;
    scrollToBottomTrigger?: number;
  }

  let { beanId, store: externalStore, setupRunning = false, scrollToBottomTrigger = 0 }: Props = $props();
  let internalScrollTrigger = $state(0);
  const combinedScrollTrigger = $derived(scrollToBottomTrigger + internalScrollTrigger);

  const ownStore = new AgentChatStore();
  const store = $derived(externalStore ?? ownStore);

  // Subscribe to agent session updates (skip if parent owns the store)
  $effect(() => {
    if (!externalStore) ownStore.subscribe(beanId);
  });

  onDestroy(() => {
    if (!externalStore) ownStore.unsubscribe();
  });

  const messages = $derived(store.session?.messages ?? []);
  const status = $derived(store.session?.status ?? null);
  const isRunning = $derived(status === 'RUNNING');
  const sessionError = $derived(store.session?.error ?? null);
  const systemStatus = $derived(store.session?.systemStatus ?? null);
  const planMode = $derived(store.session?.planMode ?? false);
  const agentMode = $derived<'plan' | 'act'>(planMode ? 'plan' : 'act');
  let thinkingPhrase = $state(pickThinkingPhrase());
  $effect(() => {
    if (isRunning) thinkingPhrase = pickThinkingPhrase();
  });
  const activityLabel = $derived(systemStatus ? `${systemStatus}...` : thinkingPhrase);
  const pendingInteraction = $derived(store.session?.pendingInteraction ?? null);
  const subagentActivities = $derived(store.session?.subagentActivities ?? []);

  function setAgentMode(mode: 'plan' | 'act') {
    store.setPlanMode(beanId, mode === 'plan');
    store.setActMode(beanId, mode === 'act');
  }

  async function approveInteraction() {
    // Enable act mode so the resumed process gets --dangerously-skip-permissions.
    // Without this, the process would restart in plan mode and loop.
    // IMPORTANT: Must await mode changes before sending — if sendMessage arrives
    // at the backend first, the process respawns in plan mode and loops.
    await store.setPlanMode(beanId, false);
    await store.setActMode(beanId, true);
    store.sendMessage(beanId, 'yes, proceed');
  }
</script>

<div class="flex h-full flex-col bg-surface">
  <AgentMessages {messages} {isRunning} {activityLabel} {subagentActivities} {setupRunning} scrollToBottomTrigger={combinedScrollTrigger} />

  <!-- Error banner -->
  {#if sessionError || store.error}
    <div class="border-t border-danger/20 bg-danger/10 px-4 py-2 text-danger">
      {sessionError || store.error}
    </div>
  {/if}

  {#if pendingInteraction}
    <PendingInteraction
      interaction={pendingInteraction}
      onApprove={approveInteraction}
      onSendMessage={(msg) => store.sendMessage(beanId, msg)}
    />
  {/if}

  <AgentComposer
    {beanId}
    {isRunning}
    hasMessages={messages.length > 0}
    {agentMode}
    {systemStatus}
    {subagentActivities}
    onSend={(text, images) => { internalScrollTrigger++; store.sendMessage(beanId, text, images); }}
    onStop={() => store.stop(beanId)}
    onSetMode={setAgentMode}
    onCompact={() => store.sendMessage(beanId, '/compact')}
    onClear={() => store.clearSession(beanId)}
  />
</div>
