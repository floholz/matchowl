<!-- Home: your pools first (the head-to-head duel is the biggest thing
     on the page), then Tip now (matches locking soonest, in the thumb
     zone), the forecast deadline, Live, yesterday's points, and the teams
     you follow as a sideways strip. Friend challenges (invites to answer,
     open ones) sit between the pools and Tip now. Nothing else.
     Signed-out visitors are sent to sign in by the layout. -->
<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { feedStore, type FeedMatch } from '$lib/feed.svelte';
	import { otherLegView } from '$lib/tips.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { tournamentStore } from '$lib/tournament.svelte';
	import { h2hStore } from '$lib/h2h.svelte';
	import { api, type PoolSummary } from '$lib/api';
	import MatchRow from '$lib/components/MatchRow.svelte';
	import DuelCard from '$lib/components/DuelCard.svelte';
	import SupportCard from '$lib/components/SupportCard.svelte';
	import { appConfig } from '$lib/appconfig.svelte'; // loaded by SupportCard
	import { teamsStore, crestUrl, type NextMatch } from '$lib/teams.svelte';
	import { challengesStore } from '$lib/challenges.svelte';
	import ChallengeCard from '$lib/components/ChallengeCard.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import { Telescope, ChevronRight, ChevronUp, ChevronDown } from '@lucide/svelte';

	let openId = $state('');

	$effect(() => {
		if (!auth.isAuthed || feedStore.loading || feedStore.error) return;
		// Home always looks at my matches.
		if (feedStore.scope !== 'mine') feedStore.setScope('mine').catch(() => {});
		else if (!feedStore.loaded) feedStore.load().catch(() => {});
		tournamentStore.ready().catch(() => {});
	});

	// ---- pools ----
	// Head-to-head pools come from the shared h2h store as duel cards;
	// classic pools keep the compact "where am I, who leads" row.
	interface LeagueLine {
		league: PoolSummary;
		rank: number;
		total: number;
		points: number;
		leader: string;
		leaderPoints: number;
	}
	let leagues = $state<LeagueLine[]>([]);
	let leaguesLoaded = $state(false);
	$effect(() => {
		if (!auth.isAuthed) return;
		h2hStore.load().catch(() => {});
		api
			.myPools()
			.then(async ({ pools: ls }) => {
				const mine = ls.filter((l) => l.inviteCode !== 'GLOBAL' && l.status !== 'finished' && l.mode !== 'h2h');
				const lines = await Promise.all(
					mine.map(async (league) => {
						const { rows } = await api.leaderboard(league.id).catch(() => ({ rows: [] }));
						const i = rows.findIndex((r) => r.userId === auth.user?.id);
						return {
							league,
							rank: i + 1,
							total: rows.length,
							points: i >= 0 ? rows[i].total : 0,
							leader: rows[0]?.name ?? '',
							leaderPoints: rows[0]?.total ?? 0
						};
					})
				);
				leagues = lines;
			})
			.catch(() => {})
			.finally(() => (leaguesLoaded = true));
	});
	let duels = $derived(h2hStore.pools);

	// ---- challenges (friends only, so verified accounts) ----
	$effect(() => {
		if (auth.user?.verified) challengesStore.load().catch(() => {});
	});
	/** Invites to answer first, then the ones still to be played. */
	let challengeCards = $derived([...challengesStore.incoming, ...challengesStore.open].slice(0, 4));
	let challengeMore = $derived(challengesStore.incoming.length + challengesStore.open.length - challengeCards.length);

	// ---- your teams ----
	$effect(() => {
		if (auth.isAuthed) teamsStore.load().catch(() => {});
	});
	/** "Tue 24 · at Bodø/Glimt", "Live · vs Ried". */
	function nextLine(n: NextMatch | null): string {
		if (!n) return 'No match scheduled';
		const opp = `${n.home ? 'vs' : 'at'} ${n.opponent.name}`;
		if (n.status === 'live') return `Live · ${opp}`;
		return `${dayOf(n.kickoff)} ${time(n.kickoff)} · ${opp}`;
	}
	/** Results are capped: the rest is one tap away on Matches. */
	const RECENT_MAX = 5;

	// ---- matches ----
	const byKickoff = (a: FeedMatch, b: FeedMatch) =>
		new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime();
	let open = $derived(
		feedStore.matches
			.filter(
				(m) =>
					!feedStore.finished(m) &&
					m.status !== 'live' &&
					m.homeTeam &&
					m.awayTeam &&
					new Date(m.kickoff).getTime() > serverClock.now()
			)
			.sort(byKickoff)
	);
	/** Tip now looks a week ahead: the current matchday, midweek games
	 *  included — not the next open pairing months out. */
	const TIP_WINDOW_MS = 7 * 86400_000;
	let soon = $derived(open.filter((m) => new Date(m.kickoff).getTime() - serverClock.now() <= TIP_WINDOW_MS));
	/** Only what your pools and challenges ask you to tip. */
	let untipped = $derived(
		soon.filter((m) => !m.myTip && (m.reasons.includes('pool') || m.reasons.includes('challenge')))
	);
	let tipNow = $derived(untipped.slice(0, 3));
	let live = $derived(feedStore.matches.filter((m) => m.status === 'live').sort(byKickoff));
	/** The most recent day with results: "Yesterday" when it is. */
	let recent = $derived.by(() => {
		const days = feedStore.days.filter((d) => d.matches.some((m) => feedStore.finished(m)));
		const d = days[days.length - 1];
		if (!d) return null;
		const y = new Date(serverClock.now() - 86400_000);
		const yKey = `${y.getFullYear()}-${String(y.getMonth() + 1).padStart(2, '0')}-${String(y.getDate()).padStart(2, '0')}`;
		const matches = d.matches.filter((m) => feedStore.finished(m));
		const points = matches.reduce((s, m) => s + (m.myTip?.points ?? 0), 0);
		return { label: d.key === yKey ? 'Yesterday' : d.isToday ? 'Today' : d.label, matches, points };
	});

	function tipFor(m: FeedMatch) {
		return m.myTip ? { match: m.id, ...m.myTip } : null;
	}
	/** Short competition code for the status column (BL, UCL, …). */
	function codeOf(m: FeedMatch): string {
		// The short name when the admin set one ("UCL", "Prem"), else the
		// full name — the row ellipses what does not fit ("La Li…").
		const t = tournamentStore.list.find((x) => x.id === m.tournament.id);
		const c = t?.competition;
		return c?.shortName || c?.name || m.tournament.shortName || m.tournament.name || '';
	}
	function legOf(m: FeedMatch) {
		return m.leg ? otherLegView(m, m.leg, m.leg.first, `/m/${m.leg.id}`) : null;
	}
	const time = (iso: string) =>
		new Date(iso).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
	/** "Tue 15" — the day a match is on; "today" when it is. Built by hand
	 *  so the order never flips with the locale, and short enough for the
	 *  row's status column. */
	function dayOf(iso: string): string {
		const d = new Date(iso);
		const now = new Date(serverClock.now());
		const key = (x: Date) => `${x.getFullYear()}-${x.getMonth()}-${x.getDate()}`;
		if (key(d) === key(now)) return 'today';
		return `${d.toLocaleDateString(undefined, { weekday: 'short' })} ${d.getDate()}`;
	}
	const dayTime = (iso: string) => `${dayOf(iso)} ${time(iso)}`;
	function locksIn(iso: string): string {
		const ms = new Date(iso).getTime() - serverClock.now();
		const days = Math.floor(ms / 86400_000);
		if (days >= 2) return `locks in ${days} days`;
		const hours = Math.floor(ms / 3600_000);
		if (hours >= 2) return `locks in ${hours} hours`;
		return 'locks soon';
	}
