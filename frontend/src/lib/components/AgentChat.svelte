<script lang="ts">
	import { AgentChatStore } from '$lib/agentChat.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { ui } from '$lib/uiState.svelte';
	import { renderMarkdown } from '$lib/markdown';
	import { onDestroy } from 'svelte';

	interface Props {
		beanId: string;
	}

	let { beanId }: Props = $props();

	const store = new AgentChatStore();

	let inputText = $state('');
	let messagesEl: HTMLDivElement | undefined = $state();
	let renderedMessages = $state<Map<string, string>>(new Map());

	// Subscribe to agent session updates
	$effect(() => {
		store.subscribe(beanId);
	});

	onDestroy(() => {
		store.unsubscribe();
	});

	const messages = $derived(store.session?.messages ?? []);
	const status = $derived(store.session?.status ?? null);
	const isRunning = $derived(status === 'RUNNING');
	const sessionError = $derived(store.session?.error ?? null);
	const systemStatus = $derived(store.session?.systemStatus ?? null);
	const planMode = $derived(store.session?.planMode ?? false);
	const yoloMode = $derived(store.session?.yoloMode ?? false);
	const agentMode = $derived<'plan' | 'act' | 'yolo'>(
		planMode ? 'plan' : yoloMode ? 'yolo' : 'act'
	);

	function setAgentMode(mode: 'plan' | 'act' | 'yolo') {
		store.setPlanMode(beanId, mode === 'plan');
		store.setYoloMode(beanId, mode === 'yolo');
	}
	const activityLabel = $derived(systemStatus ? `${systemStatus}...` : 'thinking...');
	const pendingInteraction = $derived(store.session?.pendingInteraction ?? null);

	// Render plan content as markdown when available
	let renderedPlanContent = $state<string | null>(null);
	$effect(() => {
		const content = pendingInteraction?.planContent;
		if (content) {
			renderMarkdown(content).then((html) => {
				renderedPlanContent = html;
			});
		} else {
			renderedPlanContent = null;
		}
	});

	function approveInteraction() {
		if (!pendingInteraction) return;
		store.sendMessage(beanId, 'yes, proceed');
	}

	function approveInteractionWithYolo() {
		if (!pendingInteraction) return;
		store.setYoloMode(beanId, true);
		store.sendMessage(beanId, 'yes, proceed');
	}

	function rejectInteraction() {
		if (!pendingInteraction) return;
		if (pendingInteraction.type === 'EXIT_PLAN') {
			// Rejected exiting plan → go back to plan mode
			store.setPlanMode(beanId, true);
		} else {
			// Rejected entering plan → go back to work mode
			store.setPlanMode(beanId, false);
		}
	}

	function allowPermission() {
		store.resolvePermission(beanId, true);
	}

	function alwaysAllowPermission() {
		store.resolvePermission(beanId, true, true);
	}

	function denyPermission() {
		store.resolvePermission(beanId, false);
	}

	function formatToolInput(toolName: string | null, toolInput: string | null): string {
		if (!toolInput) return '';
		try {
			const input = JSON.parse(toolInput);
			switch (toolName) {
				case 'Bash':
					return input.command ?? '';
				case 'Write':
				case 'Read':
					return input.file_path ?? '';
				case 'Edit':
					return input.file_path
						? `${input.file_path}\n${input.old_string ?? ''} → ${input.new_string ?? ''}`
						: '';
				case 'Grep':
					return input.pattern ? `/${input.pattern}/ ${input.path ?? ''}`.trim() : '';
				case 'Glob':
					return input.pattern ?? '';
				default:
					return JSON.stringify(input, null, 2);
			}
		} catch {
			return toolInput;
		}
	}

	// Auto-scroll to bottom when messages change
	$effect(() => {
		messages.length;
		if (messagesEl) {
			requestAnimationFrame(() => {
				if (messagesEl) {
					messagesEl.scrollTop = messagesEl.scrollHeight;
				}
			});
		}
	});

	// Render markdown for assistant messages (including the one being streamed).
	// The key includes content length, so each new delta triggers a re-render.
	$effect(() => {
		for (let i = 0; i < messages.length; i++) {
			const msg = messages[i];
			if (msg.role !== 'ASSISTANT') continue;

			const key = `${i}:${msg.content.length}`;
			if (!renderedMessages.has(key)) {
				renderMarkdown(msg.content).then((html) => {
					renderedMessages = new Map(renderedMessages).set(key, html);
				});
			}
		}
	});

	function getRenderedContent(index: number): string | null {
		const msg = messages[index];
		if (!msg || msg.role !== 'ASSISTANT') return null;
		const key = `${index}:${msg.content.length}`;
		return renderedMessages.get(key) ?? null;
	}

	async function send() {
		const text = inputText.trim();
		if (!text) return;

		inputText = '';
		await store.sendMessage(beanId, text);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			send();
		}
	}

	function handleBeanLinkClick(e: MouseEvent) {
		const target = (e.target as HTMLElement).closest<HTMLElement>('[data-bean-id]');
		if (!target) return;
		e.preventDefault();
		const linkedBean = beansStore.get(target.dataset.beanId!);
		if (linkedBean) ui.selectBean(linkedBean);
	}
