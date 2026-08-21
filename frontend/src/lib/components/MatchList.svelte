<!-- The full fixture list of the selected season (tournamentStore.current,
     loaded into tipsStore): every match as a TipCard, grouped by day /
     group / knockout round, with a "Now" button that jumps to the next
     kickoff. Lives in the competition hub's Matches tab. -->
<script lang="ts">
	import { tipsStore, type Match } from '$lib/tips.svelte';
	import { tournamentStore } from '$lib/tournament.svelte';
	import TipCard from './TipCard.svelte';
	import GroupStandings from './GroupStandings.svelte';
	import { bestThirds } from '$lib/standings';
	import { serverClock } from '$lib/serverclock.svelte';
	import { LocateFixed } from '@lucide/svelte';
	import { tick } from 'svelte';

	let tab = $state<'all' | 'group' | 'ko'>('all');

	// Accordion: only one match's tip inputs are open at a time.
	let openId = $state('');

	// Projected best extra-qualifier teams (per the tournament structure, e.g.
	// WC2026's 8 best thirds) across all groups (empty until every group is
	// filled) — shared by every group's standings table.
	let thirdsAdv = $derived.by(() => {
		const by: Record<string, Match[]> = {};
		for (const m of tipsStore.matches)
			if (tournamentStore.isGroup(m.stage)) (by[m.groupLetter] ||= []).push(m);
		return bestThirds(Object.values(by), tipsStore.tips);
	});

	let filtered = $derived(
		tipsStore.matches.filter((m) => {
			if (tab === 'group') return tournamentStore.isGroup(m.stage);
			if (tab === 'ko') return tournamentStore.isKnockout(m.stage);
			return true;
		})
	);

	// "Now" = the next match not yet kicked off (or the last one if the
	// season is over) within the current filter.
	let nowId = $derived.by(() => {
		const now = serverClock.now();
		const next = filtered.find((m) => new Date(m.kickoff).getTime() >= now);
		return (next ?? filtered[filtered.length - 1])?.id ?? '';
	});

	function goNow() {
		document
			.getElementById(`day-${nowDayIndex}`)
			?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	}

	// Groups tab: by group letter (A..L). Knockout tab: by stage, in the
	// structure's play order. All tab: by calendar day.
	let days = $derived.by(() => {
		const byKickoff = (a: Match, b: Match) =>
			new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime();
		if (tab === 'group') {
			const byGroup: Record<string, Match[]> = {};
			for (const m of filtered) (byGroup[m.groupLetter] ||= []).push(m);
			return Object.keys(byGroup)
				.sort()
				.map((l) => [tournamentStore.groupLabel(l), byGroup[l].sort(byKickoff)] as [string, Match[]]);
		}
		if (tab === 'ko') {
			const byStage: Record<string, Match[]> = {};
			for (const m of filtered) (byStage[m.stage] ||= []).push(m);
			return tournamentStore.knockoutStages
				.filter((s) => byStage[s.code])
				.map((s) => [s.name, byStage[s.code].sort(byKickoff)] as [string, Match[]]);
		}
		return Object.entries(
			filtered.reduce<Record<string, Match[]>>((acc, m) => {
				const d = new Date(m.kickoff).toLocaleDateString(undefined, {
					weekday: 'long',
					day: 'numeric',
					month: 'long'
				});
				(acc[d] ||= []).push(m);
				return acc;
			}, {})
		);
	});

	let nowDayIndex = $derived(days.findIndex(([, ms]) => ms.some((m) => m.id === nowId)));

	// On first load, jump to the current point in the season.
	let didAutoScroll = false;
	$effect(() => {
		if (didAutoScroll || !tipsStore.loaded) return;
		const idx = nowDayIndex;
		if (idx < 0) return;
		didAutoScroll = true;
		if (idx === 0) return;
		tick().then(() =>
			document.getElementById(`day-${idx}`)?.scrollIntoView({ block: 'start' })
		);
	});
</script>

{#if tournamentStore.knockoutStages.length > 0 && tournamentStore.groupStageCode}
	<div class="seg sub">
		<button class:on={tab === 'all'} onclick={() => (tab = 'all')}>All</button>
		<button class:on={tab === 'group'} onclick={() => (tab = 'group')}
			>{tournamentStore.singleTable ? 'Table' : 'Groups'}</button
		>
		<button class:on={tab === 'ko'} onclick={() => (tab = 'ko')}>Knockout</button>
	</div>
{/if}

{#if !tipsStore.loaded}
	<p class="muted">Loading fixtures…</p>
{:else if filtered.length === 0}
	<p class="muted">Nothing here.</p>
{:else}
	{#each days as [day, ms], i (day)}
		<h3 class="day" id={`day-${i}`}>{day}</h3>
		{#each ms as m (m.id)}
			<div class="match">
				<TipCard
					match={m}
					open={openId === m.id}
					onToggle={() => (openId = openId === m.id ? '' : m.id)}
				/>
			</div>
		{/each}
		{#if tab === 'group'}
			<GroupStandings matches={ms} bestThirds={thirdsAdv} />
		{/if}
	{/each}
	<div class="fabpad"></div>
{/if}

{#if tipsStore.loaded && nowId}
	<button class="fab" onclick={goNow} aria-label="Scroll to the next match">
		<LocateFixed size={18} /> Now
	</button>
{/if}

<style>
	.seg.sub {
		margin: 0.2rem 0 0.4rem;
	}
	.day {
		margin: 1.3rem 0 0.6rem;
		font-size: 0.95rem;
		color: var(--muted);
		/* Land below the fixed top bar + the hub's sticky tab strip. */
		scroll-margin-top: calc(var(--topbar-h) + 4.2rem);
	}
	.match + .match {
		margin-top: 6px;
	}
	.fabpad {
		height: 4rem;
	}
	.fab {
		position: fixed;
		right: 1rem;
		bottom: calc(var(--nav-h) + 1rem);
		z-index: 40;
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.7rem 1rem;
		border: none;
		border-radius: var(--radius-pill);
		background: var(--accent);
		color: var(--accent-fg);
		font: 800 0.8rem var(--font);
		letter-spacing: 0.06em;
		text-transform: uppercase;
		cursor: pointer;
		box-shadow: var(--shadow-pop);
		transition:
			transform 0.12s ease,
			box-shadow 0.2s ease;
	}
	.fab:hover {
		transform: translateY(-2px);
		box-shadow: var(--glow);
	}
	@media (min-width: 900px) {
		.fab {
			bottom: 1.5rem;
			right: 1.5rem;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.fab {
			transition: none;
		}
	}
</style>
