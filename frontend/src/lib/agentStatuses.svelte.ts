import { gql } from 'urql';
import { pipe, subscribe } from 'wonka';
import { client } from './graphqlClient';

interface ActiveAgentStatus {
	beanId: string;
	status: 'IDLE' | 'RUNNING' | 'ERROR';
}

const ACTIVE_AGENT_STATUSES_SUBSCRIPTION = gql`
	subscription ActiveAgentStatuses {
		activeAgentStatuses {
			beanId
			status
		}
	}
`;

class AgentStatusesStore {
	runningBeanIds = $state<Set<string>>(new Set());

	#unsubscribe: (() => void) | null = null;

	subscribe(): void {
		if (this.#unsubscribe) return;

		const { unsubscribe } = pipe(
			client.subscription(ACTIVE_AGENT_STATUSES_SUBSCRIPTION, {}),
			subscribe(
				(result: {
					data?: { activeAgentStatuses?: ActiveAgentStatus[] };
					error?: Error;
				}) => {
					if (result.error) {
						console.error('Agent statuses subscription error:', result.error);
						return;
					}

					const statuses = result.data?.activeAgentStatuses;
					if (statuses) {
						this.runningBeanIds = new Set(
							statuses
								.filter((s) => s.status === 'RUNNING')
								.map((s) => s.beanId)
						);
					}
				}
			)
		);

		this.#unsubscribe = unsubscribe;
	}

	unsubscribe(): void {
		if (this.#unsubscribe) {
			this.#unsubscribe();
			this.#unsubscribe = null;
		}
	}

	isRunning(beanId: string): boolean {
		return this.runningBeanIds.has(beanId);
	}
}

export const agentStatusesStore = new AgentStatusesStore();