</script>

<div class="flex flex-col h-full font-mono text-sm">
	<!-- Messages area -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		bind:this={messagesEl}
		class="flex-1 overflow-y-auto p-4 space-y-3"
		onclick={handleBeanLinkClick}
	>
		{#if messages.length === 0}
			<div class="flex items-center justify-center h-full text-text-faint">
				<p>Send a message to start a conversation with the agent.</p>
			</div>
		{:else}
			{#each messages as msg, i}
				{#if msg.role === 'USER'}
					<div class="flex gap-2">
						<span class="shrink-0 text-accent font-bold select-none">&gt;</span>
						<p class="whitespace-pre-wrap text-text">{msg.content}</p>
					</div>
				{:else if msg.role === 'TOOL'}
					<div class="flex gap-2 text-text-faint text-xs">
						<span class="shrink-0 select-none">&middot;</span>
						<span>{msg.content}</span>
					</div>
				{:else if getRenderedContent(i)}
					<div class="flex gap-2">
						<span class="shrink-0 text-text-muted select-none">&middot;</span>
						<div class="agent-prose prose max-w-none text-text min-w-0">
							{@html getRenderedContent(i)}
						</div>
					</div>
				{:else if msg.content}
					<div class="flex gap-2">
						<span class="shrink-0 text-text-muted select-none">&middot;</span>
						<p class="whitespace-pre-wrap text-text">{msg.content}</p>
					</div>
				{:else if isRunning}
					<div class="flex gap-2 text-text-muted">
						<span class="shrink-0 select-none">&middot;</span>
						<span class="animate-pulse">{activityLabel}</span>
					</div>
				{/if}
			{/each}

			{#if isRunning && (messages.length === 0 || messages[messages.length - 1].role === 'USER')}
				<div class="flex gap-2 text-text-muted">
					<span class="shrink-0 select-none">&middot;</span>
					<span class="animate-pulse">{activityLabel}</span>
				</div>
			{/if}
		{/if}
	</div>

	<!-- Error banner -->
	{#if sessionError || store.error}
		<div class="px-4 py-2 bg-danger/10 text-danger border-t border-danger/20">
			{sessionError || store.error}
		</div>
	{/if}

	<!-- Pending interaction approval -->
	{#if pendingInteraction && pendingInteraction.type !== 'ASK_USER' && pendingInteraction.type !== 'PERMISSION_REQUEST'}
		<div class={[
			'border-t p-3',
			pendingInteraction.type === 'EXIT_PLAN'
				? 'border-status-in-progress-text/20 bg-status-in-progress-bg/50'
				: 'border-warning/20 bg-warning/5'
		]}>
			<p class="text-xs font-mono text-text-muted mb-2">
				{#if pendingInteraction.type === 'EXIT_PLAN'}
					Agent wants to leave plan mode and start working.
				{:else}
					Agent wants to enter plan mode to analyze before making changes.
				{/if}
			</p>

			{#if renderedPlanContent}
				<div class="mb-3 max-h-48 overflow-y-auto rounded border border-border bg-surface p-3">
					<div class="agent-prose prose max-w-none text-text text-xs min-w-0">
						{@html renderedPlanContent}
					</div>
				</div>
			{/if}

			<div class="flex gap-2">
				<button
					onclick={approveInteraction}
					class={[
						'rounded px-3 py-1 text-xs font-mono transition-colors cursor-pointer',
						pendingInteraction.type === 'EXIT_PLAN'
							? 'bg-status-in-progress-text text-white hover:opacity-90'
							: 'bg-warning text-white hover:opacity-90'
					]}
				>
					Approve
				</button>
				{#if pendingInteraction.type === 'EXIT_PLAN'}
					<button
						onclick={approveInteractionWithYolo}
						class="rounded px-3 py-1 text-xs font-mono cursor-pointer bg-danger text-white hover:opacity-90 transition-colors"
					>
						Approve with YOLO
					</button>
				{/if}
				<button
					onclick={rejectInteraction}
					class="rounded px-3 py-1 text-xs font-mono cursor-pointer border border-border text-text-muted hover:bg-surface-alt transition-colors"
				>
					Reject
				</button>
			</div>
		</div>
	{/if}

	<!-- Permission request approval -->
	{#if pendingInteraction?.type === 'PERMISSION_REQUEST'}
		<div class="border-t border-warning/30 bg-warning/5 p-3">
			<p class="text-xs font-mono text-warning font-bold mb-1">
				Permission Request: {pendingInteraction.toolName ?? 'Unknown Tool'}
			</p>

			{#if formatToolInput(pendingInteraction.toolName, pendingInteraction.toolInput)}
				<pre class="mb-3 max-h-32 overflow-y-auto rounded border border-border bg-surface p-2 text-xs text-text-muted whitespace-pre-wrap break-all">{formatToolInput(pendingInteraction.toolName, pendingInteraction.toolInput)}</pre>
			{/if}

			<div class="flex gap-2">
				<button
					onclick={allowPermission}
					class="rounded px-3 py-1 text-xs font-mono cursor-pointer bg-status-in-progress-text text-white hover:opacity-90 transition-colors"
				>
					Allow
				</button>
				<button
					onclick={alwaysAllowPermission}
					class="rounded px-3 py-1 text-xs font-mono cursor-pointer bg-accent text-accent-text hover:opacity-90 transition-colors"
				>
					Always Allow
				</button>
				<button
					onclick={denyPermission}
					class="rounded px-3 py-1 text-xs font-mono cursor-pointer border border-border text-text-muted hover:bg-surface-alt transition-colors"
				>
					Deny
				</button>
			</div>
		</div>
	{/if}

	<!-- Ask user interaction — highlight that agent is waiting for a reply -->
	{#if pendingInteraction?.type === 'ASK_USER'}
		<div class="border-t border-accent/30 bg-accent/5 px-3 py-2">
			<p class="text-xs font-mono text-accent">Agent is waiting for your answer — type your reply below.</p>
		</div>
	{/if}

	<!-- Composer -->
	<div class="border-t border-border p-3 bg-surface">
		{#if isRunning}
			<div class="flex items-center gap-2 px-1 pb-2 text-text-muted">
				<span class="agent-spinner"></span>
				<span class="text-xs font-mono">{systemStatus ? `Agent is ${systemStatus}...` : 'Agent is working...'}</span>
			</div>
		{/if}
		<div class="flex gap-2 items-end">
			<textarea
				bind:value={inputText}
				onkeydown={handleKeydown}
				placeholder="Send a message..."
				rows={1}
				class="flex-1 resize-none rounded border border-border bg-surface-alt px-3 py-2 font-mono text-sm
					text-text placeholder:text-text-faint
					focus:outline-none focus:ring-2 focus:ring-accent/40 focus:border-accent"
			></textarea>

			<button
				onclick={send}
				disabled={!inputText.trim()}
				class="shrink-0 rounded px-3 py-2 text-sm font-mono
					bg-accent text-accent-text hover:bg-accent/90 transition-colors
					disabled:opacity-50 disabled:cursor-not-allowed"
			>
				Send
			</button>

			{#if isRunning}
				<button
					onclick={() => store.stop(beanId)}
					class="shrink-0 rounded px-3 py-2 text-sm font-mono
						bg-danger text-white hover:bg-danger/90 transition-colors"
				>
					Stop
				</button>
			{/if}
		</div>

		<!-- Mode toggle + Clear -->
		<div class="flex items-center gap-3 pt-2">
			<div class={["flex", isRunning && 'opacity-50 pointer-events-none']}>
				<button
					onclick={() => setAgentMode('plan')}
					disabled={isRunning}
					class={[
						'btn-tab-sm rounded-l',
						agentMode === 'plan'
							? 'border-warning/30 bg-warning/10 text-warning'
							: 'btn-tab-sm-inactive'
					]}
				>
					Plan
				</button>
				<button
					onclick={() => setAgentMode('act')}
					disabled={isRunning}
					class={[
						'btn-tab-sm border-l-0',
						agentMode === 'act'
							? 'border-status-in-progress-text/30 bg-status-in-progress-bg text-status-in-progress-text'
							: 'btn-tab-sm-inactive'
					]}
				>
					Act
				</button>
				<button
					onclick={() => setAgentMode('yolo')}
					disabled={isRunning}
					class={[
						'btn-tab-sm rounded-r border-l-0',
						agentMode === 'yolo'
							? 'border-danger/30 bg-danger/10 text-danger'
							: 'btn-tab-sm-inactive'
					]}
				>
					YOLO
				</button>
			</div>

			<div class={["flex", (isRunning || messages.length === 0) && 'opacity-30 pointer-events-none']}>
				<button
					onclick={() => store.sendMessage(beanId, '/compact')}
					disabled={isRunning || messages.length === 0}
					class="btn-tab-sm rounded-l btn-tab-sm-inactive"
				>
					Compact
				</button>
				<button
					onclick={() => store.clearSession(beanId)}
					disabled={isRunning || messages.length === 0}
					class="btn-tab-sm rounded-r border-l-0 btn-tab-sm-inactive"
				>
					Clear
				</button>
			</div>
		</div>
	</div>
</div>

<style>
	.agent-spinner {
		display: inline-block;
		width: 12px;
		height: 12px;
		border: 2px solid currentColor;
		border-right-color: transparent;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	/* Ensure rendered markdown inherits monospace and uniform font size,
	   but exclude code blocks so Shiki highlighting renders properly */
	.agent-prose :global(*:not(pre, pre *, code)) {
		font-family: inherit;
		font-size: inherit;
	}

	.agent-prose :global(h1),
	.agent-prose :global(h2),
	.agent-prose :global(h3),
	.agent-prose :global(h4),
	.agent-prose :global(h5),
	.agent-prose :global(h6) {
		font-size: inherit;
		font-weight: bold;
	}
</style>
