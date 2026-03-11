import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import type { Page } from '@playwright/test';

type ImageEntry = { id: string; media_type: string };
type MessageEntry = { role: 'user' | 'assistant'; content: string; images?: ImageEntry[] };
type InteractionType = 'EXIT_PLAN' | 'ENTER_PLAN' | 'ASK_USER';

interface PendingInteractionConfig {
  type: InteractionType;
  planContent?: string;
}

/** Send a GraphQL query/mutation to the beans server. */
async function gql(baseURL: string, query: string) {
  const res = await fetch(`${baseURL}/api/graphql`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query })
  });
  return res.json();
}

/**
 * Builder for setting up agent session state in e2e tests.
 *
 * Handles the details of JSONL file seeding (for messages) and GraphQL
 * mutations (for session state), in the correct order.
 *
 * Usage:
 *   await agentSession('__central__', beans)
 *     .withMessages([
 *       { role: 'user', content: 'Plan a refactor' },
 *       { role: 'assistant', content: 'Here is my plan.' }
 *     ])
 *     .inPlanMode()
 *     .withPendingInteraction({ type: 'EXIT_PLAN', planContent: '# Plan\n...' })
 *     .open(page);
 */
export function agentSession(beanId: string, beans: { beansPath: string; baseURL: string }) {
  return new AgentSessionBuilder(beanId, beans.beansPath, beans.baseURL);
}

class AgentSessionBuilder {
  private messages: MessageEntry[] = [];
  private planMode = false;
  private actMode = false;
  private pendingInteraction: PendingInteractionConfig | null = null;

  constructor(
    private beanId: string,
    private beansPath: string,
    private baseURL: string
  ) {}

  /** Seed conversation messages (persisted to JSONL, loaded on subscription connect). */
  withMessages(messages: MessageEntry[]): this {
    this.messages = messages;
    return this;
  }

  /** Set the session to plan mode. */
  inPlanMode(): this {
    this.planMode = true;
    this.actMode = false;
    return this;
  }

  /** Set the session to act mode. */
  inActMode(): this {
    this.actMode = true;
    this.planMode = false;
    return this;
  }

  /** Set a pending interaction (shown as approval UI in the chat). */
  withPendingInteraction(config: PendingInteractionConfig): this {
    this.pendingInteraction = config;
    return this;
  }

  /**
   * Apply the session state and open the chat panel.
   *
   * Order matters:
   * 1. Seed JSONL file (messages loaded lazily on subscription connect)
   * 2. Navigate and open chat (triggers subscription, loads messages)
   * 3. Apply mutations (pushed to frontend via subscription)
   */
  async open(page: Page): Promise<void> {
    // 1. Seed messages to JSONL
    if (this.messages.length > 0) {
      const convDir = join(this.beansPath, '.conversations');
      mkdirSync(convDir, { recursive: true });
      const lines = this.messages.map((m) => {
        const entry: Record<string, unknown> = { type: 'message', role: m.role, content: m.content };
        if (m.images && m.images.length > 0) entry.images = m.images;
        return JSON.stringify(entry);
      });
      writeFileSync(join(convDir, `${this.beanId}.jsonl`), lines.join('\n') + '\n');
    }

    // 2. Navigate and open the chat panel
    await page.goto(this.baseURL + '/');
    await page.click('button[title="Show chat"]');

    // Wait for messages to load if we seeded any
    if (this.messages.length > 0) {
      const firstMsg = this.messages[0].content;
      const { expect } = await import('@playwright/test');
      await expect(page.locator(`text=${firstMsg}`)).toBeVisible({ timeout: 5000 });
    }

    // 3. Apply session state via GraphQL mutations
    if (this.planMode) {
      await gql(
        this.baseURL,
        `mutation { setAgentPlanMode(beanId: "${this.beanId}", planMode: true) }`
      );
    }
    if (this.actMode) {
      await gql(
        this.baseURL,
        `mutation { setAgentActMode(beanId: "${this.beanId}", actMode: true) }`
      );
    }
    if (this.pendingInteraction) {
      const planArg = this.pendingInteraction.planContent
        ? `, planContent: ${JSON.stringify(this.pendingInteraction.planContent)}`
        : '';
      await gql(
        this.baseURL,
        `mutation { setAgentPendingInteraction(beanId: "${this.beanId}", type: ${this.pendingInteraction.type}${planArg}) }`
      );
    }
  }
}
