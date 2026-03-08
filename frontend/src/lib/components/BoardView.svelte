<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { orderBetween } from '$lib/fractional';
	import { gql } from 'urql';
	import { client } from '$lib/graphqlClient';

	interface Props {
		onSelect?: (bean: Bean) => void;
		selectedId?: string | null;
	}

	let { onSelect, selectedId = null }: Props = $props();

	const columns = [
		{ status: 'draft', label: 'Draft', color: 'bg-warning/15 text-warning' },
		{ status: 'todo', label: 'Todo', color: 'bg-surface-dim text-text-muted' },
		{ status: 'in-progress', label: 'In Progress', color: 'bg-info/15 text-info' },
		{ status: 'completed', label: 'Completed', color: 'bg-success/15 text-success' }
	];

	function beansForStatus(status: string): Bean[] {
		return beansStore.all
			.filter((b) => b.status === status && b.status !== 'scrapped')
			.sort((a, b) => {
				// Beans with order come first, sorted lexicographically
				if (a.order && b.order) return a.order < b.order ? -1 : a.order > b.order ? 1 : 0;
				if (a.order && !b.order) return -1;
				if (!a.order && b.order) return 1;
				// Fallback: title
				return a.title.localeCompare(b.title, undefined, { sensitivity: 'base' });
			});
	}

	const typeBorders: Record<string, string> = {
		milestone: 'border-l-purple-400',
		epic: 'border-l-indigo-400',
		feature: 'border-l-cyan-400',
		bug: 'border-l-red-400',
		task: 'border-l-surface-dim'
	};

	const typeColors: Record<string, string> = {
		milestone: 'bg-purple-100 text-purple-700',
		epic: 'bg-indigo-100 text-indigo-700',
		feature: 'bg-cyan-100 text-cyan-700',
		bug: 'bg-red-100 text-red-700',
		task: 'bg-surface-dim text-text-muted'
	};

	const priorityIndicators: Record<string, string> = {
		critical: 'text-danger',
		high: 'text-warning',
		low: 'text-text-faint',
		deferred: 'text-text-faint opacity-60'
	};

	// Drag and drop
	let draggedBeanId = $state<string | null>(null);
	let dropTargetStatus = $state<string | null>(null);
	let dropIndex = $state<number | null>(null);

	const UPDATE_BEAN = gql`
		mutation UpdateBean($id: ID!, $input: UpdateBeanInput!) {
			updateBean(id: $id, input: $input) {
				id
				status
				order
			}
		}
	`;

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

	/**
	 * Ensure all beans in the list have order keys.
	 * Assigns evenly-spaced keys to any beans missing them,
	 * preserving the relative positions of beans that already have keys.
	 * Returns the list with orders filled in. Updates the store optimistically.
	 */
	function ensureOrdered(beans: Bean[]): Bean[] {
		const needsOrder = beans.filter((b) => !b.order);
		if (needsOrder.length === 0) return beans;

		// Assign orders to all beans based on their current visual position
		const result = [...beans];
		let key = '';
		for (let i = 0; i < result.length; i++) {
			const nextKey = i < result.length - 1 && result[i + 1].order ? result[i + 1].order : '';
			if (!result[i].order) {
				const newOrder = orderBetween(key, nextKey);
				result[i] = { ...result[i], order: newOrder };
				beansStore.optimisticUpdate(result[i].id, { order: newOrder });
				client.mutation(UPDATE_BEAN, { id: result[i].id, input: { order: newOrder } }).toPromise();
			}
			key = result[i].order;
		}
		return result;
	}

	function computeOrder(beans: Bean[], targetIndex: number, draggedId: string): string {
		// Find where the dragged bean is in the original list
		const draggedIndex = beans.findIndex((b) => b.id === draggedId);

		// Filter out the dragged bean from the list
		const filtered = beans.filter((b) => b.id !== draggedId);

		if (filtered.length === 0) {
			return orderBetween('', '');
		}

		// Adjust target index: if dragging downward in the same column,
		// the visual index is 1 too high because the dragged bean is still in the list
		let idx = targetIndex;
		if (draggedIndex >= 0 && targetIndex > draggedIndex) {
			idx--;
		}
		idx = Math.min(idx, filtered.length);

		if (idx === 0) {
			return orderBetween('', filtered[0].order);
		}
		if (idx >= filtered.length) {
			return orderBetween(filtered[filtered.length - 1].order, '');
		}

		return orderBetween(filtered[idx - 1].order, filtered[idx].order);
	}

	function onDrop(e: DragEvent, targetStatus: string, beans: Bean[]) {
		e.preventDefault();
		const targetIdx = dropIndex;
		dropTargetStatus = null;
		dropIndex = null;

		const beanId = draggedBeanId;
		draggedBeanId = null;

		if (!beanId) return;

		const bean = beansStore.get(beanId);
		if (!bean) return;

		// Ensure all beans in the target column have order keys first
		const orderedBeans = ensureOrdered(beans);

		const sameColumn = bean.status === targetStatus;
		const newOrder = computeOrder(orderedBeans, targetIdx ?? orderedBeans.length, beanId);

		// Skip if same column and order hasn't changed
		if (sameColumn && bean.order === newOrder) return;

		// Optimistic update — move the bean immediately in the local store
		const optimistic: Partial<Bean> = { order: newOrder };
		if (!sameColumn) {
			optimistic.status = targetStatus;
		}
		beansStore.optimisticUpdate(beanId, optimistic);

		// Fire mutation in background
		const input: Record<string, string> = { order: newOrder };
		if (!sameColumn) {
			input.status = targetStatus;
		}
		client.mutation(UPDATE_BEAN, { id: beanId, input }).toPromise().then((result) => {
			if (result.error) {
				console.error('Failed to update bean:', result.error);
			}
		});
	}
