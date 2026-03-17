<script lang="ts">
  import type { PendingInteraction, AskUserQuestionData, AskUserOption } from '$lib/agentChat.svelte';
  import { renderMarkdown } from '$lib/markdown';

  interface Props {
    interaction: PendingInteraction;
    onApprove: () => void;
    onSendMessage: (message: string) => void;
  }

  let { interaction, onApprove, onSendMessage }: Props = $props();

  // Render plan content as markdown when available
  let renderedPlanContent = $state<string | null>(null);
  $effect(() => {
    const content = interaction.planContent;
    if (content) {
      renderMarkdown(content).then((html) => {
        renderedPlanContent = html;
      });
    } else {
      renderedPlanContent = null;
    }
  });

  // Multi-select state for AskUserQuestion
  let multiSelectChoices = $state<Set<string>>(new Set());

  // Reset multi-select choices when the pending interaction changes
  $effect(() => {
    interaction;
    multiSelectChoices = new Set();
  });

  function handleOptionClick(q: AskUserQuestionData, opt: AskUserOption) {
    if (q.multiSelect) {
      const next = new Set(multiSelectChoices);
      if (next.has(opt.label)) {
        next.delete(opt.label);
      } else {
        next.add(opt.label);
      }
      multiSelectChoices = next;
    } else {
      onSendMessage(opt.label);
    }
  }

  function submitMultiSelect() {
    if (multiSelectChoices.size === 0) return;
    onSendMessage([...multiSelectChoices].join(', '));
  }
</script>

{#if interaction.type === 'EXIT_PLAN'}
  <div class="border-t border-status-in-progress-text/20 bg-status-in-progress-bg/50 p-3">
    <p class="mb-2 text-text-muted">
      Agent wants to leave plan mode and start working.
    </p>

    {#if renderedPlanContent}
      <div class="mb-3 max-h-96 overflow-y-auto rounded border border-border bg-surface p-3">
        <div class="agent-prose prose max-w-none min-w-0 text-text">
          {@html renderedPlanContent}
        </div>
      </div>
    {/if}

    <div class="flex items-center gap-3">
      <button
        onclick={onApprove}
        class="cursor-pointer rounded bg-status-in-progress-text px-3 py-1.5 text-white transition-colors hover:opacity-90"
      >
        Approve
      </button>
      <span class="text-text-muted">or type below to refine the plan</span>
    </div>
  </div>
{/if}

{#if interaction.type === 'ASK_USER'}
  <div class="border-t border-accent/30 bg-accent/5 px-4 py-3">
    {#if interaction.questions?.length}
      <div class="space-y-4">
        {#each interaction.questions as q}
          <div class="space-y-2">
            <div>
              {#if q.header}
                <span class="inline-block rounded bg-accent/15 px-1.5 py-0.5 text-xs font-bold text-accent">
                  {q.header}
                </span>
              {/if}
              <p class="mt-1 text-sm text-text">{q.question}</p>
            </div>
            <div class="flex flex-col gap-1.5">
              {#each q.options as opt}
                <button
                  class={[
                    'cursor-pointer rounded border px-3 py-2 text-left text-xs transition-colors',
                    q.multiSelect && multiSelectChoices.has(opt.label)
                      ? 'border-accent bg-accent/15 text-accent'
                      : 'border-border hover:border-accent/50 hover:bg-accent/5'
                  ]}
                  onclick={() => handleOptionClick(q, opt)}
                >
                  <span class="font-bold text-text">{opt.label}</span>
                  {#if opt.description}
                    <span class="ml-2 text-text-muted">{opt.description}</span>
                  {/if}
                </button>
              {/each}
            </div>
            {#if q.multiSelect && multiSelectChoices.size > 0}
              <button
                onclick={submitMultiSelect}
                class="cursor-pointer rounded bg-accent px-3 py-1.5 text-xs text-accent-text transition-colors hover:bg-accent/90"
              >
                Submit ({multiSelectChoices.size} selected)
              </button>
            {/if}
          </div>
        {/each}
      </div>
      <p class="mt-3 text-xs text-text-faint">
        Or type a custom reply below.
      </p>
    {:else}
      <p class="text-xs text-accent">
        Agent is waiting for your answer — type your reply below.
      </p>
    {/if}
  </div>
{/if}
