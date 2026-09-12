<!-- The hub's Knockout tab: one round at a time (round pills, "Now" = the
     current round), each tie as a row — teams stacked, 1st · 2nd leg
     scores, the aggregate on the LED board with the advancer dot, and a
     strip saying how / when. Single-match rounds show one score column.
     Undecided pairings are dim placeholders. -->
<script lang="ts">
	import { tipsStore, findOtherLeg, legScore, type Match } from '$lib/tips.svelte';
	import { tournamentStore } from '$lib/tournament.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import Flag from './Flag.svelte';
	import LedBoard from './LedBoard.svelte';
	import { ChevronRight, LocateFixed } from '@lucide/svelte';

	interface Tie {
		id: string;
		home: string;
		away: string;
		homeLabel: string;
		awayLabel: string;
		/** Kick-off order; two for a two-legged tie. */
		legs: Match[];
	}
	const played = (m: Match) => m.status === 'finished' || !!m.finalizedAt;
	const byKick = (a: Match, b: Match) =>
		new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime() || a.num - b.num;

	function tiesOf(code: string): Tie[] {
		const ms = tipsStore.matches.filter((m) => m.stage === code).sort(byKick);
		const seen = new Set<string>();
		const out: Tie[] = [];
		for (const m of ms) {
			if (seen.has(m.id)) continue;
			const found = findOtherLeg(tipsStore.matches, m);
			if (found) {
				seen.add(found.other.id);
				const legs = [m, found.other].sort(byKick);
				out.push({ id: legs[0].id, home: legs[0].homeTeam, away: legs[0].awayTeam, homeLabel: '', awayLabel: '', legs });
			} else {
				out.push({ id: m.id, home: m.homeTeam, away: m.awayTeam, homeLabel: m.homeLabel, awayLabel: m.awayLabel, legs: [m] });
			}
		}
		return out;
	}
	let rounds = $derived(
		tournamentStore.knockoutStages
			.map((s) => ({ code: s.code, name: s.name, ties: tiesOf(s.code) }))
			.filter((r) => r.ties.length)
	);
	/** The round in play: the one holding the next match not yet finished. */
	let currentCode = $derived.by(() => {
		const now = serverClock.now();
		const ko = tipsStore.matches
			.filter((m) => tournamentStore.isKnockout(m.stage))
			.sort(byKick);
		const next = ko.find((m) => !played(m) && new Date(m.kickoff).getTime() >= now) ?? ko.find((m) => !played(m));
		return next?.stage ?? ko[ko.length - 1]?.stage ?? '';
	});
	let selected = $state('');
	let round = $derived(rounds.find((r) => r.code === (selected || currentCode)) ?? rounds[0]);
	let twoLeg = $derived(!!round && round.ties.some((t) => t.legs.length === 2));

	const tn = (id: string) => tipsStore.team(id);
	/** A leg's score oriented to the tie's (first leg's) home / away. */
	function oriented(t: Tie, m: Match): [number, number] | null {
		if (!played(m) && m.status !== 'live') return null;
		const [h, a] = played(m) ? legScore(m) : [m.ftHome, m.ftAway];
		return m.homeTeam === t.home ? [h, a] : [a, h];
	}
	function view(t: Tie) {
		const scores = t.legs.map((m) => oriented(t, m));
		const live = t.legs.some((m) => m.status === 'live');
		const done = t.legs.every(played);
		const last = t.legs[t.legs.length - 1];
		let agg: [number, number] | null = null;
		for (const s of scores) if (s) agg = agg ? [agg[0] + s[0], agg[1] + s[1]] : [s[0], s[1]];
		const advancer: '' | 'home' | 'away' =
			done && last.advancer ? (last.advancer === t.home ? 'home' : last.advancer === t.away ? 'away' : '') : '';
		// Where to go: the next unplayed leg, else the deciding one.
		const target = t.legs.find((m) => !played(m)) ?? last;
		const day = (m: Match) =>
			new Date(m.kickoff).toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' });
		const time = (m: Match) =>
			new Date(m.kickoff).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
		const name = (side: 'home' | 'away') =>
			side === 'home' ? (tn(t.home)?.name ?? t.homeLabel) : (tn(t.away)?.name ?? t.awayLabel);
		let foot = '';
		let footStrong = '';
		if (done) {
			const how =
				last.penHome || last.penAway
					? 'on penalties'
					: last.etHome || last.etAway
						? 'after extra time'
						: t.legs.length === 2
							? 'on aggregate'
							: '';
			if (advancer) {
				footStrong = name(advancer);
				foot = ` advance${how ? ' ' + how : ''}`;
			} else if (agg) foot = agg[0] === agg[1] ? 'Level' : '';
		} else if (live) {
			const lead = agg ? (agg[0] === agg[1] ? 'level' : `${name(agg[0] > agg[1] ? 'home' : 'away')} lead${t.legs.length === 2 ? ' on aggregate' : ''}`) : '';
			footStrong = 'Live';
			foot = lead ? ` · ${lead}` : '';
		} else if (t.legs.length === 2 && scores[0]) {
			const s = scores[0]!;
			const lead = s[0] === s[1] ? 'level' : `${name(s[0] > s[1] ? 'home' : 'away')} lead`;
			foot = `2nd leg `;
			footStrong = `${day(t.legs[1])} ${time(t.legs[1])}`;
			foot += `${footStrong} · ${lead}`;
			footStrong = '';
		} else if (t.legs.length === 2) {
			foot = `1st leg ${day(t.legs[0])} · 2nd leg ${day(t.legs[1])}`;
		} else {
			foot = `${day(last)} ${time(last)}`;
		}
		return { scores, live, done, agg, advancer, target, foot, footStrong };
	}
	function goNow() {
		selected = currentCode;
	}
