<script lang="ts">
  import { beansStore } from '$lib/beans.svelte';
  import { AgentChatStore } from '$lib/agentChat.svelte';
  import { ui } from '$lib/uiState.svelte';

  let { planningView }: { planningView: 'backlog' | 'board' } = $props();
  import { backlogDrag } from '$lib/backlogDrag.svelte';
  import { matchesFilter } from '$lib/filter';
  import { onDestroy } from 'svelte';
  import BeanItem from '$lib/components/BeanItem.svelte';
  import BoardView from '$lib/components/BoardView.svelte';
  import BeanPane from '$lib/components/BeanPane.svelte';
  import SplitPane from '$lib/components/SplitPane.svelte';
  import { configStore } from '$lib/config.svelte';
  import AgentChat from '$lib/components/AgentChat.svelte';
  import ChangesPane from '$lib/components/ChangesPane.svelte';
  import FilterInput from '$lib/components/FilterInput.svelte';

  import TerminalPane from '$lib/components/TerminalPane.svelte';
  import ViewToolbar from '$lib/components/ViewToolbar.svelte';
  import { AgentActionsStore } from '$lib/agentActions.svelte';

  const CENTRAL_SESSION_ID = '__central__';

  const agentStore = new AgentChatStore();
  const actionsStore = new AgentActionsStore();

  $effect(() => {
    agentStore.subscribe(CENTRAL_SESSION_ID);
  });

  onDestroy(() => {
    agentStore.unsubscribe();
  });

  const agentBusy = $derived(agentStore.session?.status === 'RUNNING');

  // Fetch agent actions on mount and when agent finishes
  $effect(() => {
    actionsStore.fetch(CENTRAL_SESSION_ID);
  });

  $effect(() => {
    actionsStore.notifyAgentStatus(CENTRAL_SESSION_ID, agentBusy);
  });

  let filterInput = $state<FilterInput | null>(null);

  const topLevelBeans = $derived(beansStore.all.filter((b) => !b.parentId));

  const filteredTopLevelBeans = $derived.by(() => {
    const text = ui.filterText;
    if (!text) return topLevelBeans;
    return topLevelBeans.filter((bean) => {
      if (matchesFilter(bean, text)) return true;
      return beansStore.children(bean.id).some((child) => matchesFilter(child, text));
    });
  });

  function handleKeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && (e.key === 'f' || e.key === '/')) {
      e.preventDefault();
      filterInput?.focus();
      return;
    }
    if (e.key === 'Escape' && ui.currentBean && !ui.showForm) {
      ui.clearSelection();
    }
  }

  function handlePlanningClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      ui.clearSelection();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-full flex-col">
  <ViewToolbar showAgentToggle agentActive={ui.showPlanningChat} onToggleAgent={() => ui.togglePlanningChat()}>
    <button class="btn-primary" onclick={() => ui.openCreateForm()}>+ New Bean</button>

    <div class="ml-3 flex">
      <button
        onclick={() => ui.navigateToPlanningView('backlog')}
        class={[
          'btn-tab rounded-l-md',
          planningView === 'backlog' ? 'btn-tab-active' : 'btn-tab-inactive'
        ]}
      >
        Backlog
      </button>
      <button
        onclick={() => ui.navigateToPlanningView('board')}
        class={[
          'btn-tab rounded-r-md border-l-0',
          planningView === 'board' ? 'btn-tab-active' : 'btn-tab-inactive'
        ]}
      >
        Board
      </button>
    </div>
    <div class="mx-3 w-60">
      <FilterInput bind:this={filterInput} />
    </div>
    {#snippet right()}
      {#each actionsStore.actions as action (action.id)}
        <button
          class={[
            'btn-toggle btn-toggle-inactive ml-1'
          ]}
          disabled={agentBusy || !!actionsStore.executingAction}
          title={action.description ?? undefined}
          onclick={() => actionsStore.execute(CENTRAL_SESSION_ID, action.id, agentBusy)}
        >
          {action.label}
        </button>
      {/each}
    {/snippet}
  </ViewToolbar>

  <!-- Layout: [ Backlog/Board | Detail? | Changes? | Agent? | Terminal? ] -->
  <div class="flex min-h-0 flex-1 overflow-hidden">
    {#snippet mainPanel()}
      {#snippet backlogBoard()}
        <div class="flex h-full flex-col bg-surface">
          {#if planningView === 'backlog'}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="min-h-0 flex-1 overflow-auto bg-surface-alt" onclick={handlePlanningClick}>
              <div
                class="p-3"
                ondragover={(e) => backlogDrag.hoverList(e, null, filteredTopLevelBeans.length)}
                ondragleave={(e) => backlogDrag.leaveList(e, e.currentTarget, null)}
                ondrop={(e) => backlogDrag.drop(e, null, filteredTopLevelBeans)}
                role="list"
              >
                {#each filteredTopLevelBeans as bean, i (bean.id)}
                  <BeanItem
                    {bean}
                    parentId={null}
                    index={i}
                    selectedId={ui.currentBean?.id}
                    onSelect={(b) => ui.selectBean(b)}
                    filterText={ui.filterText}
                  />
                {:else}
                  {#if !beansStore.loading}
                    <p class="text-text-muted text-center py-8 text-sm">
                      {ui.filterText ? 'No matching beans' : 'No beans yet'}
                    </p>
                  {/if}
                {/each}

                <div
                  class={[
                    'mx-1 rounded-full transition-colors',
                    backlogDrag.showEndIndicator(null, filteredTopLevelBeans.length)
                      ? 'h-0.5 bg-accent'
                      : 'h-0'
                  ]}
                ></div>
              </div>
            </div>
          {:else}
            <BoardView onSelect={(b) => ui.selectBean(b)} selectedId={ui.currentBean?.id} />
          {/if}
        </div>
      {/snippet}

      {#snippet detailPanel()}
        {#if ui.currentBean}
          <BeanPane
            bean={ui.currentBean}
            onSelect={(b) => ui.selectBean(b)}
            onEdit={(b) => ui.openEditForm(b)}
            onClose={() => ui.clearSelection()}
          />
        {/if}
      {/snippet}

      <SplitPane
        direction="horizontal"
        panels={[
          { content: backlogBoard },
          {
            content: detailPanel,
            size: 480,
            collapsed: !ui.currentBean,
            persistKey: 'detail-width'
          }
        ]}
      />
    {/snippet}

    {#snippet changesPanel()}
      <ChangesPane />
    {/snippet}

    {#snippet agentPanel()}
      <AgentChat beanId={CENTRAL_SESSION_ID} store={agentStore} />
    {/snippet}

    {#snippet terminalPanel()}
      {#if ui.terminalInitialized}
        <TerminalPane sessionId={CENTRAL_SESSION_ID} />
      {/if}
    {/snippet}

    <SplitPane
      direction="horizontal"
      panels={[
        {
          content: agentPanel,
          size: 420,
          collapsed: !configStore.agentEnabled || !ui.showPlanningChat,
          persistKey: 'planning-agent'
        },
        { content: mainPanel },
        {
          content: changesPanel,
          size: 420,
          collapsed: !configStore.agentEnabled || !ui.showChanges,
          persistKey: 'planning-changes'
        },
        {
          content: terminalPanel,
          size: 480,
          collapsed: !configStore.agentEnabled || !ui.showTerminal,
          persistKey: 'planning-terminal'
        }
      ]}
    />
  </div>
</div>
