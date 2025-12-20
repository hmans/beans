<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { beansStore, type Bean } from '$lib/beans.svelte';
	import BeanItem from '$lib/components/BeanItem.svelte';
	import BeanDetail from '$lib/components/BeanDetail.svelte';

	onMount(() => {
		beansStore.subscribe();
		// Load saved pane width
		const saved = localStorage.getItem('beans-pane-width');
		if (saved) {
			paneWidth = Math.max(200, Math.min(600, parseInt(saved, 10)));
		}
	});

	onDestroy(() => {
		beansStore.unsubscribe();
	});

	// Top-level beans (no parent)
	const topLevelBeans = $derived(beansStore.all.filter((b) => !b.parentId));

	// Selected bean
	let selectedBean = $state<Bean | null>(null);

	// Keep selected bean in sync (might have been updated)
	const currentBean = $derived(selectedBean ? beansStore.get(selectedBean.id) ?? null : null);

	function selectBean(bean: Bean) {
		selectedBean = bean;
	}

	// Draggable pane
	let paneWidth = $state(350);
	let isDragging = $state(false);

	function startDrag(e: MouseEvent) {
		isDragging = true;
		e.preventDefault();
	}

	function onDrag(e: MouseEvent) {
		if (!isDragging) return;
		const newWidth = e.clientX;
		paneWidth = Math.max(200, Math.min(600, newWidth));
	}

	function stopDrag() {
		if (isDragging) {
			isDragging = false;
			localStorage.setItem('beans-pane-width', paneWidth.toString());
		}
	}
</script>

<svelte:window onmousemove={onDrag} onmouseup={stopDrag} />

<div class="h-screen flex flex-col bg-gray-100">
	{#if beansStore.error}
		<div class="m-4 rounded-lg bg-red-100 p-4 text-red-700">
			Error: {beansStore.error}
		</div>
	{:else}
		<!-- Two-pane layout -->
		<div class="flex-1 flex min-h-0">
			<!-- Left pane: Bean list -->
			<div
				class="shrink-0 bg-gray-50  overflow-auto"
				style="width: {paneWidth}px"
			>
				<div class="p-3 space-y-1">
					{#each topLevelBeans as bean (bean.id)}
						<BeanItem {bean} selectedId={currentBean?.id} onSelect={selectBean} />
					{:else}
						{#if !beansStore.loading}
							<p class="text-gray-500 text-center py-8 text-sm">No beans yet</p>
						{/if}
					{/each}
				</div>
			</div>

			<!-- Drag handle -->
			<div
				class="w-1 cursor-col-resize transition-colors shrink-0
					{isDragging ? 'bg-gray-400' : 'bg-gray-200 hover:bg-gray-300'}"
				role="slider"
				aria-orientation="horizontal"
				aria-valuenow={paneWidth}
				aria-valuemin={200}
				aria-valuemax={600}
				tabindex="0"
				onmousedown={startDrag}
			></div>

			<!-- Right pane: Bean detail -->
			<div class="flex-1 bg-white min-w-0 overflow-hidden">
				{#if currentBean}
					<BeanDetail bean={currentBean} />
				{:else}
					<div class="h-full flex items-center justify-center text-gray-400">
						<p>Select a bean to view details</p>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>
