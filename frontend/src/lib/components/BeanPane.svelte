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
	<div class="flex items-center px-4 h-10 border-b border-border shrink-0">
		<div class="flex">
			<button
				onclick={() => (activeTab = 'bean')}
				class="px-3 py-1 text-sm font-medium border transition-colors
					{hasWorktree ? 'rounded-l-md' : 'rounded-md'}
					{activeTab === 'bean'
					? 'bg-accent text-accent-text border-accent'
					: 'bg-surface border-border text-text-muted hover:bg-surface-alt'}"
			>
				Bean
			</button>
			{#if hasWorktree}
				<button
					onclick={() => (activeTab = 'chat')}
					class="px-3 py-1 text-sm font-medium rounded-r-md border border-l-0 transition-colors
						{activeTab === 'chat'
						? 'bg-accent text-accent-text border-accent'
						: 'bg-surface border-border text-text-muted hover:bg-surface-alt'}"
				>
					Chat
				</button>
			{/if}
		</div>
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
