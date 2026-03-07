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

	// Form fields
	let title = $state(bean?.title ?? '');
	let type = $state(bean?.type ?? 'task');
	let status = $state(bean?.status ?? 'todo');
	let priority = $state(bean?.priority ?? 'normal');
	let tags = $state(bean?.tags.join(', ') ?? '');
	let body = $state(bean?.body ?? '');
	let parentId = $state(bean?.parentId ?? '');

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

<dialog class="modal modal-open">
	<div class="modal-box w-11/12 max-w-2xl">
		<h3 class="text-lg font-bold">{isEdit ? 'Edit Bean' : 'New Bean'}</h3>

		{#if error}
			<div role="alert" class="alert alert-error mt-4">
				<span>{error}</span>
			</div>
		{/if}

		<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="mt-4 space-y-4">
			<!-- Title -->
			<div class="form-control">
				<label class="label" for="bean-title">Title</label>
				<input
					id="bean-title"
					type="text"
					class="input input-bordered w-full"
					bind:value={title}
					placeholder="What needs to be done?"
				/>
			</div>

			<!-- Type / Status / Priority row -->
			<div class="grid grid-cols-3 gap-3">
				<div class="form-control">
					<label class="label" for="bean-type">Type</label>
					<select id="bean-type" class="select select-bordered w-full" bind:value={type}>
						{#each types as t}
							<option value={t}>{t}</option>
						{/each}
					</select>
				</div>

				<div class="form-control">
					<label class="label" for="bean-status">Status</label>
					<select id="bean-status" class="select select-bordered w-full" bind:value={status}>
						{#each statuses as s}
							<option value={s}>{s}</option>
						{/each}
					</select>
				</div>

				<div class="form-control">
					<label class="label" for="bean-priority">Priority</label>
					<select
						id="bean-priority"
						class="select select-bordered w-full"
						bind:value={priority}
					>
						{#each priorities as p}
							<option value={p}>{p}</option>
						{/each}
					</select>
				</div>
			</div>

			<!-- Parent -->
			<div class="form-control">
				<label class="label" for="bean-parent">Parent</label>
				<select id="bean-parent" class="select select-bordered w-full" bind:value={parentId}>
					<option value="">None</option>
					{#each availableParents as p}
						<option value={p.id}>{p.title} ({p.type})</option>
					{/each}
				</select>
			</div>

			<!-- Tags -->
			<div class="form-control">
				<label class="label" for="bean-tags">Tags</label>
				<input
					id="bean-tags"
					type="text"
					class="input input-bordered w-full"
					bind:value={tags}
					placeholder="Comma-separated tags"
				/>
			</div>

			<!-- Body -->
			<div class="form-control">
				<label class="label" for="bean-body">Description (Markdown)</label>
				<textarea
					id="bean-body"
					class="textarea textarea-bordered w-full h-40 font-mono text-sm"
					bind:value={body}
					placeholder="Markdown content..."
				></textarea>
			</div>

			<!-- Actions -->
			<div class="modal-action">
				<button type="button" class="btn" onclick={onClose} disabled={submitting}>Cancel</button>
				<button type="submit" class="btn btn-primary" disabled={submitting || !title.trim()}>
					{#if submitting}
						<span class="loading loading-spinner loading-sm"></span>
					{/if}
					{isEdit ? 'Save Changes' : 'Create Bean'}
				</button>
			</div>
		</form>
	</div>
	<form method="dialog" class="modal-backdrop">
		<button onclick={onClose}>close</button>
	</form>
</dialog>
