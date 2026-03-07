<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { gql } from 'urql';
	import { client } from '$lib/graphqlClient';

	interface Props {
		onSelect?: (bean: Bean) => void;
		selectedId?: string | null;
	}

	let { onSelect, selectedId = null }: Props = $props();

	const columns = [
		{ status: 'draft', label: 'Draft', color: 'bg-yellow-400' },
		{ status: 'todo', label: 'Todo', color: 'bg-gray-400' },
		{ status: 'in-progress', label: 'In Progress', color: 'bg-blue-400' }
	];

	function beansForStatus(status: string): Bean[] {
		return beansStore.all.filter(
			(b) => b.status === status && b.status !== 'completed' && b.status !== 'scrapped'
		);
	}

	const typeBorders: Record<string, string> = {
		milestone: 'border-l-purple-400',
		epic: 'border-l-indigo-400',
		feature: 'border-l-cyan-400',
		bug: 'border-l-red-400',
		task: 'border-l-gray-300'
	};

	const statusColors: Record<string, string> = {
		todo: 'bg-gray-200 text-gray-700',
		'in-progress': 'bg-blue-200 text-blue-700',
		completed: 'bg-green-200 text-green-700',
		scrapped: 'bg-red-200 text-red-700',
		draft: 'bg-yellow-200 text-yellow-700'
	};

	const priorityIndicators: Record<string, string> = {
		critical: 'text-red-500',
		high: 'text-orange-500',
		low: 'text-gray-400',
		deferred: 'text-gray-300'
	};

	// Drag and drop
	let draggedBeanId = $state<string | null>(null);
	let dropTargetStatus = $state<string | null>(null);

	const UPDATE_BEAN_STATUS = gql`
		mutation UpdateBeanStatus($id: ID!, $status: String!) {
			updateBean(id: $id, input: { status: $status }) {
				id
				status
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
	}

	function onDragOver(e: DragEvent, status: string) {
		e.preventDefault();
		e.dataTransfer!.dropEffect = 'move';
		dropTargetStatus = status;
	}

	function onDragLeave(e: DragEvent, columnEl: HTMLElement) {
		// Only clear if we're actually leaving the column, not entering a child
		if (!columnEl.contains(e.relatedTarget as Node)) {
			dropTargetStatus = null;
		}
	}

	async function onDrop(e: DragEvent, targetStatus: string) {
		e.preventDefault();
		dropTargetStatus = null;

		const beanId = draggedBeanId;
		draggedBeanId = null;

		if (!beanId) return;

		const bean = beansStore.get(beanId);
		if (!bean || bean.status === targetStatus) return;

		const result = await client
			.mutation(UPDATE_BEAN_STATUS, { id: beanId, status: targetStatus })
			.toPromise();

		if (result.error) {
			console.error('Failed to update bean status:', result.error);
		}
	}
</script>

<div class="h-full flex gap-4 p-4 overflow-x-auto">
	{#each columns as col}
		{@const beans = beansForStatus(col.status)}
		<div class="flex flex-col min-w-[260px] w-[300px] shrink-0">
			<!-- Column header -->
			<div class="flex items-center gap-2 mb-3 px-1">
				<div class="w-2.5 h-2.5 rounded-full {col.color}"></div>
				<h2 class="text-sm font-semibold text-gray-700">{col.label}</h2>
				<span class="text-xs text-gray-400">{beans.length}</span>
			</div>

			<!-- Cards (drop zone) -->
			<div
				class="flex-1 overflow-y-auto space-y-2 rounded-lg p-2 transition-colors
					{dropTargetStatus === col.status && draggedBeanId ? 'bg-blue-50 ring-2 ring-blue-300 ring-dashed' : ''}"
				role="list"
				ondragover={(e) => onDragOver(e, col.status)}
				ondragleave={(e) => onDragLeave(e, e.currentTarget)}
				ondrop={(e) => onDrop(e, col.status)}
			>
				{#each beans as bean (bean.id)}
					<button
						onclick={() => onSelect?.(bean)}
						draggable="true"
						ondragstart={(e) => onDragStart(e, bean)}
						ondragend={onDragEnd}
						class="w-full text-left rounded-lg p-3 border-l-3 shadow-sm transition-all cursor-grab active:cursor-grabbing
							{typeBorders[bean.type] ?? 'border-l-gray-300'}
							{draggedBeanId === bean.id ? 'opacity-40' : ''}
							{selectedId === bean.id ? 'bg-blue-50 ring-1 ring-blue-300' : 'bg-white hover:shadow-md'}"
					>
						<div class="flex items-start gap-2 min-w-0">
							<span class="text-sm text-gray-900 flex-1 leading-snug">{bean.title}</span>
							{#if bean.priority && bean.priority !== 'normal' && priorityIndicators[bean.priority]}
								<span class="text-xs shrink-0 {priorityIndicators[bean.priority]}"
									>{bean.priority}</span
								>
							{/if}
						</div>
						<div class="flex items-center gap-2 mt-2">
							<code class="text-[10px] text-gray-400">{bean.id.slice(-4)}</code>
							<span
								class="text-[10px] px-1.5 py-0.5 rounded-full {statusColors[bean.type] ?? 'bg-gray-100 text-gray-600'}"
							>
								{bean.type}
							</span>
						</div>
					</button>
				{:else}
					<div class="text-center text-gray-400 text-sm py-8">No beans</div>
				{/each}
			</div>
		</div>
	{/each}
</div>
