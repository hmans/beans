<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { renderMarkdown } from '$lib/markdown';

	interface Props {
		bean: Bean;
	}

	let { bean }: Props = $props();

	// Get parent and children
	const parent = $derived(bean.parentId ? beansStore.get(bean.parentId) : null);
	const children = $derived(beansStore.children(bean.id));
	const blocking = $derived(bean.blockingIds.map((id) => beansStore.get(id)).filter(Boolean));
	const blockedBy = $derived(beansStore.blockedBy(bean.id));

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

	// Priority colors
	const priorityColors: Record<string, string> = {
		critical: 'text-red-600',
		high: 'text-orange-600',
		normal: 'text-gray-600',
		low: 'text-gray-400',
		deferred: 'text-gray-300'
	};

	// Render markdown body (async with shiki)
	let renderedBody = $state('');

	$effect(() => {
		const body = bean.body;
		if (body) {
			renderMarkdown(body).then((html) => {
				renderedBody = html;
			});
		} else {
			renderedBody = '';
		}
	});

	let copied = $state(false);

	function copyId() {
		navigator.clipboard.writeText(bean.id);
		copied = true;
		setTimeout(() => (copied = false), 1500);
	}
</script>

<div class="h-full overflow-auto p-6">
	<!-- Header -->
	<div class="mb-6">
		<div class="flex items-center gap-2 mb-2 flex-wrap">
			<button
				onclick={copyId}
				class="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 transition-colors font-mono"
				title="Copy ID to clipboard"
			>
				{bean.id}
				{#if copied}
					<span class="text-green-500">✓</span>
				{:else}
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
					</svg>
				{/if}
			</button>
			<span class="text-xs px-2 py-0.5 rounded-full {typeColors[bean.type] ?? 'bg-gray-100 text-gray-700'}">
				{bean.type}
			</span>
			<span class="text-xs px-2 py-0.5 rounded-full {statusColors[bean.status] ?? 'bg-gray-200 text-gray-800'}">
				{bean.status}
			</span>
			{#if bean.priority && bean.priority !== 'normal'}
				<span class="text-xs {priorityColors[bean.priority] ?? 'text-gray-600'}">
					{bean.priority}
				</span>
			{/if}
		</div>
		<h1 class="text-2xl font-bold text-gray-900">{bean.title}</h1>
	</div>

	<!-- Tags -->
	{#if bean.tags.length > 0}
		<div class="mb-6">
			<h2 class="text-xs font-semibold text-gray-500 uppercase mb-2">Tags</h2>
			<div class="flex gap-1 flex-wrap">
				{#each bean.tags as tag}
					<span class="text-sm px-2 py-0.5 rounded bg-gray-100 text-gray-600">{tag}</span>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Relationships -->
	{#if parent || children.length > 0 || blocking.length > 0 || blockedBy.length > 0}
		<div class="mb-6 space-y-3">
			{#if parent}
				<div>
					<h2 class="text-xs font-semibold text-gray-500 uppercase mb-1">Parent</h2>
					<div class="text-sm text-gray-700">
						<span class="font-mono text-gray-400">{parent.id}</span> {parent.title}
					</div>
				</div>
			{/if}

			{#if children.length > 0}
				<div>
					<h2 class="text-xs font-semibold text-gray-500 uppercase mb-1">Children ({children.length})</h2>
					<ul class="text-sm text-gray-700 space-y-0.5">
						{#each children as child}
							<li>
								<span class="font-mono text-gray-400">{child.id}</span> {child.title}
								<span class="text-xs px-1.5 py-0.5 rounded-full {statusColors[child.status]}">{child.status}</span>
							</li>
						{/each}
					</ul>
				</div>
			{/if}

			{#if blocking.length > 0}
				<div>
					<h2 class="text-xs font-semibold text-gray-500 uppercase mb-1">Blocking ({blocking.length})</h2>
					<ul class="text-sm text-gray-700 space-y-0.5">
						{#each blocking as b}
							{#if b}
								<li>
									<span class="font-mono text-gray-400">{b.id}</span> {b.title}
								</li>
							{/if}
						{/each}
					</ul>
				</div>
			{/if}

			{#if blockedBy.length > 0}
				<div>
					<h2 class="text-xs font-semibold text-gray-500 uppercase mb-1">Blocked By ({blockedBy.length})</h2>
					<ul class="text-sm text-gray-700 space-y-0.5">
						{#each blockedBy as b}
							<li>
								<span class="font-mono text-gray-400">{b.id}</span> {b.title}
							</li>
						{/each}
					</ul>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Body -->
	{#if bean.body}
		<div class="mb-6">
			<h2 class="text-xs font-semibold text-gray-500 uppercase mb-2">Description</h2>
			<div class="bean-body prose prose-sm max-w-none text-gray-700">
				{@html renderedBody}
			</div>
		</div>
	{/if}

	<!-- Metadata -->
	<div class="text-xs text-gray-400 space-y-1 border-t pt-4">
		<div>Created: {new Date(bean.createdAt).toLocaleString()}</div>
		<div>Updated: {new Date(bean.updatedAt).toLocaleString()}</div>
		<div>Path: {bean.path}</div>
	</div>
</div>

<style>
	/* Subtle, colorful headings */
	.bean-body :global(h1) {
		font-size: 1.25rem;
		font-weight: 600;
		color: #1e3a5f; /* slate blue */
		border-bottom: 1px solid #e2e8f0;
		padding-bottom: 0.25rem;
		margin-top: 1.5rem;
	}

	.bean-body :global(h2) {
		font-size: 1.1rem;
		font-weight: 600;
		color: #2d5a7b; /* muted teal-blue */
		margin-top: 1.25rem;
	}

	.bean-body :global(h3) {
		font-size: 1rem;
		font-weight: 600;
		color: #4a7c6f; /* sage green */
		margin-top: 1rem;
	}

	.bean-body :global(h4),
	.bean-body :global(h5),
	.bean-body :global(h6) {
		font-size: 0.9rem;
		font-weight: 600;
		color: #64748b; /* slate-500 */
		margin-top: 0.75rem;
	}

	/* Task list styling - remove bullets */
	.bean-body :global(ul:has(input[type='checkbox'])) {
		list-style: none;
		padding-left: 0;
	}

	.bean-body :global(li:has(> input[type='checkbox'])) {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		padding-left: 0;
	}

	.bean-body :global(li:has(> input[type='checkbox'])::before) {
		content: none;
	}

	.bean-body :global(input[type='checkbox']) {
		margin-top: 0.25rem;
		accent-color: #22c55e; /* green-500 */
	}

	/* Code block styling */
	.bean-body :global(pre.shiki) {
		padding: 1rem;
		border-radius: 0.5rem;
		overflow-x: auto;
		font-size: 0.875rem;
		line-height: 1.5;
		margin: 1rem 0;
	}

	.bean-body :global(pre.shiki code) {
		font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, 'Cascadia Code', Consolas,
			'Liberation Mono', 'Courier New', monospace;
	}

	/* Inline code */
	.bean-body :global(code:not(pre code)) {
		background-color: #f1f5f9;
		padding: 0.125rem 0.375rem;
		border-radius: 0.25rem;
		font-size: 0.875em;
		font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, 'Cascadia Code', Consolas,
			'Liberation Mono', 'Courier New', monospace;
	}
</style>
