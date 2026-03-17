import { client } from '$lib/graphqlClient';
import {
  AgentActionsDocument,
  ExecuteAgentActionDocument,
  type AgentActionFieldsFragment,
} from './graphql/generated';

export type AgentAction = AgentActionFieldsFragment;

const PR_POLL_INTERVAL = 10_000;

export class AgentActionsStore {
  actions = $state<AgentAction[]>([]);
  executingAction = $state<string | null>(null);
  #wasAgentBusy = false;
  #pollTimer: ReturnType<typeof setInterval> | null = null;

  async fetch(beanId: string) {
    const result = await client
      .query(AgentActionsDocument, { beanId }, { requestPolicy: 'network-only' })
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
      this.fetch(beanId);
    }
    this.#wasAgentBusy = busy;
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
