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
		{ status: 'draft', label: 'Draft', color: 'badge-warning' },
		{ status: 'todo', label: 'Todo', color: 'badge-ghost' },
		{ status: 'in-progress', label: 'In Progress', color: 'badge-info' }
	];

	function beansForStatus(status: string): Bean[] {
		return beansStore.all
			.filter((b) => b.status === status && b.status !== 'completed' && b.status !== 'scrapped')
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
		task: 'border-l-base-300'
	};

	const typeBadge: Record<string, string> = {
		milestone: 'badge-secondary',
		epic: 'badge-primary',
		feature: 'badge-accent',
		bug: 'badge-error',
		task: 'badge-ghost'
	};

	const priorityIndicators: Record<string, string> = {
		critical: 'text-error',
		high: 'text-warning',
		low: 'text-base-content/40',
		deferred: 'text-base-content/30'
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
	 * Returns the list with orders filled in (mutates nothing, fires mutations for unordered beans).
	 */
	async function ensureOrdered(beans: Bean[]): Promise<Bean[]> {
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
				// Fire mutation (don't await, let them run in parallel)
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

	async function onDrop(e: DragEvent, targetStatus: string, beans: Bean[]) {
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
		const orderedBeans = await ensureOrdered(beans);

		const sameColumn = bean.status === targetStatus;
		const newOrder = computeOrder(orderedBeans, targetIdx ?? orderedBeans.length, beanId);

		// Skip if same column and order hasn't changed
		if (sameColumn && bean.order === newOrder) return;

		const input: Record<string, string> = { order: newOrder };
		if (!sameColumn) {
			input.status = targetStatus;
		}

		const result = await client.mutation(UPDATE_BEAN, { id: beanId, input }).toPromise();

		if (result.error) {
			console.error('Failed to update bean:', result.error);
		}
	}
</script>

<div class="h-full flex gap-4 p-4 overflow-x-auto">
	{#each columns as col}
		{@const beans = beansForStatus(col.status)}
		<div class="flex flex-col min-w-[260px] w-[300px] shrink-0">
			<!-- Column header -->
			<div class="flex items-center gap-2 mb-3 px-1">
				<span class="badge badge-sm {col.color}">{col.label}</span>
				<span class="text-xs text-base-content/40">{beans.length}</span>
			</div>

			<!-- Cards (drop zone) -->
			<div
				class="flex-1 overflow-y-auto space-y-2 rounded-xl p-2 transition-colors
					{dropTargetStatus === col.status && draggedBeanId
					? 'bg-primary/10 ring-2 ring-primary/30 ring-dashed'
					: ''}"
				role="list"
				ondragover={(e) => onColumnDragOver(e, col.status, beans.length)}
				ondragleave={(e) => onDragLeave(e, e.currentTarget)}
				ondrop={(e) => onDrop(e, col.status, beans)}
			>
				{#each beans as bean, index (bean.id)}
					<!-- Drop indicator line -->
					{#if dropTargetStatus === col.status && dropIndex === index && draggedBeanId && draggedBeanId !== bean.id}
						<div class="h-0.5 bg-primary rounded-full mx-1"></div>
					{/if}

					<div
						class="card card-border bg-base-100 shadow-sm border-l-3 transition-all cursor-pointer
							{typeBorders[bean.type] ?? 'border-l-base-300'}
							{draggedBeanId === bean.id ? 'opacity-40' : 'hover:shadow-md'}
							{selectedId === bean.id ? 'ring-1 ring-primary bg-primary/5' : ''}"
						draggable="true"
						ondragstart={(e) => onDragStart(e, bean)}
						ondragend={onDragEnd}
						ondragover={(e) => onCardDragOver(e, col.status, index)}
						role="listitem"
					>
						<button class="card-body p-3 text-left cursor-pointer" onclick={() => onSelect?.(bean)}>
							<div class="flex items-start gap-2 min-w-0">
								<span class="text-sm text-base-content flex-1 leading-snug">{bean.title}</span>
								{#if bean.priority && bean.priority !== 'normal' && priorityIndicators[bean.priority]}
									<span class="text-xs shrink-0 {priorityIndicators[bean.priority]}">
										{bean.priority}
									</span>
								{/if}
							</div>
							<div class="flex items-center gap-2 mt-1">
								<code class="text-[10px] text-base-content/40">{bean.id.slice(-4)}</code>
								<span class="badge badge-xs {typeBadge[bean.type] ?? 'badge-ghost'}">
									{bean.type}
								</span>
							</div>
						</button>
					</div>
				{:else}
					<div class="text-center text-base-content/30 text-sm py-8">No beans</div>
				{/each}

				<!-- Drop indicator at end -->
				{#if dropTargetStatus === col.status && dropIndex === beans.length && draggedBeanId}
					<div class="h-0.5 bg-primary rounded-full mx-1"></div>
				{/if}
			</div>
		</div>
	{/each}
</div>
