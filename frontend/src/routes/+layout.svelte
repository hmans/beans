<script lang="ts">
  import './layout.css';
  import favicon from '$lib/assets/favicon.svg';
  import { preloadHighlighter } from '$lib/markdown';
  import { page } from '$app/state';
  import { onMount, onDestroy } from 'svelte';
  import { beansStore } from '$lib/beans.svelte';
  import { worktreeStore } from '$lib/worktrees.svelte';
  import { agentStatusesStore } from '$lib/agentStatuses.svelte';
  import { ui } from '$lib/uiState.svelte';
  import BeanForm from '$lib/components/BeanForm.svelte';
  import Sidebar from '$lib/components/Sidebar.svelte';

  preloadHighlighter();

  let { data, children } = $props();

  // Initialize UI state from load function data (runs before first render)
  $effect.pre(() => {
    ui.showPlanningChat = data.showPlanningChat;
    ui.showChanges = data.showChanges;
    ui.filterText = data.filterText;
    if (data.selectedBeanId) {
      ui.selectedBeanId = data.selectedBeanId;
    }
  });

  // Sync UIState from URL path on every navigation
  $effect(() => {
    ui.syncFromUrl(page.url.pathname);
  });

  // Fall back to planning view if the active workspace's worktree is removed
  $effect(() => {
    if (!ui.isPlanning && !worktreeStore.hasWorktree(ui.activeView)) {
      ui.navigateTo('planning');
    }
  });

  onMount(() => {
    beansStore.subscribe();
    worktreeStore.subscribe();
    agentStatusesStore.subscribe();
  });

  onDestroy(() => {
    beansStore.unsubscribe();
    worktreeStore.unsubscribe();
    agentStatusesStore.unsubscribe();
  });
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="flex h-screen flex-col bg-surface-alt">
  {#if beansStore.error}
    <div class="m-4">
      <div class="rounded-lg border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
        Error: {beansStore.error}
      </div>
    </div>
  {:else}
    <div class="flex min-h-0 flex-1">
      <Sidebar />
      <div class="flex min-h-0 min-w-0 flex-1 flex-col">
        {@render children()}
      </div>
    </div>
  {/if}
</div>

{#if ui.showForm}
  <BeanForm
    bean={ui.editingBean}
    onClose={() => ui.closeForm()}
    onSaved={(bean) => ui.selectBean(bean)}
  />
{/if}
