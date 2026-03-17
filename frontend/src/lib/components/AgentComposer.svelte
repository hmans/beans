<script lang="ts">
  import type { SubagentActivity } from '$lib/agentChat.svelte';

  const MAX_IMAGE_SIZE = 5 * 1024 * 1024;
  const ALLOWED_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];

  interface Props {
    beanId: string;
    isRunning: boolean;
    hasMessages: boolean;
    agentMode: 'plan' | 'act';
    effort: string;
    systemStatus: string | null;
    subagentActivities: SubagentActivity[];
    quickReplies: string[];
    onSend: (message: string, images?: { data: string; mediaType: string }[]) => void;
    onStop: () => void;
    onSetMode: (mode: 'plan' | 'act') => void;
    onSetEffort: (effort: string) => void;
    onCompact: () => void;
    onClear: () => void;
  }

  let {
    beanId,
    isRunning,
    hasMessages,
    agentMode,
    effort,
    systemStatus,
    subagentActivities,
    quickReplies,
    onSend,
    onStop,
    onSetMode,
    onSetEffort,
    onCompact,
    onClear
  }: Props = $props();

  const inputStorageKey = $derived(`agent-chat-input:${beanId}`);
  let inputText = $state('');
  let pendingImages = $state<{ data: string; mediaType: string; preview: string }[]>([]);
  let isDragging = $state(false);
  let fileInputEl: HTMLInputElement | undefined = $state();
  let textareaEl: HTMLTextAreaElement | undefined = $state();

  // Focus the textarea when switching to a new bean/workspace
  $effect(() => {
    beanId;
    textareaEl?.focus();
  });

  // Load persisted composer input when beanId changes
  $effect(() => {
    inputText = localStorage.getItem(inputStorageKey) ?? '';
  });

  // Persist composer input to localStorage so it survives navigation/reloads
  $effect(() => {
    if (inputText) {
      localStorage.setItem(inputStorageKey, inputText);
    } else {
      localStorage.removeItem(inputStorageKey);
    }
  });

  function addImageFile(file: File) {
    if (!ALLOWED_IMAGE_TYPES.includes(file.type)) return;
    if (file.size > MAX_IMAGE_SIZE) return;

    const preview = URL.createObjectURL(file);
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      // Strip the data URL prefix to get raw base64
      const base64 = result.split(',')[1];
      pendingImages = [...pendingImages, { data: base64, mediaType: file.type, preview }];
    };
    reader.readAsDataURL(file);
  }

  function removeImage(index: number) {
    URL.revokeObjectURL(pendingImages[index].preview);
    pendingImages = pendingImages.filter((_, i) => i !== index);
  }

  function handlePaste(e: ClipboardEvent) {
    if (!e.clipboardData) return;
    const items = Array.from(e.clipboardData.items);
    const imageItems = items.filter((item) => ALLOWED_IMAGE_TYPES.includes(item.type));
    if (imageItems.length === 0) return;
    // Only prevent default text paste when there's no text content
    // (i.e., this is a screenshot paste, not a rich-text copy with inline images)
    const hasText = items.some((item) => item.type === 'text/plain');
    if (!hasText) e.preventDefault();
    for (const item of imageItems) {
      const file = item.getAsFile();
      if (file) addImageFile(file);
    }
  }

  function handleFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files) return;
    for (const file of input.files) {
      addImageFile(file);
    }
    input.value = '';
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    isDragging = true;
  }

  function handleDragLeave() {
    isDragging = false;
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
    if (!e.dataTransfer?.files) return;
    for (const file of e.dataTransfer.files) {
      if (ALLOWED_IMAGE_TYPES.includes(file.type)) {
        addImageFile(file);
      }
    }
  }

  function send() {
    const text = inputText.trim();
    if (!text && pendingImages.length === 0) return;
    const images =
      pendingImages.length > 0
        ? pendingImages.map(({ data, mediaType }) => ({ data, mediaType }))
        : undefined;
    for (const img of pendingImages) URL.revokeObjectURL(img.preview);
    pendingImages = [];
    inputText = '';
    onSend(text, images);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      send();
    } else if (e.key === 'Tab' && e.shiftKey && !isRunning) {
      e.preventDefault();
      onSetMode(agentMode === 'plan' ? 'act' : 'plan');
    } else if (e.key === 'Escape' && isRunning) {
      e.preventDefault();
      onStop();
    }
  }
</script>

