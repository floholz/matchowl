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
	import { media } from '$lib/media.svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import MatchGroup from '$lib/components/MatchGroup.svelte';
	import MatchRow from '$lib/components/MatchRow.svelte';
	import MatchDetail from '$lib/components/MatchDetail.svelte';
	import { tick } from 'svelte';
	import { ChevronUp, ChevronDown, ChevronLeft, ChevronRight, Check, Globe, Radio } from '@lucide/svelte';

	pageChrome(() => ({ title: 'Matches', wide: true }));

	// Desktop: ?m= opens the match in the detail panel beside the list.
	// Below the desktop rule the same link is the match page route.
	let selected = $derived($page.url.searchParams.get('m') ?? '');
	function select(id: string) {
		if (media.panel) goto(`/matches?m=${id}`, { noScroll: true, keepFocus: true });
		else goto(`/m/${id}`);
	}
	function closePanel() {
		goto('/matches', { replaceState: true, noScroll: true, keepFocus: true });
	}
	$effect(() => {
		if (selected && !media.panel) goto(`/m/${selected}`, { replaceState: true });
	});
	/** Rail: one competition only ('' = all in scope). */
	let tid = $state('');

	let openId = $state('');
	let scrolled = $state(false);
	let liveOnly = $state(false);
	/** Day strip highlight: the day section currently at the top. */
	let activeKey = $state('');
	let stripEl = $state<HTMLElement | null>(null);
	/** Which edge today's button has scrolled past in the strip ('' = visible). */
	let todayOff = $state<'' | 'left' | 'right'>('');

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
		if (!liveOnly && !tid) return feedStore.days;
		const keep = (m: FeedMatch) => (!liveOnly || m.status === 'live') && (!tid || m.tournament.id === tid);
		return feedStore.days
			.map((d) => ({
				...d,
				matches: d.matches.filter(keep),
				groups: d.groups
					.map((g) => ({ ...g, matches: g.matches.filter(keep) }))
					.filter((g) => g.matches.length)
			}))
			.filter((d) => d.matches.length);
	});
	/** Rail lists: what you play, and the rest (opens the "all" scope). */
	let notPlaying = $derived(
		tournamentStore.list.filter(
			(t) => t.status !== 'draft' && !feedStore.playing.some((p) => p.id === t.id)
		)
	);
	async function pickTournament(id: string, playing: boolean) {
		tid = tid === id ? '' : id;
		if (tid && !playing && feedStore.scope !== 'all') await setScope('all');
	}
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
	// Today is the strip's anchor: once its button scrolls out of the strip,
	// a "Today" chip pins to that edge so it is always one tap away.
	$effect(() => {
		const strip = stripEl;
		if (!strip) return;
		const check = () => {
			const today = strip.querySelector<HTMLElement>('.dayb.today');
			if (!today) return (todayOff = '');
			const s = strip.getBoundingClientRect();
			const t = today.getBoundingClientRect();
			todayOff = t.right < s.left + 8 ? 'left' : t.left > s.right - 8 ? 'right' : '';
		};
		strip.addEventListener('scroll', check, { passive: true });
		const ro = new ResizeObserver(check);
		ro.observe(strip);
		check();
		return () => {
			strip.removeEventListener('scroll', check);
			ro.disconnect();
		};
	});
	function goToday() {
		if (days.some((d) => d.key === feedStore.todayKey)) goDay(feedStore.todayKey);
		else {
			goDay(anchorKey);
			stripEl
				?.querySelector<HTMLElement>('.dayb.today')
				?.scrollIntoView({ block: 'nearest', inline: 'center', behavior: 'smooth' });
		}
	}

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

