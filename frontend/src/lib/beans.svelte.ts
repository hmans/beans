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
type ChangeType = 'INITIAL' | 'INITIAL_SYNC_COMPLETE' | 'CREATED' | 'UPDATED' | 'DELETED';

/**
 * Bean change event from GraphQL subscription
 */
interface BeanChangeEvent {
	type: ChangeType;
	beanId: string;
	bean: Bean | null;
}

/**
 * GraphQL subscription for bean changes with initial state support
 */
const BEAN_CHANGED_SUBSCRIPTION = gql`
	subscription BeanChanged($includeInitial: Boolean!) {
		beanChanged(includeInitial: $includeInitial) {
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

	/** Loading state (true until first non-initial event or subscription fully synced) */
	loading = $state(true);

	/** Error state */
	error = $state<string | null>(null);

	/** Whether subscription is connected */
	connected = $state(false);

	/** Whether initial sync is complete */
	#initialSyncDone = false;

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
	 * Start subscription to bean changes with initial state.
	 * This is the primary method to initialize the store - it subscribes to changes
	 * and receives all current beans as initial events, eliminating race conditions.
	 */
	subscribe(): void {
		if (this.#unsubscribe) {
			return; // Already subscribed
		}

		this.loading = true;
		this.error = null;
		this.#initialSyncDone = false;

		const { unsubscribe } = pipe(
			client.subscription(BEAN_CHANGED_SUBSCRIPTION, { includeInitial: true }),
			subscribe((result: { data?: { beanChanged?: BeanChangeEvent }; error?: Error }) => {
				if (result.error) {
					console.error('Subscription error:', result.error);
					this.connected = false;
					this.error = result.error.message;
					this.loading = false;
					return;
				}

				this.connected = true;

				const event = result.data?.beanChanged as BeanChangeEvent | undefined;
				if (!event) return;

				switch (event.type) {
					case 'INITIAL':
						if (event.bean) {
							this.beans.set(event.bean.id, event.bean);
						}
						break;
					case 'INITIAL_SYNC_COMPLETE':
						// Server explicitly signals initial sync is done
						this.#initialSyncDone = true;
						this.loading = false;
						break;
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
