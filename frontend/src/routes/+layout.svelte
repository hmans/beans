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

<div class="h-screen flex flex-col bg-base-200">
	{#if beansStore.error}
		<div class="m-4">
			<div role="alert" class="alert alert-error">
				<span>Error: {beansStore.error}</span>
			</div>
		</div>
	{:else}
		<!-- Nav bar -->
		<div class="flex items-center px-4 pt-2 bg-base-100 border-b border-base-200">
			<nav role="tablist" class="tabs tabs-border flex-1">
				<a href={backlogHref} role="tab" class="tab {isBacklog ? 'tab-active' : ''}">Backlog</a>
				<a href={boardHref} role="tab" class="tab {isBoard ? 'tab-active' : ''}">Board</a>
				{#each worktreeStore.worktrees as wt}
					{@const wtBean = beansStore.get(wt.beanId)}
					<a
						href="/worktree/{wt.beanId}{beanParam}"
						role="tab"
						class="tab gap-1 {worktreeId === wt.beanId ? 'tab-active' : ''}"
						title={wtBean?.title ?? wt.beanId}
					>
						{wtBean?.title ?? wt.beanId.slice(-4)}
						<button
							class="btn btn-ghost btn-xs btn-circle opacity-50 hover:opacity-100"
							title="Close worktree"
							onclick={(e) => closeWorktree(e, wt.beanId)}
						>
							&#x2715;
						</button>
					</a>
				{/each}
			</nav>
			<button class="btn btn-primary btn-sm" onclick={() => ui.openCreateForm()}>
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
