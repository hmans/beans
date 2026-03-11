<script lang="ts">
  import type { AgentMessage, SubagentActivity } from '$lib/agentChat.svelte';
  import { beansStore } from '$lib/beans.svelte';
  import { ui } from '$lib/uiState.svelte';
  import { renderMarkdown } from '$lib/markdown';
  import { fade } from 'svelte/transition';

  interface Props {
    messages: AgentMessage[];
    isRunning: boolean;
    activityLabel: string;
    subagentActivities: SubagentActivity[];
  }

  let { messages, isRunning, activityLabel, subagentActivities }: Props = $props();

  let messagesEl: HTMLDivElement | undefined = $state();
  let renderedMessages = $state<Map<string, string>>(new Map());
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

  function handleBeanLinkClick(e: MouseEvent) {
    const target = (e.target as HTMLElement).closest<HTMLElement>('[data-bean-id]');
    if (!target) return;
    e.preventDefault();
    const linkedBean = beansStore.get(target.dataset.beanId!);
    if (linkedBean) ui.selectBean(linkedBean);
  }

  function scrollToBottom() {
    if (messagesEl) {
      messagesEl.scrollTop = messagesEl.scrollHeight;
    }
  }
</script>

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
            <div>
              {#if msg.content}
                <p class="whitespace-pre-wrap text-text">{msg.content}</p>
              {/if}
              {#if msg.images.length > 0}
                <div class="mt-2 flex flex-wrap gap-2">
                  {#each msg.images as img}
                    <img
                      src={img.url}
                      alt="Attached image"
                      class="max-h-48 max-w-xs rounded border border-border"
                    />
                  {/each}
                </div>
              {/if}
            </div>
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
      onclick={scrollToBottom}
    >
      &#8595;
    </button>
  {/if}
</div>

<style>
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
