import { gql } from 'urql';
import { SvelteMap } from 'svelte/reactivity';
import { client } from './graphqlClient';

/**
 * Bean type matching the GraphQL schema
 */
export interface Bean {
	id: string;
	slug: string | null;
	path: string;
	title: string;
	status: string;
	type: string;
	priority: string;
	tags: string[];
	createdAt: string;
	updatedAt: string;
	body: string;
	parentId: string | null;
	blockingIds: string[];
}

/**
 * GraphQL query to fetch all beans
 */
const BEANS_QUERY = gql`
	query GetBeans {
		beans {
			id
			slug
			path
			title
			status
			type
			priority
			tags
			createdAt
			updatedAt
			body
			parentId
			blockingIds
		}
	}
`;

/**
 * Svelte 5 runes-style stateful store for beans.
 * Frontend equivalent of beancore on the backend.
 */
export class BeansStore {
	/** All beans indexed by ID */
	beans = $state(new SvelteMap<string, Bean>());

	/** Loading state */
	loading = $state(false);

	/** Error state */
	error = $state<string | null>(null);

	/** All beans as an array (derived) */
	get all(): Bean[] {
		return Array.from(this.beans.values());
	}

	/** Count of beans */
	get count(): number {
		return this.beans.size;
	}

	/**
	 * Load all beans from the GraphQL API
	 */
	async load(): Promise<void> {
		this.loading = true;
		this.error = null;

		try {
			const result = await client.query(BEANS_QUERY, {}).toPromise();

			if (result.error) {
				this.error = result.error.message;
				return;
			}

			if (result.data?.beans) {
				// Clear and repopulate the map
				this.beans.clear();
				for (const bean of result.data.beans as Bean[]) {
					this.beans.set(bean.id, bean);
				}
			}
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Unknown error';
		} finally {
			this.loading = false;
		}
	}

	/**
	 * Get a bean by ID
	 */
	get(id: string): Bean | undefined {
		return this.beans.get(id);
	}

	/**
	 * Get beans filtered by status
	 */
	byStatus(status: string): Bean[] {
		return this.all.filter((b) => b.status === status);
	}

	/**
	 * Get beans filtered by type
	 */
	byType(type: string): Bean[] {
		return this.all.filter((b) => b.type === type);
	}

	/**
	 * Get children of a bean (beans with this bean as parent)
	 */
	children(parentId: string): Bean[] {
		return this.all.filter((b) => b.parentId === parentId);
	}

	/**
	 * Get beans that are blocking a given bean
	 */
	blockedBy(beanId: string): Bean[] {
		return this.all.filter((b) => b.blockingIds.includes(beanId));
	}
}

/**
 * Singleton instance of the beans store
 */
export const beansStore = new BeansStore();
