import { gql } from 'urql';
import { pipe, subscribe } from 'wonka';
import { client } from './graphqlClient';

export interface AgentMessage {
	role: 'USER' | 'ASSISTANT' | 'TOOL';
	content: string;
}

export type InteractionType = 'EXIT_PLAN' | 'ENTER_PLAN' | 'ASK_USER' | 'PERMISSION_REQUEST';

export interface PendingInteraction {
	type: InteractionType;
	planContent: string | null;
	toolName: string | null;
	toolInput: string | null;
}

export interface AgentSession {
	beanId: string;
	agentType: string;
	status: 'IDLE' | 'RUNNING' | 'ERROR';
	messages: AgentMessage[];
	error: string | null;
	planMode: boolean;
	yoloMode: boolean;
	systemStatus: string | null;
	pendingInteraction: PendingInteraction | null;
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
			planMode
			yoloMode
			systemStatus
			pendingInteraction {
				type
				planContent
				toolName
				toolInput
			}
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

const SET_AGENT_PLAN_MODE = gql`
	mutation SetAgentPlanMode($beanId: ID!, $planMode: Boolean!) {
		setAgentPlanMode(beanId: $beanId, planMode: $planMode)
	}
`;

const SET_AGENT_YOLO_MODE = gql`
	mutation SetAgentYoloMode($beanId: ID!, $yoloMode: Boolean!) {
		setAgentYoloMode(beanId: $beanId, yoloMode: $yoloMode)
	}
`;

const CLEAR_AGENT_SESSION = gql`
	mutation ClearAgentSession($beanId: ID!) {
		clearAgentSession(beanId: $beanId)
	}
`;

const RESOLVE_PERMISSION = gql`
	mutation ResolvePermission($beanId: ID!, $allow: Boolean!, $always: Boolean) {
		resolvePermission(beanId: $beanId, allow: $allow, always: $always)
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
						const prev = this.session;

						// Log new messages (user/tool appear instantly, so count-based works)
						const prevLen = prev?.messages.length ?? 0;
						for (let i = prevLen; i < session.messages.length; i++) {
							const msg = session.messages[i];
							if (msg.role !== 'ASSISTANT') {
								console.debug(`[agent:${msg.role}]`, msg.content);
							}
						}

						// Log completed assistant messages when turn finishes
						if (prev?.status === 'RUNNING' && session.status === 'IDLE') {
							for (const msg of session.messages.slice(prevLen > 0 ? prevLen - 1 : 0)) {
								if (msg.role === 'ASSISTANT' && msg.content) {
									console.debug('[agent:ASSISTANT]', msg.content);
								}
							}
						}

						if (session.systemStatus && session.systemStatus !== prev?.systemStatus) {
							console.debug('[agent:system]', session.systemStatus);
						}
						if (session.error && session.error !== prev?.error) {
							console.debug('[agent:error]', session.error);
						}

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

	async setPlanMode(beanId: string, planMode: boolean): Promise<boolean> {
		const result = await client
			.mutation(SET_AGENT_PLAN_MODE, { beanId, planMode })
			.toPromise();

		if (result.error) {
			this.error = result.error.message;
			return false;
		}

		return true;
	}

	async setYoloMode(beanId: string, yoloMode: boolean): Promise<boolean> {
		const result = await client
			.mutation(SET_AGENT_YOLO_MODE, { beanId, yoloMode })
			.toPromise();

		if (result.error) {
			this.error = result.error.message;
			return false;
		}

		return true;
	}

	async resolvePermission(beanId: string, allow: boolean, always?: boolean): Promise<boolean> {
		const result = await client
			.mutation(RESOLVE_PERMISSION, { beanId, allow, always: always ?? null })
			.toPromise();

		if (result.error) {
			this.error = result.error.message;
			return false;
		}

		return true;
	}

	async clearSession(beanId: string): Promise<boolean> {
		const result = await client
			.mutation(CLEAR_AGENT_SESSION, { beanId })
			.toPromise();

		if (result.error) {
			this.error = result.error.message;
			return false;
		}

		return true;
	}
}
