<script lang="ts">
	// The app's front door: the owl mark and wordmark over a title, with the
	// page's card(s) below. Used by every signed-out screen — sign in,
	// register, forgot password, the three email-link confirm pages and the
	// pool invite — so they all read as one place.
	import type { Snippet } from 'svelte';
	import Logo from './Logo.svelte';

	let {
		title,
		lead = '',
		children
	}: { title: string; lead?: string; children: Snippet } = $props();
</script>

<div class="auth">
	<div class="brand"><Logo size={44} /></div>
	<h1>{title}</h1>
	{#if lead}<p class="muted lead">{lead}</p>{/if}
	{@render children()}
</div>

<style>
	.auth {
		max-width: 380px;
		margin: 8dvh auto 0;
		padding: 0 1rem 3rem;
	}
	.brand {
		margin-bottom: 1.4rem;
	}
	h1 {
		margin: 0;
		font-size: 1.8rem;
	}
	.lead {
		margin: 0.25rem 0 0;
	}
	.auth > :global(.card),
	.auth > :global(form.card) {
		margin-top: 1.25rem;
	}
	@media (min-width: 900px) {
		.auth {
			margin-top: 14dvh;
		}
	}
</style>
