<script lang="ts">
	import type { Bean } from '$lib/beans.svelte';
	import { beansStore } from '$lib/beans.svelte';
	import { gql } from 'urql';
	import { client } from '$lib/graphqlClient';

	interface Props {
		bean?: Bean | null;
		onClose: () => void;
		onSaved?: (bean: Bean) => void;
	}

	let { bean = null, onClose, onSaved }: Props = $props();

	const isEdit = $derived(!!bean);

	// Form fields — intentionally capture initial prop values for local editing
	/* eslint-disable svelte/valid-compile */
	// svelte-ignore state_referenced_locally
	let title = $state(bean?.title ?? '');
	// svelte-ignore state_referenced_locally
	let type = $state(bean?.type ?? 'task');
	// svelte-ignore state_referenced_locally
	let status = $state(bean?.status ?? 'todo');
	// svelte-ignore state_referenced_locally
	let priority = $state(bean?.priority ?? 'normal');
	// svelte-ignore state_referenced_locally
	let tags = $state(bean?.tags.join(', ') ?? '');
	// svelte-ignore state_referenced_locally
	let body = $state(bean?.body ?? '');
	// svelte-ignore state_referenced_locally
	let parentId = $state(bean?.parentId ?? '');
	/* eslint-enable svelte/valid-compile */

	let submitting = $state(false);
	let error = $state<string | null>(null);

	const types = ['task', 'bug', 'feature', 'epic', 'milestone'];
	const statuses = ['draft', 'todo', 'in-progress', 'completed', 'scrapped'];
	const priorities = ['critical', 'high', 'normal', 'low', 'deferred'];

	// Available parents (all beans except current bean and its descendants)
	const availableParents = $derived(
		beansStore.all.filter((b) => {
			if (!bean) return true;
			if (b.id === bean.id) return false;
			// Simple cycle check: don't allow own children as parent
			let current: Bean | undefined = b;
			while (current) {
				if (current.parentId === bean.id) return false;
				current = current.parentId ? beansStore.get(current.parentId) : undefined;
			}
			return true;
		})
	);

	const CREATE_BEAN = gql`
		mutation CreateBean($input: CreateBeanInput!) {
			createBean(input: $input) {
				id
				title
				status
				type
				priority
				tags
				body
				parentId
				blockingIds
				slug
				path
				createdAt
				updatedAt
			}
		}
	`;

	const UPDATE_BEAN = gql`
		mutation UpdateBean($id: ID!, $input: UpdateBeanInput!) {
			updateBean(id: $id, input: $input) {
				id
				title
				status
				type
				priority
				tags
				body
				parentId
				blockingIds
				slug
				path
				createdAt
				updatedAt
			}
		}
	`;

	function parseTags(raw: string): string[] {
		return raw
			.split(',')
			.map((t) => t.trim())
			.filter(Boolean);
	}

	async function handleSubmit() {
		if (!title.trim()) {
			error = 'Title is required';
			return;
		}

		submitting = true;
		error = null;

		const input: Record<string, unknown> = {
			title: title.trim(),
			type,
			status,
			priority,
			body: body || null,
			tags: parseTags(tags),
			parent: parentId || null
		};

		let result;
		if (isEdit && bean) {
			result = await client.mutation(UPDATE_BEAN, { id: bean.id, input }).toPromise();
		} else {
			result = await client.mutation(CREATE_BEAN, { input }).toPromise();
		}

		submitting = false;

		if (result.error) {
			error = result.error.message;
			return;
		}

		const saved = result.data?.createBean ?? result.data?.updateBean;
		if (saved) {
			onSaved?.(saved);
		}
		onClose();
	}
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<dialog class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 open" open>
	<div class="bg-surface rounded-xl shadow-xl w-11/12 max-w-2xl p-6">
		<h3 class="text-lg font-bold text-text">{isEdit ? 'Edit Bean' : 'New Bean'}</h3>

		{#if error}
			<div class="mt-4 rounded-lg border border-danger/30 bg-danger/10 text-danger px-4 py-3 text-sm">
				{error}
			</div>
		{/if}

		<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="mt-4 space-y-4">
			<!-- Title -->
			<div>
				<label class="block text-sm font-medium text-text-muted mb-1" for="bean-title">Title</label>
				<input
					id="bean-title"
					type="text"
					class="w-full px-3 py-2 rounded-md border border-border bg-surface text-text text-sm focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent"
					bind:value={title}
					placeholder="What needs to be done?"
				/>
			</div>

			<!-- Type / Status / Priority row -->
			<div class="grid grid-cols-3 gap-3">
				<div>
					<label class="block text-sm font-medium text-text-muted mb-1" for="bean-type">Type</label>
					<select id="bean-type" class="w-full px-3 py-2 rounded-md border border-border bg-surface text-text text-sm focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent" bind:value={type}>
						{#each types as t}
							<option value={t}>{t}</option>
						{/each}
					</select>
				</div>

				<div>
					<label class="block text-sm font-medium text-text-muted mb-1" for="bean-status">Status</label>
					<select id="bean-status" class="w-full px-3 py-2 rounded-md border border-border bg-surface text-text text-sm focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent" bind:value={status}>
						{#each statuses as s}
							<option value={s}>{s}</option>
						{/each}
					</select>
				</div>

				<div>
					<label class="block text-sm font-medium text-text-muted mb-1" for="bean-priority">Priority</label>
					<select
						id="bean-priority"
						class="w-full px-3 py-2 rounded-md border border-border bg-surface text-text text-sm focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent"
						bind:value={priority}
					>
						{#each priorities as p}
							<option value={p}>{p}</option>
						{/each}
					</select>
				</div>
			</div>

			<!-- Parent -->
			<div>
				<label class="block text-sm font-medium text-text-muted mb-1" for="bean-parent">Parent</label>
				<select id="bean-parent" class="w-full px-3 py-2 rounded-md border border-border bg-surface text-text text-sm focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent" bind:value={parentId}>
					<option value="">None</option>
					{#each availableParents as p}
						<option value={p.id}>{p.title} ({p.type})</option>
					{/each}
				</select>
			</div>

			<!-- Tags -->
			<div>
				<label class="block text-sm font-medium text-text-muted mb-1" for="bean-tags">Tags</label>
				<input
					id="bean-tags"
					type="text"
					class="w-full px-3 py-2 rounded-md border border-border bg-surface text-text text-sm focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent"
					bind:value={tags}
					placeholder="Comma-separated tags"
				/>
			</div>

			<!-- Body -->
			<div>
				<label class="block text-sm font-medium text-text-muted mb-1" for="bean-body">Description (Markdown)</label>
				<textarea
					id="bean-body"
					class="w-full px-3 py-2 rounded-md border border-border bg-surface text-text text-sm font-mono h-40 focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent resize-y"
					bind:value={body}
					placeholder="Markdown content..."
				></textarea>
			</div>

			<!-- Actions -->
			<div class="flex justify-end gap-2 pt-2">
				<button type="button" class="px-4 py-2 text-sm font-medium rounded-md border border-border text-text-muted hover:bg-surface-alt transition-colors" onclick={onClose} disabled={submitting}>Cancel</button>
				<button type="submit" class="px-4 py-2 text-sm font-medium rounded-md bg-accent text-accent-text hover:opacity-90 transition-opacity disabled:opacity-50 flex items-center gap-2" disabled={submitting || !title.trim()}>
					{#if submitting}
						<span class="inline-block w-4 h-4 border-2 border-accent-text/30 border-t-accent-text rounded-full animate-spin"></span>
					{/if}
					{isEdit ? 'Save Changes' : 'Create Bean'}
				</button>
			</div>
		</form>
	</div>
	<!-- Backdrop -->
	<button class="fixed inset-0 -z-10" onclick={onClose} tabindex="-1" aria-label="Close"></button>
</dialog>
