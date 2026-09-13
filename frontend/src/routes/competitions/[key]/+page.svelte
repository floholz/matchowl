<!-- Competition hub: compact header (back · crest · name · season · Playing)
     drawn into the mobile top bar, underline tabs Overview · Matches · Table ·
     Knockout · Forecast (only those that apply), one pane each. -->
<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { pb } from '$lib/pb';
	import { auth } from '$lib/auth.svelte';
	import {
		tournamentStore,
		defaultSeason,
		competitionLogoUrl,
		seasonLabel,
		type Tournament
	} from '$lib/tournament.svelte';
	import { tipsStore, type Match } from '$lib/tips.svelte';
	import { forecastStore } from '$lib/forecast.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { describeSeason, dateSpan } from '$lib/describe';
	import PageChrome from '$lib/components/PageChrome.svelte';
	import { shell } from '$lib/shell.svelte';
	import { feedStore } from '$lib/feed.svelte';
	import MatchRow from '$lib/components/MatchRow.svelte';
	import MatchList from '$lib/components/MatchList.svelte';
	import Standings from '$lib/components/Standings.svelte';
	import KnockoutTies from '$lib/components/KnockoutTies.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import { Check, Plus, Target, ChevronDown, ChevronRight, ChevronLeft, Lock } from '@lucide/svelte';

	type Tab = 'overview' | 'matches' | 'table' | 'knockout' | 'forecast';

	let key = $derived($page.params.key ?? '');
	let sParam = $derived($page.url.searchParams.get('s') ?? '');
	/** Deep link to one match on the Matches tab. */
	let mParam = $derived($page.url.searchParams.get('m') ?? '');

	let ready = $state(false);
	let playing = $state<Set<string>>(new Set());
	let busy = $state(false);

	$effect(() => {
		tournamentStore.ready().then(() => (ready = true));
	});
	// Your played competitions — re-read after every saved tip, since the
	// first tip on a season auto-subscribes you to it.
	$effect(() => {
		void tipsStore.saved;
		if (!auth.isAuthed) return;
		pb.send('/api/me/tournaments', { method: 'GET' })
			.then((r) => (playing = new Set((r.tournaments ?? []).map((t: Tournament) => t.id))))
			.catch(() => {});
	});
	function goBack(e: MouseEvent) {
		if (!shell.hasFrom) return;
		e.preventDefault();
		history.back();
	}

	let seasons = $derived(ready ? tournamentStore.seasonsOf(key) : []);
	let competition = $derived(seasons[0]?.competition ?? null);
	let season = $derived(seasons.find((t) => t.slug === sParam) ?? defaultSeason(seasons));

	// Point the shared stores at the selected season and load its fixtures.
	$effect(() => {
		const s = season;
		if (!s) return;
		tournamentStore.select(s.slug);
		tipsStore.load().catch(() => {});
	});
	let loaded = $derived(!!season && tipsStore.loaded && tournamentStore.current?.id === season.id);

	// ---- tabs: only those that apply ----
	let hasTable = $derived(tournamentStore.groupStageCode !== '');
	let hasKnockout = $derived(tournamentStore.knockoutStages.length > 0);
	let hasForecast = $derived(!!season?.forecastSpec?.mode && season.forecastSpec.mode !== 'none');
	let tabs = $derived.by((): { id: Tab; label: string }[] => {
		const out: { id: Tab; label: string }[] = [
			{ id: 'overview', label: 'Overview' },
			{ id: 'matches', label: 'Matches' }
		];
		if (hasTable) out.push({ id: 'table', label: tournamentStore.singleTable ? 'Table' : 'Groups' });
		if (hasKnockout) out.push({ id: 'knockout', label: 'Knockout' });
		if (hasForecast) out.push({ id: 'forecast', label: 'Forecast' });
		return out;
	});
	let tab = $derived.by((): Tab => {
		let t = $page.url.searchParams.get('tab') ?? '';
		if (t === 'standings') t = hasTable ? 'table' : 'knockout';
		return tabs.some((x) => x.id === t) ? (t as Tab) : 'overview';
	});
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
		const i = tabs.findIndex((x) => x.id === tab);
		const next = tabs[i + (dx < 0 ? 1 : -1)];
		if (next) setTab(next.id);
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
			// Home and Matches read the feed; make them follow.
			if (feedStore.loaded) feedStore.load().catch(() => {});
		} finally {
			busy = false;
		}
	}
	let isPlaying = $derived(!!season && playing.has(season.id));
	let canPlay = $derived(auth.isAuthed && !!season && (season.status !== 'archived' || isPlaying));

	// ---- overview ----
	const played = (m: Match) => m.status === 'finished' || !!m.finalizedAt;
	let teamCount = $derived(loaded ? Object.keys(tipsStore.teams).length : 0);
	let description = $derived(competition?.description || (season ? describeSeason(season, teamCount) : ''));
	/** Up to three rows: live first, else the next kick-offs, else the final. */
	let spotlight = $derived.by<{ label: string; sub: string; matches: Match[] }>(() => {
		if (!loaded) return { label: '', sub: '', matches: [] };
		const ms = tipsStore.matches;
		const live = ms.filter((m) => m.status === 'live');
		if (live.length) return { label: 'Live now', sub: '', matches: live.slice(0, 3) };
		const now = serverClock.now();
		const upcoming = ms.filter((m) => new Date(m.kickoff).getTime() >= now && !played(m));
		if (upcoming.length) {
			const first = upcoming[0];
			const day = new Date(first.kickoff);
			const sub = `${first.roundLabel} · ${day.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' })}`;
			return {
				label: 'Next up',
				sub,
				matches: upcoming.filter((m) => new Date(m.kickoff).toDateString() === day.toDateString()).slice(0, 3)
			};
		}
		const finals = ms.filter((m) => m.stage === tournamentStore.championStageCode);
		if (finals.length) return { label: 'The final', sub: '', matches: finals.slice(-1) };
		return { label: '', sub: '', matches: [] };
	});
	let openId = $state('');
	let myPoints = $derived(
		loaded ? tipsStore.matches.reduce((sum, m) => sum + (tipsStore.tips[m.id] ? (tipsStore.scores[m.id] ?? 0) : 0), 0) : 0
	);
	let myTips = $derived(loaded ? Object.keys(tipsStore.tips).length : 0);
	let exact = $derived(
		loaded
			? tipsStore.matches.filter((m) => {
					const t = tipsStore.tips[m.id];
					return t && played(m) && t.ftHome === m.ftHome && t.ftAway === m.ftAway;
				}).length
			: 0
	);

	// ---- forecast summary (overview card + Forecast tab) ----
	let fcLoaded = $derived(hasForecast && forecastStore.loaded && forecastStore.loadedFor === season?.id);
	$effect(() => {
		if (!hasForecast || !loaded || !auth.isAuthed) return;
		forecastStore.load().catch(() => {});
	});
	let fcCalls = $derived(forecastStore.spec.mode === 'calls' ? (forecastStore.spec.calls ?? []) : []);
	const picksOf = (k: string): string[] => {
		const v = forecastStore.calls[k];
		return Array.isArray(v) ? v : v ? [v] : [];
	};
	let fcPlaced = $derived(fcCalls.filter((c) => picksOf(c.key).length > 0).length);
	let fcHas = $derived(!!forecastStore.recId);
	let fcLockText = $derived.by(() => {
		if (!fcLoaded) return '';
		if (forecastStore.locked) return 'Locked';
		const at = forecastStore.tournamentStart ? new Date(forecastStore.tournamentStart) : null;
		if (!at) return 'Locks at kick-off';
		const ms = at.getTime() - serverClock.now();
		const d = Math.floor(ms / 86400_000);
		const h = Math.floor((ms % 86400_000) / 3600_000);
		const when = at.toLocaleString(undefined, { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
		return ms > 0 ? `Locks ${when} · ${d > 0 ? `${d}d ${h}h` : `${h}h`}` : 'Locks at kick-off';
	});
	let fcHref = $derived(season ? `/forecast?t=${season.slug}` : '/forecast');
	let fcStatus = $derived.by(() => {
		if (!fcLoaded) return '';
		if (forecastStore.spec.mode === 'calls') return `${fcPlaced} of ${fcCalls.length} calls placed`;
		return fcHas ? 'Placed' : 'Not placed yet';
	});

	const statusLabel: Record<string, string> = { active: 'live', upcoming: 'upcoming', finished: 'finished', archived: 'archived', draft: 'draft' };
	function initials(name: string): string {
		return name.split(/\s+/).filter((w) => /^[A-Z0-9]/.test(w)).map((w) => w[0]).join('').slice(0, 3);
	}
</script>

{#snippet head()}
	{#if competition && season}
		<a class="topbar-back" href="/competitions" aria-label="Back to competitions" onclick={goBack}><ChevronLeft size={22} /></a>
		<span class="hcrest" class:ph={!competitionLogoUrl(competition)}>
			{#if competitionLogoUrl(competition)}<img src={competitionLogoUrl(competition)} alt="" />{:else}{initials(competition.name)}{/if}
		</span>
		<span class="htxt grow">
			<b>{competition.name}</b>
			<span class="hsub">
				<label class="season" class:single={seasons.length < 2}>
					<select value={season.slug} disabled={seasons.length < 2} onchange={(e) => pickSeason((e.currentTarget as HTMLSelectElement).value)} aria-label="Season">
						{#each seasons as s (s.id)}<option value={s.slug}>{seasonLabel(s)}</option>{/each}
					</select>
					<ChevronDown size={13} />
				</label>
				{#if season.status === 'active'}<span class="pill live">Live</span>{:else}<span class="pill">{statusLabel[season.status] ?? season.status}</span>{/if}
			</span>
		</span>
		{#if canPlay}
			<button class="chip" class:on={isPlaying} disabled={busy} onclick={togglePlay}>
				{#if isPlaying}<Check size={15} /> Playing{:else}<Plus size={15} /> Play{/if}
			</button>
		{/if}
	{/if}
{/snippet}

<PageChrome bar={head} title={competition?.name ?? 'Competition'} back="/competitions" />

{#if ready && !competition}
	<div class="card miss muted">No such competition. <a href="/competitions">Back to competitions</a></div>
{:else if competition && season}
	<div class="hub">
		<!-- Desktop keeps the global top bar; the same header sits in the page. -->
		<header class="hubhead">{@render head()}</header>
		<div class="subbar tabrow">
			<div class="utabs" role="tablist">
				{#each tabs as t (t.id)}
					<button class="utab" class:on={tab === t.id} role="tab" aria-selected={tab === t.id} onclick={() => setTab(t.id)}>{t.label}</button>
				{/each}
			</div>
		</div>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="pane" ontouchstart={touchStart} ontouchend={touchEnd}>
			{#if tab === 'overview'}
				{#if !loaded}
					<p class="muted">Loading…</p>
				{:else}
					{#if spotlight.matches.length}
						<div class="sec"><h2>{spotlight.label}</h2>{#if spotlight.sub}<span class="muted small">{spotlight.sub}</span>{/if}<button class="more" onclick={() => setTab('matches')}>Matches <ChevronRight size={14} /></button></div>
						<div class="card rows">
							{#each spotlight.matches as m (m.id)}
								<MatchRow match={m} open={openId === m.id} onToggle={() => (openId = openId === m.id ? '' : m.id)} href={`/m/${m.id}`} />
							{/each}
						</div>
					{/if}
					{#if hasForecast && auth.isAuthed}
						<a class="card fc" href={fcHref}>
							<span class="fc-ic"><Target size={20} /></span>
							<span class="fc-txt">
								<b>Forecast{fcStatus ? ` · ${fcStatus}` : ''}</b>
								<span class="muted">{fcLockText || 'Your one-shot call on the whole season'}</span>
							</span>
							<ChevronRight size={18} class="cv" />
						</a>
					{/if}
					{#if auth.isAuthed}
						<div class="stats">
							<div class="card stat"><span class="n digits">{myPoints}</span><span class="l">Your points</span></div>
							<div class="card stat"><span class="n digits">{myTips}<small>/{tipsStore.matches.length}</small></span><span class="l">Tipped</span></div>
							<div class="card stat"><span class="n digits">{exact}</span><span class="l">Exact</span></div>
						</div>
					{/if}
					{#if hasTable}
						<div class="sec"><h2>{tournamentStore.singleTable ? tournamentStore.stageName(tournamentStore.groupStageCode) : 'Groups'}</h2><button class="more" onclick={() => setTab('table')}>{tournamentStore.singleTable ? 'Table' : 'Groups'} <ChevronRight size={14} /></button></div>
						<Standings compact view="groups" limit={6} />
					{:else if hasKnockout}
						<div class="sec"><h2>Knockout</h2><button class="more" onclick={() => setTab('knockout')}>Bracket <ChevronRight size={14} /></button></div>
						<Standings compact view="bracket" />
					{/if}
					<div class="sec"><h2>About</h2></div>
					<p class="desc">{description}</p>
					<p class="muted dates">{#if !description.includes(dateSpan(season.startsAt, season.endsAt))}{dateSpan(season.startsAt, season.endsAt)} · {/if}{#if competition.country && competition.country !== 'World'}{competition.country} · {/if}{competition.teamKind === 'club' ? 'Clubs' : 'National teams'}</p>
				{/if}
			{:else if tab === 'matches'}
				<MatchList focusId={mParam} />
			{:else if tab === 'table'}
				<Standings view="groups" />
			{:else if tab === 'knockout'}
				<KnockoutTies />
			{:else if tab === 'forecast'}
				{#if !auth.isAuthed}
					<div class="card quiet muted">Sign in to place a forecast.</div>
				{:else if !fcLoaded}
					<p class="muted">Loading…</p>
				{:else}
					<div class="card fc" class:locked={forecastStore.locked}>
						<span class="fc-ic">{#if forecastStore.locked}<Lock size={18} />{:else}<Target size={20} />{/if}</span>
						<span class="fc-txt"><b>Your forecast · {fcStatus}</b><span class="muted">{fcLockText}</span></span>
					</div>
					{#if forecastStore.spec.mode === 'calls'}
						{#each fcCalls as c (c.key)}
							{@const picks = picksOf(c.key)}
							<a class="card call" href={fcHref}>
								<span class="callh">
									<span class="t"><b>{c.name}</b><span class="muted small">{c.type === 'team' ? 'one pick' : `pick ${c.count ?? '?'}`} · {c.points} pt{c.type === 'teamset' ? ' each' : ''}</span></span>
									<span class="spacer"></span>
									{#if c.type === 'team'}
										<span class="pill" class:ok={picks.length > 0}>{picks.length ? 'placed' : 'open'}</span>
									{:else}
										<span class="pill" class:ok={picks.length >= (c.count ?? 0)}>{picks.length} / {c.count ?? '?'}</span>
									{/if}
								</span>
								<span class="picks">
									{#each picks as id (id)}
										{@const t = forecastStore.team(id)}
										<span class="pchip"><Flag iso2={t?.iso2 ?? ''} code={t?.fifaCode ?? ''} logo={t?.logo ?? ''} size={18} /> {t?.name ?? '?'}</span>
									{/each}
									{#if !forecastStore.locked && picks.length < (c.type === 'team' ? 1 : (c.count ?? 0))}
										<span class="pchip empty"><Plus size={14} /> {c.type === 'team' ? 'Pick' : `${(c.count ?? 0) - picks.length} more`}</span>
									{/if}
								</span>
							</a>
						{/each}
					{:else}
						<a class="card quiet" href={fcHref}>
							<span class="qtxt"><b>Groups, best thirds and the whole bracket.</b><span class="muted">{forecastStore.locked ? 'Locked — see how it went.' : 'Open the builder to place or change it.'}</span></span>
							<ChevronRight size={18} class="cv" />
						</a>
					{/if}
					<a class="btn" class:secondary={forecastStore.locked} href={fcHref} style="margin-top:0.9rem">{forecastStore.locked ? 'View forecast' : fcHas ? 'Edit forecast' : 'Make your forecast'}</a>
				{/if}
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
	/* ---- header (mobile: in the top bar via PageChrome; desktop: in-page) ---- */
	.hubhead {
		display: none;
		align-items: center;
		gap: 0.75rem;
		padding: 0.2rem 0 0.9rem;
	}
	@media (min-width: 900px) {
		.hubhead {
			display: flex;
		}
		/* The header sits above the tab row here, so no pull-up. */
		.hub :global(.subbar) {
			margin-top: 0;
		}
	}
	:global(.hcrest) {
		width: 34px;
		height: 34px;
		flex: none;
		border-radius: 50%;
		overflow: hidden;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		background: #fff;
	}
	:global(.hcrest img) {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}
	:global(.hcrest.ph) {
		background: var(--surface-2);
		color: var(--muted);
		font-size: 0.6rem;
		font-weight: 800;
	}
	:global(.htxt) {
		display: flex;
		flex-direction: column;
		gap: 0.05rem;
		min-width: 0;
		line-height: 1.15;
	}
	:global(.htxt.grow) {
		flex: 1;
	}
	:global(.htxt > b) {
		font-size: 0.95rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	:global(.hsub) {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}
	:global(.hsub .pill) {
		font-size: 0.56rem;
		padding: 0.1rem 0.4rem;
	}
	:global(.season) {
		position: relative;
		display: inline-flex;
		align-items: center;
		color: var(--muted);
	}
	:global(.season select) {
		appearance: none;
		-webkit-appearance: none;
		padding: 0 1.1rem 0 0;
		border: none;
		background: transparent;
		font: inherit;
		font-weight: 600;
		font-size: 0.78rem;
		color: inherit;
		color-scheme: dark;
		cursor: pointer;
		max-width: 30vw;
		text-overflow: ellipsis;
	}
	:global(.season option) {
		background: var(--surface);
		color: var(--text);
		font-weight: 600;
	}
	:global(.season > svg) {
		position: absolute;
		right: 0;
		pointer-events: none;
	}
	:global(.season.single select) {
		padding-right: 0;
		cursor: default;
	}
	:global(.season.single > svg) {
		display: none;
	}
	:global(.topbar-custom .chip),
	.hubhead :global(.chip) {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		height: 32px;
		padding: 0 0.75rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		font: inherit;
		font-weight: 700;
		font-size: 0.78rem;
		white-space: nowrap;
		cursor: pointer;
		flex: none;
	}
	:global(.topbar-custom .chip.on),
	.hubhead :global(.chip.on) {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.hubhead :global(.topbar-back) {
		margin-left: 0;
	}

	/* ---- panes ---- */
	.pane {
		min-height: 40vh;
	}
	.sec {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		margin: 1.1rem 0.15rem 0.55rem;
	}
	.sec:first-child {
		margin-top: 0.2rem;
	}
	.sec h2 {
		font-size: 1.15rem;
	}
	.small {
		font-size: 0.8rem;
	}
	.more {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
		gap: 0.1rem;
		border: none;
		background: transparent;
		color: var(--accent);
		font: inherit;
		font-size: 0.82rem;
		font-weight: 600;
		cursor: pointer;
		padding: 0;
	}
	.card.rows {
		padding: 0;
	}
	.card.fc,
	.card.quiet {
		display: flex;
		align-items: center;
		gap: 0.8rem;
		padding: 0.8rem 0.9rem;
		margin-top: 0.75rem;
		color: var(--text);
	}
	.card.fc {
		border-color: color-mix(in srgb, var(--accent) 35%, var(--border));
	}
	.card.fc.locked {
		border-color: var(--border);
	}
	.fc-ic {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 38px;
		height: 38px;
		flex: none;
		border-radius: var(--radius-sm);
		color: var(--accent);
		background: color-mix(in srgb, var(--accent) 14%, transparent);
	}
	.fc-txt,
	.qtxt {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		min-width: 0;
		font-size: 0.92rem;
	}
	.fc-txt .muted,
	.qtxt .muted {
		font-size: 0.82rem;
	}
	.card :global(.cv) {
		margin-left: auto;
		color: var(--muted);
	}
	.stats {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 0.5rem;
		margin-top: 0.75rem;
	}
	.stat {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		padding: 0.75rem 0.9rem;
		margin: 0;
	}
	.stat .n {
		font-size: 1.4rem;
	}
	.stat .n small {
		font-size: 0.75rem;
		color: var(--muted);
	}
	.stat .l {
		font-size: 0.72rem;
		font-weight: 600;
		color: var(--muted);
	}
	.desc {
		margin: 0;
		font-size: 0.92rem;
		line-height: 1.45;
	}
	.dates {
		margin: 0.4rem 0 0;
		font-size: 0.8rem;
	}
	/* ---- forecast tab ---- */
	.card.call {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		padding: 0.8rem 0.9rem;
		margin-top: 0.75rem;
		color: var(--text);
	}
	.callh {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.callh .t {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
	}
	.callh b {
		font-size: 0.95rem;
	}
	.picks {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
	}
	.pchip {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		height: 32px;
		padding: 0 0.7rem 0 0.4rem;
		border-radius: var(--radius-pill);
		border: 1px solid color-mix(in srgb, var(--accent) 55%, var(--border));
		background: color-mix(in srgb, var(--accent) 12%, transparent);
		font-weight: 600;
		font-size: 0.82rem;
		white-space: nowrap;
	}
	.pchip.empty {
		border-style: dashed;
		border-color: color-mix(in srgb, var(--accent) 60%, transparent);
		background: transparent;
		color: var(--accent);
		padding: 0 0.7rem;
	}
</style>
