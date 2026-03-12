<script lang="ts">
  import { gql } from 'urql';
  import { onMount, onDestroy } from 'svelte';
  import { changesStore, type FileChange } from '$lib/changes.svelte';
  import { ui } from '$lib/uiState.svelte';
  import { client } from '$lib/graphqlClient';

  import SplitPane from '$lib/components/SplitPane.svelte';

  interface Props {
    path?: string;
  }

  let { path }: Props = $props();

  type Tab = 'unstaged' | 'all';
  let activeTab = $state<Tab>('all');

  const FILE_DIFF_QUERY = gql`
    query FileDiff($filePath: String!, $staged: Boolean!, $path: String) {
      fileDiff(filePath: $filePath, staged: $staged, path: $path)
    }
  `;

  const ALL_FILE_DIFF_QUERY = gql`
    query AllFileDiff($filePath: String!, $path: String) {
      allFileDiff(filePath: $filePath, path: $path)
    }
  `;

  // Diff view state
  let selectedFile = $state<{ path: string; staged: boolean } | null>(null);
  let diffContent = $state<string>('');
  let diffLoading = $state(false);

  function selectFile(change: FileChange) {
    const key = { path: change.path, staged: change.staged };
    // Toggle off if clicking the same file
    if (selectedFile?.path === key.path && selectedFile?.staged === key.staged) {
      selectedFile = null;
      diffContent = '';
      return;
    }
    selectedFile = key;
    if (activeTab === 'all') {
      fetchAllDiff(key.path);
    } else {
      fetchDiff(key.path, key.staged);
    }
  }

  async function fetchDiff(filePath: string, staged: boolean) {
    diffLoading = true;
    const result = await client
      .query(FILE_DIFF_QUERY, { filePath, staged, path: path ?? null })
      .toPromise();

    // Guard against stale response if user clicked a different file while loading
    if (selectedFile?.path !== filePath || selectedFile?.staged !== staged) return;

    if (result.error) {
      console.error('Failed to fetch diff:', result.error);
      diffContent = '';
    } else {
      diffContent = result.data?.fileDiff ?? '';
    }
    diffLoading = false;
  }

  async function fetchAllDiff(filePath: string) {
    diffLoading = true;
    const result = await client
      .query(ALL_FILE_DIFF_QUERY, { filePath, path: path ?? null })
      .toPromise();

    if (selectedFile?.path !== filePath) return;

    if (result.error) {
      console.error('Failed to fetch all diff:', result.error);
      diffContent = '';
    } else {
      diffContent = result.data?.allFileDiff ?? '';
    }
    diffLoading = false;
  }

  // Clear selection when the selected file disappears from the active changes list
  $effect(() => {
    if (selectedFile) {
      const list = activeTab === 'all' ? changesStore.allChanges : changesStore.changes;
      const stillExists = list.some(
        (c) => c.path === selectedFile!.path && (activeTab === 'all' || c.staged === selectedFile!.staged)
      );
      if (!stillExists) {
        selectedFile = null;
        diffContent = '';
      }
    }
  });

  // When switching tabs, clear the diff selection
  let prevTab = $state<Tab>(activeTab);
  $effect(() => {
    if (activeTab !== prevTab) {
      prevTab = activeTab;
      selectedFile = null;
      diffContent = '';
    }
  });

  const stagedChanges = $derived(changesStore.changes.filter((c) => c.staged));
  const unstagedChanges = $derived(changesStore.changes.filter((c) => !c.staged));
  const hasUnstagedChanges = $derived(changesStore.changes.length > 0);
  const displayChanges = $derived(activeTab === 'all' ? changesStore.allChanges : changesStore.changes);
  const totalCount = $derived(displayChanges.length);


  onMount(() => {
    changesStore.startPolling(path);
  });

  onDestroy(() => {
    changesStore.stopPolling();
  });

  function statusColor(status: string): string {
    switch (status) {
      case 'added':
      case 'untracked':
        return 'text-success';
      case 'deleted':
        return 'text-danger';
      case 'renamed':
        return 'text-accent';
      default:
        return 'text-warning';
    }
  }

  function statusLabel(status: string): string {
    switch (status) {
      case 'modified':
        return 'M';
      case 'added':
        return 'A';
      case 'deleted':
        return 'D';
      case 'untracked':
        return '?';
      case 'renamed':
        return 'R';
      default:
        return '?';
    }
  }

  function fileName(filePath: string): string {
    return filePath.split('/').pop() ?? filePath;
  }

  function dirName(filePath: string): string {
    const parts = filePath.split('/');
    if (parts.length <= 1) return '';
    return parts.slice(0, -1).join('/') + '/';
  }

  function isFileSelected(change: FileChange): boolean {
    if (activeTab === 'all') {
      return selectedFile?.path === change.path;
    }
    return selectedFile?.path === change.path && selectedFile?.staged === change.staged;
  }

  interface DiffLine {
    type: 'add' | 'del' | 'hunk' | 'context' | 'header';
    content: string;
    oldNum: number | null;
    newNum: number | null;
  }

  const parsedDiffLines = $derived.by(() => {
    if (!diffContent) return [];
    const raw = diffContent.split('\n');
    const lines: DiffLine[] = [];
    let oldNum = 0;
    let newNum = 0;

    for (const line of raw) {
      if (line.startsWith('diff --git') || line.startsWith('index ') || line.startsWith('--- ') || line.startsWith('+++ ')) {
        // Skip raw git headers
        continue;
      }
      if (line.startsWith('@@')) {
        const match = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@(.*)/);
        if (match) {
          oldNum = parseInt(match[1]);
          newNum = parseInt(match[2]);
          lines.push({ type: 'hunk', content: match[3]?.trim() || '', oldNum: null, newNum: null });
        }
        continue;
      }
      if (line.startsWith('+')) {
        lines.push({ type: 'add', content: line.slice(1), oldNum: null, newNum: newNum++ });
      } else if (line.startsWith('-')) {
        lines.push({ type: 'del', content: line.slice(1), oldNum: oldNum++, newNum: null });
      } else {
        lines.push({ type: 'context', content: line.startsWith(' ') ? line.slice(1) : line, oldNum: oldNum++, newNum: newNum++ });
      }
    }
    return lines;
  });

  function diffLineClass(type: DiffLine['type']): string {
    if (type === 'add') return 'diff-add';
    if (type === 'del') return 'diff-del';
    if (type === 'hunk') return 'diff-hunk';
    return '';
  }

