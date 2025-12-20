<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { beansStore } from '$lib/beans.svelte';

	onMount(async () => {
		await beansStore.load();
		beansStore.subscribe();
	});

	onDestroy(() => {
		beansStore.unsubscribe();
	});

	// Status colors
	const statusColors: Record<string, string> = {
		'todo': 'bg-gray-200 text-gray-800',
		'in-progress': 'bg-blue-200 text-blue-800',
		'completed': 'bg-green-200 text-green-800',
		'scrapped': 'bg-red-200 text-red-800',
		'draft': 'bg-yellow-200 text-yellow-800'
	};

	// Type colors
	const typeColors: Record<string, string> = {
		'milestone': 'bg-purple-100 text-purple-700',
		'epic': 'bg-indigo-100 text-indigo-700',
		'feature': 'bg-cyan-100 text-cyan-700',
		'bug': 'bg-red-100 text-red-700',
		'task': 'bg-gray-100 text-gray-700'
	};
</script>

<div class="min-h-screen bg-gray-50 p-8">
	<header class="mb-8">
		<h1 class="text-3xl font-bold text-gray-900">Beans</h1>
		<p class="text-gray-600">
			{beansStore.count} beans
			{#if beansStore.loading}
				<span class="text-blue-600">· Loading...</span>
			{/if}
			{#if beansStore.connected}
				<span class="text-green-600">· Live</span>
			{/if}
		</p>
	</header>

	{#if beansStore.error}
		<div class="rounded-lg bg-red-100 p-4 text-red-700">
			Error: {beansStore.error}
		</div>
	{:else}
		<div class="grid gap-4">
			{#each beansStore.all as bean (bean.id)}
				<div class="rounded-lg bg-white p-4 shadow-sm hover:shadow-md transition-shadow">
					<div class="flex items-start justify-between gap-4">
						<div class="flex-1">
							<div class="flex items-center gap-2 mb-1">
								<code class="text-xs text-gray-400">{bean.id}</code>
								<span class="text-xs px-2 py-0.5 rounded-full {typeColors[bean.type] ?? 'bg-gray-100 text-gray-700'}">
									{bean.type}
								</span>
								<span class="text-xs px-2 py-0.5 rounded-full {statusColors[bean.status] ?? 'bg-gray-200 text-gray-800'}">
									{bean.status}
								</span>
							</div>
							<h2 class="text-lg font-medium text-gray-900">{bean.title}</h2>
							{#if bean.tags.length > 0}
								<div class="flex gap-1 mt-2">
									{#each bean.tags as tag}
										<span class="text-xs px-2 py-0.5 rounded bg-gray-100 text-gray-600">
											{tag}
										</span>
									{/each}
								</div>
							{/if}
						</div>
						<div class="text-right text-xs text-gray-400">
							{new Date(bean.updatedAt).toLocaleDateString()}
						</div>
					</div>
				</div>
			{:else}
				{#if !beansStore.loading}
					<p class="text-gray-500 text-center py-8">No beans yet</p>
				{/if}
			{/each}
		</div>
	{/if}
</div>
