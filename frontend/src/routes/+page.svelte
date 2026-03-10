<script lang="ts">
	import { beansStore } from '$lib/beans.svelte';
	import { ui } from '$lib/uiState.svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import PlanningView from '$lib/components/PlanningView.svelte';
	import WorkspaceView from '$lib/components/WorkspaceView.svelte';

	const workspaceBean = $derived(
		!ui.isPlanning ? (beansStore.get(ui.activeView) ?? null) : null
	);
</script>

<div class="flex h-full min-h-0">
	<Sidebar />

	<div class="flex min-h-0 min-w-0 flex-1 flex-col">
		{#if ui.isPlanning}
			<PlanningView />
		{:else if workspaceBean}
			{#key workspaceBean.id}
				<WorkspaceView bean={workspaceBean} />
			{/key}
		{/if}
	</div>
</div>
