import type { Plugin } from "@opencode-ai/plugin";

/**
 * Beans Prime Plugin for OpenCode
 *
 * This plugin injects the output of `beans prime` into OpenCode's system prompt,
 * giving the AI context about your project's beans (issues/tasks). It runs on:
 *
 * - Chat session start: Adds bean context to the system prompt
 * - Session compaction: Re-injects context when the session is compacted
 *
 * Plugin Location Options:
 *
 * 1. Project-local (current): .opencode/plugin/beans-prime.ts
 *    - Only available in this project
 *    - Committed to version control, shared with collaborators
 *
 * 2. User-global: ~/.opencode/plugin/beans-prime.ts
 *    - Available in all your projects that use beans
 *    - Personal configuration, not shared
 */

export const BeansPrimePlugin: Plugin = async ({ $ }) => {
  const prime = await $`beans prime`.text();

  return {
    "experimental.chat.system.transform": async (_, output) => {
      if (prime) {
        output.system.push(prime);
      }
    },
    "experimental.session.compacting": async (_, output) => {
      if (prime) {
        output.context.push(prime);
      }
    },
  };
};

export default BeansPrimePlugin;
