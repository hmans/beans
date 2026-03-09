<script lang="ts">
	interface Props {
		title: string;
		message: string;
		confirmLabel?: string;
		cancelLabel?: string;
		danger?: boolean;
		onConfirm: () => void;
		onCancel: () => void;
	}

	let {
		title,
		message,
		confirmLabel = 'Confirm',
		cancelLabel = 'Cancel',
		danger = false,
		onConfirm,
		onCancel
	}: Props = $props();

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onCancel();
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) onCancel();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
	onclick={handleBackdropClick}
>
	<div class="bg-surface rounded-lg shadow-xl border border-border w-full max-w-sm mx-4 p-5">
		<h2 class="text-base font-semibold text-text mb-2">{title}</h2>
		<p class="text-sm text-text-muted mb-5">{message}</p>
		<div class="flex justify-end gap-2">
			<button
				class="px-3 py-1.5 text-sm font-medium rounded-md border border-border text-text-muted hover:bg-surface-alt transition-colors"
				onclick={onCancel}
			>
				{cancelLabel}
			</button>
			<button
				class="px-3 py-1.5 text-sm font-medium rounded-md transition-colors
					{danger
					? 'bg-danger text-white hover:opacity-90'
					: 'bg-accent text-accent-text hover:opacity-90'}"
				onclick={onConfirm}
			>
				{confirmLabel}
			</button>
		</div>
	</div>
</div>
