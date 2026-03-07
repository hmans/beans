<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { preloadHighlighter } from '$lib/markdown';
	import { onMount, onDestroy } from 'svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { ui } from '$lib/uiState.svelte';
	import { page } from '$app/state';
	import BeanForm from '$lib/components/BeanForm.svelte';

	preloadHighlighter();

	onMount(() => {
		beansStore.subscribe();
		ui.loadPaneWidth();
		ui.loadFromUrl();
	});

	onDestroy(() => {
		beansStore.unsubscribe();
	});

	const isBacklog = $derived(page.url.pathname === '/');
	const isBoard = $derived(page.url.pathname === '/board');

	// Preserve bean selection when navigating between tabs
	const beanParam = $derived(ui.selectedBeanId ? `?bean=${ui.selectedBeanId}` : '');
	const backlogHref = $derived(`/${beanParam}`);
	const boardHref = $derived(`/board${beanParam}`);

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
