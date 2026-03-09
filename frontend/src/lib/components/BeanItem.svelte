<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { worktreeStore } from '$lib/worktrees.svelte';
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
	const hasWorktree = $derived(worktreeStore.hasWorktree(bean.id));

	const statusColors: Record<string, string> = {
		draft: 'bg-status-draft-bg text-status-draft-text',
		todo: 'bg-status-todo-bg text-status-todo-text',
		'in-progress': 'bg-status-in-progress-bg text-status-in-progress-text',
		completed: 'bg-status-completed-bg text-status-completed-text',
		scrapped: 'bg-status-scrapped-bg text-status-scrapped-text'
	};

	const typeBorders: Record<string, string> = {
		milestone: 'border-l-purple-400 dark:border-l-purple-500',
		epic: 'border-l-indigo-400 dark:border-l-indigo-500',
		feature: 'border-l-cyan-400 dark:border-l-cyan-500',
		bug: 'border-l-red-400 dark:border-l-red-500',
		task: 'border-l-surface-dim'
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
		class="w-full text-left rounded-lg p-2 border-l-3 transition-all
			{hasWorktree ? 'border-l-success' : typeBorders[bean.type] ?? 'border-l-surface-dim'}
			{isSelected ? 'bg-accent/10 ring-1 ring-accent' : 'bg-surface hover:bg-surface-alt'}"
	>
		<div class="flex items-center gap-2 min-w-0">
			<code class="text-[10px] text-text-faint shrink-0">{bean.id.slice(-4)}</code>
			<span class="text-sm text-text truncate flex-1">{bean.title}</span>
			<span class="text-[10px] px-1.5 py-0.5 rounded-full font-medium shrink-0
				{statusColors[bean.status] ?? 'bg-status-todo-bg text-status-todo-text'}">
				{bean.status}
			</span>
			{#if children.length > 0}
				<span class="text-[10px] text-text-faint shrink-0">+{children.length}</span>
			{/if}
		</div>
	</button>

	{#if children.length > 0}
		<div class="ml-4 mt-1 space-y-1 border-l border-border pl-2">
			{#each children as child (child.id)}
				<BeanItem bean={child} depth={depth + 1} {selectedId} {onSelect} />
			{/each}
		</div>
	{/if}
</div>
