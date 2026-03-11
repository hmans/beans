<script lang="ts">
  import { page } from '$app/state';
  import { beansStore } from '$lib/beans.svelte';
  import WorkspaceView from '$lib/components/WorkspaceView.svelte';

  const bean = $derived(beansStore.get(page.params.beanId) ?? null);
</script>

{#if bean}
  {#key bean.id}
    <WorkspaceView {bean} />
  {/key}
{/if}
