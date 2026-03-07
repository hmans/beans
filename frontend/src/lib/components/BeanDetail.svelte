<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { worktreeStore } from '$lib/worktrees.svelte';
	import { renderMarkdown } from '$lib/markdown';

	interface Props {
		bean: Bean;
		onSelect?: (bean: Bean) => void;
		onEdit?: (bean: Bean) => void;
	}

	let { bean, onSelect, onEdit }: Props = $props();

	const parent = $derived(bean.parentId ? beansStore.get(bean.parentId) : null);
	const children = $derived(beansStore.children(bean.id));
	const blocking = $derived(
		bean.blockingIds.map((id) => beansStore.get(id)).filter((b): b is Bean => b !== undefined)
	);
	const blockedBy = $derived(beansStore.blockedBy(bean.id));

	const statusBadge: Record<string, string> = {
		todo: 'badge-ghost',
		'in-progress': 'badge-info',
		completed: 'badge-success',
		scrapped: 'badge-error',
		draft: 'badge-warning'
	};

	const typeBadge: Record<string, string> = {
		milestone: 'badge-secondary',
		epic: 'badge-primary',
		feature: 'badge-accent',
		bug: 'badge-error',
		task: 'badge-ghost'
	};

	const typeBorders: Record<string, string> = {
		milestone: 'border-l-purple-400',
		epic: 'border-l-indigo-400',
		feature: 'border-l-cyan-400',
		bug: 'border-l-red-400',
		task: 'border-l-base-300'
	};

	const priorityBadge: Record<string, string> = {
		critical: 'badge-error',
		high: 'badge-warning',
		normal: 'badge-ghost',
		low: 'badge-ghost opacity-60',
		deferred: 'badge-ghost opacity-40'
	};

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

	const canStartWork = $derived(
		(bean.status === 'todo' || bean.status === 'draft') && !worktreeStore.hasWorktree(bean.id)
	);

	let startingWork = $state(false);

	async function startWork() {
		startingWork = true;
		await worktreeStore.createWorktree(bean.id);
		startingWork = false;
	}
</script>

