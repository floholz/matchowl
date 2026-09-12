<!-- Matches: every match of the competitions you play ("Mine", default)
     or of every competition ("All"), day by day, one card per competition
     per day, today anchored. A Live chip appears only while something is
     live; the sticky day strip scrolls the list and follows it. Only lists
     matches — deadlines and suggestions live on Home / Competitions. -->
<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { feedStore, type FeedMatch, type FeedDay } from '$lib/feed.svelte';
	import { otherLegView } from '$lib/tips.svelte';
	import { tournamentStore, competitionLogoUrl } from '$lib/tournament.svelte';
	import { pageChrome } from '$lib/shell.svelte';
	import MatchGroup from '$lib/components/MatchGroup.svelte';
	import MatchRow from '$lib/components/MatchRow.svelte';
	import { tick } from 'svelte';
	import { ChevronUp, ChevronDown, Check } from '@lucide/svelte';

	pageChrome(() => ({ title: 'Matches' }));

	let openId = $state('');
	let scrolled = $state(false);
	let liveOnly = $state(false);
	/** Day strip highlight: the day section currently at the top. */
	let activeKey = $state('');
	let stripEl = $state<HTMLElement | null>(null);

	$effect(() => {
		if (auth.isAuthed && !feedStore.loaded && !feedStore.loading && !feedStore.error) {
			feedStore.load().then(scrollToToday).catch(() => {});
			tournamentStore.ready().catch(() => {});
		}
	});

	let liveCount = $derived(feedStore.matches.filter((m) => m.status === 'live').length);
	$effect(() => {
		if (liveCount === 0) liveOnly = false;
	});
	let days = $derived.by((): FeedDay[] => {
		if (!liveOnly) return feedStore.days;
		return feedStore.days
			.map((d) => {
				const matches = d.matches.filter((m) => m.status === 'live');
				return { ...d, matches, groups: d.groups
					.map((g) => ({ ...g, matches: g.matches.filter((m) => m.status === 'live') }))
					.filter((g) => g.matches.length) };
			})
			.filter((d) => d.matches.length);
	});
	let strip = $derived.by(() => {
		const has = new Set(days.map((d) => d.key));
		return feedStore.strip.map((d) => ({ ...d, has: has.has(d.key) }));
	});

	/** The section today's anchor lands on: today itself, else the first
	 *  future day (there may be no matches today). */
	let anchorKey = $derived.by(() => {
		const today = days.find((d) => d.isToday);
		if (today) return today.key;
		return days.find((d) => d.key > feedStore.todayKey)?.key ?? days[0]?.key ?? '';
	});

	// Land on today (or the first upcoming day) once, after first render.
	async function scrollToToday() {
		if (scrolled) return;
		scrolled = true;
		await tick();
		goDay(anchorKey, 'instant');
	}
	function goDay(key: string, behavior: ScrollBehavior = 'smooth') {
		const el = document.getElementById(`day-${key}`);
		if (!el) return;
		activeKey = key;
		el.scrollIntoView({ block: 'start', behavior });
	}
	// Follow the scroll: the topmost day section below the sticky chrome.
	$effect(() => {
		if (!feedStore.loaded) return;
		const sections = Array.from(document.querySelectorAll<HTMLElement>('section.day'));
		const onScroll = () => {
			const top = (document.querySelector('.subbar')?.getBoundingClientRect().bottom ?? 120) + 8;
			let cur = sections[0];
			for (const s of sections) if (s.getBoundingClientRect().top <= top) cur = s;
			const key = cur?.dataset.key ?? '';
			if (key && key !== activeKey) activeKey = key;
		};
		window.addEventListener('scroll', onScroll, { passive: true });
		onScroll();
		return () => window.removeEventListener('scroll', onScroll);
	});
	// Keep the highlighted day visible in the strip.
	$effect(() => {
		const key = activeKey;
		const el = stripEl?.querySelector<HTMLElement>(`[data-key="${key}"]`);
		el?.scrollIntoView({ block: 'nearest', inline: 'center', behavior: 'smooth' });
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
	async function setScope(scope: 'mine' | 'all') {
		await feedStore.setScope(scope).catch(() => {});
		await tick();
		goDay(anchorKey, 'instant');
	}
</script>

<div class="subbar">
	<div class="chips">
		<button class="chip" class:on={feedStore.scope === 'mine'} onclick={() => setScope('mine')}>
			{#if feedStore.scope === 'mine'}<Check size={14} />{/if} Mine
		</button>
		<button class="chip" class:on={feedStore.scope === 'all'} onclick={() => setScope('all')}>
			{#if feedStore.scope === 'all'}<Check size={14} />{/if} All
		</button>
		<span class="spacer"></span>
		{#if liveCount > 0}
			<button class="chip live" class:on={liveOnly} onclick={() => (liveOnly = !liveOnly)}>
				<span class="dot"></span> Live · {liveCount}
			</button>
		{/if}
	</div>
	<div class="days" bind:this={stripEl}>
		{#each strip as d (d.key)}
			<button
				class="dayb"
				class:on={d.key === activeKey}
				class:today={d.isToday}
				disabled={!d.has}
				data-key={d.key}
				onclick={() => goDay(d.key)}
			>
				<span>{d.isToday ? 'Today' : d.weekday}</span>
				<b class="digits">{d.day}</b>
				{#if d.has}<i class="mark"></i>{/if}
			</button>
		{/each}
	</div>
</div>

<div class="matches">
	{#if feedStore.error && !feedStore.loaded}
		<div class="card empty">
			<p><b>Couldn't load your matches.</b></p>
			<p class="muted">{feedStore.error}</p>
			<button class="btn" onclick={() => feedStore.load().catch(() => {})}>Retry</button>
		</div>
	{:else if feedStore.loaded && days.length === 0}
		<div class="card empty">
			{#if liveOnly}
				<p><b>Nothing live right now.</b></p>
			{:else if feedStore.scope === 'mine'}
				<p><b>No matches yet.</b></p>
				<p class="muted">Play a competition and its matches show up here, day by day.</p>
				<a class="btn" href="/competitions">Browse competitions</a>
			{:else}
				<p><b>No matches in this window.</b></p>
			{/if}
		</div>
	{:else if feedStore.loaded}
		<button class="btn ghost more" onclick={() => feedStore.earlier()}>
			<ChevronUp size={16} /> Earlier results
		</button>

		{#each days as day (day.key)}
			<section class="day" id={`day-${day.key}`} data-key={day.key}>
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
	.subbar {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}
	.chips {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.chip {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		height: 34px;
		padding: 0 0.9rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--muted);
		font: inherit;
		font-weight: 700;
		font-size: 0.82rem;
		white-space: nowrap;
		cursor: pointer;
	}
	.chip.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.chip.live {
		color: var(--live);
		border-color: color-mix(in srgb, var(--live) 45%, var(--border));
	}
	.chip.live.on {
		background: var(--live);
		color: var(--bg);
	}
	.dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: currentColor;
		box-shadow: 0 0 8px currentColor;
	}
	.days {
		display: flex;
		gap: 6px;
		overflow-x: auto;
		scrollbar-width: none;
		margin: 0 calc(-1 * var(--shell-x, 1rem));
		padding: 0 var(--shell-x, 1rem);
		scroll-padding: 0 var(--shell-x, 1rem);
	}
	.days::-webkit-scrollbar {
		display: none;
	}
	.dayb {
		position: relative;
		flex: none;
		width: 52px;
		height: 50px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 2px;
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--muted);
		font: inherit;
		font-size: 0.7rem;
		font-weight: 600;
		cursor: pointer;
		padding: 0;
	}
	.dayb b {
		font-size: 0.95rem;
		color: var(--text);
	}
	.dayb:disabled {
		cursor: default;
		opacity: 0.55;
	}
	.dayb.today {
		color: var(--accent);
	}
	.dayb.on {
		background: var(--accent);
		color: var(--accent-fg);
	}
	.dayb.on b {
		color: var(--accent-fg);
	}
	.mark {
		position: absolute;
		bottom: 5px;
		width: 5px;
		height: 5px;
		border-radius: 50%;
		background: var(--accent);
	}
	.dayb.on .mark {
		background: var(--accent-fg);
	}
	.day + .day {
		margin-top: 0.4rem;
	}
	.day {
		/* land below the fixed top bar + the sticky chips/day strip */
		scroll-margin-top: calc(var(--topbar-h) + 7.4rem);
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
