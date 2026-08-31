<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { pb } from '$lib/pb';
	import { auth } from '$lib/auth.svelte';
	import {
		tournamentStore,
		defaultSeason,
		competitionLogoUrl,
		type Tournament
	} from '$lib/tournament.svelte';
	import { tipsStore, type Match } from '$lib/tips.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { describeSeason, dateSpan } from '$lib/describe';
	import TipCard from '$lib/components/TipCard.svelte';
	import MatchList from '$lib/components/MatchList.svelte';
	import Standings from '$lib/components/Standings.svelte';
	import { Check, Plus, Telescope, ChevronDown, ChevronRight } from '@lucide/svelte';

	type Tab = 'overview' | 'matches' | 'standings';
	const TABS: Tab[] = ['overview', 'matches', 'standings'];

	let key = $derived($page.params.key ?? '');
	let sParam = $derived($page.url.searchParams.get('s') ?? '');
	/** Deep link to one match on the Matches tab (e.g. "view the other leg"). */
	let mParam = $derived($page.url.searchParams.get('m') ?? '');
	let tab = $derived.by<Tab>(() => {
		const t = $page.url.searchParams.get('tab') ?? '';
		return (TABS as string[]).includes(t) ? (t as Tab) : 'overview';
	});

	let ready = $state(false);
	let playing = $state<Set<string>>(new Set());
	let busy = $state(false);

	$effect(() => {
		tournamentStore.ready().then(() => (ready = true));
		if (auth.isAuthed)
			pb.send('/api/me/tournaments', { method: 'GET' })
				.then((r) => (playing = new Set((r.tournaments ?? []).map((t: Tournament) => t.id))))
				.catch(() => {});
	});

	let seasons = $derived(ready ? tournamentStore.seasonsOf(key) : []);
	let competition = $derived(seasons[0]?.competition ?? null);
	/** The selected season: ?s= when it names one of ours, else the
	 *  default (running > next upcoming > latest). */
	let season = $derived(seasons.find((t) => t.slug === sParam) ?? defaultSeason(seasons));

	// Point the shared stores at the selected season and load its fixtures.
	$effect(() => {
		const s = season;
		if (!s) return;
		tournamentStore.select(s.slug);
		tipsStore.load().catch(() => {});
	});

	/** Fixtures in the store belong to the selected season (the store is
	 *  shared and may still hold the previous one while loading). */
	let loaded = $derived(!!season && tipsStore.loaded && tournamentStore.current?.id === season.id);

	function href(s: string, t: Tab): string {
		const q = new URLSearchParams();
		if (s) q.set('s', s);
		if (t !== 'overview') q.set('tab', t);
		const qs = q.toString();
		return `/competitions/${key}${qs ? `?${qs}` : ''}`;
	}
	function setTab(t: Tab) {
		goto(href(sParam, t), { replaceState: true, noScroll: true, keepFocus: true });
	}
	function pickSeason(slug: string) {
		goto(href(slug, tab), { noScroll: true, keepFocus: true });
	}

	// Swipe between tabs (touch only; a horizontal flick wider than it is tall).
	let sx = 0;
	let sy = 0;
	function touchStart(e: TouchEvent) {
		sx = e.touches[0].clientX;
		sy = e.touches[0].clientY;
	}
	function touchEnd(e: TouchEvent) {
		const dx = e.changedTouches[0].clientX - sx;
		const dy = e.changedTouches[0].clientY - sy;
		if (Math.abs(dx) < 70 || Math.abs(dx) < Math.abs(dy) * 1.5) return;
		const next = TABS[TABS.indexOf(tab) + (dx < 0 ? 1 : -1)];
		if (next) setTab(next);
	}

	async function togglePlay() {
		if (!season) return;
		busy = true;
		try {
			const was = playing.has(season.id);
			await pb.send(`/api/tournaments/${season.slug}/play`, { method: was ? 'DELETE' : 'POST' });
			if (was) playing.delete(season.id);
			else playing.add(season.id);
			playing = new Set(playing);
		} finally {
			busy = false;
		}
	}

	// ---- overview ----
	function played(m: Match) {
		return m.status === 'finished' || !!m.finalizedAt;
	}
	let teamCount = $derived(loaded ? Object.keys(tipsStore.teams).length : 0);
	let description = $derived(
		competition?.description || (season ? describeSeason(season, teamCount) : '')
	);

	/** What to put up top: live matches, else the next kickoffs (same
	 *  day, max 3), else — season over — the final. */
	let spotlight = $derived.by<{ label: string; matches: Match[] }>(() => {
		if (!loaded) return { label: '', matches: [] };
		const ms = tipsStore.matches;
		const live = ms.filter((m) => m.status === 'live');
		if (live.length) return { label: 'Live now', matches: live.slice(0, 3) };
		const now = serverClock.now();
		const upcoming = ms.filter((m) => new Date(m.kickoff).getTime() >= now && !played(m));
		if (upcoming.length) {
			const day = new Date(upcoming[0].kickoff).toDateString();
			return {
				label: 'Next up',
				matches: upcoming.filter((m) => new Date(m.kickoff).toDateString() === day).slice(0, 3)
			};
		}
		const finals = ms.filter((m) => m.stage === tournamentStore.championStageCode);
		if (finals.length) return { label: 'The final', matches: finals.slice(-1) };
		return { label: '', matches: [] };
	});
	let openId = $state('');

	let myPoints = $derived(
		loaded
			? tipsStore.matches.reduce(
					(sum, m) => sum + (tipsStore.tips[m.id] ? (tipsStore.scores[m.id] ?? 0) : 0),
					0
				)
			: 0
	);
	let myTips = $derived(loaded ? Object.keys(tipsStore.tips).length : 0);
	let playedCount = $derived(loaded ? tipsStore.matches.filter(played).length : 0);
	let hasForecast = $derived(!!season?.forecastSpec?.mode && season.forecastSpec.mode !== 'none');

	const statusLabel: Record<string, string> = {
		active: 'live',
		upcoming: 'upcoming',
		finished: 'finished',
		archived: 'archived',
		draft: 'draft'
	};
	function initials(name: string): string {
		return name
			.split(/\s+/)
			.filter((w) => /^[A-Z0-9]/.test(w))
			.map((w) => w[0])
			.join('')
			.slice(0, 3);
	}
	let isPlaying = $derived(!!season && playing.has(season.id));
