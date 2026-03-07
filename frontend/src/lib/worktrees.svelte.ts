import { gql } from 'urql';
import { pipe, subscribe } from 'wonka';
import { client } from './graphqlClient';

export interface Worktree {
	beanId: string;
	branch: string;
	path: string;
}

const WORKTREES_SUBSCRIPTION = gql`
	subscription WorktreesChanged {
		worktreesChanged {
			beanId
			branch
			path
		}
	}
`;

const CREATE_WORKTREE = gql`
	mutation CreateWorktree($beanId: ID!) {
		createWorktree(beanId: $beanId) {
			beanId
			branch
			path
		}
	}
`;

class WorktreeStore {
	worktrees = $state<Worktree[]>([]);
	loading = $state(false);
	error = $state<string | null>(null);

	#unsubscribe: (() => void) | null = null;

	subscribe(): void {
		if (this.#unsubscribe) return;

		const { unsubscribe } = pipe(
			client.subscription(WORKTREES_SUBSCRIPTION, {}),
			subscribe((result: { data?: { worktreesChanged?: Worktree[] }; error?: Error }) => {
				if (result.error) {
					console.error('Worktree subscription error:', result.error);
					this.error = result.error.message;
					return;
				}

				const wts = result.data?.worktreesChanged;
				if (wts) {
					this.worktrees = wts;
				}
			})
		);

		this.#unsubscribe = unsubscribe;
	}

	unsubscribe(): void {
		if (this.#unsubscribe) {
			this.#unsubscribe();
			this.#unsubscribe = null;
		}
	}

	async createWorktree(beanId: string): Promise<boolean> {
		this.loading = true;
		this.error = null;

		const result = await client.mutation(CREATE_WORKTREE, { beanId }).toPromise();

		this.loading = false;

		if (result.error) {
			this.error = result.error.message;
			return false;
		}

		return true;
	}

	hasWorktree(beanId: string): boolean {
		return this.worktrees.some((wt) => wt.beanId === beanId);
	}
}

export const worktreeStore = new WorktreeStore();
