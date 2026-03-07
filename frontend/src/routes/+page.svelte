<script lang="ts">
	import { beansStore } from '$lib/beans.svelte';
	import { ui } from '$lib/uiState.svelte';
	import BeanItem from '$lib/components/BeanItem.svelte';
	import BeanDetail from '$lib/components/BeanDetail.svelte';

	const topLevelBeans = $derived(beansStore.all.filter((b) => !b.parentId));
</script>

<div class="flex-1 flex min-h-0">
	<!-- Left pane: Bean list -->
	<div class="shrink-0 bg-base-100 overflow-auto" style="width: {ui.paneWidth}px">
		<div class="p-3 space-y-1">
			{#each topLevelBeans as bean (bean.id)}
				<BeanItem {bean} selectedId={ui.currentBean?.id} onSelect={(b) => ui.selectBean(b)} />
			{:else}
				{#if !beansStore.loading}
					<p class="text-base-content/50 text-center py-8 text-sm">No beans yet</p>
				{/if}
			{/each}
		</div>
	</div>

	<!-- Drag handle -->
	<div
		class="w-1 cursor-col-resize transition-colors shrink-0
			{ui.isDragging ? 'bg-base-300' : 'bg-base-200 hover:bg-base-300'}"
		role="slider"
		aria-orientation="horizontal"
		aria-valuenow={ui.paneWidth}
		aria-valuemin={200}
		aria-valuemax={600}
		tabindex="0"
		onmousedown={(e) => ui.startDrag(e)}
	></div>

	<!-- Right pane: Bean detail -->
	<div class="flex-1 bg-base-100 min-w-0 overflow-hidden">
		{#if ui.currentBean}
			<BeanDetail
				bean={ui.currentBean}
				onSelect={(b) => ui.selectBean(b)}
				onEdit={(b) => ui.openEditForm(b)}
			/>
		{:else}
			<div class="h-full flex items-center justify-center text-base-content/30">
				<p>Select a bean to view details</p>
			</div>
		{/if}
	</div>
</div>
