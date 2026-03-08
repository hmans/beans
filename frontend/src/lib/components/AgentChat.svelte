<script lang="ts">
	import { AgentChatStore } from '$lib/agentChat.svelte';
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

	// Render markdown for assistant messages
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
		if (!text || isRunning) return;

		inputText = '';
		await store.sendMessage(beanId, text);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			send();
		}
	}
</script>

<div class="flex flex-col h-full">
	<!-- Messages area -->
	<div
		bind:this={messagesEl}
		class="flex-1 overflow-y-auto p-4 space-y-4"
	>
		{#if messages.length === 0}
			<div class="flex items-center justify-center h-full text-text-faint">
				<p class="text-sm">Send a message to start a conversation with the agent.</p>
			</div>
		{:else}
			{#each messages as msg, i}
				{#if msg.role === 'USER'}
					<div class="flex justify-end">
						<div class="max-w-[80%] rounded-lg px-4 py-2 bg-accent text-accent-text">
							<p class="text-sm whitespace-pre-wrap">{msg.content}</p>
						</div>
					</div>
				{:else if getRenderedContent(i)}
					<div class="flex justify-start">
						<div class="max-w-[80%] rounded-lg px-4 py-2 bg-surface-dim prose prose-sm max-w-none text-text">
							{@html getRenderedContent(i)}
						</div>
					</div>
				{:else if msg.content}
					<div class="flex justify-start">
						<div class="max-w-[80%] rounded-lg px-4 py-2 bg-surface-dim">
							<p class="text-sm whitespace-pre-wrap text-text">{msg.content}</p>
						</div>
					</div>
				{:else if isRunning}
					<div class="flex justify-start">
						<div class="rounded-lg px-4 py-2 bg-surface-dim">
							<div class="flex items-center gap-2 text-text-muted">
								<span class="inline-block w-1.5 h-1.5 rounded-full bg-accent animate-pulse"></span>
								<span class="text-sm">Thinking...</span>
							</div>
						</div>
					</div>
				{/if}
			{/each}

			{#if isRunning && (messages.length === 0 || messages[messages.length - 1].role === 'USER')}
				<div class="flex justify-start">
					<div class="rounded-lg px-4 py-2 bg-surface-dim">
						<div class="flex items-center gap-2 text-text-muted">
							<span class="inline-block w-1.5 h-1.5 rounded-full bg-accent animate-pulse"></span>
							<span class="text-sm">Thinking...</span>
						</div>
					</div>
				</div>
			{/if}
		{/if}
	</div>

	<!-- Error banner -->
	{#if sessionError || store.error}
		<div class="px-4 py-2 bg-danger/10 text-danger text-sm border-t border-danger/20">
			{sessionError || store.error}
		</div>
	{/if}

	<!-- Composer -->
	<div class="border-t border-border p-3 bg-surface">
		<div class="flex gap-2 items-end">
			<textarea
				bind:value={inputText}
				onkeydown={handleKeydown}
				placeholder={isRunning ? 'Agent is working...' : 'Send a message...'}
				disabled={isRunning}
				rows={1}
				class="flex-1 resize-none rounded-lg border border-border bg-surface-alt px-3 py-2 text-sm
					text-text placeholder:text-text-faint
					focus:outline-none focus:ring-2 focus:ring-accent/40 focus:border-accent
					disabled:opacity-50 disabled:cursor-not-allowed"
			></textarea>

			{#if isRunning}
				<button
					onclick={() => store.stop(beanId)}
					class="shrink-0 rounded-lg px-4 py-2 text-sm font-medium
						bg-danger text-white hover:bg-danger/90 transition-colors"
				>
					Stop
				</button>
			{:else}
				<button
					onclick={send}
					disabled={!inputText.trim()}
					class="shrink-0 rounded-lg px-4 py-2 text-sm font-medium
						bg-accent text-accent-text hover:bg-accent/90 transition-colors
						disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Send
				</button>
			{/if}
		</div>
	</div>
</div>
