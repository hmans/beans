<script lang="ts">
  import { AgentChatStore } from '$lib/agentChat.svelte';
  import { beansStore } from '$lib/beans.svelte';
  import { ui } from '$lib/uiState.svelte';
  import { renderMarkdown } from '$lib/markdown';
  import { onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';

  interface Props {
    beanId: string;
    store?: AgentChatStore;
  }

  let { beanId, store: externalStore }: Props = $props();

  const ownStore = new AgentChatStore();
  const store = $derived(externalStore ?? ownStore);

  let inputText = $state('');
  let messagesEl: HTMLDivElement | undefined = $state();
  let renderedMessages = $state<Map<string, string>>(new Map());

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

  function setAgentMode(mode: 'plan' | 'act') {
    store.setPlanMode(beanId, mode === 'plan');
    store.setActMode(beanId, mode === 'act');
  }
  const activityLabel = $derived(systemStatus ? `${systemStatus}...` : 'thinking...');
  const pendingInteraction = $derived(store.session?.pendingInteraction ?? null);
  const subagentActivities = $derived(store.session?.subagentActivities ?? []);

  // Render plan content as markdown when available
  let renderedPlanContent = $state<string | null>(null);
  $effect(() => {
    const content = pendingInteraction?.planContent;
    if (content) {
      renderMarkdown(content).then((html) => {
        renderedPlanContent = html;
      });
    } else {
      renderedPlanContent = null;
    }
  });

  function approveInteraction() {
    if (!pendingInteraction) return;
    store.sendMessage(beanId, 'yes, proceed');
  }

  function approveInteractionWithAct() {
    if (!pendingInteraction) return;
    store.setActMode(beanId, true);
    store.sendMessage(beanId, 'yes, proceed');
  }

  function rejectInteraction() {
    if (!pendingInteraction) return;
    if (pendingInteraction.type === 'EXIT_PLAN') {
      // Rejected exiting plan → go back to plan mode
      store.setPlanMode(beanId, true);
    } else {
      // Rejected entering plan → go back to work mode
      store.setPlanMode(beanId, false);
    }
  }

  // Track whether the user is scrolled to the bottom of the messages area.
  // When they scroll up, we stop auto-scrolling so they can read earlier messages.
  let stuckToBottom = $state(true);

  function handleMessagesScroll() {
    if (!messagesEl) return;
    const { scrollTop, scrollHeight, clientHeight } = messagesEl;
    stuckToBottom = scrollHeight - scrollTop - clientHeight < 20;
  }

  // Auto-scroll to bottom when messages change, but only if the user
  // hasn't scrolled up to read earlier messages.
  $effect(() => {
    messages.length;
    if (messagesEl && stuckToBottom) {
      requestAnimationFrame(() => {
        if (messagesEl) {
          messagesEl.scrollTop = messagesEl.scrollHeight;
        }
      });
    }
  });

  // Render markdown for assistant messages (including the one being streamed).
  // The key includes content length, so each new delta triggers a re-render.
  $effect(() => {
    for (let i = 0; i < messages.length; i++) {
      const msg = messages[i];
      if (msg.role !== 'ASSISTANT') continue;

      const key = `${i}:${msg.content.length}`;
      if (!renderedMessages.has(key)) {
        renderMarkdown(msg.content).then((html) => {
          renderedMessages = new Map(renderedMessages).set(key, html);
        });
      }
    }
  });

  function getRenderedContent(index: number): string | null {
    const msg = messages[index];
    if (!msg || msg.role !== 'ASSISTANT') return null;
    const key = `${index}:${msg.content.length}`;
    return renderedMessages.get(key) ?? null;
  }

  async function send() {
    const text = inputText.trim();
    if (!text) return;

    inputText = '';
    await store.sendMessage(beanId, text);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  function handleBeanLinkClick(e: MouseEvent) {
    const target = (e.target as HTMLElement).closest<HTMLElement>('[data-bean-id]');
    if (!target) return;
    e.preventDefault();
    const linkedBean = beansStore.get(target.dataset.beanId!);
    if (linkedBean) ui.selectBean(linkedBean);
  }
</script>

<div class="flex h-full flex-col font-mono text-sm">
  <!-- Messages area -->
  <div class="relative min-h-0 flex-1">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      bind:this={messagesEl}
      class="h-full space-y-3 overflow-y-auto p-4"
      onclick={handleBeanLinkClick}
      onscroll={handleMessagesScroll}
    >
      {#if messages.length === 0}
        <div class="flex h-full items-center justify-center text-text-faint">
          <p>Send a message to start a conversation with the agent.</p>
        </div>
      {:else}
        {#each messages as msg, i}
          {#if msg.role === 'USER'}
            <div class="flex gap-2">
              <span class="shrink-0 font-bold text-accent select-none">&gt;</span>
              <p class="whitespace-pre-wrap text-text">{msg.content}</p>
            </div>
          {:else if msg.role === 'TOOL'}
            <div class="flex gap-2 text-xs text-text-faint">
              <span class="shrink-0 select-none">&middot;</span>
              <span>{msg.content}</span>
            </div>
          {:else if getRenderedContent(i)}
            <div class="flex gap-2">
              <span class="shrink-0 text-text-muted select-none">&middot;</span>
              <div class="agent-prose prose max-w-none min-w-0 text-text">
                {@html getRenderedContent(i)}
              </div>
            </div>
          {:else if msg.content}
            <div class="flex gap-2">
              <span class="shrink-0 text-text-muted select-none">&middot;</span>
              <p class="whitespace-pre-wrap text-text">{msg.content}</p>
            </div>
          {:else if isRunning}
            <div class="flex gap-2 text-text-muted">
              <span class="shrink-0 select-none">&middot;</span>
              <span class="animate-pulse">{activityLabel}</span>
            </div>
          {/if}
        {/each}

        {#if isRunning && subagentActivities.length === 0 && (messages.length === 0 || messages[messages.length - 1].role === 'USER')}
          <div class="flex gap-2 text-text-muted">
            <span class="shrink-0 select-none">&middot;</span>
            <span class="animate-pulse">{activityLabel}</span>
          </div>
        {/if}

        {#each subagentActivities as activity (activity.taskId)}
          <div class="flex gap-2 text-xs text-text-faint">
            <span class="shrink-0 select-none">&middot;</span>
            <span class="animate-pulse">
              <span class="text-text-muted">#{activity.index}</span>
              {activity.description || 'Subagent'}{activity.currentTool ? ` — ${activity.currentTool}` : ''}
            </span>
          </div>
        {/each}
      {/if}
    </div>

    {#if !stuckToBottom}
      <button
        transition:fade={{ duration: 150 }}
        class="absolute right-3 bottom-3 flex size-8 cursor-pointer items-center justify-center rounded-full border border-border bg-surface-alt text-text-muted shadow-md transition-colors hover:text-text"
        onclick={() => {
          if (messagesEl) {
            messagesEl.scrollTop = messagesEl.scrollHeight;
          }
        }}
      >
        &#8595;
      </button>
    {/if}
  </div>

  <!-- Error banner -->
  {#if sessionError || store.error}
    <div class="border-t border-danger/20 bg-danger/10 px-4 py-2 text-danger">
      {sessionError || store.error}
    </div>
  {/if}

  <!-- Pending interaction approval (plan mode transitions) -->
  {#if pendingInteraction && pendingInteraction.type !== 'ASK_USER'}
    <div
      class={[
        'border-t p-3',
        pendingInteraction.type === 'EXIT_PLAN'
          ? 'border-status-in-progress-text/20 bg-status-in-progress-bg/50'
          : 'border-warning/20 bg-warning/5'
      ]}
    >
      <p class="mb-2 font-mono text-xs text-text-muted">
        {#if pendingInteraction.type === 'EXIT_PLAN'}
          Agent wants to leave plan mode and start working.
        {:else}
          Agent wants to enter plan mode to analyze before making changes.
        {/if}
      </p>

      {#if renderedPlanContent}
        <div class="mb-3 max-h-48 overflow-y-auto rounded border border-border bg-surface p-3">
          <div class="agent-prose prose max-w-none min-w-0 text-xs text-text">
            {@html renderedPlanContent}
          </div>
        </div>
      {/if}

      <div class="flex gap-2">
        <button
          onclick={approveInteraction}
          class={[
            'cursor-pointer rounded px-3 py-1 font-mono text-xs transition-colors',
            pendingInteraction.type === 'EXIT_PLAN'
              ? 'bg-status-in-progress-text text-white hover:opacity-90'
              : 'bg-warning text-white hover:opacity-90'
          ]}
        >
          Approve
        </button>
        {#if pendingInteraction.type === 'EXIT_PLAN'}
          <button
            onclick={approveInteractionWithAct}
            class="cursor-pointer rounded bg-danger px-3 py-1 font-mono text-xs text-white transition-colors hover:opacity-90"
          >
            Approve + Act
          </button>
        {/if}
        <button
          onclick={rejectInteraction}
          class="cursor-pointer rounded border border-border px-3 py-1 font-mono text-xs text-text-muted transition-colors hover:bg-surface-alt"
        >
          Reject
        </button>
      </div>
    </div>
  {/if}

  <!-- Ask user interaction — highlight that agent is waiting for a reply -->
  {#if pendingInteraction?.type === 'ASK_USER'}
    <div class="border-t border-accent/30 bg-accent/5 px-3 py-2">
      <p class="font-mono text-xs text-accent">
        Agent is waiting for your answer — type your reply below.
      </p>
    </div>
  {/if}

  <!-- Composer -->
  <div class="border-t border-border bg-surface p-3">
    {#if isRunning}
      <div class="flex items-center gap-2 px-1 pb-2 text-text-muted">
        <span class="agent-spinner"></span>
        <span class="font-mono text-xs">
          {#if subagentActivities.length > 0}
            {subagentActivities.length} subagent{subagentActivities.length > 1 ? 's' : ''} working...
          {:else if systemStatus}
            Agent is {systemStatus}...
          {:else}
            Agent is working...
          {/if}
        </span>
      </div>
    {/if}
    <div class="flex items-end gap-2">
      <textarea
        bind:value={inputText}
        onkeydown={handleKeydown}
        placeholder="Send a message..."
        rows={1}
        class="flex-1 resize-none rounded border border-border bg-surface-alt px-3 py-2 font-mono text-sm
					text-text placeholder:text-text-faint
					focus:border-accent focus:ring-2 focus:ring-accent/40 focus:outline-none"
      ></textarea>

      <button
        onclick={send}
        disabled={!inputText.trim()}
        class="inline-flex shrink-0 items-center gap-1.5 rounded bg-accent px-3 py-2 font-mono
					text-sm text-accent-text transition-colors hover:bg-accent/90
					disabled:cursor-not-allowed disabled:opacity-50"
      >
        <span class="icon-[uil--message] size-4"></span>
        Send
      </button>

      {#if isRunning}
        <button
          onclick={() => store.stop(beanId)}
          class="inline-flex shrink-0 items-center gap-1.5 rounded bg-danger px-3 py-2 font-mono
						text-sm text-white transition-colors hover:bg-danger/90"
        >
          <span class="icon-[uil--stop-circle] size-4"></span>
          Stop
        </button>
      {/if}
    </div>

    <!-- Mode toggle + Clear -->
    <div class="flex items-center gap-3 pt-2">
      <div class={['flex', isRunning && 'pointer-events-none opacity-50']}>
        <button
          onclick={() => setAgentMode('plan')}
          disabled={isRunning}
          class={[
            'btn-tab-sm rounded-l',
            agentMode === 'plan'
              ? 'border-warning/30 bg-warning/10 text-warning'
              : 'btn-tab-sm-inactive'
          ]}
        >
          <span class="icon-[uil--eye] size-3"></span>
          Plan
        </button>
        <button
          onclick={() => setAgentMode('act')}
          disabled={isRunning}
          class={[
            'btn-tab-sm rounded-r border-l-0',
            agentMode === 'act'
              ? 'border-danger/30 bg-danger/10 text-danger'
              : 'btn-tab-sm-inactive'
          ]}
        >
          <span class="icon-[uil--play] size-3"></span>
          Act
        </button>
      </div>

      <div
        class={['flex', (isRunning || messages.length === 0) && 'pointer-events-none opacity-30']}
      >
        <button
          onclick={() => store.sendMessage(beanId, '/compact')}
          disabled={isRunning || messages.length === 0}
          class="btn-tab-sm btn-tab-sm-inactive rounded-l"
        >
          <span class="icon-[uil--compress-arrows] size-3"></span>
          Compact
        </button>
        <button
          onclick={() => store.clearSession(beanId)}
          disabled={isRunning || messages.length === 0}
          class="btn-tab-sm btn-tab-sm-inactive rounded-r border-l-0"
        >
          <span class="icon-[uil--trash-alt] size-3"></span>
          Clear
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .agent-spinner {
    display: inline-block;
    width: 12px;
    height: 12px;
    border: 2px solid currentColor;
    border-right-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  /* Ensure rendered markdown inherits monospace and uniform font size,
	   but exclude code blocks so Shiki highlighting renders properly */
  .agent-prose :global(*:not(pre, pre *, code)) {
    font-family: inherit;
    font-size: inherit;
  }

  .agent-prose :global(h1),
  .agent-prose :global(h2),
  .agent-prose :global(h3),
  .agent-prose :global(h4),
  .agent-prose :global(h5),
  .agent-prose :global(h6) {
    font-size: inherit;
    font-weight: bold;
  }
</style>
