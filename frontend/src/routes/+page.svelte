<script lang="ts">
	import { beansStore } from '$lib/beans.svelte';
	import { ui } from '$lib/uiState.svelte';
	import BeanItem from '$lib/components/BeanItem.svelte';
	import BoardView from '$lib/components/BoardView.svelte';
	import BeanPane from '$lib/components/BeanPane.svelte';
	import SplitPane from '$lib/components/SplitPane.svelte';

	const topLevelBeans = $derived(beansStore.all.filter((b) => !b.parentId));
</script>

<SplitPane direction="horizontal" side="end" persistKey="detail-width" initialSize={480}>
	{#snippet children()}
		<div class="flex flex-col h-full">
			<!-- Toggle bar -->
			<div class="flex items-center px-4 h-10 border-b border-border bg-surface shrink-0">
				<div class="flex">
					<button
						onclick={() => ui.setPlanningView('backlog')}
						class="px-3 py-1 text-sm font-medium rounded-l-md border transition-colors
							{ui.planningView === 'backlog'
							? 'bg-accent text-accent-text border-accent'
							: 'bg-surface border-border text-text-muted hover:bg-surface-alt'}"
					>
						Backlog
					</button>
					<button
						onclick={() => ui.setPlanningView('board')}
						class="px-3 py-1 text-sm font-medium rounded-r-md border border-l-0 transition-colors
							{ui.planningView === 'board'
							? 'bg-accent text-accent-text border-accent'
							: 'bg-surface border-border text-text-muted hover:bg-surface-alt'}"
					>
						Board
					</button>
				</div>
				<div class="flex-1"></div>
				<button
					class="px-3 py-1.5 text-sm font-medium bg-accent text-accent-text rounded-md hover:opacity-90 transition-opacity"
					onclick={() => ui.openCreateForm()}
				>
					+ New Bean
				</button>
			</div>

			{#if ui.planningView === 'backlog'}
				<div class="flex-1 overflow-auto bg-surface">
					<div class="p-3 space-y-1">
						{#each topLevelBeans as bean (bean.id)}
							<BeanItem
								{bean}
								selectedId={ui.currentBean?.id}
								onSelect={(b) => ui.selectBean(b)}
							/>
						{:else}
							{#if !beansStore.loading}
								<p class="text-text-muted text-center py-8 text-sm">No beans yet</p>
							{/if}
						{/each}
					</div>
				</div>
			{:else}
				<div class="flex-1 min-h-0 bg-surface-alt">
					<BoardView
						onSelect={(b) => ui.selectBean(b)}
						selectedId={ui.currentBean?.id}
					/>
				</div>
			{/if}
		</div>
	{/snippet}

	{#snippet aside()}
		{#if ui.currentBean}
			<BeanPane
				bean={ui.currentBean}
				onSelect={(b) => ui.selectBean(b)}
				onEdit={(b) => ui.openEditForm(b)}
			/>
		{:else}
			<div class="h-full flex items-center justify-center text-text-faint bg-surface">
				<p>Select a bean to view details</p>
			</div>
		{/if}
	{/snippet}
</SplitPane>
