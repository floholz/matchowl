<!-- Matches: every match of the competitions you play, day by day, one
     card per competition per day, today anchored. Only lists matches —
     deadlines and suggestions live on Home / Competitions. -->
<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { feedStore, type FeedMatch } from '$lib/feed.svelte';
	import { otherLegView } from '$lib/tips.svelte';
	import { tournamentStore, competitionLogoUrl } from '$lib/tournament.svelte';
	import { pageChrome } from '$lib/shell.svelte';
	import MatchGroup from '$lib/components/MatchGroup.svelte';
	import MatchRow from '$lib/components/MatchRow.svelte';
	import { tick } from 'svelte';
	import { ChevronUp, ChevronDown } from '@lucide/svelte';

	pageChrome(() => ({ title: 'Matches' }));

	let openId = $state('');
	let scrolled = $state(false);

	$effect(() => {
		if (auth.isAuthed && !feedStore.loaded && !feedStore.loading && !feedStore.error) {
			feedStore.load().then(scrollToToday).catch(() => {});
			tournamentStore.ready().catch(() => {});
		}
	});

	// Land on today (or the first upcoming day) once, after first render.
	async function scrollToToday() {
		if (scrolled) return;
		scrolled = true;
		await tick();
		document
			.getElementById('feed-today')
			?.scrollIntoView({ block: 'start', behavior: 'instant' as ScrollBehavior });
	}

	/** The section today's anchor lands on: today itself, else the first
	 *  future day (there may be no matches today). */
	let anchorKey = $derived.by(() => {
		const days = feedStore.days;
		const today = days.find((d) => d.isToday);
		if (today) return today.key;
		return days.find((d) => d.key > feedStore.todayKey)?.key ?? '';
	});

	function tipFor(m: FeedMatch) {
		return m.myTip ? { match: m.id, ...m.myTip } : null;
	}
	/** Crest of a feed tournament's competition (from the tournament list). */
	function logoOf(tid: string): string {
		const t = tournamentStore.list.find((x) => x.id === tid);
		return t ? competitionLogoUrl(t.competition) : '';
	}
	/** Card sub-line: the stage, plus the round when the day's matches share one. */
	function roundOf(ms: FeedMatch[]): string {
		const stage = ms[0]?.stageName ?? '';
		const rounds = new Set(ms.map((m) => m.roundLabel));
		const round = rounds.size === 1 ? ms[0].roundLabel : '';
		return [stage, round !== stage ? round : ''].filter(Boolean).join(' · ');
	}
</script>

<div class="matches">
	{#if feedStore.error && !feedStore.loaded}
		<div class="card empty">
			<p><b>Couldn't load your matches.</b></p>
			<p class="muted">{feedStore.error}</p>
			<button class="btn" onclick={() => feedStore.load().catch(() => {})}>Retry</button>
		</div>
	{:else if feedStore.loaded && feedStore.days.length === 0}
		<div class="card empty">
			<p><b>No matches yet.</b></p>
			<p class="muted">Play a competition and its matches show up here, day by day.</p>
			<a class="btn" href="/competitions">Browse competitions</a>
		</div>
	{:else if feedStore.loaded}
		<button class="btn ghost more" onclick={() => feedStore.earlier()}>
			<ChevronUp size={16} /> Earlier results
		</button>

		{#each feedStore.days as day (day.key)}
			<section class="day" id={day.key === anchorKey ? 'feed-today' : undefined}>
				<h2 class="day-h" class:today={day.isToday}>
					{day.isToday ? `Today · ${day.label}` : day.label}
				</h2>
				{#each day.groups as g (g.tournament.id)}
					<MatchGroup
						name={g.tournament.shortName || g.tournament.name}
						round={roundOf(g.matches)}
						logo={logoOf(g.tournament.id)}
						href={`/competitions/${g.tournament.competition}?s=${g.tournament.slug}`}
					>
						{#each g.matches as m (m.id)}
							<MatchRow
								match={m}
								team={(id) => feedStore.team(id)}
								tip={tipFor(m)}
								knockout={m.knockout}
								points={m.myTip?.points}
								onSave={(t) => feedStore.saveTip(m, t)}
								open={openId === m.id}
								onToggle={() => (openId = openId === m.id ? '' : m.id)}
								href={`/m/${m.id}`}
								leg={m.leg
									? otherLegView(
											m,
											m.leg,
											m.leg.first,
											`/competitions/${m.tournament.competition}?s=${m.tournament.slug}&tab=matches&m=${m.leg.id}`
										)
									: null}
							/>
						{/each}
					</MatchGroup>
				{/each}
			</section>
		{/each}

		<button class="btn ghost more" onclick={() => feedStore.later()}>
			<ChevronDown size={16} /> Later fixtures
		</button>
	{:else}
		<p class="muted">Loading…</p>
	{/if}
</div>

<style>
	.day + .day {
		margin-top: 0.4rem;
	}
	.day {
		/* room for the fixed header when the today-anchor scrolls here */
		scroll-margin-top: calc(var(--topbar-h) + 0.8rem);
	}
	.day-h {
		font-family: var(--font);
		font-size: 0.76rem;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--muted);
		margin: 0.2rem 0.15rem 0.55rem;
	}
	.day-h.today {
		color: var(--accent);
	}
	.btn.ghost.more {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.35rem;
		margin: 0 0 1rem;
	}
	.btn.ghost.more:last-child {
		margin: 0.6rem 0 0;
	}
	.empty {
		text-align: center;
		padding: 2rem 1.2rem;
	}
	.empty .btn {
		margin-top: 1rem;
	}
</style>