</script>

{#if !tipsStore.loaded}
	<p class="muted">Loading…</p>
{:else if rounds.length === 0}
	<div class="card empty muted">The knockout rounds appear here once the draw is made.</div>
{:else if round}
	<div class="subbar pills">
		<div class="chips">
			{#each rounds as r (r.code)}
				<button class="chip" class:on={r.code === round.code} onclick={() => (selected = r.code)}>{r.name}</button>
			{/each}
			{#if round.code !== currentCode}
				<button class="chip now" onclick={goNow}><LocateFixed size={14} /> Now</button>
			{/if}
		</div>
	</div>
	<div class="card ties">
		<div class="head">
			<b>{round.name}</b>
			<span class="cols" aria-hidden="true">
				{#if twoLeg}<span>1st</span><span>2nd</span><span class="w">Agg</span>{:else}<span class="w">Score</span>{/if}
			</span>
		</div>
		{#each round.ties as t (t.id)}
			{@const v = view(t)}
			{@const H = tn(t.home)}
			{@const A = tn(t.away)}
			<a class="tie" class:twoleg={twoLeg} class:dim={!t.home || !t.away} href={`/m/${v.target.id}`}>
				<span class="teams">
					<span class="team" class:ph={!H}>
						{#if H}<Flag iso2={H.iso2} code={H.fifaCode} logo={H.logo} size={20} />{:else}<span class="phc">?</span>{/if}
						<span class="tn">{H?.name ?? t.homeLabel ?? '—'}</span>
					</span>
					<span class="team" class:ph={!A}>
						{#if A}<Flag iso2={A.iso2} code={A.fifaCode} logo={A.logo} size={20} />{:else}<span class="phc">?</span>{/if}
						<span class="tn">{A?.name ?? t.awayLabel ?? '—'}</span>
					</span>
				</span>
				{#if twoLeg}
					{#each [0, 1] as i (i)}
						{@const s = t.legs[i] ? v.scores[i] : null}
						{@const isLive = t.legs[i]?.status === 'live'}
						<span class="legs" class:live={isLive}>
							{#if s}<b>{s[0]}</b><b>{s[1]}</b>{:else}<span>–</span><span>–</span>{/if}
						</span>
					{/each}
				{/if}
				<LedBoard home={v.agg?.[0] ?? null} away={v.agg?.[1] ?? null} live={v.live} advancer={v.advancer} />
				<span class="foot">
					<span class="ftxt"
						>{#if v.footStrong}<b class:livec={v.footStrong === 'Live'}>{v.footStrong}</b>{/if}{v.foot}</span
					>
					<ChevronRight size={14} />
				</span>
			</a>
		{/each}
	</div>
{/if}

<style>
	.pills {
		top: calc(var(--topbar-h) + 2.6rem);
		z-index: 19;
	}
	.chips {
		display: flex;
		gap: 0.4rem;
		overflow-x: auto;
		scrollbar-width: none;
	}
	.chips::-webkit-scrollbar {
		display: none;
	}
	.chip {
		flex: none;
		height: 32px;
		padding: 0 0.8rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--muted);
		font: inherit;
		font-weight: 700;
		font-size: 0.8rem;
		white-space: nowrap;
		cursor: pointer;
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
	}
	.chip.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.chip.now {
		margin-left: auto;
		color: var(--accent);
	}
	.card.ties {
		padding: 0;
	}
	.card.empty {
		padding: 1.2rem;
		text-align: center;
	}
	.head {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.65rem 0.75rem 0.65rem 0.9rem;
		border-bottom: 1px solid var(--border);
		font-size: 0.85rem;
	}
	.cols {
		margin-left: auto;
		display: flex;
		font-size: 0.58rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.cols span {
		width: 38px;
		text-align: center;
	}
	.cols span.w {
		width: 52px;
	}
	.tie {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 44px;
		align-items: center;
		gap: 0 8px;
		padding: 8px 10px 8px 14px;
		border-bottom: 1px solid var(--border);
		color: var(--text);
	}
	.tie.twoleg {
		grid-template-columns: minmax(0, 1fr) 30px 30px 44px;
	}
	.tie:last-child {
		border-bottom: none;
	}
	.tie.dim {
		opacity: 0.75;
	}
	.teams {
		display: flex;
		flex-direction: column;
		gap: 7px;
		min-width: 0;
	}
	.team {
		display: flex;
		align-items: center;
		gap: 8px;
		font-weight: 600;
		font-size: 14px;
		line-height: 19px;
		min-width: 0;
	}
	.team.ph {
		color: var(--muted);
		font-weight: 500;
	}
	.phc {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: var(--surface-2);
		color: var(--muted);
		font-size: 11px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex: none;
	}
	.tn {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.legs {
		display: flex;
		flex-direction: column;
		gap: 7px;
		align-items: center;
		width: 30px;
		font-family: var(--font-mono);
		font-weight: 700;
		font-size: 13px;
		line-height: 19px;
		color: var(--muted);
	}
	.legs b {
		color: var(--text);
	}
	.legs.live b {
		color: var(--live);
	}
	.foot {
		grid-column: 1 / -1;
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 11.5px;
		color: var(--muted);
		padding: 7px 2px 1px;
		margin-top: 4px;
		border-top: 1px dashed var(--border);
	}
	.ftxt {
		flex: 1;
		white-space: pre-wrap;
	}
	.foot b {
		color: var(--text);
	}
	.foot b.livec {
		color: var(--live);
	}
</style>
