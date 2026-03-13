import { gql } from 'urql';
import { pipe, subscribe } from 'wonka';
import { client } from './graphqlClient';
import { generateWorkspaceName } from '$lib/nameGenerator';

export const MAIN_WORKSPACE_ID = '__central__';

export interface Worktree {
  id: string;
  name: string | null;
  description: string | null;
  branch: string;
  path: string;
  setupStatus: 'RUNNING' | 'DONE' | 'FAILED' | null;
  setupError: string | null;
}

const WORKTREE_FIELDS = `
  id
  name
  description
  branch
  path
  setupStatus
  setupError
`;

const WORKTREES_SUBSCRIPTION = gql`
  subscription WorktreesChanged {
    worktreesChanged {
      ${WORKTREE_FIELDS}
    }
  }
`;

const CREATE_WORKTREE = gql`
  mutation CreateWorktree($name: String!) {
    createWorktree(name: $name) {
      ${WORKTREE_FIELDS}
    }
  }
`;

const REMOVE_WORKTREE = gql`
  mutation RemoveWorktree($id: ID!) {
    removeWorktree(id: $id)
  }
`;

export interface WorktreeStatus {
  hasChanges: boolean;
  hasUnmergedCommits: boolean;
}

const WORKTREES_QUERY = gql`
  query Worktrees {
    worktrees {
      id
      hasChanges
      hasUnmergedCommits
    }
  }
`;

class WorktreeStore {
  worktrees = $state<Worktree[]>([]);
  initialized = $state(false);
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
          this.initialized = true;
          return;
        }

        const wts = result.data?.worktreesChanged;
        if (wts) {
          this.worktrees = wts;
          this.initialized = true;
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

  async createWorktree(): Promise<Worktree | null> {
    this.loading = true;
    this.error = null;

    const name = generateWorkspaceName();
    const result = await client.mutation(CREATE_WORKTREE, { name }).toPromise();

    this.loading = false;

    if (result.error) {
      this.error = result.error.message;
      return null;
    }

    const wt = result.data?.createWorktree ?? null;

    // Eagerly add to local state so the layout guard doesn't redirect
    // back to planning before the subscription delivers the update.
    if (wt && !this.worktrees.some((w) => w.id === wt.id)) {
      this.worktrees = [...this.worktrees, wt];
    }

    return wt;
  }

  async removeWorktree(id: string): Promise<boolean> {
    this.loading = true;
    this.error = null;

    // Eagerly remove from local state so the sidebar updates immediately
    // without waiting for the subscription to deliver the new list.
    const previous = this.worktrees;
    this.worktrees = this.worktrees.filter((wt) => wt.id !== id);

    const result = await client.mutation(REMOVE_WORKTREE, { id }).toPromise();

    this.loading = false;

    if (result.error) {
      // Restore on failure so the item reappears
      this.worktrees = previous;
      this.error = result.error.message;
      return false;
    }

    return true;
  }

  hasWorktree(id: string): boolean {
    return this.worktrees.some((wt) => wt.id === id);
  }

  /** Fetch fresh git status for a specific worktree (on-demand, not cached). */
  async getWorktreeStatus(id: string): Promise<WorktreeStatus | null> {
    const result = await client.query(WORKTREES_QUERY, {}, { requestPolicy: 'network-only' }).toPromise();
    if (result.error) return null;
    const wt = result.data?.worktrees?.find((w: { id: string }) => w.id === id);
    return wt ? { hasChanges: wt.hasChanges, hasUnmergedCommits: wt.hasUnmergedCommits } : null;
  }
}

export const worktreeStore = new WorktreeStore();
