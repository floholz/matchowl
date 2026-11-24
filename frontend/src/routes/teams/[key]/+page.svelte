<!-- Team page: crest, country and a follow star, then Overview (next
     matches, latest results, the competitions it plays with its table
     line) and Matches (every match of the season, day by day). Following
     puts the team's matches into "my matches". -->
<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import { feedStore, type FeedMatch, localDayKey } from '$lib/feed.svelte';
	import { otherLegView, type Team } from '$lib/tips.svelte';
	import { teamsStore, teamMatches, crestUrl, type TeamPage } from '$lib/teams.svelte';
	import { tournamentStore, competitionLogoUrl } from '$lib/tournament.svelte';
	import { countryName } from '$lib/countries';
	import { serverClock } from '$lib/serverclock.svelte';
	import PageChrome from '$lib/components/PageChrome.svelte';
	import MatchRow from '$lib/components/MatchRow.svelte';
	import MatchGroup from '$lib/components/MatchGroup.svelte';
	import TabPager from '$lib/components/TabPager.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import { Star, ChevronRight } from '@lucide/svelte';

	type Tab = 'overview' | 'matches';
	const TABS: { id: Tab; label: string }[] = [
		{ id: 'overview', label: 'Overview' },
		{ id: 'matches', label: 'Matches' }
	];

	let key = $derived($page.params.key ?? '');
	let tab = $derived<Tab>($page.url.searchParams.get('tab') === 'matches' ? 'matches' : 'overview');
	function setTab(t: Tab) {
		const u = new URL($page.url);
		if (t === 'overview') u.searchParams.delete('tab');
		else u.searchParams.set('tab', t);
		goto(`${u.pathname}${u.search}`, { replaceState: true, noScroll: true, keepFocus: true });
	}

	let info = $state<TeamPage | null>(null);
	let missing = $state(false);
	let matches = $state<FeedMatch[]>([]);
	let teams = $state<Record<string, Team>>({});
	let matchesLoaded = $state(false);
	let following = $state(false);
	let followBusy = $state(false);
	let openId = $state('');

	$effect(() => {
		const k = key;
		if (!auth.isAuthed || !k) return;
		info = null;
		missing = false;
		matchesLoaded = false;
		teamsStore
			.page(k)
			.then((p) => {
				info = p;
				following = p.following;
			})
			.catch(() => (missing = true));
		teamMatches(k, 400, 400)
			.then((r) => {
				matches = r.matches;
				teams = r.teams;
			})
			.catch(() => {})
			.finally(() => (matchesLoaded = true));
		tournamentStore.ready().catch(() => {});
	});

	async function toggleFollow() {
		if (!info || followBusy) return;
		followBusy = true;
		const on = !following;
		following = on;
		try {
			await teamsStore.setFollowing(info.team.key, on);
			info.followers += on ? 1 : -1;
			// "My matches" changed: let the feed refetch next time it shows.
			if (feedStore.scope === 'mine') feedStore.load().catch(() => {});
		} catch {
			following = !on;
		} finally {
			followBusy = false;
		}
	}

	const finished = (m: FeedMatch) => m.status === 'finished' || !!m.finalizedAt;
	let upcoming = $derived(matches.filter((m) => !finished(m)));
	let results = $derived(matches.filter(finished).reverse());
	let nextUp = $derived(upcoming.slice(0, 3));
	let latest = $derived(results.slice(0, 3));

	/** The Matches tab: day by day, one card per competition per day. */
	let days = $derived.by(() => {
		const out: { key: string; label: string; groups: { t: FeedMatch['tournament']; matches: FeedMatch[] }[] }[] = [];
		for (const m of matches) {
			const k = localDayKey(m.kickoff);
			let d = out[out.length - 1];
			if (!d || d.key !== k) {
				d = {
					key: k,
					label: new Date(m.kickoff).toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' }),
					groups: []
				};
				out.push(d);
			}
			let g = d.groups.find((x) => x.t.id === m.tournament.id);
			if (!g) d.groups.push((g = { t: m.tournament, matches: [] }));
			g.matches.push(m);
		}
		return out;
	});
	// Land the Matches tab on the next match.
	$effect(() => {
		if (tab !== 'matches' || !matchesLoaded) return;
		const todayKey = localDayKey(new Date(serverClock.now()).toISOString());
		const target = days.find((d) => d.key >= todayKey) ?? days[days.length - 1];
		if (!target) return;
		requestAnimationFrame(() => {
			const el = document.getElementById(`tday-${target.key}`);
			const line = (document.querySelector('.subbar')?.getBoundingClientRect().bottom ?? 120) + 4;
			if (el) window.scrollTo({ top: el.getBoundingClientRect().top + window.scrollY - line, behavior: 'instant' });
		});
	});

	function tipFor(m: FeedMatch) {
		return m.myTip ? { match: m.id, ...m.myTip } : null;
	}
	async function saveTip(m: FeedMatch, t: Parameters<typeof feedStore.saveTip>[1]) {
		await feedStore.saveTip(m, t);
		const i = matches.findIndex((x) => x.id === m.id);
		if (i >= 0) matches[i] = { ...matches[i], myTip: { ...t } };
	}
	function logoOf(tid: string): string {
		const t = tournamentStore.list.find((x) => x.id === tid);
		return t ? competitionLogoUrl(t.competition) : '';
	}
	function dayOf(iso: string): string {
		const d = new Date(iso);
		return `${d.toLocaleDateString(undefined, { weekday: 'short' })} ${d.getDate()} ${d.toLocaleDateString(undefined, { month: 'short' })}`;
	}
	const ordinal = (n: number) => {
		const s = ['th', 'st', 'nd', 'rd'];
		const v = n % 100;
		return n + (s[(v - 20) % 10] || s[v] || s[0]);
	};
	const gd = (n: number) => (n > 0 ? `+${n}` : `${n}`);
	let meta = $derived.by(() => {
		if (!info) return '';
		const t = info.team;
		const parts = [];
		if (t.national) parts.push('National team');
		else {
			if (t.country) parts.push(countryName(t.country));
			parts.push('club');
		}
		if (info.followers > 0) parts.push(`${info.followers} follower${info.followers === 1 ? '' : 's'}`);
		return parts.join(' · ');
	});
