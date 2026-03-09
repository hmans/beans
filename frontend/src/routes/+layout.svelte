<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { preloadHighlighter } from '$lib/markdown';
	import { onMount, onDestroy } from 'svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { worktreeStore } from '$lib/worktrees.svelte';
	import { ui } from '$lib/uiState.svelte';
	import BeanForm from '$lib/components/BeanForm.svelte';

	preloadHighlighter();

	let { data, children } = $props();

	// Initialize UI state from load function data (runs before first render)
	ui.planningView = data.planningView;
	if (data.selectedBeanId) {
		ui.selectedBeanId = data.selectedBeanId;
	}

	onMount(() => {
		beansStore.subscribe();
		worktreeStore.subscribe();
	});

	onDestroy(() => {
		beansStore.unsubscribe();
		worktreeStore.unsubscribe();
	});
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="h-screen flex flex-col bg-surface-alt">
	{#if beansStore.error}
		<div class="m-4">
			<div class="rounded-lg border border-danger/30 bg-danger/10 text-danger px-4 py-3 text-sm">
				Error: {beansStore.error}
			</div>
		</div>
	{:else}
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
