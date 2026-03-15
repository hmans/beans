<script lang="ts">
  import type { AgentMessage, SubagentActivity } from '$lib/agentChat.svelte';
  import { beansStore } from '$lib/beans.svelte';
  import { ui } from '$lib/uiState.svelte';
  import { renderMarkdown } from '$lib/markdown';
  import { fade } from 'svelte/transition';
  import { decryptText } from '$lib/actions/decryptText';

  interface Props {
    messages: AgentMessage[];
    isRunning: boolean;
    activityLabel: string;
    subagentActivities: SubagentActivity[];
    setupRunning?: boolean;
    scrollToBottomTrigger?: number;
  }

  let { messages, isRunning, activityLabel, subagentActivities, setupRunning = false, scrollToBottomTrigger = 0 }: Props = $props();

  // Track how many messages existed after the initial load so only
  // genuinely new messages get the decrypt animation.  The prop may be
  // empty at mount (data arrives via subscription), so we wait for the
  // first non-empty snapshot before considering the component "settled".
  let settledMessageCount = $state(-1);

  $effect(() => {
    if (settledMessageCount === -1 && messages.length > 0) {
      settledMessageCount = messages.length;
    }
  });

  let messagesEl: HTMLDivElement | undefined = $state();
  let renderedMessages = $state<Map<string, string>>(new Map());
  let stuckToBottom = $state(true);
  let expandedDiffs = $state<Set<number>>(new Set());

  function toggleDiff(index: number) {
    const next = new Set(expandedDiffs);
    if (next.has(index)) {
      next.delete(index);
    } else {
      next.add(index);
    }
    expandedDiffs = next;
  }

  function diffLineClass(line: string): string {
    if (line.startsWith('+') && !line.startsWith('+++')) return 'diff-add';
    if (line.startsWith('-') && !line.startsWith('---')) return 'diff-del';
    if (line.startsWith('@@')) return 'diff-hunk';
    return '';
  }

  function handleMessagesScroll() {
    if (!messagesEl) return;
    const { scrollTop, scrollHeight, clientHeight } = messagesEl;
    stuckToBottom = scrollHeight - scrollTop - clientHeight < 20;
  }

  // Auto-scroll to bottom when messages change or markdown finishes rendering,
  // but only if the user hasn't scrolled up to read earlier messages.
  $effect(() => {
    messages.length;
    renderedMessages.size;
    if (messagesEl && stuckToBottom) {
      requestAnimationFrame(() => {
        if (messagesEl) {
          messagesEl.scrollTop = messagesEl.scrollHeight;
        }
      });
    }
  });

  // Render markdown for assistant and user messages (including the one being streamed).
  // The key includes content length, so each new delta triggers a re-render.
  $effect(() => {
    for (let i = 0; i < messages.length; i++) {
      const msg = messages[i];
      if (msg.role !== 'ASSISTANT' && msg.role !== 'USER') continue;

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
    if (!msg || (msg.role !== 'ASSISTANT' && msg.role !== 'USER')) return null;
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

  // Force scroll to bottom when triggered externally (e.g. action buttons, composer send)
  $effect(() => {
    if (scrollToBottomTrigger > 0) {
      stuckToBottom = true;
      scrollToBottom();
    }
  });

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
    {#if setupRunning}
      <div class="rounded-lg border border-border bg-surface px-3 py-2 text-text-muted animate-pulse">
        <p>Workspace setup is running...</p>
      </div>
    {/if}

    {#if messages.length === 0 && !setupRunning}
      <div class="flex h-full items-center justify-center text-text-faint">
        <p>Send a message to start a conversation with the agent.</p>
      </div>
    {:else if messages.length > 0}
      {#each messages as msg, i}
        {#if msg.role === 'USER'}
          <div class="rounded-lg border border-border bg-surface-alt px-3 py-2">
            <div>
              {#if getRenderedContent(i)}
                <div class="agent-prose prose max-w-none min-w-0 text-text">
                  {@html getRenderedContent(i)}
                </div>
              {:else if msg.content}
                <p class="whitespace-pre-wrap text-text">{msg.content}</p>
              {/if}
              {#if msg.images.length > 0}
                <div class="mt-2 flex flex-wrap gap-2">
                  {#each msg.images as img}
                    <img
                      src={img.url}
                      alt="Attachment"
                      class="max-h-48 max-w-xs rounded border border-border"
                    />
                  {/each}
                </div>
              {/if}
            </div>
          </div>
        {:else if msg.role === 'INFO'}
          <div class="rounded-lg border border-border bg-surface px-3 py-2 text-text-muted">
            <p class="whitespace-pre-wrap" use:decryptText={{ text: msg.content, immediate: settledMessageCount === -1 || i < settledMessageCount }}></p>
          </div>
        {:else if msg.role === 'TOOL'}
          <div class="flex gap-2 text-text-faint">
            <span class="shrink-0 select-none">&middot;</span>
            <div class="min-w-0">
              {#if msg.diff}
                <button
                  class="flex cursor-pointer gap-2 text-left hover:text-text-muted"
                  onclick={() => toggleDiff(i)}
                >
                  <span class="shrink-0 select-none">{expandedDiffs.has(i) ? '▾' : '▸'}</span>
                  <span use:decryptText={{ text: msg.content, immediate: settledMessageCount === -1 || i < settledMessageCount }}></span>
                </button>
              {:else}
                <span use:decryptText={{ text: msg.content, immediate: settledMessageCount === -1 || i < settledMessageCount }}></span>
              {/if}
              {#if msg.diff && expandedDiffs.has(i)}
                <pre class="mt-1 max-h-64 overflow-auto rounded border border-border bg-surface-alt p-2 font-mono leading-relaxed">{#each msg.diff.split('\n') as line}<span class={diffLineClass(line)}>{line}
</span>{/each}</pre>
              {/if}
            </div>
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
            <span use:decryptText={activityLabel}></span>
          </div>
        {/if}
      {/each}

      {#if isRunning && subagentActivities.length === 0 && (messages.length === 0 || messages[messages.length - 1].role === 'USER')}
        <div class="flex gap-2 text-text-muted">
          <span class="shrink-0 select-none">&middot;</span>
          <span use:decryptText={activityLabel}></span>
        </div>
      {/if}

      {#each subagentActivities as activity (activity.taskId)}
        {@const activityText = `#${activity.index} ${activity.description || 'Subagent'}${activity.currentTool ? ` — ${activity.currentTool}` : ''}`}
        <div class="text-text-faint">
          <span use:decryptText={activityText}></span>
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
