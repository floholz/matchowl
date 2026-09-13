<!-- Matches: every match of the competitions you play ("Playing", default)
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
	import { goto, afterNavigate } from '$app/navigation';
	import MatchGroup from '$lib/components/MatchGroup.svelte';
	import MatchRow from '$lib/components/MatchRow.svelte';
	import MatchDetail from '$lib/components/MatchDetail.svelte';
	import { tick, untrack } from 'svelte';
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
			feedStore.load().catch(() => {});
			tournamentStore.ready().catch(() => {});
		}
	});
	// Land on today once the feed is there — whether this page loaded it or
	// Home did before we navigated here (then it is loaded on arrival).
	$effect(() => {
		if (feedStore.loaded && !scrolled) scrollToToday();
	});
	// Tapping the Matches nav item while already here: SvelteKit re-navigates
	// to the same URL and resets the scroll to the top (the oldest loaded
	// day). Treat it as "take me to today" instead.
	afterNavigate((nav) => {
		if (
			nav.type === 'link' &&
			nav.from?.url.pathname === '/matches' &&
			nav.to?.url.pathname === '/matches' &&
			!nav.to.url.search &&
			scrolled
		) {
			requestAnimationFrame(() => requestAnimationFrame(() => goDay(anchorKey, 'instant')));
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

	// Land on today (or the first upcoming day) once, after first render:
	// list and strip both jump, then `scrolled` switches the strip to smooth.
	async function scrollToToday() {
		if (scrolled) return;
		await tick();
		// SvelteKit resets the window to the top right after a client-side
		// navigation has rendered; two frames later that has happened and
		// our landing sticks (a fresh load never hit this — the fetch took
		// longer than the reset).
		await new Promise<void>((r) => requestAnimationFrame(() => requestAnimationFrame(() => r())));
		goDay(anchorKey, 'instant');
		stripEl
			?.querySelector<HTMLElement>(`[data-key="${anchorKey}"]`)
			?.scrollIntoView({ block: 'nearest', inline: 'center', behavior: 'instant' });
		scrolled = true;
	}
	/** Scroll so the day section starts right under the sticky chrome (top
	 *  bar + subbar) — measured, not guessed, so the follow logic (which
	 *  reads the same line) agrees on which day is current. */
	function goDay(key: string, behavior: ScrollBehavior = 'smooth') {
		const el = document.getElementById(`day-${key}`);
		if (!el) return;
		activeKey = key;
		const line = (document.querySelector('.subbar')?.getBoundingClientRect().bottom ?? 120) + 4;
		window.scrollTo({ top: el.getBoundingClientRect().top + window.scrollY - line, behavior });
	}
	/** Strip tap: an unloaded day grows the window out to it first; a day
	 *  without matches lands on the nearest day that has some. */
	async function tapDay(d: { key: string; offset: number; loaded: boolean }) {
		if (!d.loaded) {
			await feedStore.extendTo(d.offset).catch(() => {});
			await tick();
		}
		const keys = days.map((x) => x.key);
		const after = keys.find((k) => k >= d.key);
		const before = [...keys].reverse().find((k) => k <= d.key);
		// Prefer the direction of the tap, else whatever is nearest.
		const target = (d.offset >= 0 ? after ?? before : before ?? after) ?? '';
		if (!target) return;
		goDay(target, d.loaded ? 'smooth' : 'instant');
		// Freshly loaded rows settle (crests, cards) after the first jump.
		if (!d.loaded) setTimeout(() => goDay(target, 'instant'), 300);
	}
	// Infinite scroll forwards: the bottom loader fetches more as it comes
	// into view, once the list has landed on today. Earlier results stay a
	// tap (auto-loading upwards would fire on first paint and move the
	// landing off today). The button stays as the visible state.
	let bottomEl = $state<HTMLElement | null>(null);
	/** Load further fixtures when the bottom loader is within reach (600px
	 *  below the viewport counts — the next days arrive before you get
	 *  there) and nothing is loading. */
	function maybeLoadLater() {
		const el = bottomEl;
		if (!el || !scrolled || feedStore.loading || !feedStore.canLater) return;
		if (el.getBoundingClientRect().top < window.innerHeight + 600) feedStore.later().catch(() => {});
	}
	$effect(() => {
		const el = bottomEl;
		if (!el || !scrolled) return;
		const io = new IntersectionObserver(() => maybeLoadLater(), { rootMargin: '0px 0px 600px 0px' });
		io.observe(el);
		return () => io.disconnect();
	});
	// Infinite scroll backwards, the same way, once the list has landed on
	// today. Days get inserted above the viewport, which would shift what
	// you are looking at; the scroll is compensated by the added height by
	// hand (Chrome's scroll anchoring would do it, Safari has none — see the
	// overflow-anchor rule on the column).
	let topEl = $state<HTMLElement | null>(null);
	/** Bottom edge of the sticky chrome in viewport coordinates. */
	function chromeLine() {
		return (document.querySelector('.subbar')?.getBoundingClientRect().bottom ?? 120) + 4;
	}
	/** One upward load at a time, compensation included: the store's own
	 *  `loading` flag drops before the scroll is put back, and a second load
	 *  started in that gap measured its anchor against a half-moved page. */
	let earlierBusy = false;
	async function loadEarlier() {
		if (earlierBusy || feedStore.loading || !feedStore.canEarlier) return;
		earlierBusy = true;
		try {
			// Anchor on the first day section still on screen: after the load
			// it is put back exactly where it was, whatever got inserted above.
			const line = chromeLine();
			const anchor = Array.from(document.querySelectorAll<HTMLElement>('section.day')).find(
				(s) => s.getBoundingClientRect().bottom > line
			);
			const key = anchor?.dataset.key;
			const top = anchor?.getBoundingClientRect().top ?? 0;
			await feedStore.earlier().catch(() => {});
			await tick();
			const el = key ? document.getElementById(`day-${key}`) : null;
			if (el) window.scrollBy({ top: el.getBoundingClientRect().top - top, behavior: 'instant' });
		} finally {
			earlierBusy = false;
		}
		// Still in reach after the page was put back? Keep filling.
		maybeLoadEarlier();
	}
	function maybeLoadEarlier() {
		const el = topEl;
		if (!el || !scrolled || earlierBusy || feedStore.loading || !feedStore.canEarlier) return;
		if (el.getBoundingClientRect().bottom > chromeLine()) loadEarlier();
	}
	// The loader counts as "in view" only below the sticky chrome (a negative
	// top margin): on landing it sits right above today, under the bar, and
	// must not fire until you actually scroll up into it.
	$effect(() => {
		const el = topEl;
		if (!el || !scrolled) return;
		const io = new IntersectionObserver(() => maybeLoadEarlier(), {
			rootMargin: `-${Math.round(chromeLine())}px 0px 0px 0px`
		});
		io.observe(el);
		return () => io.disconnect();
	});
	// An intersection that happened while another load was running (a scope
	// switch, a far strip tap) was swallowed — the observer only reports
	// crossings. Look again whenever a load settles; that also keeps filling
	// until both loaders are out of reach.
	$effect(() => {
		if (!feedStore.loading)
			untrack(() => {
				maybeLoadLater();
				maybeLoadEarlier();
			});
	});
	// Follow the scroll: the topmost day section below the sticky chrome.
	$effect(() => {
		if (!feedStore.loaded) return;
		const onScroll = () => {
			// Query every time: days get inserted above and below as you scroll.
			const sections = Array.from(document.querySelectorAll<HTMLElement>('section.day'));
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
	// Keep the highlighted day visible in the strip. Until the list has
	// landed on today the strip jumps (no animated scroll across weeks on
	// first paint); afterwards it glides along with the list.
	$effect(() => {
		const key = activeKey;
		const el = stripEl?.querySelector<HTMLElement>(`[data-key="${key}"]`);
		el?.scrollIntoView({
			block: 'nearest',
			inline: 'center',
			behavior: scrolled ? 'smooth' : 'instant'
		});
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
	<button class="raillink" class:on={feedStore.scope === 'mine' && !liveOnly && !tid} onclick={() => { tid = ''; liveOnly = false; setScope('mine'); }}><Check size={16} /> Playing</button>
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
			{#if feedStore.scope === 'mine'}<Check size={14} />{/if} Playing
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
				class:unloaded={!d.loaded}
				class:none={d.loaded && !d.has}
				data-key={d.key}
				onclick={() => tapDay(d)}
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
		{#if feedStore.canEarlier}
			<button class="btn ghost more" bind:this={topEl} onclick={loadEarlier} disabled={feedStore.loading}>
				<ChevronUp size={16} /> {feedStore.loading ? 'Loading…' : 'Earlier results'}
			</button>
		{/if}

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

		{#if feedStore.canLater}
			<button class="btn ghost more" bind:this={bottomEl} onclick={() => feedStore.later()} disabled={feedStore.loading}>
				<ChevronDown size={16} /> {feedStore.loading ? 'Loading…' : 'Later fixtures'}
			</button>
		{/if}
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
		/* Earlier days are inserted above the viewport; the page compensates
		   the scroll itself (loadEarlier), so the browser must not also. */
		overflow-anchor: none;
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
	/* Loaded but empty: dim. Not loaded yet: dimmer, still a tap away. */
	.dayb.none {
		opacity: 0.55;
	}
	.dayb.unloaded {
		opacity: 0.35;
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
