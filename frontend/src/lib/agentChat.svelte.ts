import { gql } from 'urql';
import { pipe, subscribe } from 'wonka';
import { client } from './graphqlClient';

export interface AgentMessage {
	role: 'USER' | 'ASSISTANT' | 'TOOL';
	content: string;
}

export interface AgentSession {
	beanId: string;
	agentType: string;
	status: 'IDLE' | 'RUNNING' | 'ERROR';
	messages: AgentMessage[];
	error: string | null;
}

const AGENT_SESSION_SUBSCRIPTION = gql`
	subscription AgentSessionChanged($beanId: ID!) {
		agentSessionChanged(beanId: $beanId) {
			beanId
			agentType
			status
			messages {
				role
				content
			}
			error
		}
	}
`;

const SEND_AGENT_MESSAGE = gql`
	mutation SendAgentMessage($beanId: ID!, $message: String!) {
		sendAgentMessage(beanId: $beanId, message: $message)
	}
`;

const STOP_AGENT = gql`
	mutation StopAgent($beanId: ID!) {
		stopAgent(beanId: $beanId)
	}
`;

export class AgentChatStore {
	session = $state<AgentSession | null>(null);
	sending = $state(false);
	error = $state<string | null>(null);

	#beanId: string | null = null;
	#unsubscribe: (() => void) | null = null;

	subscribe(beanId: string): void {
		// If already subscribed to the same bean, skip
		if (this.#unsubscribe && this.#beanId === beanId) return;

		// Clean up previous subscription
		this.unsubscribe();
		this.#beanId = beanId;

		const { unsubscribe } = pipe(
			client.subscription(AGENT_SESSION_SUBSCRIPTION, { beanId }),
			subscribe(
				(result: { data?: { agentSessionChanged?: AgentSession }; error?: Error }) => {
					if (result.error) {
						console.error('Agent session subscription error:', result.error);
						this.error = result.error.message;
						return;
					}

					const session = result.data?.agentSessionChanged;
					if (session) {
						this.session = session;
						this.error = null;
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
		this.#beanId = null;
	}

	async sendMessage(beanId: string, message: string): Promise<boolean> {
		this.sending = true;
		this.error = null;

		const result = await client
			.mutation(SEND_AGENT_MESSAGE, { beanId, message })
			.toPromise();

		this.sending = false;

		if (result.error) {
			this.error = result.error.message;
			return false;
		}

		return true;
	}

	async stop(beanId: string): Promise<boolean> {
		const result = await client.mutation(STOP_AGENT, { beanId }).toPromise();

		if (result.error) {
			this.error = result.error.message;
			return false;
		}

		return true;
	}
}
