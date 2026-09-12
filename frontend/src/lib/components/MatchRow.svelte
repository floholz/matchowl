<!-- The match row — the core unit of every list. Stacked home over away
     with crest + name; a status column on the left (kick-off time, or
     FT / AET / PEN, or LIVE); the LED score board and the orange tip
     capsule on the right, always in the same columns. Tapping the capsule
     opens an inline stepper drawer (saves on close); the chevron leads to
     the match page. Two-legged ties get a lug on the board and a strip
     under the row (next leg date, or aggregate · other leg · who advances).

     Like TipCard: reads/writes through tipsStore by default (the
     per-season pages); the feed passes team/tip/knockout/points/leg +
     onSave explicitly because its matches span tournaments. -->
<script lang="ts">
	import {
		tipsStore,
		isLocked,
		teamsResolved,
		findOtherLeg,
		otherLegView,
		legScore,
		type Match,
		type Team,
		type Tip,
		type OtherLeg
	} from '$lib/tips.svelte';
	import { tournamentStore } from '$lib/tournament.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { tieStrip } from '$lib/tie';
	import Flag from './Flag.svelte';
	import LedBoard from './LedBoard.svelte';
	import TipCapsule from './TipCapsule.svelte';
	import Stepper from './Stepper.svelte';
	import { ChevronRight, Check } from '@lucide/svelte';
	import { untrack } from 'svelte';

	let {
		match,
		open = false,
		onToggle,
		team = undefined,
		tip = undefined,
		knockout = undefined,
		points = undefined,
		onSave = undefined,
		leg = undefined,
		href = '',
		sub = '',
		onSelect = undefined,
		selected = false
	}: {
		match: Match;
		/** The editor drawer is open (the parent keeps one open at a time). */
		open?: boolean;
		onToggle?: () => void;
		team?: (id: string) => Team | undefined;
		tip?: Tip | null;
		knockout?: boolean;
		points?: number;
		onSave?: (t: Omit<Tip, 'id' | 'match'>) => Promise<void>;
		/** Two-legged tie context; `undefined` (store mode) computes it from
		 *  the season's full match list, the feed passes it explicitly. */
		leg?: OtherLeg | null;
		/** Match page link (the chevron + the teams). */
		href?: string;
		/** Second line of the status column (e.g. the competition code on
		 *  Home). Defaults to the countdown inside the last hours. */
		sub?: string;
		/** Desktop lists: open the match in the detail panel instead of
		 *  following `href`. */
		onSelect?: () => void;
		/** The row whose match the panel shows. */
		selected?: boolean;
	} = $props();

	const teamOf = (id: string) => (team ? team(id) : tipsStore.team(id));

	// Row-local clock: ticks per second only inside the countdown window,
	// otherwise a single timeout wakes it at the window's edge.
	const COUNTDOWN_MS = 3 * 3600_000;
	let now = $state(serverClock.now());
	let kickoffMs = $derived(new Date(match.kickoff).getTime());
	let played = $derived(match.status === 'finished' || !!match.finalizedAt);
	let live = $derived(match.status === 'live');
	// A match that is underway or done is locked whatever its kickoff says.
	let locked = $derived(isLocked(match) || now >= kickoffMs || live || played);
	let untilKickoff = $derived(kickoffMs - now);
	let countdown = $derived(!locked && untilKickoff <= COUNTDOWN_MS);
	let resolved = $derived(teamsResolved(match));
	let home = $derived(teamOf(match.homeTeam));
	let away = $derived(teamOf(match.awayTeam));
	let existing = $derived(tip !== undefined ? tip : tipsStore.tips[match.id]);
	let isKO = $derived(knockout ?? tournamentStore.isKnockout(match.stage));
	let pts = $derived(points ?? tipsStore.scores[match.id]);
	let editable = $derived(!locked && !live && !played && (!isKO || resolved));

	$effect(() => {
		if (locked || live || played) return;
		const ms = kickoffMs - untrack(() => now);
		if (ms > COUNTDOWN_MS) {
			const t = setTimeout(() => (now = serverClock.now()), ms - COUNTDOWN_MS + 50);
			return () => clearTimeout(t);
		}
		const id = setInterval(() => (now = serverClock.now()), 1000);
		return () => clearInterval(id);
	});

	// ---- two-legged ties ----
	let legView = $derived.by((): OtherLeg | null => {
		if (leg !== undefined) return leg;
		if (!isKO) return null;
		const found = findOtherLeg(tipsStore.matches, match);
		if (!found) return null;
		return otherLegView(match, found.other, found.first, `/m/${found.other.id}`);
	});
	// First legs are tipped like group matches (draws allowed, no advancer).
	let phased = $derived(isKO && !(legView?.first ?? false));
	// ---- labels ----
	function label(side: 'home' | 'away') {
		const t = side === 'home' ? home : away;
		if (t) return { name: t.name, iso2: t.iso2, code: t.fifaCode, logo: t.logo, ph: false };
		const raw = side === 'home' ? match.homeLabel : match.awayLabel;
		return { name: raw || '—', iso2: '', code: '', logo: '', ph: true };
	}
	let H = $derived(label('home'));
	let A = $derived(label('away'));
	let sideOf = (id: string): '' | 'home' | 'away' =>
		id && id === match.homeTeam ? 'home' : id && id === match.awayTeam ? 'away' : '';

	const kickoffTime = $derived(
		new Date(match.kickoff).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
	);
	const countdownText = $derived.by(() => {
		const s = Math.max(0, Math.floor(untilKickoff / 1000));
		const h = Math.floor(s / 3600);
		const m = Math.floor((s % 3600) / 60);
		if (h >= 1) return `in ${h}h ${String(m).padStart(2, '0')}m`;
		return `in ${String(m).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`;
	});
	/** How the match ended, for the status column. */
	let ending = $derived(
		match.penHome || match.penAway ? 'PEN' : match.etHome || match.etAway ? 'AET' : 'FT'
	);
	let statusSub = $derived(sub || (played || live ? '' : countdown ? countdownText : locked ? 'locked' : ''));

	// ---- board ----
	let board = $derived.by((): [number, number] | null => {
		if (played) return legScore(match);
		if (live) return [match.ftHome, match.ftAway];
		return null;
	});
	let boardAdv = $derived(played && phased ? sideOf(match.advancer) : '');

	// ---- capsule ----
	let tipScore = $derived(existing ? legScore(existing) : null);
	let tipAdv = $derived(existing && phased ? sideOf(existing.advancer) : '');
	let capState = $derived.by(() => {
		if (isKO && !resolved) return 'unavailable' as const;
		if (!locked) return 'open' as const;
		if (!existing) return 'locked' as const;
		if (played) return (pts ?? 0) > 0 ? ('hit' as const) : ('miss' as const);
		return 'frozen' as const;
	});

	// ---- editor drawer ----
	let ftH = $state(0);
	let ftA = $state(0);
	let etH = $state(0);
	let etA = $state(0);
	let pen = $state('');
	let busy = $state(false);
	let msg = $state('');
	let dirty = $state(false);

	// Seed the editor from the saved tip whenever it changes.
	$effect(() => {
		const t = existing;
		ftH = t?.ftHome ?? 0;
		ftA = t?.ftAway ?? 0;
		etH = t?.etHome ?? 0;
		etA = t?.etAway ?? 0;
		pen = t?.penWinner ?? '';
		dirty = false;
	});
	// Keep ET >= FT (cumulative) as the user edits FT.
	$effect(() => {
		if (etH < ftH) etH = ftH;
		if (etA < ftA) etA = ftA;
	});
	let ftTie = $derived(phased && ftH === ftA);
	let etTie = $derived(ftTie && etH === etA);
	let advancerId = $derived(
		!phased
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
	let advancerName = $derived(advancerId ? (teamOf(advancerId)?.name ?? '—') : '');
	/** A knockout tip that predicts a draw needs the penalty pick to be complete. */
	let incomplete = $derived(etTie && !pen);

	async function save(): Promise<boolean> {
		msg = '';
		busy = true;
		try {
			const t = { ftHome: ftH, ftAway: ftA, etHome: etH, etAway: etA, penWinner: pen, advancer: '' };
			if (onSave) await onSave(t);
			else await tipsStore.save({ id: existing?.id, match: match.id, ...t });
			dirty = false;
			return true;
		} catch (e: unknown) {
			msg = (e as { message?: string })?.message ?? 'Could not save this tip.';
			return false;
		} finally {
			busy = false;
		}
	}
	/** Untipped: the button always saves what is shown (0:0 included).
	 *  Tipped: only a change needs a request; otherwise it just closes. */
	let needsSave = $derived(dirty || !existing);
	async function done() {
		if (needsSave && !incomplete) {
			if (!(await save())) return;
		}
		onToggle?.();
	}
	// Saves on close: the parent flips `open` off (another row opened, or
	// the capsule was tapped again) — flush an unsaved, complete tip.
	let wasOpen = false;
	$effect(() => {
		const o = open;
		if (wasOpen && !o) untrack(() => {
			if (dirty && !incomplete && !busy) save();
		});
		wasOpen = o;
	});
	const mark = () => (dirty = true);

	// ---- tie strip ----
	let strip = $derived(
		legView
			? tieStrip({
					match,
					leg: legView,
					played,
					live,
					homeName: H.name,
					awayName: A.name,
					teamName: (id) => teamOf(id)?.name ?? '—'
				})
			: null
	);
	function pick(e: MouseEvent) {
		if (!onSelect) return;
		e.preventDefault();
		onSelect();
	}
</script>

<div class="mr" class:live class:played class:editing={open} class:selected id={`m-${match.id}`}>
	<!-- An <a> when there is a link (the click handler only intercepts it
	     for the desktop panel), a plain wrapper otherwise. -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<svelte:element this={href ? 'a' : 'div'} class="main" href={href || undefined} onclick={href ? pick : undefined}>
		<span class="when" class:islive={live}>
			{#if live}
				<b>Live</b><span class="ldot"></span>
			{:else if played}
				<b>{ending}</b>{#if statusSub}<span>{statusSub}</span>{/if}
			{:else}
				<b>{kickoffTime}</b>{#if statusSub}<span>{statusSub}</span>{/if}
			{/if}
		</span>
		<span class="teams">
			<span class="team" class:ph={H.ph}>
				<Flag iso2={H.iso2} code={H.code} logo={H.logo} size={20} />
				<span class="tn">{H.name}</span>
			</span>
			<span class="team" class:ph={A.ph}>
				<Flag iso2={A.iso2} code={A.code} logo={A.logo} size={20} />
				<span class="tn">{A.name}</span>
			</span>
		</span>
	</svelte:element>
	<span class="right">
		<LedBoard
			home={board?.[0] ?? null}
			away={board?.[1] ?? null}
			{live}
			advancer={boardAdv}
			leg={legView ? (legView.first ? '1st' : '2nd') : ''}
		/>
		<TipCapsule
			home={tipScore?.[0] ?? null}
			away={tipScore?.[1] ?? null}
			state={capState}
			advancer={tipAdv}
			active={open}
			onclick={editable ? () => onToggle?.() : undefined}
		/>
		{#if played}
			<span class="pts digits" class:ok={(pts ?? 0) > 0}
				>{existing && pts !== undefined ? `${pts > 0 ? '+' : ''}${pts}` : existing ? '' : '0'}</span
			>
		{:else if href}
			<a class="go" {href} aria-label="Match details" onclick={pick}><ChevronRight size={16} /></a>
		{:else}
			<span class="pts"></span>
		{/if}
	</span>

	{#if open && editable}
		<div class="drawer">
			<div class="enter">
				<Stepper bind:value={ftH} onchange={mark} />
				<span class="sep">:</span>
				<Stepper bind:value={ftA} onchange={mark} />
			</div>
			{#if ftTie}
				<div class="phase">After extra time</div>
				<div class="enter">
					<Stepper bind:value={etH} min={ftH} onchange={mark} />
					<span class="sep">:</span>
					<Stepper bind:value={etA} min={ftA} onchange={mark} />
				</div>
			{/if}
			{#if etTie}
				<div class="phase">Penalties — who goes through?</div>
				<div class="pens">
					<button
						type="button"
						class="pen"
						class:sel={pen === match.homeTeam}
						onclick={() => {
							pen = match.homeTeam;
							mark();
						}}>{H.name}</button
					>
					<button
						type="button"
						class="pen"
						class:sel={pen === match.awayTeam}
						onclick={() => {
							pen = match.awayTeam;
							mark();
						}}>{A.name}</button
					>
				</div>
			{/if}
			<div class="foot">
				{#if msg}
					<span class="error">{msg}</span>
				{:else if isKO && advancerName}
					<span class="muted">Advances: <b>{advancerName}</b></span>
				{:else}
					<span class="muted">{dirty ? 'Unsaved' : existing ? 'Saved' : 'Not placed'}</span>
				{/if}
				<span class="spacer"></span>
				<button type="button" class="btn slim" onclick={done} disabled={busy || incomplete}>
					<Check size={15} />
					{busy ? 'Saving…' : needsSave ? 'Save' : 'Done'}
				</button>
			</div>
		</div>
	{/if}

	{#if strip && legView}
		<a class="strip" href={legView.href}>
			{#each strip as p, i (i)}{#if p.b}<b>{p.t}</b>{:else if p.n}<span class="digits n">{p.t}</span
					>{:else}{p.t}{/if}{/each}
			<ChevronRight size={14} class="cv" />
		</a>
	{/if}
</div>

<style>
	.mr {
		display: grid;
		grid-template-columns: 46px 1fr auto;
		align-items: center;
		gap: 6px;
		min-height: 60px;
		padding: 7px 10px 7px 12px;
		border-bottom: 1px solid var(--border);
		transition: background 0.15s ease;
	}
	.mr:last-child {
		border-bottom: none;
	}
	.mr.editing {
		background: color-mix(in srgb, var(--accent) 5%, transparent);
	}
	.mr.selected {
		background: color-mix(in srgb, var(--accent) 6%, transparent);
		box-shadow: inset 3px 0 0 var(--accent);
	}
	.main {
		display: contents;
		color: inherit;
	}
	.when {
		display: flex;
		flex-direction: column;
		gap: 1px;
		font-size: 12px;
		line-height: 1.15;
		color: var(--muted);
		min-width: 0;
	}
	.when b {
		color: var(--text);
		font-weight: 700;
		white-space: nowrap;
	}
	.when.islive,
	.when.islive b {
		color: var(--live);
	}
	.ldot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--live);
		box-shadow: 0 0 8px var(--live);
		animation: pulse 1.4s ease-in-out infinite;
	}
	@keyframes pulse {
		50% {
			opacity: 0.35;
		}
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
	.tn {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	a.main:hover .tn {
		color: var(--accent);
	}
	.right {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.pts {
		width: 34px;
		text-align: center;
		font-size: 12px;
		color: var(--muted);
	}
	.pts.ok {
		color: var(--accent);
	}
	.go {
		width: 34px;
		height: 34px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		color: var(--muted);
		border-radius: var(--radius-sm);
	}
	.go:hover {
		color: var(--accent);
		background: var(--surface-2);
	}

	/* ---- editor drawer ---- */
	.drawer {
		grid-column: 1 / -1;
		display: flex;
		flex-direction: column;
		gap: 0.55rem;
		padding: 10px 0 4px;
		margin-top: 4px;
		border-top: 1px dashed var(--border);
	}
	.enter {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.8rem;
	}
	.sep {
		font-weight: 800;
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
		padding: 0.55rem 0.6rem;
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
	.foot {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		font-size: 0.8rem;
	}
	.foot b {
		color: var(--text);
	}
	.btn.slim {
		width: auto;
		padding: 0.5rem 0.9rem;
		font-size: 0.78rem;
		gap: 0.3rem;
	}

	/* ---- tie strip ---- */
	.strip {
		grid-column: 1 / -1;
		display: flex;
		align-items: center;
		gap: 4px;
		flex-wrap: wrap;
		padding: 7px 2px 1px;
		margin-top: 3px;
		border-top: 1px dashed var(--border);
		font-size: 11.5px;
		color: var(--muted);
		white-space: pre-wrap;
	}
	.strip b,
	.strip .n {
		color: var(--text);
		font-weight: 700;
	}
	.strip :global(.cv) {
		margin-left: auto;
		color: var(--muted);
	}
</style>
