import { gql } from 'urql';
import { client } from '$lib/graphqlClient';

export interface AgentAction {
  id: string;
  label: string;
  description: string | null;
}

const AGENT_ACTIONS_QUERY = gql`
  query AgentActions($beanId: ID!) {
    agentActions(beanId: $beanId) {
      id
      label
      description
    }
  }
`;

const EXECUTE_AGENT_ACTION = gql`
  mutation ExecuteAgentAction($beanId: ID!, $actionId: ID!) {
    executeAgentAction(beanId: $beanId, actionId: $actionId)
  }
`;

export class AgentActionsStore {
  actions = $state<AgentAction[]>([]);
  executingAction = $state<string | null>(null);
  #wasAgentBusy = false;

  async fetch(beanId: string) {
    const result = await client.query(AGENT_ACTIONS_QUERY, { beanId }).toPromise();
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

  async execute(beanId: string, actionId: string, agentBusy: boolean) {
    if (agentBusy) return;
    this.executingAction = actionId;
    try {
      await client.mutation(EXECUTE_AGENT_ACTION, { beanId, actionId }).toPromise();
    } finally {
      this.executingAction = null;
    }
  }
}
