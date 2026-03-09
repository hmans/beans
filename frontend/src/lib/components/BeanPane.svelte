<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { worktreeStore } from '$lib/worktrees.svelte';
	import BeanDetail from './BeanDetail.svelte';
	import AgentChat from './AgentChat.svelte';

	interface Props {
		bean: Bean;
		onSelect?: (bean: Bean) => void;
		onEdit?: (bean: Bean) => void;
		onClose?: () => void;
	}

	let { bean, onSelect, onEdit, onClose }: Props = $props();

	const hasWorktree = $derived(worktreeStore.hasWorktree(bean.id));

	// Track explicit tab selection per bean; activeTab derives from it
	let tabSelection = $state<{ beanId: string; tab: 'bean' | 'chat' } | null>(null);
	const activeTab = $derived.by(() => {
		if (tabSelection?.beanId === bean.id) {
			// Fall back to 'bean' if chat tab selected but worktree was removed
			if (tabSelection.tab === 'chat' && !hasWorktree) return 'bean';
			return tabSelection.tab;
		}
		return 'bean';
	});

	function setTab(tab: 'bean' | 'chat') {
		tabSelection = { beanId: bean.id, tab };
	}
</script>

<div class="flex flex-col h-full bg-surface">
	<!-- Tab bar -->
	<div class="flex items-center px-4 h-10 border-b border-border shrink-0">
		<div class="flex">
			<button
				onclick={() => setTab('bean')}
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
					onclick={() => setTab('chat')}
					class="px-3 py-1 text-sm font-medium rounded-r-md border border-l-0 transition-colors
						{activeTab === 'chat'
						? 'bg-accent text-accent-text border-accent'
						: 'bg-surface border-border text-text-muted hover:bg-surface-alt'}"
				>
					Chat
				</button>
			{/if}
		</div>
		{#if onClose}
			<div class="flex-1"></div>
			<button
				onclick={onClose}
				class="w-6 h-6 flex items-center justify-center rounded text-text-muted hover:text-text hover:bg-surface-alt transition-colors"
				title="Close"
			>
				&#x2715;
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
