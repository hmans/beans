<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { beansStore } from '$lib/beans.svelte';
	import BeanItem from '$lib/components/BeanItem.svelte';

	onMount(() => {
		beansStore.subscribe();
	});

	onDestroy(() => {
		beansStore.unsubscribe();
	});

	// Top-level beans (no parent)
	const topLevelBeans = $derived(beansStore.all.filter((b) => !b.parentId));
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
		<div class="space-y-3">
			{#each topLevelBeans as bean (bean.id)}
				<BeanItem {bean} />
			{:else}
				{#if !beansStore.loading}
					<p class="text-gray-500 text-center py-8">No beans yet</p>
				{/if}
			{/each}
		</div>
	{/if}
</div>
