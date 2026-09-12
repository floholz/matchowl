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
	import { tick, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { shell } from '$lib/shell.svelte';

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
	// Filters live in the URL (?round=stage|label, ?team=id): empty = all.
	// Only the user sets them; coming back restores them with the page.
	let roundKey = $derived($page.url.searchParams.get('round') ?? '');
	let team = $derived($page.url.searchParams.get('team') ?? '');
	function setFilter(k: 'round' | 'team', v: string) {
		const u = new URL($page.url);
		u.searchParams.delete('round');
		u.searchParams.delete('team');
		if (v) u.searchParams.set(k, v);
		goto(`${u.pathname}${u.search}`, { replaceState: true, noScroll: true, keepFocus: true });
	}
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
		goDay(nowDayIndex);
	}
	/** Day strip: the competition's match days within the current filter
	 *  (a season has long gaps, so only days with fixtures are shown). */
	let strip = $derived(
		days.map(([, ms], i) => {
			const d = new Date(ms[0].kickoff);
			return {
				i,
				weekday: d.toLocaleDateString(undefined, { weekday: 'short' }),
				day: d.getDate(),
				month: d.toLocaleDateString(undefined, { month: 'short' }),
				isToday: d.toDateString() === new Date(serverClock.now()).toDateString(),
				isNow: i === nowDayIndex
			};
		})
	);
	let activeDay = $state(-1);
	let stripEl = $state<HTMLElement | null>(null);
	function goDay(i: number, behavior: ScrollBehavior = 'smooth') {
		if (i < 0) return;
		activeDay = i;
		document.getElementById(`day-${i}`)?.scrollIntoView({ behavior, block: 'start' });
	}
	// Follow the list: the topmost day section below the sticky chrome.
	$effect(() => {
		if (!tipsStore.loaded) return;
		const onScroll = () => {
			const sections = Array.from(document.querySelectorAll<HTMLElement>('h3.day'));
			const top = (document.querySelector('.pills')?.getBoundingClientRect().bottom ?? 160) + 8;
			let cur = -1;
			sections.forEach((sec, i) => {
				if (sec.getBoundingClientRect().top <= top) cur = i;
			});
			if (cur >= 0 && cur !== activeDay) activeDay = cur;
		};
		window.addEventListener('scroll', onScroll, { passive: true });
		onScroll();
		return () => window.removeEventListener('scroll', onScroll);
	});
	$effect(() => {
		const i = activeDay;
		stripEl
			?.querySelector<HTMLElement>(`[data-i="${i}"]`)
			?.scrollIntoView({ block: 'nearest', inline: 'center', behavior: 'smooth' });
	});
	// Land on the current day: on first load, and whenever the user changes
	// a filter. Coming back (history) restores the saved position instead.
	const memory = (globalThis as { __matchListPos?: Map<string, number> }).__matchListPos ??=
		new Map<string, number>();
	const memKey = () => `${tournamentStore.current?.id}|${roundKey}|${team}`;
	let lastFilter = '';
	$effect(() => {
		const key = `${roundKey}|${team}`;
		if (!tipsStore.loaded) return;
		const first = lastFilter === '';
		if (key === lastFilter) return;
		lastFilter = key;
		const remembered = memory.get(memKey());
		if (first && shell.navType === 'popstate' && remembered !== undefined) {
			tick().then(() => window.scrollTo({ top: remembered, behavior: 'instant' as ScrollBehavior }));
			return;
		}
		tick().then(() => goDay(nowDayIndex, first ? 'instant' : 'smooth'));
	});
	onDestroy(() => {
		if (typeof window !== 'undefined') memory.set(memKey(), window.scrollY);
	});

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
		if (roundKey || team) setFilter('round', '');
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
				onchange={(e) => setFilter('round', (e.currentTarget as HTMLSelectElement).value)}
				aria-label="Matchday"
			>
				<option value="">All matchdays</option>
				{#each rounds as r (r.key)}<option value={r.key}>{r.label}</option>{/each}
			</select>
			<span class="lbl">{team ? 'Matchday' : (rounds.find((r) => r.key === roundKey)?.label ?? 'All matchdays')}</span>
			<ChevronDown size={14} />
		</label>
		<label class="chip sel" class:on={!!team}>
			<select value={team} onchange={(e) => setFilter('team', (e.currentTarget as HTMLSelectElement).value)} aria-label="Team">
				<option value="">By team</option>
				{#each teams as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
			</select>
			<span class="lbl">{team ? (tipsStore.team(team)?.name ?? 'Team') : 'By team'}</span>
			{#if team}<X size={14} />{:else}<ChevronDown size={14} />{/if}
		</label>
		<span class="spacer"></span>
		{#if nowId && activeDay !== nowDayIndex}
			<button class="chip" onclick={goNow}><LocateFixed size={14} /> Now</button>
		{/if}
	</div>
	{#if strip.length > 1}
		<div class="days" bind:this={stripEl}>
			{#each strip as d (d.i)}
				<button class="dayb" class:on={d.i === activeDay} class:today={d.isToday} class:now={d.isNow} data-i={d.i} onclick={() => goDay(d.i)}>
					<span>{d.isToday ? 'Today' : d.weekday}</span>
					<b class="digits">{d.day}</b>
					<small>{d.month}</small>
				</button>
			{/each}
		</div>
	{/if}
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
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.days {
		display: flex;
		gap: 6px;
		overflow-x: auto;
		scrollbar-width: none;
		margin: 0 calc(-1 * var(--shell-x, 1rem));
		padding: 0 var(--shell-x, 1rem);
	}
	.days::-webkit-scrollbar {
		display: none;
	}
	.dayb {
		flex: none;
		width: 54px;
		height: 54px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1px;
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--muted);
		font: inherit;
		font-size: 0.68rem;
		font-weight: 600;
		cursor: pointer;
		padding: 0;
		line-height: 1.1;
	}
	.dayb b {
		font-size: 0.95rem;
		color: var(--text);
	}
	.dayb small {
		font-size: 0.6rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}
	.dayb.today,
	.dayb.now {
		color: var(--accent);
	}
	.dayb.on {
		background: var(--accent);
		color: var(--accent-fg);
	}
	.dayb.on b {
		color: var(--accent-fg);
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
		/* Land below the fixed top bar + the hub's sticky tab strip + pills + strip. */
		scroll-margin-top: calc(var(--topbar-h) + 10.4rem);
	}
</style>