</script>

{#snippet fileRow(change: FileChange)}
  <button
    class={[
      'flex w-full cursor-pointer items-center gap-1.5 px-3 py-0.5 text-left hover:bg-surface-alt',
      isFileSelected(change) && 'bg-surface-alt'
    ]}
    onclick={() => selectFile(change)}
  >
    <span class={['w-3 shrink-0 text-center font-mono font-bold', statusColor(change.status)]}>
      {statusLabel(change.status)}
    </span>
    <span class="min-w-0 flex-1 truncate" title={change.path}>
      <span class="text-text-muted">{dirName(change.path)}</span><span class="text-text">{fileName(change.path)}</span>
    </span>
    {#if change.additions > 0 || change.deletions > 0}
      <span class="shrink-0 font-mono">
        {#if change.additions > 0}<span class="text-success">+{change.additions}</span>{/if}
        {#if change.deletions > 0}<span class={[change.additions > 0 && 'ml-1', 'text-danger']}>-{change.deletions}</span>{/if}
      </span>
    {/if}
  </button>
{/snippet}

{#snippet fileList()}
  <div class="flex-1 overflow-auto">
    {#if totalCount === 0}
      <p class="px-3 py-4 text-center text-text-muted">No changes</p>
    {:else if activeTab === 'all'}
      {#each changesStore.allChanges as change (change.path)}
        {@render fileRow(change)}
      {/each}
    {:else}
      {#if stagedChanges.length > 0}
        <div class="px-3 pt-2 pb-1 font-medium text-text-muted">Staged</div>
        {#each stagedChanges as change (change.path + ':staged')}
          {@render fileRow(change)}
        {/each}
      {/if}

      {#if unstagedChanges.length > 0}
        {#if stagedChanges.length > 0}
          <div class="px-3 pt-2 pb-1 font-medium text-text-muted">Unstaged</div>
        {/if}
        {#each unstagedChanges as change (change.path + ':unstaged')}
          {@render fileRow(change)}
        {/each}
      {/if}
    {/if}
  </div>
{/snippet}

{#snippet diffView()}
  <div class="flex h-full flex-col border-t border-border">
    <div class="flex items-center justify-between px-3 py-1.5">
      <span class="truncate font-mono text-xs text-text-muted">
        {selectedFile?.path}
        {#if activeTab !== 'all' && selectedFile?.staged}
          <span class="text-text-faint">(staged)</span>
        {/if}
      </span>
      <button
        class="btn-icon shrink-0 cursor-pointer"
        onclick={() => { selectedFile = null; diffContent = ''; }}
        aria-label="Close diff"
      >
        <span class="iconify lucide--x size-3.5"></span>
      </button>
    </div>
    <div class="flex-1 overflow-auto bg-surface-alt">
      {#if diffLoading}
        <p class="px-3 py-4 text-center text-text-muted">Loading...</p>
      {:else if diffContent === ''}
        <p class="px-3 py-4 text-center text-text-muted">No diff available</p>
      {:else}
        <table class="diff-table w-full font-mono text-sm">
          <tbody>
            {#each parsedDiffLines as line, i (i)}
              {#if line.type === 'hunk'}
                <tr class="diff-hunk">
                  <td class="diff-gutter-hunk"></td>
                  <td class="px-3 py-1">{line.content || '...'}</td>
                </tr>
              {:else}
                <tr class={diffLineClass(line.type)}>
                  <td class="diff-gutter">{line.newNum ?? line.oldNum ?? ''}</td>
                  <td class="whitespace-pre pr-3">{#if line.type === 'add'}<span class="diff-indicator">+</span>{:else if line.type === 'del'}<span class="diff-indicator">-</span>{:else}<span class="diff-indicator"> </span>{/if}{line.content}</td>
                </tr>
              {/if}
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  </div>
{/snippet}

{#snippet tabSwitcher()}
  <div class="flex px-3 pt-3 pb-1.5">
    <div class="flex w-full">
      <button
        class={[
          'btn-tab-sm flex-1 rounded-l-md',
          activeTab === 'all' ? 'btn-tab-active' : 'btn-tab-inactive'
        ]}
        onclick={() => { activeTab = 'all'; }}
      >
        All Changes
        {#if changesStore.allChanges.length > 0}
          <span class="ml-1 opacity-60">({changesStore.allChanges.length})</span>
        {/if}
      </button>
      <button
        class={[
          'btn-tab-sm flex-1 rounded-r-md border-l-0',
          activeTab === 'unstaged'
            ? 'btn-tab-active'
            : hasUnstagedChanges
              ? 'btn-tab-inactive'
              : 'btn-tab-inactive opacity-50 cursor-not-allowed'
        ]}
        disabled={!hasUnstagedChanges}
        onclick={() => { activeTab = 'unstaged'; }}
      >
        Unstaged
        {#if changesStore.changes.length > 0}
          <span class="ml-1 opacity-60">({changesStore.changes.length})</span>
        {/if}
      </button>
    </div>
  </div>
{/snippet}

<div class="flex h-full flex-col bg-surface">
  {@render tabSwitcher()}

  {#if selectedFile}
    <SplitPane
      direction="vertical"
      panels={[
        { content: fileList, size: 200, minSize: 60, persistKey: 'changes-diff' },
        { content: diffView }
      ]}
    />
  {:else}
    {@render fileList()}
  {/if}

</div>