</script>

{#snippet row(m: FeedMatch, sub = '')}
	<MatchRow
		match={m}
		team={(id) => teams[id]}
		tip={tipFor(m)}
		knockout={m.knockout}
		points={m.myTip?.points}
		onSave={(t) => saveTip(m, t)}
		open={openId === m.id}
		onToggle={() => (openId = openId === m.id ? '' : m.id)}
		href={`/m/${m.id}`}
		leg={m.leg ? otherLegView(m, m.leg, m.leg.first, `/m/${m.leg.id}`) : null}
		{sub}
	/>
{/snippet}

<PageChrome title={info?.team.name ?? 'Team'} back="/matches" />

{#if missing}
	<div class="card miss muted">No such team. <a href="/search">Search for one</a></div>
{:else if info}
	{@const t = info.team}
	<div class="teampage">
		<header class="hero">
			<span class="crest">
				{#if t.national && t.iso2}
					<Flag iso2={t.iso2} code={t.fifaCode} logo={crestUrl(t.crest)} size={60} />
				{:else if t.crest}
					<img src={crestUrl(t.crest)} alt="" />
				{:else}
					<span class="ph">{t.fifaCode || t.name.slice(0, 3).toUpperCase()}</span>
				{/if}
			</span>
			<span class="htxt">
				<h1>{t.name}</h1>
				<span class="meta muted">
					{#if t.country && !t.national}<Flag iso2={t.country} size={18} />{/if}
					{meta}
				</span>
			</span>
			<button
				class="follow"
				class:on={following}
				onclick={toggleFollow}
				disabled={followBusy}
				aria-pressed={following}
				aria-label={following ? `Unfollow ${t.name}` : `Follow ${t.name}`}
				title={following ? 'Following: its matches are in My matches' : 'Follow: put its matches in My matches'}
			>
				<Star size={22} fill={following ? 'currentColor' : 'none'} />
			</button>
		</header>
		<div class="subbar tabrow">
			<div class="utabs" role="tablist">
				{#each TABS as tb (tb.id)}
					<button class="utab" class:on={tab === tb.id} role="tab" aria-selected={tab === tb.id} onclick={() => setTab(tb.id)}>{tb.label}</button>
				{/each}
			</div>
		</div>

		{#snippet paneFor(id: Tab)}
			{#if id === 'overview'}
				{#if !matchesLoaded}
					<p class="muted">Loading…</p>
				{:else}
					{#if nextUp.length}
						<div class="sec"><h2>Next matches</h2><button class="more" onclick={() => setTab('matches')}>Matches <ChevronRight size={14} /></button></div>
						<div class="card rows">
							{#each nextUp as m (m.id)}{@render row(m, dayOf(m.kickoff))}{/each}
						</div>
					{/if}
					{#if latest.length}
						<div class="sec"><h2>Latest results</h2></div>
						<div class="card rows">
							{#each latest as m (m.id)}{@render row(m, dayOf(m.kickoff))}{/each}
						</div>
					{/if}
					{#if !nextUp.length && !latest.length}
						<div class="card quiet muted">No matches of {t.name} in the seasons Matchowl covers right now.</div>
					{/if}
				{/if}
				{#if info?.seasons.length}
					<div class="sec"><h2>Competitions</h2></div>
					<div class="card rows">
						{#each info?.seasons ?? [] as s (s.teamId)}
							<a class="comp" href={s.tournament.competition ? `/competitions/${s.tournament.competition}?s=${s.tournament.slug}` : '#'}>
								{#if logoOf(s.tournament.id)}<img class="clogo" src={logoOf(s.tournament.id)} alt="" />{/if}
								<span class="ctxt">
									<b>{s.tournament.name}</b>
									<span class="muted">
										{#if s.table}
											{ordinal(s.table.rank)} of {s.table.size}{s.table.group && s.table.size < 10 ? ` in group ${s.table.group}` : ''} · {s.table.played} played · W{s.table.won} D{s.table.drawn} L{s.table.lost} · GD {gd(s.table.gf - s.table.ga)}
										{:else}
											{s.tournament.status === 'archived' ? 'Finished' : s.tournament.status === 'upcoming' ? 'Starts soon' : 'Knockout'}
										{/if}
									</span>
								</span>
								{#if s.table}<span class="pts digits" title="Points">{s.table.pts}</span>{/if}
								<ChevronRight size={16} class="cv" />
							</a>
						{/each}
					</div>
				{/if}
			{:else}
				{#if !matchesLoaded}
					<p class="muted">Loading…</p>
				{:else if !days.length}
					<div class="card quiet muted">No matches of {t.name} in the seasons Matchowl covers right now.</div>
				{:else}
					{#each days as d (d.key)}
						<section class="day" id={`tday-${d.key}`}>
							<h2 class="day-h">{d.label}</h2>
							{#each d.groups as g (g.t.id)}
								<MatchGroup
									name={g.t.shortName || g.t.name}
									round={g.matches[0].roundLabel || g.matches[0].stageName}
									logo={logoOf(g.t.id)}
									href={`/competitions/${g.t.competition}?s=${g.t.slug}`}
								>
									{#each g.matches as m (m.id)}{@render row(m)}{/each}
								</MatchGroup>
							{/each}
						</section>
					{/each}
				{/if}
			{/if}
		{/snippet}
		<TabPager tabs={TABS.map((x) => x.id)} current={tab} onchange={setTab} pane={paneFor} />
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
		gap: 0.9rem;
		padding: 0.4rem 0 0.9rem;
	}
	.crest {
		width: 64px;
		height: 64px;
		flex: none;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}
	.crest img {
		width: 64px;
		height: 64px;
		object-fit: contain;
	}
	.crest .ph {
		width: 56px;
		height: 56px;
		border-radius: 50%;
		background: var(--surface-2);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-weight: 800;
		color: var(--muted);
	}
	.htxt {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		min-width: 0;
		flex: 1;
	}
	.htxt h1 {
		font-size: 1.5rem;
		line-height: 1.15;
		overflow-wrap: anywhere;
	}
	.meta {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.85rem;
	}
	.follow {
		width: 46px;
		height: 46px;
		flex: none;
		border-radius: 50%;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border: 1px solid var(--border);
		background: transparent;
		color: var(--muted);
		cursor: pointer;
		transition:
			color 0.15s ease,
			background 0.15s ease,
			border-color 0.15s ease;
	}
	.follow:hover {
		color: var(--accent);
	}
	.follow.on {
		color: var(--accent);
		border-color: var(--accent);
		background: color-mix(in srgb, var(--accent) 12%, transparent);
	}
	.sec {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		margin: 1.2rem 0.15rem 0.55rem;
	}
	.sec h2 {
		font-size: 1.2rem;
	}
	.sec .more {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
		gap: 0.1rem;
		font-size: 0.82rem;
		font-weight: 600;
		background: none;
		border: 0;
		color: var(--accent);
		cursor: pointer;
		padding: 0;
	}
	.card.rows {
		padding: 0;
	}
	.card.quiet {
		padding: 0.9rem 1rem;
		font-size: 0.9rem;
		margin-top: 1rem;
	}
	.comp {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 0.9rem;
		border-bottom: 1px solid var(--border);
		color: var(--text);
	}
	.comp:last-child {
		border-bottom: none;
	}
	.clogo {
		width: 26px;
		height: 26px;
		object-fit: contain;
		flex: none;
	}
	.ctxt {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		min-width: 0;
		flex: 1;
		font-size: 0.92rem;
	}
	.ctxt .muted {
		font-size: 0.8rem;
	}
	.pts {
		font-size: 1rem;
		padding: 0.3rem 0.6rem;
		border-radius: var(--radius-sm);
		background: var(--board-bg);
		color: var(--board-led, var(--warning));
	}
	.comp :global(.cv) {
		color: var(--muted);
		flex: none;
	}
	.day {
		margin-top: 1rem;
	}
	.day-h {
		font-family: var(--font);
		font-weight: 700;
		font-size: 0.76rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--muted);
		margin: 0 0.15rem 0.5rem;
	}
</style>