</script>

<div class="h-full flex gap-4 p-4 overflow-x-auto">
	{#each columns as col}
		{@const beans = beansForStatus(col.status)}
		<div class="flex flex-col min-w-[260px] w-[300px] shrink-0">
			<!-- Column header -->
			<div class="flex items-center gap-2 mb-3 px-1">
				<span class="text-[11px] px-2 py-0.5 rounded-full font-medium {col.color}">{col.label}</span>
				<span class="text-xs text-text-faint">{beans.length}</span>
			</div>

			<!-- Cards (drop zone) -->
			<div
				class="flex-1 overflow-y-auto rounded-xl p-2 transition-colors
					{dropTargetStatus === col.status && draggedBeanId
					? 'bg-accent/10 ring-2 ring-accent/30'
					: ''}"
				role="list"
				ondragover={(e) => onColumnDragOver(e, col.status, beans.length)}
				ondragleave={(e) => onDragLeave(e, e.currentTarget)}
				ondrop={(e) => onDrop(e, col.status, beans)}
			>
				{#each beans as bean, index (bean.id)}
					<!-- Drop indicator (always present, transparent unless active) -->
					<div
						class="h-0.5 rounded-full mx-1 my-1 transition-colors
							{dropTargetStatus === col.status && draggedBeanId && draggedBeanId !== bean.id && dropIndex === index
							? 'bg-accent' : 'bg-transparent'}"
					></div>

					<div
						class="rounded-lg border border-border bg-surface shadow-sm border-l-3 transition-all cursor-pointer
							{typeBorders[bean.type] ?? 'border-l-surface-dim'}
							{draggedBeanId === bean.id ? 'opacity-40' : 'hover:shadow-md'}
							{selectedId === bean.id ? 'ring-1 ring-accent bg-accent/5' : ''}"
						draggable="true"
						ondragstart={(e) => onDragStart(e, bean)}
						ondragend={onDragEnd}
						ondragover={(e) => onCardDragOver(e, col.status, index)}
						role="listitem"
					>
						<button class="p-3 text-left cursor-pointer w-full" onclick={() => onSelect?.(bean)}>
							<div class="flex items-start gap-2 min-w-0">
								<span class="text-sm text-text flex-1 leading-snug">{bean.title}</span>
								{#if bean.priority && bean.priority !== 'normal' && priorityIndicators[bean.priority]}
									<span class="text-xs shrink-0 {priorityIndicators[bean.priority]}">
										{bean.priority}
									</span>
								{/if}
							</div>
							<div class="flex items-center gap-2 mt-1">
								<code class="text-[10px] text-text-faint">{bean.id.slice(-4)}</code>
								<span class="text-[10px] px-1.5 py-0.5 rounded-full font-medium {typeColors[bean.type] ?? 'bg-surface-dim text-text-muted'}">
									{bean.type}
								</span>
							</div>
						</button>
					</div>
				{:else}
					<div class="text-center text-text-faint text-sm py-8">No beans</div>
				{/each}

				<!-- Drop indicator at end (always present) -->
				<div
					class="h-0.5 rounded-full mx-1 my-1 transition-colors
						{dropTargetStatus === col.status && draggedBeanId && dropIndex === beans.length
						? 'bg-accent' : 'bg-transparent'}"
				></div>
			</div>
		</div>
	{/each}
</div>
