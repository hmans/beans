<script lang="ts">
  import type { Snippet } from 'svelte';
  import { ui } from '$lib/uiState.svelte';
  import { configStore } from '$lib/config.svelte';

  interface Props {
    children?: Snippet;
    right?: Snippet;
    showAgentToggle?: boolean;
    agentActive?: boolean;
    onToggleAgent?: () => void;
  }

  let { children, right, showAgentToggle = false, agentActive = false, onToggleAgent }: Props = $props();
</script>

<div class="toolbar bg-surface-alt">
  {#if configStore.agentEnabled}
    {#if showAgentToggle && onToggleAgent}
      <button
        onclick={onToggleAgent}
        class={['btn-toggle', agentActive ? 'btn-toggle-active' : 'btn-toggle-inactive']}
        title={agentActive ? 'Hide agent' : 'Show agent'}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          class="h-4 w-4"
        >
          <path
            fill-rule="evenodd"
            d="M10 3c-4.31 0-8 3.033-8 7 0 2.024.978 3.825 2.499 5.085a3.478 3.478 0 01-.522 1.756.75.75 0 00.584 1.143 5.976 5.976 0 003.936-1.108c.487.082.99.124 1.503.124 4.31 0 8-3.033 8-7s-3.69-7-8-7z"
            clip-rule="evenodd"
          />
        </svg>
        Agent
      </button>
    {/if}
    <button
      onclick={() => ui.toggleChanges()}
      class={['btn-toggle ml-1', ui.showChanges ? 'btn-toggle-active' : 'btn-toggle-inactive']}
      title={ui.showChanges ? 'Hide changes' : 'Show changes'}
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
      Changes
    </button>
    <button
      onclick={() => ui.toggleTerminal()}
      class={['btn-toggle ml-1', ui.showTerminal ? 'btn-toggle-active' : 'btn-toggle-inactive']}
      title={ui.showTerminal ? 'Hide terminal' : 'Show terminal'}
    >
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
        <path fill-rule="evenodd" d="M3.25 3A2.25 2.25 0 001 5.25v9.5A2.25 2.25 0 003.25 17h13.5A2.25 2.25 0 0019 14.75v-9.5A2.25 2.25 0 0016.75 3H3.25zm.943 8.752a.75.75 0 01.055-1.06L6.128 9l-1.88-1.693a.75.75 0 111.004-1.114l2.5 2.25a.75.75 0 010 1.114l-2.5 2.25a.75.75 0 01-1.06-.055zM9.75 10.25a.75.75 0 000 1.5h2.5a.75.75 0 000-1.5h-2.5z" clip-rule="evenodd" />
      </svg>
      Terminal
    </button>
    <div class="mr-3"></div>
  {/if}
  {#if children}
    {@render children()}
  {/if}
  <div class="flex-1"></div>
  {#if right}
    {@render right()}
  {/if}
</div>
