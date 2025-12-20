<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { preloadHighlighter } from '$lib/markdown';

	// Start loading the syntax highlighter early
	preloadHighlighter();

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>
{@render children()}
