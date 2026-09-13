<script lang="ts">
	// /welcome used to show the marketing landing inside the app. The landing
	// now lives on the site; in-app help is /help. Old links land there.
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	onMount(() => goto('/help', { replaceState: true }));
</script>
