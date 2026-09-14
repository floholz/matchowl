<!-- The match page (and the desktop detail panel — same component, so
     nothing is designed twice). Hero with crests and the kick-off or the
     LED board large, the tie strip for two-legged ties, your tip (big
     steppers before lock, capsule + line + points breakdown after),
     friends' picks (list after kick-off), bots, and the mini table for the
     group. Resolves the match's season into the shared stores itself. -->
<script lang="ts">
	import { pb } from '$lib/pb';
	import { tournamentStore } from '$lib/tournament.svelte';
	import {
		tipsStore,
		isLocked,
		teamsResolved,
		findOtherLeg,
		otherLegView,
		legScore,
		type Match,
		type Tip,
		type FriendTip,
		type PerfectScorers,
		type OtherLeg
	} from '$lib/tips.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { defaultScoring, scoreTip, type ScoringConfig } from '$lib/scoring';
	import { tieStrip } from '$lib/tie';
	import { groupTable } from '$lib/standings';
	import Flag from './Flag.svelte';
	import LedBoard from './LedBoard.svelte';
	import TipCapsule from './TipCapsule.svelte';
	import Stepper from './Stepper.svelte';
	import H2HPickButtons from './H2HPickButtons.svelte';
	import Avatar from './Avatar.svelte';
	import { Check, Lock, Bot, Target, ChevronRight, ChevronDown, X } from '@lucide/svelte';
	import { untrack } from 'svelte';

	let {
		id,
		onClose = undefined,
		onSaved = undefined
	}: {
		id: string;
		/** Panel mode: shows a header with a close button. */
		onClose?: () => void;
		/** A tip was saved — lists elsewhere can mirror it. */
		onSaved?: (t: Omit<Tip, 'id' | 'match'>) => void;
	} = $props();

	// ---- resolve the match: its season → the shared stores ----
	let missing = $state(false);
	let seasonId = $state('');
	$effect(() => {
		const mid = id;
		missing = false;
		seasonId = '';
		if (!mid) return;
		(async () => {
			await tournamentStore.ready();
			const rec = await pb.collection('matches').getOne(mid).catch(() => null);
			const t = rec && tournamentStore.list.find((x) => x.id === rec.tournament);
			if (!t) {
				missing = true;
				return;
			}
			tournamentStore.select(t.slug);
			seasonId = t.id;
			await tipsStore.load().catch(() => {});
		})();
	});
	let season = $derived(tournamentStore.current);
	let loaded = $derived(!!seasonId && tipsStore.loaded && season?.id === seasonId);
	let match = $derived(loaded ? tipsStore.matches.find((m) => m.id === id) : undefined);

	let cfg = $state<ScoringConfig | null>(null);
	$effect(() => {
		defaultScoring().then((c) => (cfg = c));
	});

	// ---- state (mirrors MatchRow) ----
	const COUNTDOWN_MS = 3 * 3600_000;
	let now = $state(serverClock.now());
	let kickoffMs = $derived(match ? new Date(match.kickoff).getTime() : 0);
	let played = $derived(!!match && (match.status === 'finished' || !!match.finalizedAt));
	let live = $derived(match?.status === 'live');
	let locked = $derived(!!match && (isLocked(match) || now >= kickoffMs || live || played));
	let resolved = $derived(!!match && teamsResolved(match));
	let isKO = $derived(!!match && tournamentStore.isKnockout(match.stage));
	let existing = $derived(match ? tipsStore.tips[match.id] : undefined);
	let pts = $derived(match ? tipsStore.scores[match.id] : undefined);
	let legView = $derived.by((): OtherLeg | null => {
		if (!match || !isKO) return null;
		const found = findOtherLeg(tipsStore.matches, match);
		if (!found) return null;
		return otherLegView(match, found.other, found.first, `/m/${found.other.id}`);
	});
	let phased = $derived(isKO && !(legView?.first ?? false));
	let editable = $derived(!!match && !locked && (!isKO || resolved));

	$effect(() => {
		if (!match || locked) return;
		const ms = kickoffMs - untrack(() => now);
		if (ms > COUNTDOWN_MS) {
			const t = setTimeout(() => (now = serverClock.now()), ms - COUNTDOWN_MS + 50);
			return () => clearTimeout(t);
		}
		const i = setInterval(() => (now = serverClock.now()), 1000);
		return () => clearInterval(i);
	});

	const teamOf = (tid: string) => tipsStore.team(tid);
	const nameOf = (tid: string) => teamOf(tid)?.name ?? '—';
	function label(side: 'home' | 'away') {
		const m = match;
		const t = m ? teamOf(side === 'home' ? m.homeTeam : m.awayTeam) : undefined;
		if (t) return { name: t.name, iso2: t.iso2, code: t.fifaCode, logo: t.logo, ph: false };
		const raw = m ? (side === 'home' ? m.homeLabel : m.awayLabel) : '';
		return { name: raw || '—', iso2: '', code: '', logo: '', ph: true };
	}
	let H = $derived(label('home'));
	let A = $derived(label('away'));
	const sideOf = (tid: string): '' | 'home' | 'away' =>
		match && tid ? (tid === match.homeTeam ? 'home' : tid === match.awayTeam ? 'away' : '') : '';

	let kickoffTime = $derived(
		match ? new Date(match.kickoff).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) : ''
	);
	let kickoffDay = $derived(
		match
			? new Date(match.kickoff).toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' })
			: ''
	);
	let untilText = $derived.by(() => {
		const s = Math.max(0, Math.floor((kickoffMs - now) / 1000));
		const d = Math.floor(s / 86400);
		const h = Math.floor((s % 86400) / 3600);
		const m = Math.floor((s % 3600) / 60);
		if (d >= 1) return `${d}d ${h}h`;
		if (h >= 1) return `${h}h ${String(m).padStart(2, '0')}m`;
		return `${String(m).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`;
	});
	let ending = $derived(
		!match ? '' : match.penHome || match.penAway ? 'PEN' : match.etHome || match.etAway ? 'AET' : 'FT'
	);
	let wentET = $derived(!!match && (match.etHome !== 0 || match.etAway !== 0));
	let wentPens = $derived(!!match && (match.penHome !== 0 || match.penAway !== 0));
	let board = $derived.by((): [number, number] | null => {
		if (!match) return null;
		if (played) return legScore(match);
		if (live) return [match.ftHome, match.ftAway];
		return null;
	});
	let boardAdv = $derived(match && played && phased ? sideOf(match.advancer) : '');
	let strip = $derived(
		match && legView
			? tieStrip({ match, leg: legView, played, live, homeName: H.name, awayName: A.name, teamName: nameOf })
			: null
	);

	// ---- your tip ----
	let tipScore = $derived(existing ? legScore(existing) : null);
	let tipAdv = $derived(existing && phased ? sideOf(existing.advancer) : '');
	let capState = $derived.by(() => {
		if (isKO && !resolved) return 'unavailable' as const;
		if (!locked) return 'open' as const;
		if (!existing) return 'locked' as const;
		if (played) return (pts ?? breakdown?.total ?? 0) > 0 ? ('hit' as const) : ('miss' as const);
		return 'frozen' as const;
	});
	let breakdown = $derived(
		match && existing && played && cfg ? scoreTip(cfg, phased, match, existing) : null
	);
	/** "2:1, Bayern win" / "1:1, draw" / "1:0, Bayern through on penalties". */
	let tipLine = $derived.by(() => {
		if (!match || !existing || !tipScore) return '';
		const [h, a] = tipScore;
		const score = `${h}:${a}`;
		if (phased) {
			const adv = existing.advancer ? nameOf(existing.advancer) : '';
			if (!adv) return score;
			const how = existing.ftHome === existing.ftAway && existing.etHome === existing.etAway ? ' on penalties' : '';
			return `${score}, ${adv} through${how}`;
		}
		return `${score}, ${h === a ? 'draw' : `${h > a ? H.name : A.name} win`}`;
	});

	// editor
	let ftH = $state(0);
	let ftA = $state(0);
	let etH = $state(0);
	let etA = $state(0);
	let pen = $state('');
	let busy = $state(false);
	let msg = $state('');
	let dirty = $state(false);
	let savedOk = $state(false);
	$effect(() => {
		const t = existing;
		ftH = t?.ftHome ?? 0;
		ftA = t?.ftAway ?? 0;
		etH = t?.etHome ?? 0;
		etA = t?.etAway ?? 0;
		pen = t?.penWinner ?? '';
		dirty = false;
	});
	$effect(() => {
		if (etH < ftH) etH = ftH;
		if (etA < ftA) etA = ftA;
	});
	let ftTie = $derived(phased && ftH === ftA);
	let etTie = $derived(ftTie && etH === etA);
	let advancerId = $derived(
		!phased || !match
			? ''
			: ftH !== ftA
				? ftH > ftA
					? match.homeTeam
					: match.awayTeam
				: etH !== etA
					? etH > etA
						? match.homeTeam
						: match.awayTeam
					: pen
	);
	let incomplete = $derived(etTie && !pen);
	let outcomeHint = $derived.by(() => {
		if (!match) return '';
		if (phased) return advancerId ? `${nameOf(advancerId)} through` : 'pick who goes through';
		return ftH === ftA ? 'Draw' : `${ftH > ftA ? H.name : A.name} win`;
	});
	const mark = () => {
		dirty = true;
		savedOk = false;
	};
	async function save() {
		if (!match) return;
		msg = '';
		busy = true;
		try {
			const t = { ftHome: ftH, ftAway: ftA, etHome: etH, etAway: etA, penWinner: pen, advancer: '' };
			await tipsStore.save({ id: existing?.id, match: match.id, ...t });
			dirty = false;
			savedOk = true;
			onSaved?.(t);
		} catch (e: unknown) {
			msg = (e as { message?: string })?.message ?? 'Could not save this tip.';
		} finally {
			busy = false;
		}
	}

	// ---- friends' picks + bots (after kick-off) ----
	let friends = $state<FriendTip[] | null>(null);
	let bots = $state<FriendTip[]>([]);
	let botsOpen = $state(false);
	let perfect = $state<PerfectScorers | null>(null);
	let friendsFor = '';
	$effect(() => {
		const m = match;
		if (!m || !locked) return;
		if (friendsFor === m.id) return;
		friendsFor = m.id;
		friends = null;
		tipsStore
			.friends(m.id)
			.then((r) => {
				friends = r.tips;
				bots = r.bots;
				perfect = r.perfect;
			})
			.catch(() => {
				friends = [];
				bots = [];
				perfect = null;
			});
	});
	let maxPts = $derived(cfg ? cfg.match.tendency + cfg.match.exact + cfg.match.totalGoals + cfg.match.goalDiff : 0);
	function friendCap(f: FriendTip): 'hit' | 'miss' | 'frozen' {
		if (f.points === undefined) return 'frozen';
		return f.points > 0 ? 'hit' : 'miss';
	}

	// ---- mini table: the two teams in their group / league table ----
	let table = $derived.by(() => {
		const m = match;
		if (!m || isKO || !m.groupLetter || !resolved) return null;
		const ms = tipsStore.matches.filter((x) => x.stage === m.stage && x.groupLetter === m.groupLetter);
		const rows = groupTable(ms, {});
		const pick = rows
			.map((r, i) => ({ ...r, pos: i + 1 }))
			.filter((r) => r.id === m.homeTeam || r.id === m.awayTeam);
		return { label: tournamentStore.groupLabel(m.groupLetter), rows: pick };
	});
	let tableHref = $derived(
		season ? `/competitions/${season.competition.key}?s=${season.slug}&tab=standings` : ''
	);
	let context = $derived(
		match
			? [tournamentStore.stageName(match.stage), match.roundLabel !== tournamentStore.stageName(match.stage) ? match.roundLabel : '']
					.filter(Boolean)
					.join(' · ')
			: ''
	);
