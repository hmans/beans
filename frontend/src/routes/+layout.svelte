<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { preloadHighlighter } from '$lib/markdown';
	import { onMount, onDestroy } from 'svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { worktreeStore } from '$lib/worktrees.svelte';
	import { ui } from '$lib/uiState.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import BeanForm from '$lib/components/BeanForm.svelte';

	preloadHighlighter();

	onMount(() => {
		beansStore.subscribe();
		worktreeStore.subscribe();
		ui.loadPaneWidth();
		ui.loadFromUrl();
	});

	onDestroy(() => {
		beansStore.unsubscribe();
		worktreeStore.unsubscribe();
	});

	const isBacklog = $derived(page.url.pathname === '/');
	const isBoard = $derived(page.url.pathname === '/board');
	const worktreeId = $derived(
		page.url.pathname.startsWith('/worktree/') ? page.url.pathname.split('/')[2] : null
	);

	// Preserve bean selection when navigating between tabs
	const beanParam = $derived(ui.selectedBeanId ? `?bean=${ui.selectedBeanId}` : '');
	const backlogHref = $derived(`/${beanParam}`);
	const boardHref = $derived(`/board${beanParam}`);

	async function closeWorktree(e: MouseEvent, beanId: string) {
		e.preventDefault();
		e.stopPropagation();
		if (worktreeId === beanId) {
			await goto(backlogHref);
		}
		worktreeStore.removeWorktree(beanId);
	}

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>
<svelte:window onmousemove={(e) => ui.onDrag(e)} onmouseup={() => ui.stopDrag()} />

<div class="h-screen flex flex-col bg-surface-alt">
	{#if beansStore.error}
		<div class="m-4">
			<div class="rounded-lg border border-danger/30 bg-danger/10 text-danger px-4 py-3 text-sm">
				Error: {beansStore.error}
			</div>
		</div>
	{:else}
		<!-- Nav bar -->
		<div class="flex items-center px-4 pt-2 bg-surface border-b border-border">
			<nav class="flex gap-0 flex-1">
				<a
					href={backlogHref}
					class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
						{isBacklog ? 'border-accent text-accent' : 'border-transparent text-text-muted hover:text-text hover:border-border'}"
				>Backlog</a>
				<a
					href={boardHref}
					class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
						{isBoard ? 'border-accent text-accent' : 'border-transparent text-text-muted hover:text-text hover:border-border'}"
				>Board</a>
				{#each worktreeStore.worktrees as wt}
					{@const wtBean = beansStore.get(wt.beanId)}
					<a
						href="/worktree/{wt.beanId}{beanParam}"
						class="px-4 py-2 text-sm font-medium border-b-2 transition-colors flex items-center gap-1
							{worktreeId === wt.beanId ? 'border-accent text-accent' : 'border-transparent text-text-muted hover:text-text hover:border-border'}"
						title={wtBean?.title ?? wt.beanId}
					>
						{wtBean?.title ?? wt.beanId.slice(-4)}
						<button
							class="ml-1 w-4 h-4 rounded-full text-[10px] leading-none opacity-50 hover:opacity-100 hover:bg-surface-dim"
							title="Close worktree"
							onclick={(e) => closeWorktree(e, wt.beanId)}
						>
							&#x2715;
						</button>
					</a>
				{/each}
			</nav>
			<button
				class="px-3 py-1.5 text-sm font-medium bg-accent text-accent-text rounded-md hover:opacity-90 transition-opacity"
				onclick={() => ui.openCreateForm()}
			>
				+ New Bean
			</button>
		</div>

		<!-- Page content -->
		{@render children()}
	{/if}
</div>

{#if ui.showForm}
	<BeanForm
		bean={ui.editingBean}
		onClose={() => ui.closeForm()}
		onSaved={(bean) => ui.selectBean(bean)}
	/>
{/if}
