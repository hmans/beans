<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import BeanItem from './BeanItem.svelte';

	interface Props {
		bean: Bean;
		depth?: number;
		selectedId?: string | null;
		onSelect?: (bean: Bean) => void;
	}

	let { bean, depth = 0, selectedId = null, onSelect }: Props = $props();

	const children = $derived(beansStore.children(bean.id));
	const isSelected = $derived(selectedId === bean.id);

	const statusBadge: Record<string, string> = {
		todo: 'badge-ghost',
		'in-progress': 'badge-info',
		completed: 'badge-success',
		scrapped: 'badge-error',
		draft: 'badge-warning'
	};

	const typeBorders: Record<string, string> = {
		milestone: 'border-l-purple-400',
		epic: 'border-l-indigo-400',
		feature: 'border-l-cyan-400',
		bug: 'border-l-red-400',
		task: 'border-l-base-300'
	};

	function handleClick(e: MouseEvent) {
		e.stopPropagation();
		onSelect?.(bean);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onSelect?.(bean);
		}
	}
</script>

<div class="bean-item">
	<button
		onclick={handleClick}
		onkeydown={handleKeydown}
		class="w-full text-left rounded-lg p-2 border-l-3 transition-all cursor-pointer
			{typeBorders[bean.type] ?? 'border-l-base-300'}
			{isSelected ? 'bg-primary/10 ring-1 ring-primary' : 'bg-base-100 hover:bg-base-200'}"
	>
		<div class="flex items-center gap-2 min-w-0">
			<code class="text-[10px] text-base-content/40 shrink-0">{bean.id.slice(-4)}</code>
			<span class="text-sm text-base-content truncate flex-1">{bean.title}</span>
			<span class="badge badge-xs {statusBadge[bean.status] ?? 'badge-ghost'} shrink-0">
				{bean.status}
			</span>
			{#if children.length > 0}
				<span class="text-[10px] text-base-content/40 shrink-0">+{children.length}</span>
			{/if}
		</div>
	</button>

	{#if children.length > 0}
		<div class="ml-4 mt-1 space-y-1 border-l border-base-200 pl-2">
			{#each children as child (child.id)}
				<BeanItem bean={child} depth={depth + 1} {selectedId} {onSelect} />
			{/each}
		</div>
	{/if}
</div>
