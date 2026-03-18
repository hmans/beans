import { client } from '$lib/graphqlClient';
import { FileDiffDocument, AllFileDiffDocument } from '$lib/graphql/generated';

export interface DiffSelection {
	path: string;
	staged: boolean;
}

export interface DiffLine {
	type: 'add' | 'del' | 'hunk' | 'context' | 'header';
	content: string;
	oldNum: number | null;
	newNum: number | null;
}

function parseDiff(raw: string): DiffLine[] {
	if (!raw) return [];
	const lines: DiffLine[] = [];
	let oldNum = 0;
	let newNum = 0;

	for (const line of raw.split('\n')) {
		if (
			line.startsWith('diff --git') ||
			line.startsWith('index ') ||
			line.startsWith('--- ') ||
			line.startsWith('+++ ')
		) {
			continue;
		}
		if (line.startsWith('@@')) {
			const match = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@(.*)/);
			if (match) {
				oldNum = parseInt(match[1]);
				newNum = parseInt(match[2]);
				lines.push({ type: 'hunk', content: match[3]?.trim() || '', oldNum: null, newNum: null });
			}
			continue;
		}
		if (line.startsWith('+')) {
			lines.push({ type: 'add', content: line.slice(1), oldNum: null, newNum: newNum++ });
		} else if (line.startsWith('-')) {
			lines.push({ type: 'del', content: line.slice(1), oldNum: oldNum++, newNum: null });
		} else {
			lines.push({
				type: 'context',
				content: line.startsWith(' ') ? line.slice(1) : line,
				oldNum: oldNum++,
				newNum: newNum++
			});
		}
	}
	return lines;
}

class DiffSelectionStore {
	selected = $state<DiffSelection | null>(null);
	diffContent = $state('');
	diffLoading = $state(false);
	parsedLines = $derived(parseDiff(this.diffContent));

	#currentTab: 'all' | 'unstaged' = 'all';
	#worktreePath: string | undefined;

	setContext(tab: 'all' | 'unstaged', worktreePath: string | undefined) {
		this.#currentTab = tab;
		this.#worktreePath = worktreePath;
	}

	toggle(path: string, staged: boolean) {
		if (this.selected?.path === path && this.selected?.staged === staged) {
			this.clear();
			return;
		}
		this.selected = { path, staged };
		if (this.#currentTab === 'all') {
			this.#fetchAllDiff(path);
		} else {
			this.#fetchDiff(path, staged);
		}
	}

	clear() {
		this.selected = null;
		this.diffContent = '';
	}

	async #fetchDiff(filePath: string, staged: boolean) {
		this.diffLoading = true;
		const result = await client
			.query(FileDiffDocument, { filePath, staged, path: this.#worktreePath ?? null })
			.toPromise();

		if (this.selected?.path !== filePath || this.selected?.staged !== staged) return;

		if (result.error) {
			console.error('Failed to fetch diff:', result.error);
			this.diffContent = '';
		} else {
			this.diffContent = result.data?.fileDiff ?? '';
		}
		this.diffLoading = false;
	}

	async #fetchAllDiff(filePath: string) {
		this.diffLoading = true;
		const result = await client
			.query(AllFileDiffDocument, { filePath, path: this.#worktreePath ?? null })
			.toPromise();

		if (this.selected?.path !== filePath) return;

		if (result.error) {
			console.error('Failed to fetch all diff:', result.error);
			this.diffContent = '';
		} else {
			this.diffContent = result.data?.allFileDiff ?? '';
		}
		this.diffLoading = false;
	}
}

export const diffSelectionStore = new DiffSelectionStore();
