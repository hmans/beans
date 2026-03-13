<script lang="ts">
  import type { Bean } from '$lib/beans.svelte';
  import { beansStore, sortBeans } from '$lib/beans.svelte';
  import { applyDrop } from '$lib/dragOrder';
  import { matchesFilter } from '$lib/filter';
  import { client } from '$lib/graphqlClient';
  import { ui } from '$lib/uiState.svelte';
  import { typeBorders } from '$lib/styles';
  import { fade } from 'svelte/transition';
  import { gql } from 'urql';
  import BeanCard from './BeanCard.svelte';
  import ConfirmModal from './ConfirmModal.svelte';

  interface Props {
    onSelect?: (bean: Bean) => void;
    selectedId?: string | null;
  }

  let { onSelect, selectedId = null }: Props = $props();

  let confirmingArchiveAll = $state(false);
  let archivingAll = $state(false);

  const ARCHIVE_BEAN = gql`
    mutation ArchiveBean($id: ID!) {
      archiveBean(id: $id)
    }
  `;

  async function archiveAll() {
    archivingAll = true;
    const completedBeans = beansForStatus('completed');
    for (const bean of completedBeans) {
      await client.mutation(ARCHIVE_BEAN, { id: bean.id }).toPromise();
    }
    archivingAll = false;
    confirmingArchiveAll = false;
  }

  const columns = [
    { status: 'todo', label: 'Todo', color: 'bg-status-todo-bg text-status-todo-text' },
    {
      status: 'in-progress',
      label: 'In Progress',
      color: 'bg-status-in-progress-bg text-status-in-progress-text'
    },
    {
      status: 'completed',
      label: 'Completed',
      color: 'bg-status-completed-bg text-status-completed-text'
    }
  ];

  function beansForStatus(status: string): Bean[] {
    // sortBeans already handles order → priority → type → title sorting
    return sortBeans(
      beansStore.all.filter(
        (b) => b.status === status && b.status !== 'scrapped' && matchesFilter(b, ui.filterText)
      )
    );
  }

  // Scroll fade indicators per column
  let scrollState = $state<Record<string, { top: boolean; bottom: boolean }>>({});

  function trackScroll(el: HTMLElement, status: string) {
    function update() {
      scrollState[status] = {
        top: el.scrollTop > 0,
        bottom: el.scrollTop + el.clientHeight < el.scrollHeight - 1
      };
    }
    requestAnimationFrame(update);
    el.addEventListener('scroll', update, { passive: true });
    const resizeObs = new ResizeObserver(update);
    resizeObs.observe(el);
    const mutationObs = new MutationObserver(update);
    mutationObs.observe(el, { childList: true, subtree: true });
    return {
      destroy() {
        el.removeEventListener('scroll', update);
        resizeObs.disconnect();
        mutationObs.disconnect();
      }
    };
  }

  // Drag and drop
  let draggedBeanId = $state<string | null>(null);
  let dropTargetStatus = $state<string | null>(null);
  let dropIndex = $state<number | null>(null);

  function onDragStart(e: DragEvent, bean: Bean) {
    draggedBeanId = bean.id;
    e.dataTransfer!.effectAllowed = 'move';
    e.dataTransfer!.setData('text/plain', bean.id);
  }

  function onDragEnd() {
    draggedBeanId = null;
    dropTargetStatus = null;
    dropIndex = null;
  }

  function onCardDragOver(e: DragEvent, status: string, index: number) {
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer!.dropEffect = 'move';
    dropTargetStatus = status;

    // Determine if we're in the top or bottom half of the card
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const midY = rect.top + rect.height / 2;
    dropIndex = e.clientY < midY ? index : index + 1;
  }

  function onColumnDragOver(e: DragEvent, status: string, beanCount: number) {
    e.preventDefault();
    e.dataTransfer!.dropEffect = 'move';
    dropTargetStatus = status;
    // If dragging over empty space at the bottom, drop at end
    if (dropIndex === null || dropTargetStatus !== status) {
      dropIndex = beanCount;
    }
  }

  function onDragLeave(e: DragEvent, columnEl: HTMLElement) {
    if (!columnEl.contains(e.relatedTarget as Node)) {
      dropTargetStatus = null;
      dropIndex = null;
    }
  }

  function onDrop(e: DragEvent, targetStatus: string, beans: Bean[]) {
    e.preventDefault();
    const targetIdx = dropIndex;
    dropTargetStatus = null;
    dropIndex = null;

    const beanId = draggedBeanId;
    draggedBeanId = null;

    if (!beanId) return;

    applyDrop(beans, beanId, targetIdx ?? beans.length, { newStatus: targetStatus });
  }
