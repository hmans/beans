<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { worktreeStore } from '$lib/worktrees.svelte';
	import BeanDetail from './BeanDetail.svelte';
	import AgentChat from './AgentChat.svelte';

	interface Props {
		bean: Bean;
		onSelect?: (bean: Bean) => void;
		onEdit?: (bean: Bean) => void;
	}

	let { bean, onSelect, onEdit }: Props = $props();

	let activeTab = $state<'bean' | 'chat'>('bean');
	const hasWorktree = $derived(worktreeStore.hasWorktree(bean.id));

	// Reset to bean tab when selected bean changes or worktree is removed
	let prevBeanId = $state(bean.id);
	$effect(() => {
		if (bean.id !== prevBeanId) {
			prevBeanId = bean.id;
			activeTab = 'bean';
		}
	});
	$effect(() => {
		if (activeTab === 'chat' && !hasWorktree) {
			activeTab = 'bean';
		}
	});
</script>

<div class="flex flex-col h-full bg-surface">
	<!-- Tab bar -->
	<div class="flex border-b border-border px-2 pt-1 shrink-0">
		<button
			onclick={() => (activeTab = 'bean')}
			class="px-3 py-1.5 text-xs font-medium border-b-2 transition-colors
				{activeTab === 'bean'
				? 'border-accent text-accent'
				: 'border-transparent text-text-muted hover:text-text'}"
		>
			Bean
		</button>
		{#if hasWorktree}
			<button
				onclick={() => (activeTab = 'chat')}
				class="px-3 py-1.5 text-xs font-medium border-b-2 transition-colors
					{activeTab === 'chat'
					? 'border-accent text-accent'
					: 'border-transparent text-text-muted hover:text-text'}"
			>
				Chat
			</button>
		{/if}
	</div>

	<!-- Tab content -->
	<div class="flex-1 min-h-0">
		{#if activeTab === 'bean'}
			<BeanDetail {bean} {onSelect} {onEdit} />
		{:else if activeTab === 'chat' && hasWorktree}
			<AgentChat beanId={bean.id} />
		{/if}
	</div>
</div>