<div class="layout" class:withpanel={media.panel && !!selected}>
<aside class="rail">
	<div class="railh">Show</div>
	<button class="raillink" class:on={feedStore.scope === 'mine' && !liveOnly && !tid} onclick={() => { tid = ''; liveOnly = false; setScope('mine'); }}><Check size={16} /> My competitions</button>
	<button class="raillink" class:on={feedStore.scope === 'all' && !liveOnly && !tid} onclick={() => { tid = ''; liveOnly = false; setScope('all'); }}><Globe size={16} /> Everything</button>
	{#if liveCount > 0}
		<button class="raillink live" class:on={liveOnly} onclick={() => (liveOnly = !liveOnly)}><Radio size={16} /> Live · {liveCount}</button>
	{/if}
	{#if feedStore.playing.length}
		<div class="railh">Playing</div>
		{#each feedStore.playing as t (t.id)}
			<button class="raillink" class:on={tid === t.id} onclick={() => pickTournament(t.id, true)}>{t.shortName || t.name}</button>
		{/each}
	{/if}
	{#if notPlaying.length}
		<div class="railh">Not playing</div>
		{#each notPlaying as t (t.id)}
			<button class="raillink dim" class:on={tid === t.id} onclick={() => pickTournament(t.id, false)}>{t.shortName || t.name}</button>
		{/each}
	{/if}
</aside>
<div class="col">
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
	<div class="dayswrap" class:offl={todayOff === 'left'} class:offr={todayOff === 'right'}>
	{#if todayOff}
		<button class="todaypin {todayOff}" onclick={goToday} aria-label="Back to today">
			{#if todayOff === 'left'}<ChevronLeft size={14} />{/if}
			Today
			{#if todayOff === 'right'}<ChevronRight size={14} />{/if}
		</button>
	{/if}
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
								onSelect={() => select(m.id)}
								selected={selected === m.id}
								leg={m.leg ? otherLegView(m, m.leg, m.leg.first, `/m/${m.leg.id}`) : null}
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
</div>
{#if media.panel && selected}
	<aside class="panel">
		<MatchDetail id={selected} onClose={closePanel} onSaved={(t) => feedStore.applyTip(selected, t)} />
	</aside>
{/if}
</div>

<style>
	/* ---- desktop: rail · list · panel ----
	   900–1099: nav only, a match opens as its route.
	   1100–1359: list · panel (the chips row stays, no rail).
	   ≥ 1360: rail · list (≤ 880px) · panel, centred as a set.
	   ≥ 1984: rail + list stay put when the panel opens (room on the right). */
	.rail,
	.panel {
		display: none;
	}
	.col {
		min-width: 0;
	}
	@media (min-width: 900px) {
		.layout {
			display: grid;
			grid-template-columns: minmax(0, 880px);
			justify-content: center;
			gap: 1.5rem;
			align-items: start;
		}
		.subbar {
			margin-left: 0;
			margin-right: 0;
			padding-left: 0;
			padding-right: 0;
			margin-top: calc(-1 * var(--shell-gap, 2rem));
		}
		.days {
			margin: 0;
			padding: 0;
		}
		.day {
			scroll-margin-top: calc(var(--topbar-h) + 7.4rem);
		}
	}
	@media (min-width: 1100px) {
		.layout.withpanel {
			grid-template-columns: minmax(0, 880px) 380px;
		}
		.panel {
			display: block;
			position: sticky;
			top: calc(var(--topbar-h) + var(--shell-gap, 2rem));
			max-height: calc(100vh - var(--topbar-h) - 2 * var(--shell-gap, 2rem));
			overflow-y: auto;
			padding: 0.9rem 1rem 1.2rem;
			background:
				linear-gradient(180deg, rgba(255, 255, 255, 0.025), transparent 40%),
				var(--surface);
			border: 1px solid var(--border);
			border-radius: var(--radius);
		}
		.layout.withpanel .panel {
			display: block;
		}
	}
	@media (min-width: 1360px) {
		.layout {
			grid-template-columns: var(--rail-w) minmax(0, 880px);
		}
		.layout.withpanel {
			grid-template-columns: var(--rail-w) minmax(0, 880px) 380px;
		}
		.rail {
			display: block;
			position: sticky;
			top: calc(var(--topbar-h) + var(--shell-gap, 2rem));
		}
		/* The rail replaces the chips; the day strip stays above the list. */
		.subbar .chips {
			display: none;
		}
		.day {
			scroll-margin-top: calc(var(--topbar-h) + 4.6rem);
		}
	}
	@media (min-width: 1984px) {
		/* Closed set = 240 + 24 + 880 = 1144px, centred. Keep that left
		   edge when the panel opens; it fits from 1984px up. */
		.layout.withpanel {
			justify-content: start;
			padding-left: calc((100% - 1144px) / 2);
		}
	}
	.railh {
		font-size: 0.66rem;
		font-weight: 700;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--muted);
		margin: 1.1rem 0.75rem 0.35rem;
	}
	.railh:first-child {
		margin-top: 0.2rem;
	}
	.raillink {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		width: 100%;
		padding: 0.55rem 0.75rem;
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--muted);
		font: inherit;
		font-weight: 600;
		font-size: 0.86rem;
		text-align: left;
		cursor: pointer;
	}
	.raillink:hover {
		color: var(--text);
		background: var(--surface-2);
	}
	.raillink.on {
		color: var(--accent);
		background: color-mix(in srgb, var(--accent) 12%, transparent);
	}
	.raillink.live {
		color: var(--live);
	}
	.raillink.dim {
		opacity: 0.75;
	}

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
	.dayswrap {
		position: relative;
	}
	/* Fade the strip under the pinned chip so it reads as an overlay. */
	.dayswrap::before,
	.dayswrap::after {
		content: '';
		position: absolute;
		top: 0;
		bottom: 0;
		width: 84px;
		pointer-events: none;
		opacity: 0;
		transition: opacity 0.15s ease;
		z-index: 1;
	}
	.dayswrap::before {
		left: calc(-1 * var(--shell-x, 1rem));
		background: linear-gradient(90deg, var(--bg) 40%, transparent);
	}
	.dayswrap::after {
		right: calc(-1 * var(--shell-x, 1rem));
		background: linear-gradient(270deg, var(--bg) 40%, transparent);
	}
	.dayswrap.offl::before,
	.dayswrap.offr::after {
		opacity: 0.9;
	}
	.todaypin {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		z-index: 2;
		display: inline-flex;
		align-items: center;
		gap: 0.15rem;
		height: 30px;
		padding: 0 0.6rem;
		border: 1px solid color-mix(in srgb, var(--accent) 55%, var(--border));
		border-radius: var(--radius-pill);
		background: var(--surface);
		color: var(--accent);
		font: inherit;
		font-size: 0.72rem;
		font-weight: 800;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		cursor: pointer;
		box-shadow: var(--shadow-pop);
	}
	.todaypin.left {
		left: 0;
	}
	.todaypin.right {
		right: 0;
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
