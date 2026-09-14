<!-- The head-to-head side of a pool: this matchday's duel, the W-D-L table
     and a browser over every round the pool has played. Data comes from
     /api/pools/{id}/h2h (open rounds carry provisional scores). -->
<script lang="ts">
	import { api, type H2HOverview, type H2HPair, type H2HPerson, type H2HPicks, type H2HPickTeam } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import { pb } from '$lib/pb';
	import { teamLogoUrl } from '$lib/tips.svelte';
	import Avatar from './Avatar.svelte';
	import { Bot, Ghost as GhostIcon, Star, Ban } from '@lucide/svelte';

	let { poolId }: { poolId: string } = $props();

	let data = $state<H2HOverview | null>(null);
	let error = $state('');
	let selected = $state('');

	$effect(() => {
		const id = poolId;
		data = null;
		error = '';
		api
			.h2h(id)
			.then((d) => {
				if (id !== poolId) return;
				data = d;
				selected = d.current;
			})
			.catch(() => (error = 'Could not load the head-to-head.'));
	});

	const me = $derived(auth.user?.id ?? '');
	const avatarUrl = (p: H2HPerson | null) =>
		p?.avatar ? pb.files.getURL({ id: p.userId, collectionName: 'users' }, p.avatar) : null;
	const when = (iso: string) =>
		iso ? new Date(iso).toLocaleString(undefined, { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : '';
	const day = (iso: string) => (iso ? new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' }) : '');
	/** "Regular Season - 12" → "Matchday 12"; anything else stays. */
	const roundName = (r: { label: string; num: number }) => (r.num > 0 && /\d+\s*$/.test(r.label) ? `Matchday ${r.num}` : r.label);

	/** The latest round (open, else the last closed) — the headline. */
	let latest = $derived(data ? (data.rounds.find((r) => r.key === data!.current) ?? null) : null);
	let round = $derived(data ? (data.rounds.find((r) => r.key === selected) ?? null) : null);
	const mine = (r: { pairs: { a: H2HPerson; b: H2HPerson | null }[] } | null) =>
		r?.pairs.find((p) => p.a.userId === me || p.b?.userId === me) ?? null;
	/** My side first. */
	function faced(p: H2HPair): { me: H2HPerson; them: H2HPerson | null; mine: number; theirs: number; pts: number } {
		return p.a.userId === me
			? { me: p.a, them: p.b, mine: p.scoreA, theirs: p.scoreB, pts: p.ptsA }
			: { me: p.b!, them: p.a, mine: p.scoreB, theirs: p.scoreA, pts: p.ptsB };
	}
	const verdict = (pts: number, open: boolean) =>
		open ? (pts === 3 ? 'ahead' : pts === 1 ? 'level' : 'behind') : pts === 3 ? 'won' : pts === 1 ? 'draw' : 'lost';
	let myLatest = $derived(latest ? (mine(latest) as H2HPair | null) : null);
	let myNext = $derived(data?.next ? mine(data.next) : null);
	let seasonStarted = $derived(!!data && data.rounds.length > 0);
	let rows = $derived(data?.table ?? []);

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
		} catch (e: unknown) {
			const msg = (e as { response?: { error?: string } })?.response?.error;
			pickErr = msg || 'Could not place that.';
		} finally {
			pickBusy = '';
		}
	}
	const logo = (t: H2HPickTeam | null) => (t ? teamLogoUrl(t.id, t.logo) : '');
	const kick = (iso: string) => new Date(iso).toLocaleString(undefined, { weekday: 'short', hour: '2-digit', minute: '2-digit' });
</script>

{#if error}
	<p class="error">{error}</p>
{:else if !data}
	<p class="muted">Loading…</p>
{:else}
	<section class="card duel">
		{#if latest && myLatest}
			{@const f = faced(myLatest)}
			{@const open = latest.status === 'open'}
			<div class="dhead">
				<b>{roundName(latest)}</b>
				<span class="muted small">
					{#if open}
						{latest.counted} of {latest.matches} matches in · closes {when(latest.closesAt)}
					{:else}
						final
					{/if}
				</span>
			</div>
			<div class="vs" class:open>
				<span class="side me">
					<Avatar name={f.me.name} src={avatarUrl(f.me)} size={36} />
					<span class="sname">You</span>
				</span>
				<span class="score digits" class:win={f.pts === 3} class:draw={f.pts === 1} class:loss={f.pts === 0 && (f.mine > 0 || f.theirs > 0)}>
					{f.mine}<i>–</i>{f.theirs}
				</span>
				<span class="side them">
					{#if f.them}
						<Avatar name={f.them.name} src={avatarUrl(f.them)} size={36} />
						<span class="sname">{f.them.name}</span>
					{:else}
						<span class="ghost"><GhostIcon size={20} /></span>
						<span class="sname">The Ghost</span>
					{/if}
				</span>
			</div>
			<div class="muted small dfoot">
				{#if open}
					You are {verdict(f.pts, true)}{f.them ? '' : ' of the Ghost, who scores the mean of everyone else'}.
				{:else}
					You {verdict(f.pts, false)} this matchday{f.pts === 3 ? ' · +3' : f.pts === 1 ? ' · +1' : ''}.
				{/if}
				{#if data.next && myNext}
					{@const n = myNext.a.userId === me ? myNext.b : myNext.a}
					Next: {n ? n.name : 'the Ghost'} on {roundName(data.next)}, {day(data.next.firstKickoff)}.
				{/if}
			</div>
		{:else if latest}
			<div class="dhead"><b>{roundName(latest)}</b><span class="muted small">{latest.status === 'open' ? 'in play' : 'final'}</span></div>
			<p class="muted small dfoot">
				You are not paired this matchday — you join from
				{data.next ? `${roundName(data.next)} (${day(data.next.firstKickoff)})` : 'the next one'}{#if myNext}, against {(myNext.a.userId === me ? myNext.b : myNext.a)?.name ?? 'the Ghost'}{/if}.
			</p>
		{:else}
			<div class="dhead"><b>Kicks off with {roundName({ label: data.firstRound.label, num: data.next?.num ?? 0 })}</b><span class="muted small">{when(data.firstRound.firstKickoff)}</span></div>
			<p class="muted small dfoot">
				{#if myNext}
					First up: you against {(myNext.a.userId === me ? myNext.b : myNext.a)?.name ?? 'the Ghost'}. Every matchday pairs you with a pool mate; your tip points decide the duel.
				{:else}
					Every matchday pairs you with a pool mate; your tip points decide the duel.
				{/if}
			</p>
		{/if}
	</section>

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
				{#each rows as r, i (r.userId)}
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
		<p class="muted small note">
			3 for a win, 1 for a draw. Ties on points go to the season's tip points. Rounds close 24 h after the matchday's last scheduled kick-off.
		</p>
	</section>

	{#if data.rounds.length}
		<section class="card rounds">
			<div class="rchips">
				{#each data.rounds as r (r.key)}
					<button class="rchip" class:on={r.key === selected} class:live={r.status === 'open'} onclick={() => (selected = r.key)}>{r.num > 0 ? r.num : r.label}</button>
				{/each}
				{#if data.next}
					<button class="rchip next" class:on={data.next.key === selected} onclick={() => (selected = data!.next!.key)}>{data.next.num > 0 ? data.next.num : 'next'}</button>
				{/if}
			</div>
			{#if round}
				<div class="rhead">
					<b>{roundName(round)}</b>
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
								{#if p.savesA}<span class="mark" title="Save calls revealed"><Star size={11} />{p.savesA}</span>{/if}
								{#if p.bannedA}<span class="mark ban" title="A match banned by the rival"><Ban size={11} /></span>{/if}
							</span>
							<span class="ps digits"><b class:hi={p.ptsA === 3}>{p.scoreA}</b><i>–</i><b class:hi={p.ptsB === 3}>{p.scoreB}</b></span>
							<span class="pb" class:won={p.ptsB === 3} class:lost={p.ptsB === 0 && p.ptsA === 3}>
								{#if p.bannedB}<span class="mark ban" title="A match banned by the rival"><Ban size={11} /></span>{/if}
								{#if p.savesB}<span class="mark" title="Save calls revealed"><Star size={11} />{p.savesB}</span>{/if}
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
					<b>{roundName(data.next)}</b>
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

	{#if pickRound}
		<section class="card calls">
			{#if pickErr && !picks}
				<p class="error">{pickErr}</p>
			{:else if !picks}
				<p class="muted small">Loading your calls…</p>
			{:else}
				<div class="rhead">
					<b>Your calls · {roundName(picks.round)}</b>
					<span class="muted small">
						{#if picks.closed}
							closed
						{:else}
							<Star size={12} /> {picks.saveCallsLeft} of {picks.saveCalls} left · <Ban size={12} /> {picks.ghost ? 'no rival' : picks.mine.ban ? 'placed' : 'open'}
						{/if}
					</span>
				</div>
				<p class="muted small chelp">
					{#if picks.ghost}
						A save call counts double for you. You play the Ghost this matchday, so there is no one to ban.
					{:else if picks.paired}
						A save call counts double for you. Your ban takes a match away from {picks.rival?.name ?? 'your rival'} — they only see it once it kicks off. A ban on a save call cancels the double.
					{:else}
						You are not paired this matchday.
					{/if}
				</p>
				{#if pickErr}<p class="error small">{pickErr}</p>{/if}
				<ul class="cmatches">
					{#each picks.matches as m (m.id)}
						<li class:locked={m.locked}>
							<span class="cteams">
								<span class="ct"><img src={logo(m.home)} alt="" class="crest" />{m.home?.name ?? '?'}</span>
								<span class="ct"><img src={logo(m.away)} alt="" class="crest" />{m.away?.name ?? '?'}</span>
							</span>
							<span class="cwhen muted small">
								{#if m.ftHome !== undefined}<span class="digits">{m.ftHome}–{m.ftAway}</span>{:else}{kick(m.kickoff)}{/if}
								{#if m.rivalSaved}<span class="mark" title="{picks.rival?.name} saved this"><Star size={11} /> {picks.rival?.name}</span>{/if}
								{#if m.rivalBanned}<span class="mark ban" title="{picks.rival?.name} banned this for you"><Ban size={11} /> banned</span>{/if}
							</span>
							<span class="cbtns">
								<button class="pk" class:on={m.saved} disabled={m.locked || picks.closed || !!pickBusy || (!m.saved && picks.saveCallsLeft === 0)} aria-label={m.saved ? 'Take the save call back' : 'Save call'} title={m.saved ? 'Take the save call back' : 'Save call: counts double for you'} onclick={() => setPick(m, 'save')}><Star size={15} /></button>
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
{/if}

<style>
	.card {
		margin-bottom: 0.8rem;
	}
	.duel {
		padding: 0.9rem 1rem;
	}
	.dhead,
	.rhead {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 0.6rem;
		flex-wrap: wrap;
	}
	.vs {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 0.6rem;
		margin: 0.8rem 0 0.5rem;
	}
	.side {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.3rem;
		min-width: 0;
	}
	.sname {
		font-size: 0.8rem;
		font-weight: 700;
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.score {
		font-size: 1.9rem;
		font-weight: 800;
		letter-spacing: 0.02em;
		display: inline-flex;
		align-items: baseline;
		gap: 0.15rem;
	}
	.score i {
		font-style: normal;
		color: var(--muted);
		font-size: 1.2rem;
	}
	.score.win {
		color: var(--accent);
	}
	.score.loss {
		color: var(--muted);
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
	.dfoot {
		margin: 0.2rem 0 0;
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
	.rchips {
		display: flex;
		gap: 0.35rem;
		overflow-x: auto;
		padding-bottom: 0.4rem;
		margin-bottom: 0.4rem;
		scrollbar-width: none;
	}
	.rchip {
		flex: 0 0 auto;
		min-width: 34px;
		height: 30px;
		padding: 0 0.6rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface-2);
		color: var(--muted);
		font: inherit;
		font-weight: 700;
		font-size: 0.78rem;
	}
	.rchip.live {
		border-color: var(--live, #e0443e);
	}
	.rchip.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
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
	.rchip.next {
		border-style: dashed;
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
