<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import BeanItem from './BeanItem.svelte';

	interface Props {
		bean: Bean;
		depth?: number;
	}

	let { bean, depth = 0 }: Props = $props();

	let copied = $state(false);

	function copyId() {
		navigator.clipboard.writeText(bean.id);
		copied = true;
		setTimeout(() => (copied = false), 1500);
	}

	// Get children of this bean
	const children = $derived(beansStore.children(bean.id));

	// Status colors
	const statusColors: Record<string, string> = {
		todo: 'bg-gray-200 text-gray-800',
		'in-progress': 'bg-blue-200 text-blue-800',
		completed: 'bg-green-200 text-green-800',
		scrapped: 'bg-red-200 text-red-800',
		draft: 'bg-yellow-200 text-yellow-800'
	};

	// Type colors
	const typeColors: Record<string, string> = {
		milestone: 'bg-purple-100 text-purple-700',
		epic: 'bg-indigo-100 text-indigo-700',
		feature: 'bg-cyan-100 text-cyan-700',
		bug: 'bg-red-100 text-red-700',
		task: 'bg-gray-100 text-gray-700'
	};
</script>

<div class="bean-item" style="--depth: {depth}">
	<div
		class="rounded-lg bg-white p-3 shadow-sm hover:shadow-md transition-shadow border-l-4"
		class:border-purple-400={bean.type === 'milestone'}
		class:border-indigo-400={bean.type === 'epic'}
		class:border-cyan-400={bean.type === 'feature'}
		class:border-red-400={bean.type === 'bug'}
		class:border-gray-300={bean.type === 'task'}
	>
		<div class="flex items-start justify-between gap-4">
			<div class="flex-1 min-w-0">
				<div class="flex items-center gap-2 mb-1 flex-wrap">
					<button
						onclick={copyId}
						class="flex items-center gap-1 text-xs text-gray-400 hover:text-gray-600 transition-colors"
						title="Copy ID to clipboard"
					>
						<code>{bean.id}</code>
						{#if copied}
							<span class="text-green-500">✓</span>
						{:else}
							<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
							</svg>
						{/if}
					</button>
					<span
						class="text-xs px-2 py-0.5 rounded-full {typeColors[bean.type] ??
							'bg-gray-100 text-gray-700'}"
					>
						{bean.type}
					</span>
					<span
						class="text-xs px-2 py-0.5 rounded-full {statusColors[bean.status] ??
							'bg-gray-200 text-gray-800'}"
					>
						{bean.status}
					</span>
					{#if children.length > 0}
						<span class="text-xs text-gray-400">
							({children.length} child{children.length === 1 ? '' : 'ren'})
						</span>
					{/if}
				</div>
				<h2 class="text-base font-medium text-gray-900 truncate">{bean.title}</h2>
				{#if bean.tags.length > 0}
					<div class="flex gap-1 mt-1 flex-wrap">
						{#each bean.tags as tag}
							<span class="text-xs px-2 py-0.5 rounded bg-gray-100 text-gray-600">
								{tag}
							</span>
						{/each}
					</div>
				{/if}
			</div>
			<div class="text-right text-xs text-gray-400 shrink-0">
				{new Date(bean.updatedAt).toLocaleDateString()}
			</div>
		</div>
	</div>

	{#if children.length > 0}
		<div class="children ml-6 mt-2 space-y-2 border-l-2 border-gray-200 pl-4">
			{#each children as child (child.id)}
				<BeanItem bean={child} depth={depth + 1} />
			{/each}
		</div>
	{/if}
</div>
