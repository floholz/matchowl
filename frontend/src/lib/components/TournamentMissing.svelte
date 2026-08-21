<!-- Shown on the ?t=<slug> pages when the slug isn't available to this
     user (draft for non-admins, deleted, typo) — instead of silently
     falling back to the current tournament. -->
<script lang="ts">
	import { tournamentStore } from '$lib/tournament.svelte';
	import { TriangleAlert } from '@lucide/svelte';
</script>

{#if tournamentStore.missing}
	<div class="card tmiss" role="status">
		<TriangleAlert size={16} />
		<span>
			Season <code>{tournamentStore.missing}</code> isn't available (not published yet, or
			it doesn't exist) — showing <b>{tournamentStore.current?.name ?? '—'}</b> instead.
			<a href="/competitions">Browse competitions</a>
		</span>
	</div>
{/if}

<style>
	.tmiss {
		display: flex;
		align-items: flex-start;
		gap: 0.6rem;
		margin-bottom: 0.85rem;
		font-size: 0.9rem;
		color: var(--warning, #d9a441);
		border-color: color-mix(in srgb, var(--warning, #d9a441) 45%, var(--border));
	}
	.tmiss span {
		color: var(--text);
	}
	.tmiss code {
		font-size: 0.85em;
	}
	.tmiss a {
		color: var(--accent);
	}
</style>
