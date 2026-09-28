<!-- The head-to-head side of a pool: this matchday's duel, the W-D-L table
     and a browser over every round the pool has played. Data comes from
     /api/pools/{id}/h2h (open rounds carry provisional scores). -->
<script lang="ts">
	import { api, type H2HOverview, type H2HPair, type H2HPerson, type H2HPicks, type H2HPickTeam, type PoolSeason } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import { h2hStore } from '$lib/h2h.svelte';
	import { pb } from '$lib/pb';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { teamLogoUrl } from '$lib/tips.svelte';
	import Avatar from './Avatar.svelte';
	import { Bot, Ghost as GhostIcon, ShieldCheck, Ban, ChevronRight } from '@lucide/svelte';

	let {
		poolId,
		season = null,
		mode = 'overview',
		onLeaderboard = undefined
	}: {
		poolId: string;
		season?: PoolSeason | null;
		/** overview: the matchday strip, the selected matchday's duels, the
		 *  checklist and a short table linking to the leaderboard · table:
		 *  the full W-D-L table only (the pool's Leaderboard tab). */
		mode?: 'overview' | 'table';
		onLeaderboard?: () => void;
	} = $props();

	/** The competition hub's Matches tab filtered to a matchday. */
	const hubRound = (key: string) =>
		season?.competition
			? `/competitions/${season.competition.key}?s=${encodeURIComponent(season.slug)}&tab=matches&round=${encodeURIComponent(key)}`
			: '';

	let data = $state<H2HOverview | null>(null);
	let error = $state('');
	// The selected matchday lives in the URL (?md=) so coming back from a
	// match lands on the card you had open, not the latest one.
	let selected = $state($page.url.searchParams.get('md') ?? '');

	$effect(() => {
		const id = poolId;
		data = null;
		error = '';
		api
			.h2h(id)
			.then((d) => {
				if (id !== poolId) return;
				data = d;
				const known = d.rounds.some((r) => r.key === selected) || d.next?.key === selected;
				if (!known) selected = d.current || d.next?.key || '';
			})
			.catch(() => (error = 'Could not load the head-to-head.'));
	});
	$effect(() => {
		const key = selected;
		if (!data || mode !== 'overview') return;
		// The current matchday is the default and stays out of the URL.
		const want = key && key !== data.current ? key : '';
		const u = new URL($page.url);
		if ((u.searchParams.get('md') ?? '') === want) return;
		if (want) u.searchParams.set('md', want);
		else u.searchParams.delete('md');
		goto(`${u.pathname}${u.search}`, { replaceState: true, noScroll: true, keepFocus: true });
	});

	const me = $derived(auth.user?.id ?? '');
	const avatarUrl = (p: H2HPerson | null) =>
		p?.avatar ? pb.files.getURL({ id: p.userId, collectionName: 'users' }, p.avatar) : null;
	const when = (iso: string) =>
		iso ? new Date(iso).toLocaleString(undefined, { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : '';
	const day = (iso: string) => (iso ? new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' }) : '');
	/** "Regular Season - 12" → "Matchday 12"; anything else stays. */
	const roundName = (r: { label: string; num: number }) => (r.num > 0 && /\d+\s*$/.test(r.label) ? `Matchday ${r.num}` : r.label);

	let round = $derived(data ? (data.rounds.find((r) => r.key === selected) ?? null) : null);
	const mine = (r: { pairs: { a: H2HPerson; b: H2HPerson | null }[] } | null) =>
		r?.pairs.find((p) => p.a.userId === me || p.b?.userId === me) ?? null;
	/** My side first. */
	function faced(p: H2HPair): { me: H2HPerson; them: H2HPerson | null; mine: number; theirs: number; pts: number } {
		return p.a.userId === me
			? { me: p.a, them: p.b, mine: p.scoreA, theirs: p.scoreB, pts: p.ptsA }
			: { me: p.b!, them: p.a, mine: p.scoreB, theirs: p.scoreA, pts: p.ptsB };
	}
	let myNext = $derived(data?.next ? mine(data.next) : null);
	let seasonStarted = $derived(!!data && data.rounds.length > 0);
	let rows = $derived(data?.table ?? []);
	/** The overview's short table: the top three, plus me when I am lower. */
	let shownRows = $derived.by(() => {
		const all = rows.map((r, i) => ({ r, i }));
		if (mode === 'table' || all.length <= 4) return all;
		const top = all.slice(0, 3);
		const mine = all.find((x) => x.r.userId === me);
		return mine && mine.i >= 3 ? [...top, mine] : top;
	});
	/** What the next matchday still wants from me (from the shared store). */
	let nextTodo = $derived.by(() => {
		const n = h2hStore.pools.find((p) => p.poolId === poolId)?.next;
		if (!n) return '';
		const parts: string[] = [];
		if (n.untipped) parts.push(`${n.untipped} to tip`);
		if ((n.saveCallsLeft ?? 0) > 0) parts.push(`Save Call ${n.saveCallsLeft} left`);
		if (n.paired && !n.ghost && n.banPlaced === false) parts.push('ban open');
		return parts.join(' · ');
	});
	$effect(() => {
		if (auth.isAuthed) h2hStore.load().catch(() => {});
	});
	// Centre the selected card in the strip (instantly on load, smoothly after).
	let stripEl = $state<HTMLElement | null>(null);
	let scrolledOnce = false;
	$effect(() => {
		const key = selected;
		const el = stripEl;
		if (!el || !key) return;
		const card = el.querySelector<HTMLElement>(`[data-key="${CSS.escape(key)}"]`);
		if (!card) return;
		const left = card.offsetLeft - (el.clientWidth - card.clientWidth) / 2;
		el.scrollTo({ left, behavior: scrolledOnce ? 'smooth' : 'instant' });
		scrolledOnce = true;
	});

	// ---- picks: save calls and the ban, for the selected round while it
	// takes them (open, or the one that opens next) ----
	let picks = $state<H2HPicks | null>(null);
	let pickErr = $state('');
	let pickBusy = $state('');
	let pickRound = $derived.by(() => {
		if (!data) return '';
		if (selected === data.next?.key) return selected;
		const r = data.rounds.find((r) => r.key === selected);
		return r && r.status === 'open' ? r.key : '';
	});
	$effect(() => {
		const key = pickRound;
		const id = poolId;
		picks = null;
		pickErr = '';
		if (!key) return;
		api
			.h2hPicks(id, key)
			.then((p) => {
				if (key === pickRound) picks = p;
			})
			.catch(() => (pickErr = 'Could not load your calls.'));
	});
	async function setPick(m: { id: string; saved: boolean; banned: boolean }, kind: 'save' | 'ban') {
		if (!picks || pickBusy) return;
		pickBusy = m.id + kind;
		pickErr = '';
		try {
			picks = await api.h2hSetPick(poolId, m.id, kind, kind === 'save' ? !m.saved : !m.banned);
			h2hStore.invalidate();
		} catch (e: unknown) {
			const msg = (e as { response?: { error?: string } })?.response?.error;
			pickErr = msg || 'Could not place that.';
		} finally {
			pickBusy = '';
		}
	}
	const logo = (t: H2HPickTeam | null) => (t ? teamLogoUrl(t.id, t.logo) : '');
	let untipped = $derived(picks ? picks.matches.filter((m) => !m.tip && !m.locked).length : 0);
	const kick = (iso: string) => new Date(iso).toLocaleString(undefined, { weekday: 'short', hour: '2-digit', minute: '2-digit' });
</script>

{#if error}
	<p class="error">{error}</p>
{:else if !data}
	<p class="muted">Loading…</p>
{:else}
	<!-- The matchday strip: one card per matchday — played ones with my
	     result, the open one with the live score, the next one with the
	     rival and what is left to do — so last week and next week are one
	     flick apart, never a mode switch. The selected card's duels and
	     checklist follow, then the table. -->
	{#if mode === 'overview'}
	<div class="strip" bind:this={stripEl}>
		{#each data.rounds as r (r.key)}
			{@const p = mine(r) as H2HPair | null}
			{@const open = r.status === 'open'}
			{@const f = p ? faced(p) : null}
			<button
				class="mcard"
				class:on={r.key === selected}
				class:live={open}
				class:win={!!f && f.pts === 3}
				class:draw={!!f && f.pts === 1}
				class:loss={!!f && f.pts === 0 && (f.mine > 0 || f.theirs > 0)}
				data-key={r.key}
				onclick={() => (selected = r.key)}
			>
				<span class="mhead"><b>{roundName(r)}</b><span class="mst" class:islive={open}>{open ? 'in play' : 'final'}</span></span>
				{#if f}
					<span class="mscore digits">{f.mine}<i>–</i>{f.theirs}</span>
					<span class="mwho">
						{#if f.them}<Avatar name={f.them.name} src={avatarUrl(f.them)} size={18} />{:else}<span class="ghost xs"><GhostIcon size={11} /></span>{/if}
						<span class="mname">{f.them ? f.them.name : 'The Ghost'}</span>
					</span>
					<span class="mfoot">{open ? `${r.counted} of ${r.matches} in` : f.pts === 3 ? 'won · +3' : f.pts === 1 ? 'draw · +1' : 'lost'}</span>
				{:else}
					<span class="mscore digits muted">–</span>
					<span class="mfoot">not paired</span>
				{/if}
			</button>
		{/each}
		{#if data.next}
			{@const n = myNext ? (myNext.a.userId === me ? myNext.b : myNext.a) : null}
			<button class="mcard next" class:on={data.next.key === selected} data-key={data.next.key} onclick={() => (selected = data!.next!.key)}>
				<span class="mhead"><b>{roundName(data.next)}</b><span class="mst">{data.rounds.length ? 'next' : 'first'}</span></span>
				<span class="mvs">vs</span>
				<span class="mwho">
					{#if myNext}
						{#if n}<Avatar name={n.name} src={avatarUrl(n)} size={18} />{:else}<span class="ghost xs"><GhostIcon size={11} /></span>{/if}
						<span class="mname">{n ? n.name : 'The Ghost'}</span>
					{:else}
						<span class="mname muted">not paired</span>
					{/if}
				</span>
				<span class="mfoot">{#if nextTodo}<span class="todo">{nextTodo}</span>{:else}opens {day(data.next.firstKickoff)}{/if}</span>
			</button>
		{/if}
		{#if !data.rounds.length && !data.next}
			<div class="mcard">
				<span class="mhead"><b>{roundName({ label: data.firstRound.label, num: 0 })}</b></span>
				<span class="mfoot">kicks off {when(data.firstRound.firstKickoff)}</span>
			</div>
		{/if}
	</div>
	{#if !seasonStarted}
		<p class="muted small intro">Every matchday pairs you with a pool mate; your tip points decide the duel.</p>
	{/if}

	{#if round || (data.next && selected === data.next.key)}
		<section class="card rounds">
			{#if round}
				<div class="rhead">
					{#if hubRound(round.key)}<a class="rlink" href={hubRound(round.key)}><b>{roundName(round)}</b><ChevronRight size={14} /></a>{:else}<b>{roundName(round)}</b>{/if}
					<span class="muted small">
						{#if round.status === 'open'}
							in play · {round.counted} of {round.matches} matches in · closes {when(round.closesAt)}
						{:else}
							final · {round.counted} of {round.matches} matches counted{round.counted < round.matches ? ' (the rest kicked off too late)' : ''}
						{/if}
					</span>
				</div>
				<ul class="pairs">
					{#each round.pairs as p (p.a.userId)}
						<li class:mine={p.a.userId === me || p.b?.userId === me}>
							<span class="pa" class:won={p.ptsA === 3} class:lost={p.ptsA === 0 && p.ptsB === 3}>
								<Avatar name={p.a.name} src={avatarUrl(p.a)} size={24} />
								<span class="pn">{p.a.name}</span>
								{#if p.savesA}<span class="mark" title="Save Calls revealed"><ShieldCheck size={11} />{p.savesA}</span>{/if}
								{#if p.bannedA}<span class="mark ban" title="A match banned by the rival"><Ban size={11} /></span>{/if}
							</span>
							<span class="ps digits"><b class:hi={p.ptsA === 3}>{p.scoreA}</b><i>–</i><b class:hi={p.ptsB === 3}>{p.scoreB}</b></span>
							<span class="pb" class:won={p.ptsB === 3} class:lost={p.ptsB === 0 && p.ptsA === 3}>
								{#if p.bannedB}<span class="mark ban" title="A match banned by the rival"><Ban size={11} /></span>{/if}
								{#if p.savesB}<span class="mark" title="Save Calls revealed"><ShieldCheck size={11} />{p.savesB}</span>{/if}
								{#if p.b}
									<span class="pn">{p.b.name}</span>
									<Avatar name={p.b.name} src={avatarUrl(p.b)} size={24} />
								{:else}
									<span class="pn">Ghost</span>
									<span class="ghost sm"><GhostIcon size={14} /></span>
								{/if}
							</span>
						</li>
					{/each}
				</ul>
			{/if}
			{#if data.next && selected === data.next.key}
				<div class="rhead">
					{#if hubRound(data.next.key)}<a class="rlink" href={hubRound(data.next.key)}><b>{roundName(data.next)}</b><ChevronRight size={14} /></a>{:else}<b>{roundName(data.next)}</b>{/if}
					<span class="muted small">opens {when(data.next.firstKickoff)} · {data.next.matches} matches</span>
				</div>
				<ul class="pairs preview">
					{#each data.next.pairs as p (p.a.userId)}
						<li class:mine={p.a.userId === me || p.b?.userId === me}>
							<span class="pa"><Avatar name={p.a.name} src={avatarUrl(p.a)} size={24} /><span class="pn">{p.a.name}</span></span>
							<span class="ps muted">v</span>
							<span class="pb">{#if p.b}<span class="pn">{p.b.name}</span><Avatar name={p.b.name} src={avatarUrl(p.b)} size={24} />{:else}<span class="pn">Ghost</span><span class="ghost sm"><GhostIcon size={14} /></span>{/if}</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
	{/if}

	{#if pickRound && mode === 'overview'}
		<section class="card calls">
			{#if pickErr && !picks}
				<p class="error">{pickErr}</p>
			{:else if !picks}
				<p class="muted small">Loading your calls…</p>
			{:else}
				<div class="rhead">
					{#if hubRound(picks.round.key)}<a class="rlink" href={hubRound(picks.round.key)}><b>Your matchday · {roundName(picks.round)}</b><ChevronRight size={14} /></a>{:else}<b>Your matchday · {roundName(picks.round)}</b>{/if}
					<span class="muted small">
						{#if picks.closed}
							closed
						{:else}
							{#if untipped}<span class="todo">{untipped} to tip</span> · {/if}<ShieldCheck size={12} /> {picks.saveCallsLeft} of {picks.saveCalls} left · <Ban size={12} /> {picks.ghost ? 'no rival' : picks.mine.ban ? 'placed' : 'open'}
						{/if}
					</span>
				</div>
				<p class="muted small chelp">
					{#if picks.ghost}
						A Save Call counts double for you — the tip you would bet the house on. You play the Ghost this matchday, so there is no one to ban.
					{:else if picks.paired}
						A Save Call counts double for you — the tip you would bet the house on. Your ban takes a match away from {picks.rival?.name ?? 'your rival'} — they only see it once it kicks off. A ban on a Save Call cancels the double.
					{:else}
						You are not paired this matchday.
					{/if}
				</p>
				{#if pickErr}<p class="error small">{pickErr}</p>{/if}
				<ul class="cmatches">
					{#each picks.matches as m (m.id)}
						<li class:locked={m.locked}>
							<a class="cteams" href={`/m/${m.id}`}>
								<span class="ct"><img src={logo(m.home)} alt="" class="crest" />{m.home?.name ?? '?'}</span>
								<span class="ct"><img src={logo(m.away)} alt="" class="crest" />{m.away?.name ?? '?'}</span>
							</a>
							<a class="cwhen muted small" href={`/m/${m.id}`}>
								{#if m.ftHome !== undefined}<span class="digits">{m.ftHome}–{m.ftAway}</span>{:else}{kick(m.kickoff)}{/if}
								{#if m.tip}<span class="capsule digits" class:dim={m.locked}>{m.tip.ftHome}–{m.tip.ftAway}</span>{:else if !m.locked}<span class="capsule todo">no tip</span>{:else}<span class="capsule dim">–</span>{/if}
								{#if m.rivalSaved}<span class="mark" title="{picks.rival?.name} made this a Save Call"><ShieldCheck size={11} /> {picks.rival?.name}</span>{/if}
								{#if m.rivalBanned}<span class="mark ban" title="{picks.rival?.name} banned this for you"><Ban size={11} /> banned</span>{/if}
							</a>
							<span class="cbtns">
								<button class="pk" class:on={m.saved} disabled={m.locked || picks.closed || !!pickBusy || (!m.saved && picks.saveCallsLeft === 0)} aria-label={m.saved ? 'Take the Save Call back' : 'Save Call'} title={m.saved ? 'Take the Save Call back' : 'Save Call: counts double for you'} onclick={() => setPick(m, 'save')}><ShieldCheck size={15} /></button>
								{#if !picks.ghost && picks.paired}
									<button class="pk ban" class:on={m.banned} disabled={m.locked || picks.closed || !!pickBusy} aria-label={m.banned ? 'Lift the ban' : 'Ban for your rival'} title={m.banned ? 'Lift the ban' : `Ban: does not count for ${picks.rival?.name ?? 'your rival'}`} onclick={() => setPick(m, 'ban')}><Ban size={15} /></button>
								{/if}
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}

	<section class="card board">
		<table class="lb">
			<thead>
				<tr>
					<th>#</th>
					<th>Player</th>
					<th class="num" title="Matchdays played">P</th>
					<th class="num" title="Won">W</th>
					<th class="num" title="Drawn">D</th>
					<th class="num" title="Lost">L</th>
					<th class="num ext" title="Tip points this season (tiebreak)">Tips</th>
					<th class="num pts" title="3 per win, 1 per draw">Pts</th>
				</tr>
			</thead>
			<tbody>
				{#each shownRows as { r, i } (r.userId)}
					<tr class:lead={r.userId === me}>
						<td class="rank"><span class="medal" class:g={i === 0} class:s={i === 1} class:b={i === 2}>{i + 1}</span></td>
						<td class="player">
							<div class="pwrap">
								<Avatar name={r.name} src={avatarUrl(r)} size={28} />
								<span class="pname">{r.name}</span>
								{#if r.role === 'bot'}<span class="rolepill" title="Bot player"><Bot size={11} /> Bot</span>{/if}
							</div>
						</td>
						<td class="num digits">{r.played}</td>
						<td class="num digits">{r.won}</td>
						<td class="num digits">{r.drawn}</td>
						<td class="num digits">{r.lost}</td>
						<td class="num ext digits">{r.tipsPoints}</td>
						<td class="num pts digits">{r.points}</td>
					</tr>
				{/each}
			</tbody>
		</table>
		{#if mode === 'overview'}
			<button class="seeall" onclick={onLeaderboard}>Leaderboard · {rows.length} players <ChevronRight size={14} /></button>
		{:else}
			<p class="muted small note">
				3 for a win, 1 for a draw. Ties on points go to the season's tip points. Rounds close 24 h after the matchday's last scheduled kick-off.
			</p>
		{/if}
	</section>
{/if}

<style>
	.card {
		margin-bottom: 0.8rem;
	}
	.strip {
		display: flex;
		gap: 0.6rem;
		overflow-x: auto;
		scroll-snap-type: x mandatory;
		scrollbar-width: none;
		padding: 0.2rem 0.15rem 0.5rem;
	}
	/* Phones: edge to edge, so the neighbours peek in at both sides. */
	@media (max-width: 899px) {
		.strip {
			margin: 0 calc(-1 * var(--shell-x, 1rem));
			padding-inline: var(--shell-x, 1rem);
		}
	}
	.strip::-webkit-scrollbar {
		display: none;
	}
	.mcard {
		flex: 0 0 auto;
		width: min(62vw, 220px);
		scroll-snap-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.2rem;
		padding: 0.6rem 0.7rem 0.55rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--text);
		font: inherit;
		text-align: center;
		cursor: pointer;
		transition: border-color 0.15s ease;
	}
	.mcard.on {
		border-color: var(--accent);
		box-shadow: var(--glow);
	}
	.mcard.live {
		border-color: color-mix(in srgb, var(--live, #e0443e) 60%, var(--border));
	}
	.mcard.next {
		border-style: dashed;
	}
	.mhead {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 0.5rem;
		width: 100%;
		font-size: 0.8rem;
	}
	.mst {
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.mst.islive {
		color: var(--live, #e0443e);
	}
	.mscore {
		font-size: 1.7rem;
		font-weight: 800;
		letter-spacing: 0.02em;
		display: inline-flex;
		align-items: baseline;
		gap: 0.1rem;
		line-height: 1.1;
	}
	.mscore i {
		font-style: normal;
		color: var(--muted);
		font-size: 1rem;
	}
	.mcard.win .mscore {
		color: var(--accent);
	}
	.mcard.loss .mscore {
		color: var(--muted);
	}
	.mvs {
		font-size: 1.1rem;
		font-weight: 800;
		color: var(--muted);
		line-height: 1.6;
	}
	.mwho {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		max-width: 100%;
		font-size: 0.78rem;
		font-weight: 700;
	}
	.mname {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.mfoot {
		font-size: 0.72rem;
		color: var(--muted);
	}
	.intro {
		margin: 0 0 0.6rem;
	}
	.seeall {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.15rem;
		width: 100%;
		margin: 0.4rem 0 0;
		padding: 0.5rem;
		border: none;
		background: transparent;
		color: var(--accent);
		font: inherit;
		font-size: 0.82rem;
		font-weight: 600;
		cursor: pointer;
	}
	.ghost {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border-radius: 50%;
		border: 1px dashed var(--border);
		color: var(--muted);
	}
	.ghost.sm {
		width: 24px;
		height: 24px;
	}
	.ghost.xs {
		width: 18px;
		height: 18px;
	}
	.rlink {
		display: inline-flex;
		align-items: center;
		gap: 0.15rem;
		color: inherit;
		text-decoration: none;
	}
	.rlink:hover b {
		color: var(--accent);
	}
	.rhead {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 0.6rem;
		flex-wrap: wrap;
	}
	.small {
		font-size: 0.8rem;
	}
	.note {
		margin: 0.6rem 0 0;
	}
	.lb {
		width: 100%;
		border-collapse: collapse;
	}
	.lb th,
	.lb td {
		text-align: left;
		padding: 0.55rem 0.35rem;
		border-bottom: 1px solid var(--border);
	}
	.lb th {
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.lb tbody tr:last-child td {
		border-bottom: 0;
	}
	.lb .num {
		text-align: right;
	}
	.lb .pts {
		font-weight: 800;
	}
	.lb tr.lead td {
		background: color-mix(in srgb, var(--accent) 10%, transparent);
	}
	.pwrap {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		min-width: 0;
	}
	.pname {
		font-weight: 700;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.rolepill {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		padding: 0.05rem 0.4rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		font-size: 0.62rem;
		color: var(--muted);
	}
	.medal {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		font-size: 0.75rem;
		font-weight: 800;
		color: var(--muted);
	}
	.medal.g {
		background: #d4a017;
		color: #1c0e00;
	}
	.medal.s {
		background: #b8b8b8;
		color: #1c0e00;
	}
	.medal.b {
		background: #a5652d;
		color: #1c0e00;
	}
	@media (max-width: 420px) {
		.lb .ext {
			display: none;
		}
	}
	.pairs {
		list-style: none;
		margin: 0.6rem 0 0;
		padding: 0;
	}
	.pairs li {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.2rem;
		border-bottom: 1px solid var(--border);
	}
	.pairs li:last-child {
		border-bottom: 0;
	}
	.pairs li.mine {
		background: color-mix(in srgb, var(--accent) 10%, transparent);
		border-radius: var(--radius-sm);
		padding-inline: 0.4rem;
	}
	.pa,
	.pb {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		min-width: 0;
		font-size: 0.85rem;
	}
	.pb {
		justify-content: flex-end;
	}
	.pn {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.lost .pn {
		color: var(--muted);
	}
	.ps {
		font-weight: 800;
		font-size: 1rem;
		display: inline-flex;
		gap: 0.15rem;
		align-items: baseline;
	}
	.ps i {
		font-style: normal;
		color: var(--muted);
	}
	.ps b {
		color: var(--muted);
	}
	.ps b.hi {
		color: var(--accent);
	}
	.mark {
		display: inline-flex;
		align-items: center;
		gap: 0.1rem;
		padding: 0.05rem 0.3rem;
		border-radius: 999px;
		border: 1px solid var(--accent);
		color: var(--accent);
		font-size: 0.62rem;
		font-weight: 700;
	}
	.mark.ban {
		border-color: var(--live, #e0443e);
		color: var(--live, #e0443e);
	}
	.chelp {
		margin: 0.3rem 0 0.4rem;
	}
	.error.small {
		font-size: 0.8rem;
	}
	.cmatches {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.cmatches li {
		display: grid;
		grid-template-columns: 1fr auto auto;
		align-items: center;
		gap: 0.5rem;
		padding: 0.45rem 0;
		border-bottom: 1px solid var(--border);
	}
	.cmatches li:last-child {
		border-bottom: 0;
	}
	.cmatches li.locked .cteams {
		color: var(--muted);
	}
	.cteams {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		min-width: 0;
		font-size: 0.82rem;
		font-weight: 600;
		color: inherit;
		text-decoration: none;
	}
	.cwhen {
		text-decoration: none;
	}
	.capsule {
		display: inline-flex;
		align-items: center;
		height: 20px;
		padding: 0 0.45rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--accent);
		color: var(--accent);
		font-size: 0.72rem;
		font-weight: 700;
	}
	.capsule.dim {
		border-color: var(--border);
		color: var(--muted);
	}
	.capsule.todo,
	.todo {
		color: var(--accent);
		font-weight: 700;
	}
	.capsule.todo {
		border-style: dashed;
	}
	.ct {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.crest {
		width: 16px;
		height: 16px;
		object-fit: contain;
	}
	.cwhen {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 0.15rem;
		text-align: right;
	}
	.cbtns {
		display: inline-flex;
		gap: 0.3rem;
	}
	.pk {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 34px;
		height: 34px;
		border-radius: 50%;
		border: 1px solid var(--border);
		background: var(--surface-2);
		color: var(--muted);
	}
	.pk.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.pk.ban.on {
		background: var(--live, #e0443e);
		border-color: var(--live, #e0443e);
		color: #fff;
	}
	.pk:disabled {
		opacity: 0.45;
	}
</style>