</script>

<div class="md">
	{#if onClose}
		<header class="head">
			<span class="htxt">
				<b>{season?.competition?.name ?? 'Match'}</b>
				{#if context}<span class="muted">{context}</span>{/if}
			</span>
			<button class="close" onclick={onClose} aria-label="Close"><X size={18} /></button>
		</header>
	{/if}

	{#if missing}
		<div class="card"><p class="muted">No such match.</p></div>
	{:else if !match}
		<p class="muted">Loading…</p>
	{:else}
		<div class="hero">
			<div class="side" class:ph={H.ph}>
				<Flag iso2={H.iso2} code={H.code} logo={H.logo} size={56} />
				<b>{H.name}</b>
			</div>
			<div class="mid">
				{#if played || live}
					<LedBoard
						size="hero"
						home={board?.[0] ?? null}
						away={board?.[1] ?? null}
						{live}
						advancer={boardAdv}
						leg={legView ? (legView.first ? '1st' : '2nd') : ''}
					/>
					<div class="pills">
						{#if live}
							<span class="pill live">Live</span>
						{:else}
							{#if wentET}<span class="pill">AET</span>{/if}
							{#if wentPens}<span class="pill">PEN</span>{/if}
							{#if !wentET && !wentPens}<span class="pill">FT</span>{/if}
						{/if}
					</div>
					{#if wentET}
						<span class="muted line"
							>after 90’ <b class="digits">{match.ftHome}:{match.ftAway}</b>{#if wentPens}
								· after 120’ <b class="digits">{match.etHome}:{match.etAway}</b>{/if}</span
						>
					{/if}
					{#if wentPens && match.advancer}
						<span class="muted line"
							><b>{nameOf(match.advancer)}</b> win <b class="digits">{match.penHome}–{match.penAway}</b> on penalties</span
						>
					{/if}
				{:else}
					<span class="time digits">{kickoffTime}</span>
					<span class="muted line">{kickoffDay}</span>
					{#if resolved && !locked}
						<span class="pill ok">locks in {untilText}</span>
					{:else if locked}
						<span class="pill"><Lock size={11} /> locked</span>
					{/if}
				{/if}
			</div>
			<div class="side" class:ph={A.ph}>
				<Flag iso2={A.iso2} code={A.code} logo={A.logo} size={56} />
				<b>{A.name}</b>
			</div>
		</div>

		{#if strip && legView}
			<a class="card strip" href={legView.href}>
				<span
					>{#each strip as p, i (i)}{#if p.b}<b>{p.t}</b>{:else if p.n}<span class="digits n">{p.t}</span
							>{:else}{p.t}{/if}{/each}</span
				>
				<ChevronRight size={14} />
			</a>
		{/if}

		<!-- ---- your tip ---- -->
		{#if isKO && !resolved}
			<div class="card quiet muted">Tipping opens once the pairing is decided.</div>
		{:else if editable}
			<div class="card tip">
				<div class="tiphead">
					<span class="kicker">Your tip</span>
					<span class="spacer"></span>
					<span class="muted small">
						{#if savedOk}<Check size={13} /> saved{:else if dirty}unsaved{:else if existing}saved{:else}not placed{/if}
					</span>
				</div>
				<div class="enter">
					<Stepper bind:value={ftH} onchange={mark} size="lg" />
					<span class="sep">:</span>
					<Stepper bind:value={ftA} onchange={mark} size="lg" />
				</div>
				{#if ftTie}
					<div class="phase">After extra time</div>
					<div class="enter">
						<Stepper bind:value={etH} min={ftH} onchange={mark} size="lg" />
						<span class="sep">:</span>
						<Stepper bind:value={etA} min={ftA} onchange={mark} size="lg" />
					</div>
				{/if}
				{#if etTie}
					<div class="phase">Penalties — who goes through?</div>
					<div class="pens">
						<button type="button" class="pen" class:sel={pen === match.homeTeam} onclick={() => { pen = match!.homeTeam; mark(); }}>{H.name}</button>
						<button type="button" class="pen" class:sel={pen === match.awayTeam} onclick={() => { pen = match!.awayTeam; mark(); }}>{A.name}</button>
					</div>
				{/if}
				<p class="hint muted">
					<b>{outcomeHint}</b>
					{#if cfg}
						· tendency {cfg.match.tendency} pts · exact +{cfg.match.exact}
					{/if}
				</p>
				{#if msg}<p class="error">{msg}</p>{/if}
				<button class="btn" onclick={save} disabled={busy || incomplete || (!dirty && !!existing)}>
					{#if busy}Saving…{:else if !dirty && existing}<Check size={16} /> Saved{:else}Save tip{/if}
				</button>
				<H2HPickButtons matchId={match.id} size="lg" />
			</div>
		{:else if existing}
			<div class="card scored">
				<div class="tiphead">
					<span class="kicker">Your tip</span>
					<span class="spacer"></span>
					{#if played}
						<span class="ypts digits" class:ok={(pts ?? breakdown?.total ?? 0) > 0}
							>{(pts ?? breakdown?.total ?? 0) > 0 ? '+' : ''}{pts ?? breakdown?.total ?? 0}</span
						>
					{/if}
				</div>
				<div class="tiprow">
					<TipCapsule home={tipScore?.[0] ?? null} away={tipScore?.[1] ?? null} state={capState} advancer={tipAdv} />
					<div class="tiptxt">
						<span class="tipline">{tipLine}</span>
						{#if breakdown}
							<!-- Only the components the rules pay for (a 0-point rule is off). -->
							<span class="muted small"
								>result <b class:ok={breakdown.tendency > 0}>{breakdown.tendency > 0 ? '+' : ''}{breakdown.tendency}</b>
								{#if !cfg || cfg.match.goalDiff}· goal difference <b class:ok={breakdown.goalDiff > 0}>{breakdown.goalDiff > 0 ? '+' : ''}{breakdown.goalDiff}</b>{/if}
								{#if !cfg || cfg.match.totalGoals}· total goals <b class:ok={breakdown.totalGoals > 0}>{breakdown.totalGoals > 0 ? '+' : ''}{breakdown.totalGoals}</b>{/if}
								{#if !cfg || cfg.match.exact}· exact <b class:ok={breakdown.exact > 0}>{breakdown.exact > 0 ? '+' : ''}{breakdown.exact}</b>{/if}</span
							>
						{:else if played}
							<span class="muted small">points pending</span>
						{:else}
							<span class="muted small">locked · points once it’s final</span>
						{/if}
					</div>
				</div>
			</div>
		{:else}
			<div class="card quiet muted">No tip — this match was locked.</div>
		{/if}

		<!-- ---- friends' picks ---- -->
		<div class="sec">
			<h2>Friends’ picks</h2>
			{#if !locked}<span class="muted small">shown at kick-off</span>{/if}
		</div>
		{#if !locked}
			<div class="card quiet muted"><Lock size={14} /> Your league mates’ picks show once the match kicks off.</div>
		{:else if friends === null}
			<p class="muted small">Loading…</p>
		{:else if friends.length === 0}
			<div class="card quiet muted">No friends’ tips for this match.</div>
		{:else}
			<div class="card list">
				{#each friends as f (f.userId)}
					<div class="frow">
						<Avatar name={f.name} size={28} />
						<span class="fname">{f.name}</span>
						{#if f.points !== undefined && maxPts > 0 && f.points === maxPts}
							<span class="pill ok"><Target size={10} /> exact</span>
						{/if}
						<span class="spacer"></span>
						<span class="cap">
							<TipCapsule
								home={legScore(f)[0]}
								away={legScore(f)[1]}
								state={friendCap(f)}
								advancer={phased ? sideOf(f.advancer) : ''}
							/>
						</span>
						{#if f.points !== undefined}
							<span class="fpts digits" class:ok={f.points > 0}>{f.points > 0 ? '+' : ''}{f.points}</span>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
		{#if perfect}
			<p class="muted small perfect">
				<Target size={13} />
				{#if perfect.count > 0}
					Perfect tip ({perfect.points} pts): {perfect.names.join(', ')}{#if perfect.count > perfect.names.length}
						&nbsp;+{perfect.count - perfect.names.length} more{/if}
				{:else}
					No perfect tips for this match.
				{/if}
			</p>
		{/if}

		{#if locked && bots.length > 0}
			<button class="botstoggle" onclick={() => (botsOpen = !botsOpen)} aria-expanded={botsOpen}>
				<span class="pill"><Bot size={11} /> Bots</span>
				<span class="muted small">{bots.length} tip{bots.length === 1 ? '' : 's'}</span>
				<ChevronDown size={14} class="cv {botsOpen ? 'up' : ''}" />
			</button>
			{#if botsOpen}
				<div class="card list">
					{#each bots as f (f.userId)}
						<div class="frow">
							<Avatar name={f.name} size={28} />
							<span class="fname">{f.name}</span>
							<span class="spacer"></span>
							<span class="cap">
								<TipCapsule
									home={legScore(f)[0]}
									away={legScore(f)[1]}
									state={friendCap(f)}
									advancer={phased ? sideOf(f.advancer) : ''}
								/>
							</span>
							{#if f.points !== undefined}
								<span class="fpts digits" class:ok={f.points > 0}>{f.points > 0 ? '+' : ''}{f.points}</span>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		{/if}

		<!-- ---- mini table ---- -->
		{#if table && table.rows.length}
			<div class="sec">
				<h2>{table.label}</h2>
				{#if tableHref}<a class="more" href={tableHref}>Table <ChevronRight size={14} /></a>{/if}
			</div>
			<div class="card list">
				{#each table.rows as r (r.id)}
					{@const t = teamOf(r.id)}
					<div class="trow">
						<span class="pos digits muted">{r.pos}</span>
						<span class="tname"><Flag iso2={t?.iso2 ?? ''} code={t?.fifaCode ?? ''} logo={t?.logo ?? ''} size={18} /> {t?.name ?? '?'}</span>
						<span class="num digits muted">{r.pld}</span>
						<span class="num digits muted">{r.gf - r.ga > 0 ? '+' : ''}{r.gf - r.ga}</span>
						<span class="num digits">{r.pts}</span>
					</div>
				{/each}
			</div>
		{/if}
	{/if}
</div>

<style>
	.md {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.head {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0 0 0.6rem;
		border-bottom: 1px solid var(--border);
	}
	.htxt {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
		font-size: 0.92rem;
	}
	.htxt .muted {
		font-size: 0.78rem;
	}
	.close {
		margin-left: auto;
		width: 34px;
		height: 34px;
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--muted);
		cursor: pointer;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}
	.close:hover {
		background: var(--surface-2);
		color: var(--text);
	}

	/* ---- hero ---- */
	.hero {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 0.5rem;
		padding: 0.9rem 0.25rem 0.6rem;
	}
	.side {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		text-align: center;
		font-size: 0.9rem;
		min-width: 0;
	}
	.side b {
		max-width: 100%;
		overflow-wrap: anywhere;
	}
	.side.ph {
		color: var(--muted);
	}
	.mid {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.35rem;
		min-width: 96px;
	}
	.time {
		font-size: 1.7rem;
	}
	.line {
		font-size: 0.8rem;
		font-weight: 600;
		text-align: center;
	}
	.line b {
		color: var(--text);
	}
	.pills {
		display: flex;
		gap: 0.3rem;
	}

	/* ---- strip ---- */
	.strip {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.65rem 0.9rem;
		font-size: 0.85rem;
		color: var(--muted);
	}
	.strip span:first-child {
		flex: 1;
		white-space: pre-wrap;
	}
	.strip b,
	.strip .n {
		color: var(--text);
		font-weight: 700;
	}

	/* ---- tip ---- */
	.card.tip {
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
		box-shadow: var(--glow);
		display: flex;
		flex-direction: column;
		gap: 0.7rem;
	}
	.tiphead {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.small {
		font-size: 0.78rem;
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
	}
	.enter {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 1rem;
	}
	.sep {
		font-weight: 800;
		font-size: 1.4rem;
		opacity: 0.5;
	}
	.phase {
		text-align: center;
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.pens {
		display: flex;
		gap: 0.5rem;
		justify-content: center;
	}
	.pen {
		flex: 1;
		max-width: 12rem;
		padding: 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		font-weight: 700;
		cursor: pointer;
	}
	.pen.sel {
		border-color: var(--accent);
		background: color-mix(in srgb, var(--accent) 14%, transparent);
	}
	.hint {
		margin: 0;
		text-align: center;
		font-size: 0.82rem;
	}
	.hint b {
		color: var(--text);
	}
	.card.scored {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}
	.tiprow {
		display: flex;
		align-items: center;
		gap: 0.9rem;
	}
	.tiptxt {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		min-width: 0;
	}
	.tipline {
		font-weight: 700;
	}
	.tiptxt .small {
		display: inline;
		line-height: 1.5;
	}
	.tiptxt b {
		color: var(--text);
	}
	.tiptxt b.ok,
	.ypts.ok,
	.fpts.ok {
		color: var(--accent);
	}
	.ypts {
		font-size: 1.1rem;
		color: var(--muted);
	}
	.card.quiet {
		padding: 0.85rem 1rem;
		font-size: 0.9rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	/* ---- sections / lists ---- */
	.sec {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		margin: 0.6rem 0.15rem 0;
	}
	.sec h2 {
		font-size: 1.1rem;
	}
	.sec .more {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
		font-size: 0.82rem;
		font-weight: 600;
	}
	.card.list {
		padding: 0;
	}
	.frow,
	.trow {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.5rem 0.9rem;
		border-bottom: 1px solid var(--border);
		font-size: 0.9rem;
	}
	.frow:last-child,
	.trow:last-child {
		border-bottom: none;
	}
	.fname {
		font-weight: 600;
	}
	.cap {
		display: inline-flex;
		transform: scale(0.85);
		transform-origin: right center;
	}
	.fpts {
		width: 2.2rem;
		text-align: right;
		color: var(--muted);
	}
	.perfect {
		margin: 0.2rem 0.3rem 0;
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}
	.botstoggle {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		width: 100%;
		padding: 0.5rem 0.3rem;
		border: none;
		background: transparent;
		color: var(--text);
		font: inherit;
		cursor: pointer;
	}
	.botstoggle :global(.cv) {
		margin-left: auto;
		color: var(--muted);
		transition: transform 0.15s ease;
	}
	.botstoggle :global(.cv.up) {
		transform: rotate(180deg);
	}
	.trow .pos {
		width: 1.4rem;
	}
	.tname {
		flex: 1;
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		font-weight: 600;
		min-width: 0;
	}
	.trow .num {
		width: 2.2rem;
		text-align: right;
	}
</style>
