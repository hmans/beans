<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { ui } from '$lib/uiState.svelte';
	import SplitPane from './SplitPane.svelte';
	import AgentChat from './AgentChat.svelte';
	import BeanPane from './BeanPane.svelte';

	interface Props {
		bean: Bean;
	}

	let { bean }: Props = $props();
</script>

<SplitPane direction="horizontal" side="start" persistKey="workspace-chat-width" initialSize={480}>
	{#snippet aside()}
		<div class="flex h-full flex-col border-r border-border bg-surface">
			<div class="toolbar">
				<span class="text-sm font-medium text-text">Agent</span>
			</div>
			<div class="min-h-0 flex-1">
				<AgentChat beanId={bean.id} />
			</div>
		</div>
	{/snippet}

	{#snippet children()}
		<BeanPane
			{bean}
			onSelect={(b) => ui.selectBean(b)}
			onEdit={(b) => ui.openEditForm(b)}
		/>
	{/snippet}
</SplitPane>