</script>

{#if ready && !competition}
	<div class="card miss muted">
		No such competition. <a href="/competitions">Back to competitions</a>
	</div>
{:else if competition && season}
	<div class="hub">
		<header class="hero stagger">
			<div class="logo" class:ph={!competitionLogoUrl(competition)}>
				{#if competitionLogoUrl(competition)}
					<img src={competitionLogoUrl(competition)} alt="" />
				{:else}
					{initials(competition.name)}
				{/if}
			</div>
			<div class="htxt">
				<p class="kicker">
					{competition.country && competition.country !== 'World' ? `${competition.country} · ` : ''}{competition.teamKind ===
					'club'
						? 'Clubs'
						: 'National teams'}
				</p>
				<h1>{competition.name}</h1>
				<div class="seasonrow">
					<label class="season" class:single={seasons.length < 2}>
						<select
							value={season.slug}
							disabled={seasons.length < 2}
							onchange={(e) => pickSeason((e.currentTarget as HTMLSelectElement).value)}
							aria-label="Season"
						>
							{#each seasons as s (s.id)}
								<option value={s.slug}>{s.name}</option>
							{/each}
						</select>
						<ChevronDown size={14} />
					</label>
					<span class="pill" class:live={season.status === 'active'}
						>{statusLabel[season.status] ?? season.status}</span
					>
				</div>
			</div>
		</header>

		<p class="desc">{description}</p>
		{#if !description.includes(dateSpan(season.startsAt, season.endsAt))}
			<p class="muted dates">{dateSpan(season.startsAt, season.endsAt)}</p>
		{/if}

		{#if auth.isAuthed && (season.status !== 'archived' || isPlaying)}
			<button class="btn play" class:secondary={isPlaying} disabled={busy} onclick={togglePlay}>
				{#if isPlaying}<Check size={18} /> Playing — matches in your feed{:else}<Plus size={18} /> Play
					{season.shortName || season.name}{/if}
			</button>
		{/if}

		<div class="tabbar">
			<div class="seg" role="tablist">
				<button role="tab" aria-selected={tab === 'overview'} class:on={tab === 'overview'} onclick={() => setTab('overview')}>Overview</button>
				<button role="tab" aria-selected={tab === 'matches'} class:on={tab === 'matches'} onclick={() => setTab('matches')}>Matches</button>
				<button role="tab" aria-selected={tab === 'standings'} class:on={tab === 'standings'} onclick={() => setTab('standings')}
					>{tournamentStore.singleTable && !tournamentStore.knockoutStages.length ? 'Table' : 'Standings'}</button
				>
			</div>
		</div>

		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="pane" ontouchstart={touchStart} ontouchend={touchEnd}>
			{#if tab === 'overview'}
				{#if !loaded}
					<p class="muted">Loading…</p>
				{:else}
					{#if spotlight.matches.length}
						<h2 class="sec">{spotlight.label}</h2>
						{#each spotlight.matches as m (m.id)}
							<div class="match">
								<TipCard
									match={m}
									open={openId === m.id}
									onToggle={() => (openId = openId === m.id ? '' : m.id)}
								/>
							</div>
						{/each}
					{/if}

					{#if auth.isAuthed}
						<div class="stats">
							<div class="stat card">
								<span class="num digits">{myPoints}</span>
								<span class="lbl">Your points</span>
							</div>
							<div class="stat card">
								<span class="num digits">{myTips}<small>/{tipsStore.matches.length}</small></span>
								<span class="lbl">Tips placed</span>
							</div>
							<div class="stat card">
								<span class="num digits">{playedCount}<small>/{tipsStore.matches.length}</small></span>
								<span class="lbl">Matches played</span>
							</div>
						</div>
					{/if}

					{#if hasForecast}
						<a class="card fc" href={`/forecast?t=${season.slug}`}>
							<span class="fc-ic"><Telescope size={20} /></span>
							<span class="fc-txt">
								<b>Forecast</b>
								<span class="muted">Your one-shot call on the whole season</span>
							</span>
							<ChevronRight size={18} />
						</a>
					{/if}

					<div class="sechead">
						<h2 class="sec">Standings</h2>
						<button class="link" onclick={() => setTab('standings')}>Full view <ChevronRight size={14} /></button>
					</div>
					<Standings compact />
				{/if}
			{:else if tab === 'matches'}
				<MatchList focusId={mParam} />
			{:else}
				<Standings />
			{/if}
		</div>
	</div>
{:else}
	<p class="muted">Loading…</p>
{/if}

<style>
	.miss {
		text-align: center;
		padding: 2rem;
	}
	.hero {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin: 0.4rem 0 0.9rem;
	}
	.logo {
		flex: none;
		width: 72px;
		height: 72px;
		display: grid;
		place-items: center;
		border-radius: 18px;
		background: #fff;
		overflow: hidden;
		box-shadow: var(--shadow-pop);
	}
	.logo img {
		width: 82%;
		height: 82%;
		object-fit: contain;
	}
	.logo.ph {
		background: var(--surface-2);
		color: var(--muted);
		font-family: var(--font-display);
		font-size: 1.3rem;
		letter-spacing: 0.04em;
		box-shadow: none;
	}
	.htxt {
		min-width: 0;
		flex: 1;
	}
	.htxt h1 {
		margin: 0.1rem 0 0.4rem;
		font-size: 1.55rem;
		line-height: 1.1;
	}
	.seasonrow {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}
	.season {
		position: relative;
		display: inline-flex;
		align-items: center;
		color: var(--text);
	}
	.season select {
		appearance: none;
		-webkit-appearance: none;
		padding: 0.3rem 1.6rem 0.3rem 0.7rem;
		font: inherit;
		font-weight: 700;
		font-size: 0.85rem;
		color: inherit;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		cursor: pointer;
		max-width: 60vw;
		text-overflow: ellipsis;
	}
	.season :global(svg) {
		position: absolute;
		right: 0.55rem;
		pointer-events: none;
		color: var(--muted);
	}
	.season.single select {
		padding-right: 0.7rem;
		cursor: default;
	}
	.season.single :global(svg) {
		display: none;
	}
	.desc {
		margin: 0 0 0.25rem;
		font-size: 0.95rem;
		line-height: 1.45;
	}
	.dates {
		margin: 0;
		font-size: 0.85rem;
	}
	.btn.play {
		margin: 0.9rem 0 0;
	}
	/* Tab strip sticks under the top bar; the panes scroll beneath it. */
	.tabbar {
		position: sticky;
		top: var(--topbar-h);
		z-index: 20;
		margin: 0.9rem -1rem 0.9rem;
		padding: 0.5rem 1rem;
		background: color-mix(in srgb, var(--bg) 86%, transparent);
		backdrop-filter: blur(12px) saturate(1.3);
	}
	@media (min-width: 900px) {
		.tabbar {
			top: 0;
			margin: 0.9rem -2rem 0.9rem;
			padding: 0.6rem 2rem;
		}
	}
	.pane {
		min-height: 40vh;
	}
	.sec {
		font-size: 0.78rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
		margin: 1.2rem 0 0.6rem;
	}
	.sec:first-child {
		margin-top: 0;
	}
	.sechead {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
	}
	.link {
		display: inline-flex;
		align-items: center;
		gap: 0.1rem;
		background: none;
		border: none;
		padding: 0;
		color: var(--accent);
		font: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		cursor: pointer;
	}
	.match + .match {
		margin-top: 6px;
	}
	.stats {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.6rem;
		margin-top: 1rem;
	}
	.stat {
		padding: 0.75rem 0.6rem;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.15rem;
		text-align: center;
	}
	.stat .num {
		font-size: 1.35rem;
		font-weight: 800;
		color: var(--accent);
		line-height: 1;
	}
	.stat .num small {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--muted);
	}
	.stat .lbl {
		font-size: 0.66rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.fc {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		margin-top: 0.85rem;
		color: var(--text);
		text-decoration: none;
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
		background:
			linear-gradient(135deg, color-mix(in srgb, var(--accent) 10%, transparent), transparent 55%),
			var(--surface);
	}
	.fc-ic {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		border-radius: 50%;
		flex: none;
		color: var(--accent-fg);
		background: var(--accent);
	}
	.fc-txt {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		flex: 1;
		min-width: 0;
	}
	.fc-txt .muted {
		font-size: 0.85rem;
	}
</style>
