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
		{ status: 'draft', label: 'Draft', color: 'badge-warning' },
		{ status: 'todo', label: 'Todo', color: 'badge-ghost' },
		{ status: 'in-progress', label: 'In Progress', color: 'badge-info' }
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
				ondragover={(e) => onDragOver(e, col.status)}
				ondragleave={(e) => onDragLeave(e, e.currentTarget)}
				ondrop={(e) => onDrop(e, col.status)}
			>
				{#each beans as bean (bean.id)}
					<div
						class="card card-border bg-base-100 shadow-sm border-l-3 transition-all cursor-pointer
							{typeBorders[bean.type] ?? 'border-l-base-300'}
							{draggedBeanId === bean.id ? 'opacity-40' : 'hover:shadow-md'}
							{selectedId === bean.id ? 'ring-1 ring-primary bg-primary/5' : ''}"
						draggable="true"
						ondragstart={(e) => onDragStart(e, bean)}
						ondragend={onDragEnd}
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
			</div>
		</div>
	{/each}
</div>