</script>

{#snippet row(m: FeedMatch)}
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
		leg={legOf(m)}
		sub={dayOf(m.kickoff) === 'today' ? codeOf(m) : dayOf(m.kickoff)}
	/>
{/snippet}

{#if auth.isAuthed}
	<div class="home stagger">
		{#if feedStore.error && !feedStore.loaded}
			<div class="card empty">
				<p><b>Couldn't load your matches.</b></p>
				<p class="muted">{feedStore.error}</p>
				<button class="btn" onclick={() => feedStore.load().catch(() => {})}>Retry</button>
			</div>
		{:else if feedStore.loaded}
			<div class="sec">
				<h2>Your pools</h2>
				<a class="more" href="/pools">Pools <ChevronRight size={14} /></a>
			</div>
			{#each duels as pool (pool.poolId)}
				<DuelCard {pool} />
			{/each}
			{#if leagues.length}
				<div class="card rows">
					{#each leagues as l (l.league.id)}
						<a class="lrow" href={`/pools/${l.league.id}`}>
							<span class="rank digits"
								>{l.rank > 0 ? `#${l.rank}` : '–'}<small>/{l.total}</small></span
							>
							<span class="ltxt">
								<b>{l.league.name}</b>
								<span class="muted"
									>{l.leader
										? l.rank === 1
											? 'You lead'
											: `${l.leader} leads · ${l.leaderPoints} pts`
										: `${l.league.members} members`}</span
								>
							</span>
							<span class="lpts">
								<span class="digits">{l.points}</span>
								{#if l.leader && l.rank > 1}
									<span class="gap muted"><ChevronDown size={11} />{l.leaderPoints - l.points}</span>
								{:else if l.rank === 1 && l.total > 1}
									<span class="gap up"><ChevronUp size={11} />lead</span>
								{/if}
							</span>
						</a>
					{/each}
				</div>
			{:else if leaguesLoaded && h2hStore.loaded && !duels.length}
				<div class="card quiet muted">
					No pools yet. <a href="/pools">Start one</a> and invite your friends.
				</div>
			{/if}

			{#if challengeCards.length}
				<div class="sec">
					<h2>Challenges</h2>
					<a class="more" href="/friends">Friends <ChevronRight size={14} /></a>
				</div>
				<div class="chs">
					{#each challengeCards as c (c.id)}
						<ChallengeCard {c} onChange={() => feedStore.load().catch(() => {})} />
					{/each}
				</div>
				{#if challengeMore > 0}
					<a class="seeall" href="/friends">{challengeMore} more on Friends <ChevronRight size={14} /></a>
				{/if}
			{/if}

			<div class="sec">
				<h2>Tip now</h2>
				{#if untipped.length}
					<span class="muted note"
						>{untipped.length} open this week · locks {dayTime(untipped[0].kickoff)}</span
					>
				{/if}
				<a class="more" href="/matches">Matches <ChevronRight size={14} /></a>
			</div>
			{#if tipNow.length}
				<div class="card rows">
					{#each tipNow as m (m.id)}{@render row(m)}{/each}
				</div>
			{:else if soon.length}
				<div class="card quiet muted">
					Everything's tipped this week. Next kick-off {dayTime(soon[0].kickoff)}.
				</div>
			{:else if open.length}
				<div class="card quiet muted">
					Nothing to tip this week. Next kick-off {new Date(open[0].kickoff).toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' })}.
				</div>
			{:else}
				<div class="card quiet muted">
					Nothing to tip right now. <a href="/pools">Join or start a pool</a> to fill this up.
				</div>
			{/if}

			{#each feedStore.deadlines as d (d.tournament.id)}
				<a class="card deadline" href={`/forecast?t=${d.tournament.slug}`}>
					<span class="dl-ic"><Telescope size={20} /></span>
					<span class="dl-txt">
						<b>{d.hasForecast ? 'Finish your' : 'Make your'} {d.tournament.shortName || d.tournament.name} forecast</b>
						<span class="muted">One shot before kickoff · {locksIn(d.locksAt)}</span>
					</span>
					<ChevronRight size={18} class="cv" />
				</a>
			{/each}

			{#if live.length}
				<div class="sec">
					<h2>Live</h2>
					<a class="more" href="/matches">Matches <ChevronRight size={14} /></a>
				</div>
				<div class="card rows">
					{#each live as m (m.id)}{@render row(m)}{/each}
				</div>
			{/if}

			{#if recent}
				<div class="sec">
					<h2>{recent.label}</h2>
					<span class="pill" class:ok={recent.points > 0}
						>{recent.points > 0 ? '+' : ''}{recent.points} pts</span
					>
					<a class="more" href="/matches">Results <ChevronRight size={14} /></a>
				</div>
				<div class="card rows">
					{#each recent.matches.slice(0, RECENT_MAX) as m (m.id)}{@render row(m)}{/each}
					{#if recent.matches.length > RECENT_MAX}
						<a class="seeall" href="/matches">See all {recent.matches.length} results <ChevronRight size={14} /></a>
					{/if}
				</div>
			{/if}
			{#if teamsStore.mine.length}
				<div class="sec">
					<h2>Your teams</h2>
					<a class="more" href="/search">Search <ChevronRight size={14} /></a>
				</div>
				<div class="teamstrip">
					{#each teamsStore.mine as t (t.key)}
						<a class="card tcard" href={t.next ? `/m/${t.next.id}` : `/teams/${t.key}`}>
							<span class="tcrest">
								{#if t.national && t.iso2}
									<Flag iso2={t.iso2} code={t.fifaCode} logo={crestUrl(t.crest)} size={30} />
								{:else if t.crest}
									<img src={crestUrl(t.crest)} alt="" />
								{/if}
							</span>
							<b class="tname">{t.name}</b>
							<span class="muted tnext" class:live={t.next?.status === 'live'}>{nextLine(t.next)}</span>
						</a>
					{/each}
				</div>
			{/if}
		{:else}
			<p class="muted">Loading…</p>
		{/if}

		<SupportCard />
		<footer class="foot muted">
			Matchowl{#if appConfig.version}&nbsp;<span class="ver">v{appConfig.version}</span>{/if} · made by floholz ·
			<a href="/help">Help</a> · <a href="/legal/about">About</a>
		</footer>
	</div>
{/if}

<style>
	.sec {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		margin: 1.2rem 0.15rem 0.55rem;
	}
	.sec:first-child {
		margin-top: 0.2rem;
	}
	.sec h2 {
		font-size: 1.2rem;
	}
	.sec .note {
		font-size: 0.82rem;
	}
	.sec .more {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
		gap: 0.1rem;
		font-size: 0.82rem;
		font-weight: 600;
	}
	.card.rows {
		padding: 0;
	}
	.card.quiet {
		padding: 0.9rem 1rem;
		font-size: 0.9rem;
	}
	.card + .card {
		margin-top: 0.75rem;
	}
	.deadline {
		display: flex;
		align-items: center;
		gap: 0.8rem;
		padding: 0.8rem 0.9rem;
		margin-top: 0.75rem;
		color: var(--text);
		border-color: color-mix(in srgb, var(--accent) 35%, var(--border));
	}
	.dl-ic {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 38px;
		height: 38px;
		border-radius: var(--radius-sm);
		flex: none;
		color: var(--accent);
		background: color-mix(in srgb, var(--accent) 14%, transparent);
	}
	.dl-txt {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		min-width: 0;
	}
	.dl-txt b {
		font-size: 0.92rem;
	}
	.dl-txt .muted {
		font-size: 0.82rem;
	}
	.deadline :global(.cv) {
		margin-left: auto;
		color: var(--muted);
	}
	.lrow {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 0.9rem;
		border-bottom: 1px solid var(--border);
		color: var(--text);
	}
	.lrow:last-child {
		border-bottom: none;
	}
	.seeall {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.15rem;
		padding: 0.6rem;
		font-size: 0.82rem;
		font-weight: 600;
	}
	.rank {
		font-size: 1.35rem;
		width: 3rem;
		flex: none;
	}
	.rank small {
		font-size: 0.75rem;
		color: var(--muted);
		font-weight: 600;
	}
	.ltxt {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
		font-size: 0.92rem;
	}
	.ltxt .muted {
		font-size: 0.78rem;
	}
	.lpts {
		margin-left: auto;
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 0.1rem;
	}
	.lpts .digits {
		font-size: 0.95rem;
	}
	.gap {
		display: inline-flex;
		align-items: center;
		font-size: 0.72rem;
		font-weight: 700;
	}
	.gap.up {
		color: var(--success);
	}
	.empty {
		text-align: center;
		padding: 2rem 1.2rem;
	}
	.empty .btn {
		margin-top: 1rem;
	}
	.chs {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}
	/* Followed teams: a sideways strip of cards, each with its next match. */
	.teamstrip {
		display: flex;
		gap: 0.6rem;
		overflow-x: auto;
		scroll-snap-type: x proximity;
		margin: 0 calc(-1 * var(--shell-x, 1rem));
		padding: 0 var(--shell-x, 1rem) 0.2rem;
		scrollbar-width: none;
	}
	.teamstrip::-webkit-scrollbar {
		display: none;
	}
	.tcard {
		flex: none;
		width: 8.6rem;
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
		padding: 0.75rem;
		color: var(--text);
		scroll-snap-align: start;
	}
	.card.tcard + .card.tcard {
		margin-top: 0;
	}
	.tcrest {
		height: 32px;
		display: inline-flex;
		align-items: center;
	}
	.tcrest img {
		width: 32px;
		height: 32px;
		object-fit: contain;
	}
	.tname {
		font-size: 0.9rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.tnext {
		font-size: 0.76rem;
		line-height: 1.3;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.tnext.live {
		color: var(--live);
		font-weight: 700;
	}
	.foot {
		text-align: center;
		font-size: 0.75rem;
		padding: 2rem 0 1rem;
	}
</style>
