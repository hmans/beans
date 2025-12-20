import { gql, type SubscriptionHandler } from 'urql';
import { pipe, subscribe } from 'wonka';
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
 * Change type from GraphQL subscription
 */
type ChangeType = 'CREATED' | 'UPDATED' | 'DELETED';

/**
 * Bean change event from GraphQL subscription
 */
interface BeanChangeEvent {
	type: ChangeType;
	beanId: string;
	bean: Bean | null;
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
 * GraphQL subscription for bean changes
 */
const BEAN_CHANGED_SUBSCRIPTION = gql`
	subscription BeanChanged {
		beanChanged {
			type
			beanId
			bean {
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

	/** Whether subscription is connected */
	connected = $state(false);

	/** Subscription teardown function */
	#unsubscribe: (() => void) | null = null;

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
	 * Start subscription to bean changes.
	 * Call this after load() to receive real-time updates.
	 */
	subscribe(): void {
		if (this.#unsubscribe) {
			return; // Already subscribed
		}

		const { unsubscribe } = pipe(
			client.subscription(BEAN_CHANGED_SUBSCRIPTION, {}),
			subscribe((result: { data?: { beanChanged?: BeanChangeEvent }; error?: Error }) => {
				if (result.error) {
					console.error('Subscription error:', result.error);
					this.connected = false;
					return;
				}

				this.connected = true;

				const event = result.data?.beanChanged as BeanChangeEvent | undefined;
				if (!event) return;

				switch (event.type) {
					case 'CREATED':
					case 'UPDATED':
						if (event.bean) {
							this.beans.set(event.bean.id, event.bean);
						}
						break;
					case 'DELETED':
						this.beans.delete(event.beanId);
						break;
				}
			})
		);

		this.#unsubscribe = unsubscribe;
	}

	/**
	 * Stop subscription to bean changes.
	 */
	unsubscribe(): void {
		if (this.#unsubscribe) {
			this.#unsubscribe();
			this.#unsubscribe = null;
			this.connected = false;
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
