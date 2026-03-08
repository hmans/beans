<script lang="ts">
	import { ui } from '$lib/uiState.svelte';
	import BoardView from '$lib/components/BoardView.svelte';
	import BeanDetail from '$lib/components/BeanDetail.svelte';
</script>

<div class="flex-1 flex min-h-0">
	<!-- Board view -->
	<div class="flex-1 bg-surface-alt min-w-0">
		<BoardView onSelect={(b) => ui.selectBean(b)} selectedId={ui.currentBean?.id} />
	</div>

	{#if ui.currentBean}
		<!-- Drag handle -->
		<div
			class="w-1 cursor-col-resize transition-colors shrink-0
				{ui.isDragging ? 'bg-surface-dim' : 'bg-border hover:bg-surface-dim'}"
			role="slider"
			aria-orientation="horizontal"
			aria-valuenow={ui.paneWidth}
			aria-valuemin={200}
			aria-valuemax={600}
			tabindex="0"
			onmousedown={(e) => ui.startDrag(e)}
		></div>

		<div class="shrink-0 bg-surface overflow-hidden" style="width: {ui.paneWidth}px">
			<BeanDetail
				bean={ui.currentBean}
				onSelect={(b) => ui.selectBean(b)}
				onEdit={(b) => ui.openEditForm(b)}
			/>
		</div>
	{/if}
</div>
