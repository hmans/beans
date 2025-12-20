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

	// Get children of this bean
	const children = $derived(beansStore.children(bean.id));

	const isSelected = $derived(selectedId === bean.id);

	// Status colors - more compact
	const statusColors: Record<string, string> = {
		todo: 'bg-gray-200 text-gray-700',
		'in-progress': 'bg-blue-200 text-blue-700',
		completed: 'bg-green-200 text-green-700',
		scrapped: 'bg-red-200 text-red-700',
		draft: 'bg-yellow-200 text-yellow-700'
	};

	// Type border colors
	const typeBorders: Record<string, string> = {
		milestone: 'border-l-purple-400',
		epic: 'border-l-indigo-400',
		feature: 'border-l-cyan-400',
		bug: 'border-l-red-400',
		task: 'border-l-gray-300'
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
		class="w-full text-left rounded-md p-2 border-l-3 transition-all cursor-pointer
			{typeBorders[bean.type] ?? 'border-l-gray-300'}
			{isSelected ? 'bg-blue-50 ring-1 ring-blue-300' : 'bg-white hover:bg-gray-50'}"
	>
		<div class="flex items-center gap-2 min-w-0">
			<code class="text-[10px] text-gray-400 shrink-0">{bean.id.slice(-4)}</code>
			<span class="text-sm text-gray-900 truncate flex-1">{bean.title}</span>
			<span
				class="text-[10px] px-1.5 py-0.5 rounded-full shrink-0 {statusColors[bean.status] ?? 'bg-gray-200 text-gray-700'}"
			>
				{bean.status}
			</span>
			{#if children.length > 0}
				<span class="text-[10px] text-gray-400 shrink-0">+{children.length}</span>
			{/if}
		</div>
	</button>

	{#if children.length > 0}
		<div class="ml-4 mt-1 space-y-1 border-l border-gray-200 pl-2">
			{#each children as child (child.id)}
				<BeanItem bean={child} depth={depth + 1} {selectedId} {onSelect} />
			{/each}
		</div>
	{/if}
</div>