{#snippet beanCard(b: Bean)}
	<button
		onclick={() => onSelect?.(b)}
		class="w-full text-left rounded-lg p-2 border-l-2 transition-all cursor-pointer bg-base-100 hover:bg-base-200
			{typeBorders[b.type] ?? 'border-l-base-300'}"
	>
		<div class="flex items-center gap-1.5 min-w-0">
			<code class="text-[9px] text-base-content/40 shrink-0">{b.id.slice(-4)}</code>
			<span class="text-xs text-base-content truncate flex-1">{b.title}</span>
			<span class="badge badge-xs {statusBadge[b.status] ?? 'badge-ghost'} shrink-0">
				{b.status}
			</span>
		</div>
	</button>
{/snippet}

<div class="h-full overflow-auto p-6">
	<!-- Header -->
	<div class="mb-6">
		<div class="flex items-center gap-2 mb-2 flex-wrap">
			<button
				onclick={copyId}
				class="btn btn-ghost btn-xs font-mono gap-1"
				title="Copy ID to clipboard"
			>
				{bean.id}
				{#if copied}
					<span class="text-success">&#10003;</span>
				{:else}
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
						/>
					</svg>
				{/if}
			</button>
			<span class="badge badge-sm {typeBadge[bean.type] ?? 'badge-ghost'}">{bean.type}</span>
			<span class="badge badge-sm {statusBadge[bean.status] ?? 'badge-ghost'}">{bean.status}</span>
			{#if bean.priority && bean.priority !== 'normal'}
				<span class="badge badge-sm badge-outline {priorityBadge[bean.priority] ?? ''}">
					{bean.priority}
				</span>
			{/if}
		</div>
		<div class="flex items-center gap-2">
			<h1 class="text-2xl font-bold text-base-content flex-1">{bean.title}</h1>
			{#if canStartWork}
				<button
					class="btn btn-success btn-sm"
					onclick={startWork}
					disabled={startingWork}
				>
					{#if startingWork}
						<span class="loading loading-spinner loading-sm"></span>
					{/if}
					Start Work
				</button>
			{/if}
			{#if onEdit}
				<button class="btn btn-ghost btn-sm" onclick={() => onEdit(bean)}>Edit</button>
			{/if}
		</div>
	</div>

	<!-- Tags -->
	{#if bean.tags.length > 0}
		<div class="mb-6">
			<h2 class="text-xs font-semibold text-base-content/50 uppercase mb-2">Tags</h2>
			<div class="flex gap-1 flex-wrap">
				{#each bean.tags as tag}
					<span class="badge badge-sm badge-outline">{tag}</span>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Relationships -->
	{#if parent || children.length > 0 || blocking.length > 0 || blockedBy.length > 0}
		<div class="mb-6 space-y-3">
			{#if parent}
				<div>
					<h2 class="text-xs font-semibold text-base-content/50 uppercase mb-1">Parent</h2>
					{@render beanCard(parent)}
				</div>
			{/if}

			{#if children.length > 0}
				<div>
					<h2 class="text-xs font-semibold text-base-content/50 uppercase mb-1">
						Children ({children.length})
					</h2>
					<div class="space-y-0.5">
						{#each children as child}
							{@render beanCard(child)}
						{/each}
					</div>
				</div>
			{/if}

			{#if blocking.length > 0}
				<div>
					<h2 class="text-xs font-semibold text-base-content/50 uppercase mb-1">
						Blocking ({blocking.length})
					</h2>
					<div class="space-y-0.5">
						{#each blocking as b}
							{@render beanCard(b)}
						{/each}
					</div>
				</div>
			{/if}

			{#if blockedBy.length > 0}
				<div>
					<h2 class="text-xs font-semibold text-base-content/50 uppercase mb-1">
						Blocked By ({blockedBy.length})
					</h2>
					<div class="space-y-0.5">
						{#each blockedBy as b}
							{@render beanCard(b)}
						{/each}
					</div>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Body -->
	{#if bean.body}
		<div class="mb-6">
			<h2 class="text-xs font-semibold text-base-content/50 uppercase mb-2">Description</h2>
			<div class="bean-body prose prose-sm max-w-none">
				{@html renderedBody}
			</div>
		</div>
	{/if}

	<!-- Metadata -->
	<div class="divider"></div>
	<div class="text-xs text-base-content/40 space-y-1">
		<div>Created: {new Date(bean.createdAt).toLocaleString()}</div>
		<div>Updated: {new Date(bean.updatedAt).toLocaleString()}</div>
		<div>Path: {bean.path}</div>
	</div>
</div>

<style>
	.bean-body :global(h1) {
		font-size: 1.25rem;
		font-weight: 600;
		color: #1e3a5f;
		border-bottom: 1px solid #e2e8f0;
		padding-bottom: 0.25rem;
		margin-top: 1.5rem;
	}

	.bean-body :global(h2) {
		font-size: 1.1rem;
		font-weight: 600;
		color: #2d5a7b;
		margin-top: 1.25rem;
	}

	.bean-body :global(h3) {
		font-size: 1rem;
		font-weight: 600;
		color: #4a7c6f;
		margin-top: 1rem;
	}

	.bean-body :global(h4),
	.bean-body :global(h5),
	.bean-body :global(h6) {
		font-size: 0.9rem;
		font-weight: 600;
		color: #64748b;
		margin-top: 0.75rem;
	}

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
		accent-color: #22c55e;
	}

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

	.bean-body :global(code:not(pre code)) {
		background-color: #f1f5f9;
		padding: 0.125rem 0.375rem;
		border-radius: 0.25rem;
		font-size: 0.875em;
		font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, 'Cascadia Code', Consolas,
			'Liberation Mono', 'Courier New', monospace;
	}
</style>
