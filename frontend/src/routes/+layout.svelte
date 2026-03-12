<script lang="ts">
  import './layout.css';
  import favicon from '$lib/assets/favicon.svg';
  import { preloadHighlighter } from '$lib/markdown';
  import { page } from '$app/state';
  import { onMount, onDestroy } from 'svelte';
  import { beansStore } from '$lib/beans.svelte';
  import { worktreeStore } from '$lib/worktrees.svelte';
  import { agentStatusesStore } from '$lib/agentStatuses.svelte';
  import { configStore } from '$lib/config.svelte';
  import { ui } from '$lib/uiState.svelte';
  import BeanForm from '$lib/components/BeanForm.svelte';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import SplitPane from '$lib/components/SplitPane.svelte';

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

  // Fall back to planning view if agents are disabled or the workspace's worktree is removed
  $effect(() => {
    if (!ui.isPlanning && (!configStore.agentEnabled || !worktreeStore.hasWorktree(ui.activeView))) {
      ui.navigateTo('planning');
    }
  });

  onMount(() => {
    configStore.load();
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
    <SplitPane direction="horizontal" side="start" initialSize={224} minSize={150} maxSize={400} persistKey="sidebar">
      {#snippet aside()}
        <Sidebar />
      {/snippet}
      {@render children()}
    </SplitPane>
  {/if}
</div>

{#if ui.showForm}
  <BeanForm
    bean={ui.editingBean}
    onClose={() => ui.closeForm()}
    onSaved={(bean) => ui.selectBean(bean)}
  />
{/if}
