<!-- Home: the time-based hub. Tip now (matches locking soonest), the
     forecast deadline, Live, your leagues, yesterday's points. Nothing
     else. Signed-out visitors get the landing page. -->
<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { feedStore, type FeedMatch } from '$lib/feed.svelte';
	import { otherLegView } from '$lib/tips.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { tournamentStore } from '$lib/tournament.svelte';
	import { api, type LeagueSummary } from '$lib/api';
	import Landing from '$lib/components/Landing.svelte';
	import MatchRow from '$lib/components/MatchRow.svelte';
	import SupportCard from '$lib/components/SupportCard.svelte';
	import { Telescope, ChevronRight, ChevronUp, ChevronDown } from '@lucide/svelte';

	let openId = $state('');

	$effect(() => {
		if (auth.isAuthed && !feedStore.loaded && !feedStore.loading && !feedStore.error) {
			feedStore.load().catch(() => {});
			tournamentStore.ready().catch(() => {});
		}
	});

	// ---- leagues: where am I, who leads ----
	interface LeagueLine {
		league: LeagueSummary;
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
		api
			.myLeagues()
			.then(async ({ leagues: ls }) => {
				const mine = ls.filter((l) => l.inviteCode !== 'GLOBAL');
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
	let untipped = $derived(open.filter((m) => !m.myTip));
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
		const t = tournamentStore.list.find((x) => x.id === m.tournament.id);
		const s = t?.competition?.shortName || m.tournament.shortName || '';
		return s.length <= 5 ? s : '';
	}
	function legOf(m: FeedMatch) {
		return m.leg
			? otherLegView(
					m,
					m.leg,
					m.leg.first,
					`/competitions/${m.tournament.competition}?s=${m.tournament.slug}&tab=matches&m=${m.leg.id}`
				)
			: null;
	}
	const time = (iso: string) =>
		new Date(iso).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
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
		sub={codeOf(m)}
	/>
{/snippet}

{#if !auth.isAuthed}
	<Landing />
{:else}
	<div class="home stagger">
		{#if feedStore.error && !feedStore.loaded}
			<div class="card empty">
				<p><b>Couldn't load your matches.</b></p>
				<p class="muted">{feedStore.error}</p>
				<button class="btn" onclick={() => feedStore.load().catch(() => {})}>Retry</button>
			</div>
		{:else if feedStore.loaded}
			<div class="sec">
				<h2>Tip now</h2>
				{#if untipped.length}
					<span class="muted note"
						>{untipped.length} open · locks {time(untipped[0].kickoff)}</span
					>
				{/if}
				<a class="more" href="/matches">Matches <ChevronRight size={14} /></a>
			</div>
			{#if tipNow.length}
				<div class="card rows">
					{#each tipNow as m (m.id)}{@render row(m)}{/each}
				</div>
			{:else if open.length}
				<div class="card quiet muted">
					Everything's tipped. Next kick-off {time(open[0].kickoff)}.
				</div>
			{:else}
				<div class="card quiet muted">
					Nothing to tip right now. <a href="/competitions">Play a competition</a> to fill this up.
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

			<div class="sec">
				<h2>Your leagues</h2>
				<a class="more" href="/friends">Friends <ChevronRight size={14} /></a>
			</div>
			{#if leagues.length}
				<div class="card rows">
					{#each leagues as l (l.league.id)}
						<a class="lrow" href={`/friends/${l.league.id}`}>
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
			{:else if leaguesLoaded}
				<div class="card quiet muted">
					No leagues yet. <a href="/friends">Create one</a> and invite your friends.
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
					{#each recent.matches as m (m.id)}{@render row(m)}{/each}
				</div>
			{/if}
		{:else}
			<p class="muted">Loading…</p>
		{/if}

		<SupportCard />
		<footer class="foot muted">Matchowl · made by floholz</footer>
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
	.foot {
		text-align: center;
		font-size: 0.75rem;
		padding: 2rem 0 1rem;
	}
</style>
