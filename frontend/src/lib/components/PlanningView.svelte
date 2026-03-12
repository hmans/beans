<script lang="ts">
  import { beansStore } from '$lib/beans.svelte';
  import { ui } from '$lib/uiState.svelte';

  let { planningView }: { planningView: 'backlog' | 'board' } = $props();
  import { backlogDrag } from '$lib/backlogDrag.svelte';
  import { matchesFilter } from '$lib/filter';
  import BeanItem from '$lib/components/BeanItem.svelte';
  import BoardView from '$lib/components/BoardView.svelte';
  import BeanPane from '$lib/components/BeanPane.svelte';
  import SplitPane from '$lib/components/SplitPane.svelte';
  import FilterInput from '$lib/components/FilterInput.svelte';
  import PaneHeader from '$lib/components/PaneHeader.svelte';

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
  <div class="toolbar bg-surface-alt">
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
    <div class="flex-1"></div>
  </div>

  <div class="flex min-h-0 flex-1 overflow-hidden">
    <SplitPane
      direction="horizontal"
      side="end"
      persistKey="detail-width"
      initialSize={480}
      collapsed={!ui.currentBean}
    >
      {#snippet children()}
        <div class="flex h-full flex-col bg-surface">
          <PaneHeader title={planningView === 'backlog' ? 'Backlog' : 'Board'} />
          {#if planningView === 'backlog'}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="min-h-0 flex-1 overflow-auto bg-surface" onclick={handlePlanningClick}>
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
                    'mx-1 h-0.5 rounded-full transition-colors',
                    backlogDrag.showEndIndicator(null, filteredTopLevelBeans.length)
                      ? 'bg-accent'
                      : 'bg-transparent'
                  ]}
                ></div>
              </div>
            </div>
          {:else}
            <div class="min-h-0 flex-1 bg-surface-alt">
              <BoardView onSelect={(b) => ui.selectBean(b)} selectedId={ui.currentBean?.id} />
            </div>
          {/if}
        </div>
      {/snippet}

      {#snippet aside()}
        {#if ui.currentBean}
          <BeanPane
            bean={ui.currentBean}
            onSelect={(b) => ui.selectBean(b)}
            onEdit={(b) => ui.openEditForm(b)}
            onClose={() => ui.clearSelection()}
          />
        {/if}
      {/snippet}
    </SplitPane>
  </div>
</div>
