<!-- Match page /m/{id}. Bridge until the hero + big steppers design lands:
     resolves the match's season, points the shared stores at it and shows
     the existing TipCard expanded (tip editor, friends' picks, bots). -->
<script lang="ts">
	import { page } from '$app/stores';
	import { pb } from '$lib/pb';
	import { tournamentStore } from '$lib/tournament.svelte';
	import { tipsStore } from '$lib/tips.svelte';
	import TipCard from '$lib/components/TipCard.svelte';
	import { ChevronLeft } from '@lucide/svelte';

	let id = $derived($page.params.id ?? '');
	let missing = $state(false);
	let seasonId = $state('');

	$effect(() => {
		const mid = id;
		missing = false;
		seasonId = '';
		if (!mid) return;
		(async () => {
			await tournamentStore.ready();
			const rec = await pb.collection('matches').getOne(mid).catch(() => null);
			const t = rec && tournamentStore.list.find((x) => x.id === rec.tournament);
			if (!t) {
				missing = true;
				return;
			}
			tournamentStore.select(t.slug);
			seasonId = t.id;
			await tipsStore.load().catch(() => {});
		})();
	});

	let season = $derived(tournamentStore.current);
	let loaded = $derived(!!seasonId && tipsStore.loaded && season?.id === seasonId);
	let match = $derived(loaded ? tipsStore.matches.find((m) => m.id === id) : undefined);
	let hubHref = $derived(
		season ? `/competitions/${season.competition.key}?s=${season.slug}&tab=matches` : '/'
	);
</script>

<div class="mp">
	<a class="back" href={hubHref}><ChevronLeft size={18} /> {season?.competition?.name ?? 'Back'}</a>
	{#if missing}
		<div class="card"><p class="muted">No such match.</p></div>
	{:else if !match}
		<p class="muted">Loading…</p>
	{:else}
		<TipCard {match} open={true} onToggle={() => {}} />
	{/if}
</div>

<style>
	.back {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		margin: 0.2rem 0 0.9rem;
		font-weight: 700;
		font-size: 0.9rem;
	}
</style>
