<script lang="ts">
	import { beansStore } from '$lib/beans.svelte';
	import { ui } from '$lib/uiState.svelte';
	import { backlogDrag } from '$lib/backlogDrag.svelte';
	import { matchesFilter } from '$lib/filter';
	import BeanItem from '$lib/components/BeanItem.svelte';
	import BoardView from '$lib/components/BoardView.svelte';
	import BeanPane from '$lib/components/BeanPane.svelte';
	import SplitPane from '$lib/components/SplitPane.svelte';
	import AgentChat from '$lib/components/AgentChat.svelte';
	import FilterInput from '$lib/components/FilterInput.svelte';

	const CENTRAL_SESSION_ID = '__central__';

	let filterInput = $state<FilterInput | null>(null);

	const topLevelBeans = $derived(beansStore.all.filter((b) => !b.parentId));

	const filteredTopLevelBeans = $derived.by(() => {
		const text = ui.filterText;
		if (!text) return topLevelBeans;
		return topLevelBeans.filter((bean) => {
			if (matchesFilter(bean, text)) return true;
			return beansStore.children(bean.id).some((child) => matchesFilter(child, text));
		});
	});

	function handleKeydown(e: KeyboardEvent) {
		if ((e.metaKey || e.ctrlKey) && (e.key === 'f' || e.key === '/')) {
			e.preventDefault();
			filterInput?.focus();
			return;
		}
		if (e.key === 'Escape' && ui.currentBean && !ui.showForm) {
			ui.clearSelection();
		}
	}

	function handlePlanningClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			ui.clearSelection();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<SplitPane
	direction="horizontal"
	side="start"
	persistKey="chat-width"
	initialSize={420}
	collapsed={!ui.showPlanningChat}
>
	{#snippet aside()}
		<div class="flex h-full flex-col border-r border-border bg-surface">
			<div class="toolbar">
				<span class="text-sm font-medium text-text">Agent</span>
				<div class="flex-1"></div>
				<button onclick={() => ui.togglePlanningChat()} class="btn-icon" title="Close chat">
					&#x2715;
				</button>
			</div>
			<div class="min-h-0 flex-1">
				<AgentChat beanId={CENTRAL_SESSION_ID} />
			</div>
		</div>
	{/snippet}

	{#snippet children()}
		<SplitPane
			direction="horizontal"
			side="end"
			persistKey="detail-width"
			initialSize={480}
			collapsed={!ui.currentBean}
		>
			{#snippet children()}
				<div class="flex h-full flex-col">
					<div class="toolbar bg-surface">
						<button
							onclick={() => ui.togglePlanningChat()}
							class={[
								'mr-3 flex h-8 w-8 items-center justify-center rounded transition-colors',
								ui.showPlanningChat
									? 'bg-accent text-accent-text'
									: 'border border-border bg-surface text-text-muted hover:bg-surface-alt'
							]}
							title={ui.showPlanningChat ? 'Hide chat' : 'Show chat'}
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								viewBox="0 0 20 20"
								fill="currentColor"
								class="h-4 w-4"
							>
								<path
									fill-rule="evenodd"
									d="M10 3c-4.31 0-8 3.033-8 7 0 2.024.978 3.825 2.499 5.085a3.478 3.478 0 01-.522 1.756.75.75 0 00.584 1.143 5.976 5.976 0 003.936-1.108c.487.082.99.124 1.503.124 4.31 0 8-3.033 8-7s-3.69-7-8-7z"
									clip-rule="evenodd"
								/>
							</svg>
						</button>

						<div class="flex">
							<button
								onclick={() => ui.setPlanningView('backlog')}
								class={[
									'btn-tab rounded-l-md',
									ui.planningView === 'backlog' ? 'btn-tab-active' : 'btn-tab-inactive'
								]}
							>
								Backlog
							</button>
							<button
								onclick={() => ui.setPlanningView('board')}
								class={[
									'btn-tab rounded-r-md border-l-0',
									ui.planningView === 'board' ? 'btn-tab-active' : 'btn-tab-inactive'
								]}
							>
								Board
							</button>
						</div>
						<div class="mx-3 w-60">
							<FilterInput bind:this={filterInput} />
						</div>
						<div class="flex-1"></div>
						<button class="btn-primary" onclick={() => ui.openCreateForm()}>+ New Bean</button>
					</div>

					{#if ui.planningView === 'backlog'}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<div class="flex-1 overflow-auto bg-surface" onclick={handlePlanningClick}>
							<div
								class="p-3"
								ondragover={(e) => backlogDrag.hoverList(e, null, filteredTopLevelBeans.length)}
								ondragleave={(e) => backlogDrag.leaveList(e, e.currentTarget, null)}
								ondrop={(e) => backlogDrag.drop(e, null, filteredTopLevelBeans)}
								role="list"
							>
								{#each filteredTopLevelBeans as bean, i (bean.id)}
									<BeanItem
										{bean}
										parentId={null}
										index={i}
										selectedId={ui.currentBean?.id}
										onSelect={(b) => ui.selectBean(b)}
										filterText={ui.filterText}
									/>
								{:else}
									{#if !beansStore.loading}
										<p class="text-text-muted text-center py-8 text-sm">
											{ui.filterText ? 'No matching beans' : 'No beans yet'}
										</p>
									{/if}
								{/each}

								<div
									class={[
										'mx-1 h-0.5 rounded-full transition-colors',
										backlogDrag.showEndIndicator(null, filteredTopLevelBeans.length)
											? 'bg-accent'
											: 'bg-transparent'
									]}
								></div>
							</div>
						</div>
					{:else}
						<div class="min-h-0 flex-1 bg-surface-alt">
							<BoardView onSelect={(b) => ui.selectBean(b)} selectedId={ui.currentBean?.id} />
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
						onClose={() => ui.clearSelection()}
					/>
				{/if}
			{/snippet}
		</SplitPane>
	{/snippet}
</SplitPane>