<div class="p-3">
  {#if isRunning}
    <div class="flex items-center gap-2 px-1 pb-2 text-text-muted">
      <span class="agent-spinner"></span>
      <span class="text-xs">
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
  {#if quickReplies.length > 0 && !isRunning}
    <div class="flex flex-wrap gap-1.5 pb-2">
      {#each quickReplies as reply (reply)}
        <button
          type="button"
          onclick={() => onSend(reply)}
          class="cursor-pointer rounded border border-border bg-surface-alt px-3 py-1
            text-text-muted transition-colors hover:border-accent/40 hover:bg-accent/10 hover:text-accent"
        >
          {reply}
        </button>
      {/each}
    </div>
  {/if}
  <div
    class={[
      'relative flex flex-col rounded border bg-surface-alt',
      isDragging ? 'border-accent ring-2 ring-accent/40' : 'border-border'
    ]}
    role="region"
    aria-label="Message input with drag and drop for images"
    ondragover={handleDragOver}
    ondragleave={handleDragLeave}
    ondrop={handleDrop}
  >
    <textarea
      bind:this={textareaEl}
      bind:value={inputText}
      onkeydown={handleKeydown}
      onpaste={handlePaste}
      placeholder="Send a message..."
      rows={1}
      class="max-h-48 resize-none rounded bg-transparent px-3 py-2
				text-text [field-sizing:content] placeholder:text-text-faint
				focus:outline-none"
    ></textarea>
    <div class="flex items-center gap-1 px-2 pb-1.5">
      <input
        bind:this={fileInputEl}
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp"
        multiple
        class="hidden"
        onchange={handleFileInput}
      />
      <button
        type="button"
        onclick={() => fileInputEl?.click()}
        class="cursor-pointer rounded p-1 text-text-muted transition-colors hover:bg-surface hover:text-text"
        aria-label="Attach images"
      >
        <span class="icon-[uil--image-plus] size-4"></span>
      </button>
      <div class="flex-1"></div>
      {#if isRunning}
        <button
          onclick={onStop}
          class="cursor-pointer rounded p-1 text-danger transition-colors hover:bg-surface hover:text-danger"
          aria-label="Stop agent"
        >
          <span class="icon-[uil--stop-circle] size-4"></span>
        </button>
      {/if}
      <button
        onclick={send}
        disabled={!inputText.trim() && pendingImages.length === 0}
        class="cursor-pointer rounded p-1 text-text-muted transition-colors hover:bg-surface hover:text-text
					disabled:cursor-not-allowed disabled:opacity-30"
        aria-label="Send message"
      >
        <span class="icon-[uil--message] size-4"></span>
      </button>
    </div>
  </div>

  <!-- Pending image thumbnails -->
  {#if pendingImages.length > 0}
    <div class="flex flex-wrap gap-2 pt-2">
      {#each pendingImages as img, i (img.preview)}
        <div class="group relative">
          <img
            src={img.preview}
            alt="Pending attachment {i + 1}"
            class="max-h-16 rounded border border-border object-cover"
          />
          <button
            type="button"
            onclick={() => removeImage(i)}
            class="absolute -top-1.5 -right-1.5 flex size-5 cursor-pointer items-center justify-center
              rounded-full bg-danger text-xs text-white opacity-0 transition-opacity
              group-hover:opacity-100"
            aria-label="Remove image {i + 1}"
          >
            <span class="icon-[uil--times] size-3"></span>
          </button>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Effort level + Mode toggle + Clear -->
  <div class="flex items-center gap-3 pt-2">
    <div class={['flex', isRunning && 'pointer-events-none opacity-50']}>
      <button
        onclick={() => onSetEffort('low')}
        disabled={isRunning}
        class={[
          'btn-tab-sm cursor-pointer rounded-l',
          effort === 'low'
            ? 'border-accent/30 bg-accent/10 text-accent'
            : 'btn-tab-sm-inactive'
        ]}
      >
        Low
      </button>
      <button
        onclick={() => onSetEffort('medium')}
        disabled={isRunning}
        class={[
          'btn-tab-sm cursor-pointer border-l-0',
          effort === 'medium'
            ? 'border-accent/30 bg-accent/10 text-accent'
            : 'btn-tab-sm-inactive'
        ]}
      >
        Med
      </button>
      <button
        onclick={() => onSetEffort('high')}
        disabled={isRunning}
        class={[
          'btn-tab-sm cursor-pointer border-l-0',
          effort === 'high' || !effort
            ? 'border-accent/30 bg-accent/10 text-accent'
            : 'btn-tab-sm-inactive'
        ]}
      >
        High
      </button>
      <button
        onclick={() => onSetEffort('max')}
        disabled={isRunning}
        class={[
          'btn-tab-sm cursor-pointer rounded-r border-l-0',
          effort === 'max'
            ? 'border-accent/30 bg-accent/10 text-accent'
            : 'btn-tab-sm-inactive'
        ]}
      >
        Max
      </button>
    </div>

    <div class={['flex', isRunning && 'pointer-events-none opacity-50']}>
      <button
        onclick={() => onSetMode('plan')}
        disabled={isRunning}
        class={[
          'btn-tab-sm cursor-pointer rounded-l',
          agentMode === 'plan'
            ? 'border-warning/30 bg-warning/10 text-warning'
            : 'btn-tab-sm-inactive'
        ]}
      >
        <span class="icon-[uil--eye] size-3"></span>
        Plan
      </button>
      <button
        onclick={() => onSetMode('act')}
        disabled={isRunning}
        class={[
          'btn-tab-sm cursor-pointer rounded-r border-l-0',
          agentMode === 'act'
            ? 'border-success/30 bg-success/10 text-success'
            : 'btn-tab-sm-inactive'
        ]}
      >
        <span class="icon-[uil--play] size-3"></span>
        Act
      </button>
    </div>

    <div
      class={['flex', (isRunning || !hasMessages) && 'pointer-events-none opacity-30']}
    >
      <button
        onclick={onCompact}
        disabled={isRunning || !hasMessages}
        class="btn-tab-sm btn-tab-sm-inactive cursor-pointer rounded-l"
      >
        <span class="icon-[uil--compress-arrows] size-3"></span>
        Compact
      </button>
      <button
        onclick={onClear}
        disabled={isRunning || !hasMessages}
        class="btn-tab-sm btn-tab-sm-inactive cursor-pointer rounded-r border-l-0"
      >
        <span class="icon-[uil--trash-alt] size-3"></span>
        Clear
      </button>
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
</style>
