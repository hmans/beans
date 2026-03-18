import { client } from '$lib/graphqlClient';
import {
  AgentActionsDocument,
  ExecuteAgentActionDocument,
  type AgentActionFieldsFragment,
} from './graphql/generated';

export type AgentAction = AgentActionFieldsFragment;

const PR_POLL_INTERVAL = 10_000;
/** Delay before the follow-up re-fetch after agent idle, giving CI providers time to register new checks. */
const POST_IDLE_REFETCH_DELAY = 5_000;

export class AgentActionsStore {
  actions = $state<AgentAction[]>([]);
  executingAction = $state<string | null>(null);
  #wasAgentBusy = false;
  #pollTimer: ReturnType<typeof setInterval> | null = null;
  #postIdleTimer: ReturnType<typeof setTimeout> | null = null;

  async fetch(beanId: string, skipForge?: boolean) {
    const result = await client
      .query(
        AgentActionsDocument,
        { beanId, skipForge: skipForge ?? null },
        { requestPolicy: 'network-only' }
      )
      .toPromise();
    if (result.error) {
      console.error('Failed to fetch agent actions:', result.error);
      return;
    }
    if (result.data?.agentActions) {
      this.actions = result.data.agentActions;
    }
  }

  /**
   * Call this reactively with the current agent busy state.
   * Automatically re-fetches actions when the agent transitions from busy to idle.
   */
  notifyAgentStatus(beanId: string, busy: boolean) {
    if (this.#wasAgentBusy && !busy) {
      // Immediate fetch for actions that don't depend on external CI state.
      this.fetch(beanId);
      // Delayed re-fetch to pick up CI check status that GitHub may not have
      // registered yet right after the agent pushed.
      this.#clearPostIdleTimer();
      this.#postIdleTimer = setTimeout(() => this.fetch(beanId), POST_IDLE_REFETCH_DELAY);
    }
    this.#wasAgentBusy = busy;
  }

  #clearPostIdleTimer() {
    if (this.#postIdleTimer) {
      clearTimeout(this.#postIdleTimer);
      this.#postIdleTimer = null;
    }
  }

  /** Start polling agent actions to keep PR check status fresh. */
  startPolling(beanId: string) {
    this.stopPolling();
    this.#pollTimer = setInterval(() => this.fetch(beanId), PR_POLL_INTERVAL);
  }

  stopPolling() {
    if (this.#pollTimer) {
      clearInterval(this.#pollTimer);
      this.#pollTimer = null;
    }
    this.#clearPostIdleTimer();
  }

  async execute(beanId: string, actionId: string) {
    this.executingAction = actionId;
    try {
      await client.mutation(ExecuteAgentActionDocument, { beanId, actionId }).toPromise();
    } finally {
      this.executingAction = null;
    }
  }
}
