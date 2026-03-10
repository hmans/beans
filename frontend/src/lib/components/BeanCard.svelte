<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { worktreeStore } from '$lib/worktrees.svelte';
	import { agentStatusesStore } from '$lib/agentStatuses.svelte';
	import { statusColors, typeColors, typeBorders, priorityIndicators } from '$lib/styles';

	interface Props {
		bean: Bean;
		variant?: 'list' | 'board' | 'compact';
		selected?: boolean;
		onclick?: () => void;
	}

	let { bean, variant = 'list', selected = false, onclick }: Props = $props();

	const childCount = $derived(variant === 'list' ? beansStore.children(bean.id).length : 0);
	const hasWorktree = $derived(variant !== 'compact' && worktreeStore.hasWorktree(bean.id));
	const agentRunning = $derived(hasWorktree && agentStatusesStore.isRunning(bean.id));
</script>

<button
	{onclick}
	class={[
		'relative w-full cursor-pointer overflow-hidden text-left transition-all',
		variant === 'board'
			? 'p-3'
			: [
					'rounded-xs p-2',
					variant === 'compact' ? 'border-l-2' : 'border-l-3',
					hasWorktree
						? 'border-l-success'
						: (typeBorders[bean.type] ?? 'border-l-type-task-border'),
					selected ? 'bg-accent/10 ring-1 ring-accent' : 'bg-surface hover:bg-surface-alt'
				]
	]}
>
	{#if hasWorktree}
		<div
			class={[
				'absolute top-0 right-0 size-4 bg-success',
				agentRunning && 'bean-card-agent-pulse'
			]}
			style="clip-path: polygon(0 0, 100% 0, 100% 100%)"
		></div>
	{/if}

	{#if variant === 'board'}
		<!-- Board: two-row layout -->
		<div class="flex min-w-0 items-start gap-2">
			<span class="flex-1 text-sm leading-snug text-text">{bean.title}</span>
			{#if bean.priority && bean.priority !== 'normal' && priorityIndicators[bean.priority]}
				<span class={['shrink-0 text-xs', priorityIndicators[bean.priority]]}>
					{bean.priority}
				</span>
			{/if}
		</div>
		<div class="mt-1 flex items-center gap-2">
			<code class="text-[10px] text-text-faint">{bean.id.slice(-4)}</code>
			<span
				class={[
					'rounded-full px-1.5 py-0.5 text-[10px] font-medium',
					typeColors[bean.type] ?? 'bg-type-task-bg text-type-task-text'
				]}
			>
				{bean.type}
			</span>
		</div>
	{:else}
		<!-- List / Compact: single-row layout -->
		<div class="flex min-w-0 items-center gap-2">
			<code
				class={['shrink-0 text-text-faint', variant === 'compact' ? 'text-[9px]' : 'text-[10px]']}
				>{bean.id.slice(-4)}</code
			>
			<span class={['flex-1 truncate text-text', variant === 'compact' ? 'text-xs' : 'text-sm']}
				>{bean.title}</span
			>
			<span
				class={[
					'shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium',
					statusColors[bean.status] ?? 'bg-status-todo-bg text-status-todo-text'
				]}
			>
				{bean.status}
			</span>
			{#if variant === 'list' && childCount > 0}
				<span class="shrink-0 text-[10px] text-text-faint">+{childCount}</span>
			{/if}
		</div>
	{/if}
</button>

<style>
	.bean-card-agent-pulse {
		animation: agent-pulse 2s ease-in-out infinite;
	}

	@keyframes agent-pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.3;
		}
	}
</style>
