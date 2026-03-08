<script lang="ts">
	import { page } from '$app/state';
	import { beansStore } from '$lib/beans.svelte';
	import { ui } from '$lib/uiState.svelte';
	import AgentChat from '$lib/components/AgentChat.svelte';
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
	<!-- Agent chat -->
	<div class="flex-1 min-w-0">
		{#if beanId}
			<AgentChat beanId={beanId} />
		{:else}
			<div class="flex items-center justify-center h-full text-text-faint">
				<span>Worktree not found</span>
			</div>
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
