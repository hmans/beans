<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { worktreeStore } from '$lib/worktrees.svelte';
	import { agentStatusesStore } from '$lib/agentStatuses.svelte';
	import { ui } from '$lib/uiState.svelte';

	const worktreeBeans = $derived(
		worktreeStore.worktrees
			.map((wt) => beansStore.get(wt.beanId))
			.filter((b): b is Bean => b != null)
	);
</script>

<nav class="flex h-full w-56 shrink-0 flex-col border-r border-border bg-surface-alt">
	<div class="flex h-14 shrink-0 items-center border-b border-border px-3">
		<span class="text-sm font-semibold text-text">beans</span>
	</div>

	<div class="flex-1 overflow-y-auto p-2">
		<!-- Planning item -->
		<button
			onclick={() => ui.setActiveView('planning')}
			class={[
				'flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors',
				ui.isPlanning
					? 'bg-surface font-medium text-text'
					: 'text-text-muted hover:bg-surface hover:text-text'
			]}
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				viewBox="0 0 20 20"
				fill="currentColor"
				class="h-4 w-4 shrink-0"
			>
				<path
					fill-rule="evenodd"
					d="M6 4.75A.75.75 0 016.75 4h10.5a.75.75 0 010 1.5H6.75A.75.75 0 016 4.75zM6 10a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H6.75A.75.75 0 016 10zm0 5.25a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H6.75a.75.75 0 01-.75-.75zM1.99 4.75a1 1 0 011-1h.01a1 1 0 010 2h-.01a1 1 0 01-1-1zm0 5.25a1 1 0 011-1h.01a1 1 0 010 2h-.01a1 1 0 01-1-1zm0 5.25a1 1 0 011-1h.01a1 1 0 010 2h-.01a1 1 0 01-1-1z"
					clip-rule="evenodd"
				/>
			</svg>
			Planning
			{#if agentStatusesStore.isRunning('__central__')}
				<span class="ml-auto h-2 w-2 shrink-0 animate-pulse rounded-full bg-success"></span>
			{/if}
		</button>

		<!-- Workspaces section -->
		{#if worktreeBeans.length > 0}
			<div class="mb-1 mt-4 px-3 text-xs font-medium uppercase tracking-wider text-text-faint">
				Workspaces
			</div>

			{#each worktreeBeans as bean (bean.id)}
				<button
					onclick={() => ui.setActiveView(bean.id)}
					class={[
						'flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors',
						ui.activeView === bean.id
							? 'bg-surface font-medium text-text'
							: 'text-text-muted hover:bg-surface hover:text-text'
					]}
				>
					<span class="min-w-0 flex-1 truncate">{bean.title}</span>
					{#if agentStatusesStore.isRunning(bean.id)}
						<span class="h-2 w-2 shrink-0 animate-pulse rounded-full bg-success"></span>
					{/if}
				</button>
			{/each}
		{/if}
	</div>
</nav>
