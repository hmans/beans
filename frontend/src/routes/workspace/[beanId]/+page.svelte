<script lang="ts">
  import { page } from '$app/state';
  import WorkspaceView from '$lib/components/WorkspaceView.svelte';

  const worktreeId = $derived(page.params.beanId!);
</script>

{#key worktreeId}
  <WorkspaceView {worktreeId} />
{/key}