</script>

<div class="flex min-h-0 flex-1 overflow-x-auto bg-surface-alt px-4 pt-4">
  {#each columns as col (col.status)}
    {@const beans = beansForStatus(col.status)}
    <div class="flex w-75 min-w-65 shrink-0 flex-col" data-status={col.status}>
      <!-- Column header -->
      <div class="mb-3 flex items-center gap-2 px-1">
        <span class={['badge', col.color]}
          >{col.label}</span
        >
        <span class="text-xs text-text-faint">{beans.length}</span>
        {#if col.status === 'completed' && beans.length > 0}
          <button
            class="cursor-pointer text-text-faint transition-colors hover:text-text-muted"
            title="Archive all completed beans"
            onclick={() => (confirmingArchiveAll = true)}
            disabled={archivingAll}
          >
            <span class="icon-[uil--archive] size-3.5"></span>
          </button>
        {/if}
      </div>

      <!-- Cards (drop zone) with scroll fade indicators -->
      <div class="relative min-h-0 flex-1">
        <div
          class={[
            'pointer-events-none absolute inset-x-0 top-0 z-10 h-6 rounded-t-xl bg-linear-to-b from-surface-alt to-transparent transition-opacity duration-150',
            scrollState[col.status]?.top ? 'opacity-100' : 'opacity-0'
          ]}
        ></div>

        <div
          class={[
            'h-full overflow-y-auto rounded-xl p-2 transition-colors',
            dropTargetStatus === col.status && draggedBeanId && 'bg-accent/10 ring-2 ring-accent/30'
          ]}
          role="list"
          use:trackScroll={col.status}
          ondragover={(e) => onColumnDragOver(e, col.status, beans.length)}
          ondragleave={(e) => onDragLeave(e, e.currentTarget)}
          ondrop={(e) => onDrop(e, col.status, beans)}
        >
          {#each beans as bean, index (bean.id)}
            <!-- Drop indicator (always present, transparent unless active) -->
            <div
              class={[
                'mx-1 my-1 h-0.5 rounded-full transition-colors',
                dropTargetStatus === col.status &&
                draggedBeanId &&
                draggedBeanId !== bean.id &&
                dropIndex === index
                  ? 'bg-accent'
                  : 'bg-transparent'
              ]}
            ></div>

            <div
              class={[
                'overflow-hidden rounded border border-l-5 border-border bg-surface shadow transition-all',
                typeBorders[bean.type] ?? 'border-l-type-task-border',
                draggedBeanId === bean.id ? 'opacity-40' : 'hover:shadow-md',
                selectedId === bean.id && 'bg-accent/5 ring-1 ring-accent'
              ]}
              draggable="true"
              ondragstart={(e) => onDragStart(e, bean)}
              ondragend={onDragEnd}
              ondragover={(e) => onCardDragOver(e, col.status, index)}
              role="listitem"
              transition:fade={{ duration: 150 }}
            >
              <BeanCard {bean} variant="board" onclick={() => onSelect?.(bean)} />
            </div>
          {:else}
            <div class="text-center text-text-faint text-sm py-8">No beans</div>
          {/each}

          <!-- Drop indicator at end (always present) -->
          <div
            class={[
              'mx-1 my-1 h-0.5 rounded-full transition-colors',
              dropTargetStatus === col.status && draggedBeanId && dropIndex === beans.length
                ? 'bg-accent'
                : 'bg-transparent'
            ]}
          ></div>
        </div>

        <div
          class={[
            'pointer-events-none absolute inset-x-0 bottom-0 z-10 h-6 rounded-b-xl bg-linear-to-t from-surface-alt to-transparent transition-opacity duration-150',
            scrollState[col.status]?.bottom ? 'opacity-100' : 'opacity-0'
          ]}
        ></div>
      </div>
    </div>
  {/each}
</div>

{#if confirmingArchiveAll}
  {@const completedCount = beansForStatus('completed').length}
  <ConfirmModal
    title="Archive All Completed"
    message="Are you sure you want to archive all {completedCount} completed beans? This will move them to the archive directory."
    confirmLabel="Archive All"
    danger={false}
    onConfirm={archiveAll}
    onCancel={() => (confirmingArchiveAll = false)}
  />
{/if}
