<!-- The full fixture list of the selected season (tournamentStore.current,
     loaded into tipsStore): rows grouped by day, one card per day. A
     sticky pills row picks the matchday (round) or a team; "Now" jumps to
     the next kick-off when showing everything. Lives in the competition
     hub's Matches tab. -->
<script lang="ts">
	import { tipsStore, type Match } from '$lib/tips.svelte';
	import { tournamentStore, competitionLogoUrl } from '$lib/tournament.svelte';
	import MatchGroup from './MatchGroup.svelte';
	import MatchRow from './MatchRow.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { LocateFixed, ChevronDown, X } from '@lucide/svelte';
	import { tick } from 'svelte';

	let { focusId = '' }: { focusId?: string } = $props();

	// Accordion: only one match's tip inputs are open at a time.
	let openId = $state('');

	const byKickoff = (a: Match, b: Match) =>
		new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime() || a.num - b.num;
	const played = (m: Match) => m.status === 'finished' || !!m.finalizedAt;

	/** Rounds in play order (first kick-off), as "stage · round" keys. */
	let rounds = $derived.by(() => {
		const seen = new Map<string, { key: string; label: string; first: number }>();
		for (const m of [...tipsStore.matches].sort(byKickoff)) {
			const key = `${m.stage}|${m.roundLabel}`;
			if (seen.has(key)) continue;
			const stage = tournamentStore.stageName(m.stage);
			const label = m.roundLabel && m.roundLabel !== stage ? m.roundLabel : stage;
			seen.set(key, { key, label, first: new Date(m.kickoff).getTime() });
		}
		return [...seen.values()];
	});
	/** The round in play: holds the next match not yet kicked off. */
	let currentRound = $derived.by(() => {
		const now = serverClock.now();
		const ms = [...tipsStore.matches].sort(byKickoff);
		const next = ms.find((m) => new Date(m.kickoff).getTime() >= now && !played(m)) ?? ms[ms.length - 1];
		return next ? `${next.stage}|${next.roundLabel}` : '';
	});
	/** '' = all rounds. Starts on the current round once loaded. */
	let round = $state<string | null>(null);
	let roundKey = $derived(round ?? currentRound);
	let team = $state('');
	let teams = $derived(Object.values(tipsStore.teams).sort((a, b) => a.name.localeCompare(b.name)));

	let filtered = $derived(
		tipsStore.matches.filter((m) => {
			if (team) return m.homeTeam === team || m.awayTeam === team;
			if (roundKey) return `${m.stage}|${m.roundLabel}` === roundKey;
			return true;
		})
	);

	// "Now" = the next match not yet kicked off within the current filter.
	let nowId = $derived.by(() => {
		const now = serverClock.now();
		const next = filtered.find((m) => new Date(m.kickoff).getTime() >= now);
		return (next ?? filtered[filtered.length - 1])?.id ?? '';
	});

	let days = $derived(
		Object.entries(
			[...filtered].sort(byKickoff).reduce<Record<string, Match[]>>((acc, m) => {
				const d = new Date(m.kickoff).toLocaleDateString(undefined, {
					weekday: 'long',
					day: 'numeric',
					month: 'long'
				});
				(acc[d] ||= []).push(m);
				return acc;
			}, {})
		)
	);
	let nowDayIndex = $derived(days.findIndex(([, ms]) => ms.some((m) => m.id === nowId)));
	function goNow() {
		document.getElementById(`day-${nowDayIndex}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	}

	/** Card sub-line for a day: the stage, plus the round when shared. */
	function roundOf(ms: Match[]): string {
		const stages = new Set(ms.map((m) => m.stage));
		const stage = stages.size === 1 ? tournamentStore.stageName(ms[0].stage) : '';
		const rs = new Set(ms.map((m) => m.roundLabel));
		const r = rs.size === 1 ? ms[0].roundLabel : '';
		return [stage, r !== stage ? r : ''].filter(Boolean).join(' · ');
	}
	let logo = $derived(competitionLogoUrl(tournamentStore.current?.competition));
	let compName = $derived(
		tournamentStore.current?.competition?.shortName || tournamentStore.current?.competition?.name || ''
	);

	// Deep link to one match (?m= — e.g. "view the other leg"): show all
	// rounds, open its row and scroll to it.
	let lastFocus = '';
	$effect(() => {
		const id = focusId;
		if (!tipsStore.loaded || !id || id === lastFocus) return;
		if (!tipsStore.matches.some((m) => m.id === id)) return;
		lastFocus = id;
		round = '';
		team = '';
		openId = id;
		tick().then(() =>
			document.getElementById(`m-${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
		);
	});
</script>

<div class="subbar pills">
	<div class="chips">
		<label class="chip sel" class:on={!team && !!roundKey}>
			<select
				value={team ? '' : roundKey}
				onchange={(e) => {
					team = '';
					round = (e.currentTarget as HTMLSelectElement).value;
				}}
				aria-label="Matchday"
			>
				<option value="">All matchdays</option>
				{#each rounds as r (r.key)}<option value={r.key}>{r.label}</option>{/each}
			</select>
			<span class="lbl">{team ? 'Matchday' : (rounds.find((r) => r.key === roundKey)?.label ?? 'All matchdays')}</span>
			<ChevronDown size={14} />
		</label>
		<label class="chip sel" class:on={!!team}>
			<select value={team} onchange={(e) => (team = (e.currentTarget as HTMLSelectElement).value)} aria-label="Team">
				<option value="">By team</option>
				{#each teams as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
			</select>
			<span class="lbl">{team ? (tipsStore.team(team)?.name ?? 'Team') : 'By team'}</span>
			{#if team}<X size={14} />{:else}<ChevronDown size={14} />{/if}
		</label>
		<span class="spacer"></span>
		{#if !roundKey && !team && nowId}
			<button class="chip" onclick={goNow}><LocateFixed size={14} /> Now</button>
		{/if}
	</div>
</div>

{#if !tipsStore.loaded}
	<p class="muted">Loading fixtures…</p>
{:else if filtered.length === 0}
	<p class="muted">Nothing here.</p>
{:else}
	{#each days as [day, ms], i (day)}
		<h3 class="day" id={`day-${i}`}>{day}</h3>
		<MatchGroup name={compName} round={roundOf(ms)} {logo}>
			{#each ms as m (m.id)}
				<MatchRow
					match={m}
					open={openId === m.id}
					onToggle={() => (openId = openId === m.id ? '' : m.id)}
					href={`/m/${m.id}`}
				/>
			{/each}
		</MatchGroup>
	{/each}
{/if}

<style>
	.pills {
		top: calc(var(--topbar-h) + 2.6rem);
		z-index: 19;
	}
	.chips {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.chip {
		position: relative;
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		height: 32px;
		padding: 0 0.7rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--muted);
		font: inherit;
		font-weight: 700;
		font-size: 0.8rem;
		white-space: nowrap;
		cursor: pointer;
		max-width: 48vw;
	}
	.chip .lbl {
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.chip.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	/* The native select sits invisibly over the chip: real dropdown, our look. */
	.chip.sel select {
		position: absolute;
		inset: 0;
		width: 100%;
		opacity: 0;
		cursor: pointer;
		color-scheme: dark;
	}
	.chip.sel option {
		background: var(--surface);
		color: var(--text);
	}
	.day {
		margin: 1rem 0.15rem 0.55rem;
		font-family: var(--font);
		font-size: 0.76rem;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--muted);
		/* Land below the fixed top bar + the hub's sticky tab strip + pills. */
		scroll-margin-top: calc(var(--topbar-h) + 6.2rem);
	}
</style>
