<!-- Declares the page's top-bar chrome from markup, so a page can hand
     over a snippet (`bar`) for the mobile top bar. Renders nothing. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { pageChrome } from '$lib/shell.svelte';
	let {
		title = '',
		back = '',
		context = '',
		wide = false,
		bar = undefined
	}: { title?: string; back?: string; context?: string; wide?: boolean; bar?: Snippet } = $props();
	pageChrome(() => ({ title, back, context, wide, bar }));
</script>
