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
	<div class="flex-1 flex items-center justify-center text-text-faint">
		{#if bean}
			<div class="text-center">
				<h2 class="text-lg font-semibold text-text-muted">{bean.title}</h2>
				<p class="text-sm mt-1">Worktree view coming soon</p>
			</div>
		{:else}
			<span>Worktree not found</span>
		{/if}
	</div>

	<!-- Detail pane -->
	{#if selectedBean}
		<div
			class="border-l border-border overflow-hidden bg-surface shrink-0"
			style="width: {ui.paneWidth}px"
		>
			<BeanDetail
				bean={selectedBean}
				onSelect={(b) => ui.selectBean(b)}
				onEdit={(b) => ui.openEditForm(b)}
			/>
		</div>
		<div
			class="w-1 cursor-col-resize hover:bg-accent/30 transition-colors"
			role="slider"
			aria-orientation="horizontal"
			aria-valuenow={ui.paneWidth}
			aria-valuemin={200}
			aria-valuemax={600}
			tabindex="0"
			onmousedown={(e) => ui.startDrag(e)}
		></div>
	{/if}
</div>
