<script lang="ts">
	import { page } from '$app/state';
	import { beansStore } from '$lib/beans.svelte';
	import { ui } from '$lib/uiState.svelte';
	import BeanDetail from '$lib/components/BeanDetail.svelte';

	const beanId = $derived(page.params.id);
	const bean = $derived(beanId ? beansStore.get(beanId) : null);

	// Auto-select the worktree's bean in the detail pane
	$effect(() => {
		if (bean && !ui.selectedBeanId) {
			ui.selectBean(bean);
		}
	});

	const selectedBean = $derived(ui.selectedBeanId ? beansStore.get(ui.selectedBeanId) : null);
</script>

<div class="flex flex-1 min-h-0">
	<!-- Main content area (blank for now) -->
	<div class="flex-1 flex items-center justify-center text-base-content/40">
		{#if bean}
			<div class="text-center">
				<h2 class="text-lg font-semibold text-base-content/60">{bean.title}</h2>
				<p class="text-sm mt-1">Worktree view coming soon</p>
			</div>
		{:else}
			<span>Worktree not found</span>
		{/if}
	</div>

	<!-- Detail pane -->
	{#if selectedBean}
		<div
			class="border-l border-base-300 overflow-hidden bg-base-100 shrink-0"
			style="width: {ui.paneWidth}px"
		>
			<BeanDetail
				bean={selectedBean}
				onSelect={(b) => ui.selectBean(b)}
				onEdit={(b) => ui.openEditForm(b)}
			/>
		</div>
		<div
			class="w-1 cursor-col-resize hover:bg-primary/30 transition-colors"
			role="separator"
			onmousedown={(e) => ui.startDrag(e)}
		></div>
	{/if}
</div>
