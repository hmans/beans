<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { beansStore, type Bean } from '$lib/beans.svelte';
	import BeanItem from '$lib/components/BeanItem.svelte';
	import BeanDetail from '$lib/components/BeanDetail.svelte';
	import BoardView from '$lib/components/BoardView.svelte';

	type Tab = 'backlog' | 'board';
	let activeTab = $state<Tab>('backlog');

	onMount(() => {
		beansStore.subscribe();
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

<div class="h-screen flex flex-col bg-base-200">
	{#if beansStore.error}
		<div class="m-4">
			<div role="alert" class="alert alert-error">
				<span>Error: {beansStore.error}</span>
			</div>
		</div>
	{:else}
		<!-- Tab bar -->
		<div role="tablist" class="tabs tabs-border px-4 pt-2 bg-base-100">
			<button
				role="tab"
				onclick={() => (activeTab = 'backlog')}
				class="tab {activeTab === 'backlog' ? 'tab-active' : ''}"
			>
				Backlog
			</button>
			<button
				role="tab"
				onclick={() => (activeTab = 'board')}
				class="tab {activeTab === 'board' ? 'tab-active' : ''}"
			>
				Board
			</button>
		</div>

		<!-- Tab content -->
		<div class="flex-1 flex min-h-0">
			{#if activeTab === 'backlog'}
				<!-- Left pane: Bean list -->
				<div class="shrink-0 bg-base-100 overflow-auto" style="width: {paneWidth}px">
					<div class="p-3 space-y-1">
						{#each topLevelBeans as bean (bean.id)}
							<BeanItem {bean} selectedId={currentBean?.id} onSelect={selectBean} />
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
						{isDragging ? 'bg-base-300' : 'bg-base-200 hover:bg-base-300'}"
					role="slider"
					aria-orientation="horizontal"
					aria-valuenow={paneWidth}
					aria-valuemin={200}
					aria-valuemax={600}
					tabindex="0"
					onmousedown={startDrag}
				></div>

				<!-- Right pane: Bean detail -->
				<div class="flex-1 bg-base-100 min-w-0 overflow-hidden">
					{#if currentBean}
						<BeanDetail bean={currentBean} onSelect={selectBean} />
					{:else}
						<div class="h-full flex items-center justify-center text-base-content/30">
							<p>Select a bean to view details</p>
						</div>
					{/if}
				</div>
			{:else if activeTab === 'board'}
				<!-- Board view with optional detail pane -->
				<div class="flex-1 bg-base-200 min-w-0">
					<BoardView onSelect={selectBean} selectedId={currentBean?.id} />
				</div>

				{#if currentBean}
					<!-- Drag handle -->
					<div
						class="w-1 cursor-col-resize transition-colors shrink-0
							{isDragging ? 'bg-base-300' : 'bg-base-200 hover:bg-base-300'}"
						role="slider"
						aria-orientation="horizontal"
						aria-valuenow={paneWidth}
						aria-valuemin={200}
						aria-valuemax={600}
						tabindex="0"
						onmousedown={startDrag}
					></div>

					<div class="shrink-0 bg-base-100 overflow-hidden" style="width: {paneWidth}px">
						<BeanDetail bean={currentBean} onSelect={selectBean} />
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</div>
