import { pipe, subscribe } from 'wonka';
import { client } from './graphqlClient';
import {
  AgentSessionChangedDocument,
  SendAgentMessageDocument,
  StopAgentDocument,
  SetAgentPlanModeDocument,
  SetAgentActModeDocument,
  SetAgentEffortDocument,
  ClearAgentSessionDocument,
  type AgentSessionFieldsFragment,
  AgentMessageRole,
  type AgentMessage as GqlAgentMessage,
  type AgentMessageImage as GqlAgentMessageImage,
  type PendingInteraction as GqlPendingInteraction,
  type AskUserQuestion as GqlAskUserQuestion,
  type AskUserOption as GqlAskUserOption,
  type SubagentActivity as GqlSubagentActivity,
  type InteractionType,
  type ImageInput,
  type FileAttachmentInput,
} from './graphql/generated';

export type AgentMessageImage = GqlAgentMessageImage;
export type AgentMessage = GqlAgentMessage;
export type { InteractionType };
export type AskUserOption = GqlAskUserOption;
export type AskUserQuestionData = GqlAskUserQuestion;
export type PendingInteraction = GqlPendingInteraction;
export type SubagentActivity = GqlSubagentActivity;
export type AgentSession = AgentSessionFieldsFragment;
export type ImageUploadInput = ImageInput;
export type FileAttachment = FileAttachmentInput;

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
      client.subscription(AgentSessionChangedDocument, { beanId }),
      subscribe((result) => {
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
      })
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

  async sendMessage(
    beanId: string,
    message: string,
    images?: ImageUploadInput[],
    attachments?: FileAttachment[]
  ): Promise<boolean> {
    this.sending = true;
    this.error = null;

    // Optimistically append user message so it appears instantly
    if (this.session) {
      this.session = {
        ...this.session,
        messages: [
          ...this.session.messages,
          {
            role: AgentMessageRole.User,
            content: message,
            images: [],
            diff: null
          }
        ]
      };
    }

    const result = await client
      .mutation(SendAgentMessageDocument, {
        beanId,
        message,
        images: images ?? null,
        attachments: attachments?.length ? attachments : null
      })
      .toPromise();

    this.sending = false;

    if (result.error) {
      this.error = result.error.message;
      return false;
    }

    return true;
  }

  async stop(beanId: string): Promise<boolean> {
    const result = await client.mutation(StopAgentDocument, { beanId }).toPromise();

    if (result.error) {
      this.error = result.error.message;
      return false;
    }

    return true;
  }

  async setPlanMode(beanId: string, planMode: boolean): Promise<boolean> {
    const result = await client.mutation(SetAgentPlanModeDocument, { beanId, planMode }).toPromise();

    if (result.error) {
      this.error = result.error.message;
      return false;
    }

    return true;
  }

  async setActMode(beanId: string, actMode: boolean): Promise<boolean> {
    const result = await client.mutation(SetAgentActModeDocument, { beanId, actMode }).toPromise();

    if (result.error) {
      this.error = result.error.message;
      return false;
    }

    return true;
  }

  async setEffort(beanId: string, effort: string): Promise<boolean> {
    const result = await client
      .mutation(SetAgentEffortDocument, { beanId, effort })
      .toPromise();

    if (result.error) {
      this.error = result.error.message;
      return false;
    }

    return true;
  }

  async clearSession(beanId: string): Promise<boolean> {
    const result = await client.mutation(ClearAgentSessionDocument, { beanId }).toPromise();

    if (result.error) {
      this.error = result.error.message;
      return false;
    }

    return true;
  }
}
