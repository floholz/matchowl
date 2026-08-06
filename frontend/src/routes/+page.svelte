<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Confetti } from 'svelte-confetti';
	import { auth } from '$lib/auth.svelte';
	import { api, type LeagueSummary, type LeaderboardRow } from '$lib/api';
	import { appConfig } from '$lib/appconfig.svelte';
	import { tipsStore, isLocked, teamsResolved, type Match } from '$lib/tips.svelte';
	import { tournamentStore } from '$lib/tournament.svelte';
	import { countdown } from '$lib/countdown.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { pb } from '$lib/pb';
	import Avatar from '$lib/components/Avatar.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import Landing from '$lib/components/Landing.svelte';
	import Scoreline from '$lib/components/Scoreline.svelte';
	import SupportCard from '$lib/components/SupportCard.svelte';
	import {
		Telescope,
		Volleyball,
		Trophy,
		Users,
		ChevronRight,
		Check,
		Clock,
		CircleHelp,
		MessageSquareHeart
	} from '@lucide/svelte';

	type Rank = { rank: number; total: number; points: number };

	let leagues = $state<LeagueSummary[]>([]);
	let ranks = $state<Record<string, Rank | null>>({});
	let leaguesLoaded = $state(false);

	// Global is the everyone-league — pin it to the top (matches the Leagues
	// page); other leagues keep the server order (sort is stable).
	let orderedLeagues = $derived(
		[...leagues].sort(
			(a, b) =>
				Number(b.inviteCode === 'GLOBAL') - Number(a.inviteCode === 'GLOBAL')
		)
	);
	let hasForecast = $state(false);
	let forecastChecked = $state(false);

	// Global (everyone) league — final-standings podium + my overall placement.
	let globalRows = $state<LeaderboardRow[]>([]);
	let globalId = $state('');

	// Survey: prominent promo card until submitted, quiet review card after.
	let surveySubmitted = $state(false);
	let surveyChecked = $state(false); // render neither card until known

	onMount(() => {
		if (!auth.isAuthed) return;
		countdown.start();
		tipsStore.load();
		appConfig.load();
		api
			.surveyStatus()
			.then((r) => {
				surveySubmitted = r.submitted;
				surveyChecked = true;
			})
			.catch(() => {});
		// Has the user submitted their forecast yet?
		pb.collection('forecasts')
			.getList(1, 1, { filter: `user = "${auth.user?.id}"` })
			.then((r) => (hasForecast = r.items.length > 0))
			.catch(() => {})
			.finally(() => (forecastChecked = true));
		api
			.myLeagues()
			.then((r) => {
				leagues = r.leagues;
				r.leagues.forEach((l) => loadRank(l.id, l.inviteCode === 'GLOBAL'));
			})
			.catch(() => {})
			.finally(() => (leaguesLoaded = true));
	});
	onDestroy(() => countdown.stop());

	// My placement in a league: rank by total points (mirrors the Overall tab —
	// the stable sort keeps the server's tiebreak order among equal totals).
	function loadRank(id: string, isGlobal = false) {
		api
			.leaderboard(id)
			.then((res) => {
				const rows = [...res.rows].sort((a, b) => b.total - a.total);
				const i = rows.findIndex((row) => row.userId === auth.user?.id);
				ranks[id] = i >= 0 ? { rank: i + 1, total: rows.length, points: rows[i].total } : null;
				if (isGlobal) {
					globalRows = rows;
					globalId = id;
				}
			})
			.catch(() => (ranks[id] = null));
	}

	function avatarUrl(userId: string, avatar?: string | null): string | null {
		return avatar ? pb.files.getURL({ id: userId, collectionName: 'users' }, avatar) : null;
	}

	const pad = (n: number) => String(n).padStart(2, '0');
	const finishedM = (m: Match) => m.status === 'finished' || !!m.finalizedAt;
	const byKick = (a: Match, b: Match) =>
		new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime();

	function stageLabel(stage: string): string {
		return stage ? tournamentStore.stageName(stage) : '';
	}
	function roundOf(m: Match): string {
		return tournamentStore.isGroup(m.stage)
			? `Group ${m.groupLetter} · ${m.roundLabel}`
			: m.roundLabel;
	}
	function fmtKick(iso: string): string {
		return new Date(iso).toLocaleString(undefined, {
			weekday: 'short',
			day: '2-digit',
			month: 'short',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
	// A team slot — resolved team, or the KO placeholder label ("W73", "1A").
	function slot(id: string, label: string) {
		const t = id ? tipsStore.team(id) : undefined;
		return { name: t?.name ?? label ?? 'TBD', iso2: t?.iso2 ?? '', code: t?.fifaCode ?? '' };
	}

	let started = $derived(countdown.locked); // first kickoff has passed
	let total = $derived(tipsStore.matches.length);
	let finished = $derived(tipsStore.matches.filter(finishedM).length);
	let progress = $derived(total ? Math.round((finished / total) * 100) : 0);
	let allDone = $derived(started && total > 0 && finished === total);

	// Current phase = stage of the next match still to kick off.
	let phase = $derived.by(() => {
		const now = serverClock.now();
		const ms = [...tipsStore.matches].sort(byKick);
		const next = ms.find((m) => new Date(m.kickoff).getTime() >= now);
		return stageLabel(next?.stage ?? ms[ms.length - 1]?.stage ?? '');
	});

	// Next up = soonest match not yet kicked off.
	let nextMatch = $derived.by(() => {
		const now = serverClock.now();
		return [...tipsStore.matches].sort(byKick).find((m) => new Date(m.kickoff).getTime() >= now) ?? null;
	});
	let nextTipped = $derived(nextMatch ? !!tipsStore.tips[nextMatch.id] : false);

	// A running match takes over the card (earliest if several run at once).
	let liveMatches = $derived(
		tipsStore.matches.filter((m) => m.status === 'live').sort(byKick)
	);
	let liveMatch = $derived(liveMatches[0] ?? null);

	// Open matches you can still tip (teams resolved, not yet locked).
	let untipped = $derived(
		tipsStore.matches.filter((m) => teamsResolved(m) && !isLocked(m) && !tipsStore.tips[m.id]).length
	);

	// Smart next moves — only what's actually still outstanding.
	let moves = $derived.by(() => {
		const out: { href: string; icon: typeof Telescope; title: string; sub: string }[] = [];
		if (forecastChecked && !countdown.locked && !hasForecast)
			out.push({
				href: '/forecast',
				icon: Telescope,
				title: 'Fill in your forecast',
				sub: 'Your full tournament call — locks at the opening kickoff'
			});
		if (tipsStore.loaded && untipped > 0)
			out.push({
				href: '/tips',
				icon: Volleyball,
				title: `Tip ${untipped} open ${untipped === 1 ? 'match' : 'matches'}`,
				sub: 'Score predictions, editable until each kickoff'
			});
		if (leaguesLoaded && leagues.length === 0)
			out.push({
				href: '/leagues',
				icon: Trophy,
				title: 'Create or join a league',
				sub: 'Play against your friends'
			});
		return out;
	});
	let ready = $derived(forecastChecked && tipsStore.loaded && leaguesLoaded);
	let allCaught = $derived(ready && moves.length === 0);

	// The world champion = the champion stage's advancer, once played.
	let champion = $derived.by(() => {
		const final = tipsStore.matches.find(
			(m) => m.stage === tournamentStore.championStageCode
		);
		return final?.advancer ? tipsStore.team(final.advancer) : undefined;
	});

	// Post-tournament celebration: Global top three (podium order 2-1-3) + my
	// own overall finish, from the tiebreak-sorted Global leaderboard.
	let podium = $derived(
		globalRows.length >= 3 ? [globalRows[1], globalRows[0], globalRows[2]] : []
	);
	let myGlobal = $derived.by(() => {
		const i = globalRows.findIndex((r) => r.userId === auth.user?.id);
		return i >= 0 ? { rank: i + 1, total: globalRows.length } : null;
	});
	const medals = ['🥇', '🥈', '🥉'];
	const achTitle = ['You are the champion!', 'Silver — second overall!', 'Bronze — third overall!'];

	// Header subline: "{name} · {d MMM} – {d MMM}" from the current tournament
	// (loaded via tipsStore). en-GB guarantees the day-first "11 Jun" order.
	const fmtDay = (iso: string) =>
		new Date(iso).toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
	let tournamentLine = $derived.by(() => {
		const t = tournamentStore.current;
		return t ? `${t.name} · ${fmtDay(t.startsAt)} – ${fmtDay(t.endsAt)}` : '';
	});
</script>

{#if !auth.isAuthed}
	<Landing />
{:else}
	<header>
		<p class="kicker">Matchday HQ</p>
		<h1>Hi,&nbsp;{auth.user?.name}</h1>
		{#if tournamentLine}<p class="muted sd">{tournamentLine}</p>{/if}
	</header>

	<div class="stagger">
		<!-- ===== Tournament progress / pre-tournament countdown ===== -->
		<section class="card prog">
			{#if !countdown.ready || !tipsStore.loaded}
				<p class="muted">Loading…</p>
			{:else if countdown.kickoff && !countdown.locked}
				<p class="kicker2">Kickoff in</p>
				<div class="cd">
					<span class="u"><b class="digits">{pad(countdown.parts.days)}</b><i>days</i></span>
					<span class="u"><b class="digits">{pad(countdown.parts.hours)}</b><i>hrs</i></span>
					<span class="u"><b class="digits">{pad(countdown.parts.mins)}</b><i>min</i></span>
					<span class="u"><b class="digits">{pad(countdown.parts.secs)}</b><i>sec</i></span>
				</div>
				<p class="muted fine">The opening match kicks off {fmtKick(new Date(countdown.kickoff).toISOString())}.</p>
			{:else if allDone}
				<div class="wrapup">
					<span class="wi"><Trophy size={24} /></span>
					<h2>That's a wrap!</h2>
					{#if champion}
						<p class="champ">
							<Flag iso2={champion.iso2} code={champion.fifaCode} />
							<b>{champion.name}</b>
							<span class="muted">— world champions!</span>
						</p>
					{/if}
					<p class="muted wt">
						All {total} matches played. Thank you for playing this tournament on
						Matchowl — it was a blast running it for you.
					</p>
				</div>
			{:else}
				<div class="prog-head">
					<span class="phase-lbl">{phase}</span>
					<span class="pct digits">{progress}%</span>
				</div>
				<div class="bar"><span style="width:{progress}%"></span></div>
				<p class="muted fine">{finished} of {total} matches played</p>
			{/if}
		</section>

		<!-- ===== Feedback survey (until submitted) — deliberately slot 2, both
		     during the tournament and after: the answers steer whether Matchowl
		     continues, so it may be a bit pushy. -->
		{#if surveyChecked && !surveySubmitted}
			<section class="card svy">
				<a class="move" href="/survey">
					<span class="mi"><MessageSquareHeart size={20} /></span>
					<span class="mt">
						<span class="title">Got 2 minutes? The full-time survey</span>
						<span class="muted sub">
							How was it — and should this return for leagues &amp; seasons?
						</span>
					</span>
					<ChevronRight size={18} class="cr" />
				</a>
			</section>
		{/if}

		<!-- ===== Post-tournament: Global podium + personal finish ===== -->
		{#if allDone && podium.length === 3}
			<section class="card pod">
				<div class="row">
					<h3>The final standings</h3>
					<div class="spacer"></div>
					<a class="pill" href={`/leagues/${globalId}`}>Full table</a>
				</div>
				<div class="podium">
					{#each podium as p, i (p.userId)}
						{@const place = [2, 1, 3][i]}
						<div class="pstep p{place}">
							<Avatar name={p.name} src={avatarUrl(p.userId, p.avatar)} size={place === 1 ? 52 : 42} />
							<span class="pname">{p.name}</span>
							<span class="ppts digits">{p.total} pts</span>
							<div class="pblock"><span class="pmedal">{medals[place - 1]}</span></div>
						</div>
					{/each}
				</div>
				{#if myGlobal && myGlobal.rank > 3}
					<p class="muted mine">
						You finished <b class="rk">#{myGlobal.rank}</b> of {myGlobal.total} players overall.
					</p>
				{/if}
			</section>

			{#if myGlobal && myGlobal.rank <= 3}
				<section class="card ach">
					<div class="ach-confetti" aria-hidden="true">
						<Confetti
							x={[-1.2, 1.2]}
							y={[-0.4, 0.9]}
							fallDistance="60px"
							amount={70}
							duration={2600}
							colorArray={['#ff7700', '#ffd7b4', '#ffc633', '#ffffff']}
							destroyOnComplete
						/>
					</div>
					<span class="ach-medal">{medals[myGlobal.rank - 1]}</span>
					<h3>{achTitle[myGlobal.rank - 1]}</h3>
					<p class="muted">
						You finished <b class="rk">#{myGlobal.rank}</b> of {myGlobal.total} players across the
						whole tournament. {myGlobal.rank === 1
							? 'Nobody read this tournament better than you. Take a bow!'
							: 'An amazing run — wear it with pride.'}
					</p>
				</section>
			{/if}

			<SupportCard />
		{/if}

		<!-- ===== Live now / next up match ===== -->
		{#if liveMatch || nextMatch}
			{@const m = (liveMatch ?? nextMatch) as Match}
			{@const isLive = !!liveMatch}
			{@const myTip = tipsStore.tips[m.id]}
			{@const H = slot(m.homeTeam, m.homeLabel)}
			{@const A = slot(m.awayTeam, m.awayLabel)}
			<a class="card next" class:onair={isLive} href="/tips">
				<div class="row">
					<h3>{isLive ? 'Live now' : 'Next up'}</h3>
					<div class="spacer"></div>
					<span class="muted small">{roundOf(m)}</span>
				</div>
				<div class="nm">
					<span class="nm-team">
						<Flag iso2={H.iso2} code={H.code} />
						<span class="nm-name">{H.name}</span>
					</span>
					{#if isLive}
						<span class="nm-score digits">
							<Scoreline home={m.ftHome} away={m.ftAway} etHome={m.etHome} etAway={m.etAway} />
						</span>
					{:else}
						<span class="nm-vs">vs</span>
					{/if}
					<span class="nm-team right">
						<span class="nm-name">{A.name}</span>
						<Flag iso2={A.iso2} code={A.code} />
					</span>
				</div>
				<div class="nm-foot">
					{#if isLive}
						<span class="pill livep">
							<span class="dot"></span>
							Live{liveMatches.length > 1 ? ` · ${liveMatches.length} matches` : ''}
						</span>
						<div class="spacer"></div>
						{#if myTip}
							<span class="muted small">Your tip {myTip.ftHome}:{myTip.ftAway}</span>
						{/if}
					{:else}
						<span class="muted small"><Clock size={14} /> {fmtKick(m.kickoff)}</span>
						<div class="spacer"></div>
						{#if nextTipped}
							<span class="pill ok"><Check size={12} /> Tipped</span>
						{:else if teamsResolved(m)}
							<span class="pill act">Tip it →</span>
						{:else}
							<span class="pill">Teams TBD</span>
						{/if}
					{/if}
				</div>
			</a>
		{/if}

		<!-- ===== Your next moves (gone once the tournament is over) ===== -->
		{#if !allDone}
			<section class="card">
				<h3>Your next moves</h3>
				{#if !ready}
					<p class="muted">Loading…</p>
				{:else if allCaught}
					<p class="caught"><span class="ci"><Check size={18} /></span> You're all caught up — nothing to do but watch.</p>
				{:else}
					<div class="moves">
						{#each moves as m (m.href)}
							{@const Icon = m.icon}
							<a class="move" href={m.href}>
								<span class="mi"><Icon size={20} /></span>
								<span class="mt">
									<span class="title">{m.title}</span>
									<span class="muted sub">{m.sub}</span>
								</span>
								<ChevronRight size={18} class="cr" />
							</a>
						{/each}
					</div>
				{/if}
			</section>
		{/if}


		<!-- ===== Your leagues (with placement) ===== -->
		<section class="card">
			<div class="row">
				<h3>Your leagues</h3>
				<div class="spacer"></div>
				<a class="pill" href="/leagues">Manage</a>
			</div>
			{#if !leaguesLoaded}
				<p class="muted">Loading…</p>
			{:else if leagues.length === 0}
				<p class="muted">
					You're not in a league yet. <a href="/leagues">Create or join one →</a>
				</p>
			{:else}
				{#each orderedLeagues as l (l.id)}
					<a class="lrow" href={`/leagues/${l.id}`}>
						<span class="lname">{l.name}</span>
						{#if l.role === 'owner'}<span class="pill">owner</span>{/if}
						<span class="spacer"></span>
						<span class="standing" title="Your placement · players">
							<Users size={15} />
							{#if ranks[l.id]}
								<b class="rk">#{ranks[l.id]?.rank}</b><small>/{ranks[l.id]?.total}</small>
							{:else}
								<span class="cnt">{l.members}</span>
							{/if}
						</span>
					</a>
				{/each}
			{/if}
		</section>

		<!-- ===== Support (bottom placement while the tournament runs) ===== -->
		{#if !allDone}
			<SupportCard />
		{/if}

		<!-- ===== Survey follow-up: submitted answers stay editable ===== -->
		{#if surveyChecked && surveySubmitted}
			<section class="card">
				<a class="move" href="/survey">
					<span class="mi"><MessageSquareHeart size={20} /></span>
					<span class="mt">
						<span class="title">Survey sent — thank you!</span>
						<span class="muted sub">
							Something new came to mind? Review or change your answers.
						</span>
					</span>
					<ChevronRight size={18} class="cr" />
				</a>
			</section>
		{/if}

		<!-- ===== How does it work ===== -->
		<section class="card">
			<a class="move" href="/welcome">
				<span class="mi"><CircleHelp size={20} /></span>
				<span class="mt">
					<span class="title">How does it work?</span>
					<span class="muted sub">Scoring, forecasts, tips &amp; leagues — explained.</span>
				</span>
				<ChevronRight size={18} class="cr" />
			</a>
		</section>

		<p class="foot muted">
			Matchowl · made by
			<a href="https://floholz.com" target="_blank" rel="noopener">floholz</a>
			· <a href="mailto:{appConfig.contactEmail}">contact</a>
		</p>
	</div>
{/if}

<style>
	header {
		margin: 0.25rem 0 1.25rem;
	}
	h1 {
		margin: 0;
		font-size: 1.6rem;
	}
	header .muted {
		margin: 0.2rem 0 0;
	}
	.small {
		font-size: 0.85rem;
	}

	/* ---------- progress / countdown ---------- */
	.kicker2 {
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--accent);
		margin: 0;
	}
	.prog-head {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.5rem;
		margin-bottom: 0.6rem;
	}
	.phase-lbl {
		font-weight: 700;
		font-size: 1.1rem;
	}
	.pct {
		font-weight: 700;
		font-size: 1.1rem;
		color: var(--accent);
	}
	.bar {
		height: 10px;
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		overflow: hidden;
	}
	.bar > span {
		display: block;
		height: 100%;
		border-radius: var(--radius-pill);
		background: linear-gradient(90deg, var(--accent), var(--accent-2));
		transition: width 0.4s ease;
	}
	.fine {
		font-size: 0.8rem;
		margin: 0.55rem 0 0;
	}
	.cd {
		display: flex;
		gap: 0.7rem;
		margin: 0.35rem 0 0.3rem;
	}
	.cd .u {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.15rem;
		min-width: 2.4ch;
	}
	.cd .u b {
		font-size: 1.85rem;
		font-weight: 700;
		line-height: 1;
		font-variant-numeric: tabular-nums;
	}
	.cd .u i {
		font-style: normal;
		font-size: 0.58rem;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
	}

	/* ---------- post-tournament wrap-up ---------- */
	.wrapup {
		text-align: center;
		padding: 0.75rem 0.25rem 0.35rem;
	}
	.wi {
		display: grid;
		place-items: center;
		width: 46px;
		height: 46px;
		margin: 0 auto 0.6rem;
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--accent) 16%, transparent);
		color: var(--accent);
	}
	.wrapup h2 {
		margin: 0 0 0.35rem;
	}
	.champ {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		margin: 0 0 0.5rem;
		font-size: 1.05rem;
	}
	.wt {
		margin: 0 auto;
		max-width: 46ch;
		line-height: 1.5;
	}

	/* ---------- podium ---------- */
	.podium {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.6rem;
		align-items: end;
		margin-top: 1rem;
	}
	.pstep {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.3rem;
		min-width: 0;
	}
	.pname {
		max-width: 100%;
		font-weight: 700;
		font-size: 0.9rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.p1 .pname {
		font-size: 1rem;
	}
	.ppts {
		font-size: 0.75rem;
		color: var(--muted);
	}
	.pblock {
		display: grid;
		place-items: start center;
		width: 100%;
		margin-top: 0.35rem;
		padding-top: 0.45rem;
		border-radius: var(--radius-sm) var(--radius-sm) 0 0;
		border: 1px solid var(--border);
		border-bottom: none;
	}
	.pmedal {
		font-size: 1.35rem;
		line-height: 1;
	}
	/* Podium blocks: metal-tinted at the usual whisper level, height = rank.
	   Gold comes from the theme token (adapts per theme); silver and bronze are
	   intentional literal podium metals — mixed at 12–40% into theme surfaces/
	   borders they read correctly on dark, light and amoled alike. */
	.podium {
		--metal-gold: var(--gold);
		--metal-silver: #b9c2cc;
		--metal-bronze: #c98a5e;
	}
	.p1 .pblock {
		height: 74px;
		background: color-mix(in srgb, var(--metal-gold) 14%, var(--surface-2));
		border-color: color-mix(in srgb, var(--metal-gold) 40%, var(--border));
	}
	.p2 .pblock {
		height: 52px;
		background: color-mix(in srgb, var(--metal-silver) 12%, var(--surface-2));
		border-color: color-mix(in srgb, var(--metal-silver) 40%, var(--border));
	}
	.p3 .pblock {
		height: 40px;
		background: color-mix(in srgb, var(--metal-bronze) 12%, var(--surface-2));
		border-color: color-mix(in srgb, var(--metal-bronze) 40%, var(--border));
	}
	.mine {
		margin: 0.9rem 0 0;
		text-align: center;
		font-size: 0.88rem;
	}

	/* ---------- personal achievement ---------- */
	.ach {
		position: relative;
		text-align: center;
		padding: 1.6rem 1.25rem;
		background: color-mix(in srgb, var(--accent) 2%, var(--surface));
		border-color: color-mix(in srgb, var(--accent) 20%, var(--border));
	}
	.ach-confetti {
		position: absolute;
		left: 50%;
		top: 20%;
		pointer-events: none;
	}
	.ach-medal {
		display: block;
		font-size: 2.2rem;
		line-height: 1;
		margin-bottom: 0.45rem;
	}
	.ach h3 {
		margin: 0 0 0.35rem;
		font-size: 1.15rem;
	}
	.ach .muted {
		margin: 0 auto;
		max-width: 44ch;
		line-height: 1.5;
	}

	/* ---------- survey promo ---------- */
	.svy {
		background: color-mix(in srgb, var(--accent) 2%, var(--surface));
		border-color: color-mix(in srgb, var(--accent) 15%, var(--border));
	}
	.svy .move {
		padding: 0;
		border-top: none;
	}

	/* ---------- footer ---------- */
	.foot {
		text-align: center;
		font-size: 0.78rem;
		margin: 1.5rem 0 0.5rem;
	}
	.foot a {
		color: inherit;
		text-decoration: underline;
		text-underline-offset: 2px;
	}

	/* ---------- next match ---------- */
	/* The whole card is the link to /tips. */
	.next {
		display: block;
		color: var(--text);
		text-decoration: none;
	}
	/* Live variant — same whisper-level tint as TipCard's live state. */
	.next.onair {
		background: color-mix(in srgb, var(--live) 2%, var(--surface));
		border-color: color-mix(in srgb, var(--live) 22%, var(--border));
	}
	.nm-score {
		font-weight: 700;
		font-size: 1.05rem;
		white-space: nowrap;
	}
	.pill.livep {
		color: var(--bg);
		background: var(--live);
		border-color: var(--live);
	}
	.pill.livep .dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--bg);
		animation: pulse 1.1s ease-in-out infinite;
	}
	@keyframes pulse {
		50% {
			opacity: 0.25;
		}
	}
	.nm {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 0.6rem;
		padding: 0.5rem 0 0.1rem;
		color: var(--text);
	}
	.nm-team {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		min-width: 0;
	}
	.nm-team.right {
		justify-content: flex-end;
	}
	.nm-name {
		font-weight: 600;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.nm-vs {
		font-size: 0.78rem;
		color: var(--muted);
	}
	.nm-foot {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-top: 0.6rem;
	}
	.nm-foot .small {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
	}
	.pill.act {
		color: var(--accent);
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
	}

	/* ---------- next moves ---------- */
	.moves {
		margin-top: 0.6rem;
	}
	.move {
		display: flex;
		align-items: center;
		gap: 0.85rem;
		padding: 0.75rem 0;
		border-top: 1px solid var(--border);
		color: var(--text);
	}
	.move:first-child {
		border-top: none;
	}
	.mi {
		display: grid;
		place-items: center;
		width: 38px;
		height: 38px;
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--accent);
		flex: none;
	}
	.mt {
		display: flex;
		flex-direction: column;
	}
	.title {
		font-weight: 600;
	}
	.sub {
		font-size: 0.82rem;
	}
	:global(.move .cr) {
		margin-left: auto;
		color: var(--muted);
	}
	.caught {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		margin: 0.6rem 0 0;
	}
	.ci {
		display: grid;
		place-items: center;
		width: 32px;
		height: 32px;
		flex: none;
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--accent) 18%, transparent);
		color: var(--accent);
	}

	/* ---------- leagues ---------- */
	.lrow {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.7rem 0;
		border-top: 1px solid var(--border);
		color: var(--text);
	}
	.lrow:first-of-type {
		border-top: none;
	}
	.lname {
		/* Flex items refuse to shrink below their content without this. */
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	/* Combined right-hand indicator: people icon + your placement (#rank/size).
	   The /size doubles as the member count, so no separate count is shown. */
	.standing {
		display: inline-flex;
		align-items: baseline;
		gap: 0.3rem;
		color: var(--muted);
		font-variant-numeric: tabular-nums;
	}
	.standing :global(svg) {
		align-self: center;
	}
	.rk {
		color: var(--accent);
		font-weight: 700;
	}
	.standing small {
		font-size: 0.72rem;
		font-weight: 600;
	}
	.cnt {
		font-size: 0.9rem;
	}
</style>
