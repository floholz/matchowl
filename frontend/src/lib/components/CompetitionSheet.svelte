<!-- The "More" sheet of the Matches filter row: every competition with a
     running or upcoming season, pinned ones first, the rest by country
     (international competitions last). Tapping a row filters Matches to it.
     The Pin toggle on top switches to pin mode, where each row pins or
     unpins instead; pinned competitions become chips in the row. The ones
     your pools play are always pinned. -->
<script lang="ts">
	import { pins, type CompetitionEntry } from '$lib/pins.svelte';
	import { competitionLogoUrl } from '$lib/tournament.svelte';
	import { Pin, Check, ChevronRight } from '@lucide/svelte';

	let {
		current = '',
		onPick,
		onClose
	}: {
		/** Key of the competition the row is filtered to ('' = none). */
		current?: string;
		onPick: (key: string) => void;
		onClose: () => void;
	} = $props();

	let editing = $state(false);

	interface Group {
		title: string;
		items: CompetitionEntry[];
	}
	const INTERNATIONAL = 'International';
	function place(e: CompetitionEntry): string {
		const c = e.competition;
		if (c.teamKind === 'national' || !c.country || c.country === 'World' || c.country === 'Europe')
			return INTERNATIONAL;
		return c.country;
	}
	let groups = $derived.by((): Group[] => {
		const entries = [...pins.entries.values()].sort((a, b) =>
			a.competition.name.localeCompare(b.competition.name)
		);
		const pinned: CompetitionEntry[] = [];
		for (const p of pins.pinned) {
			const e = pins.entries.get(p.competition.key);
			if (e) pinned.push(e);
		}
		const rest = new Map<string, CompetitionEntry[]>();
		for (const e of entries) {
			// In pin mode everything stays where it is, so a row doesn't jump
			// away from under your finger.
			if (!editing && pins.isPinned(e.competition.key)) continue;
			const g = place(e);
			if (!rest.has(g)) rest.set(g, []);
			rest.get(g)!.push(e);
		}
		const titles = [...rest.keys()].sort((a, b) =>
			a === INTERNATIONAL ? 1 : b === INTERNATIONAL ? -1 : a.localeCompare(b)
		);
		const out: Group[] = [];
		if (pinned.length && !editing) out.push({ title: 'Pinned', items: pinned });
		for (const t of titles) out.push({ title: t, items: rest.get(t)! });
		return out;
	});

	function tap(e: CompetitionEntry) {
		if (editing) pins.toggle(e.competition.key);
		else onPick(e.competition.key);
	}
	function onKey(e: KeyboardEvent) {
		if (e.key === 'Escape') onClose();
	}
</script>

<svelte:window onkeydown={onKey} />

<button class="scrim" onclick={onClose} aria-label="Close"></button>
<div class="sheet" role="dialog" aria-modal="true" aria-label="Competitions">
	<div class="grab"></div>
	<div class="shead">
		<h2>Competitions</h2>
		<span class="spacer"></span>
		<button class="pintoggle" class:on={editing} onclick={() => (editing = !editing)} aria-pressed={editing}>
			{#if editing}Done{:else}<Pin size={14} /> Pin{/if}
		</button>
	</div>
	<p class="muted lead">
		{#if editing}
			Pinned competitions show up in the filter row. Your pools' competitions are always pinned.
		{:else}
			Pick one to see only its matches.
		{/if}
	</p>
	{#each groups as g (g.title)}
		<h3 class="gh">{g.title}</h3>
		<div class="group">
			{#each g.items as e (e.competition.key)}
				{@const key = e.competition.key}
				{@const pool = pins.poolKeys.includes(key)}
				<button class="row" class:on={!editing && current === key} onclick={() => tap(e)} disabled={editing && pool}>
					<span class="logo">
						{#if competitionLogoUrl(e.competition)}<img src={competitionLogoUrl(e.competition)} alt="" loading="lazy" />{/if}
					</span>
					<span class="name">{e.competition.name}</span>
					{#if editing}
						{#if pool}
							<span class="pill">Pool</span>
						{:else if pins.isPinned(key)}
							<span class="pinpill on">Pinned</span>
						{:else}
							<span class="pinpill">Pin</span>
						{/if}
					{:else}
						<span class="muted sub">{pool ? 'Your pool' : place(e) === INTERNATIONAL ? '' : e.competition.country}</span>
						{#if current === key}<Check size={16} class="ck" />{/if}
					{/if}
				</button>
			{/each}
		</div>
	{/each}
	{#if !editing}
		<a class="all" href="/competitions" onclick={onClose}>All competitions and seasons <ChevronRight size={14} /></a>
	{/if}
</div>

<style>
	.scrim {
		position: fixed;
		inset: 0;
		z-index: 60;
		border: none;
		padding: 0;
		background: rgba(0, 0, 0, 0.55);
		cursor: default;
	}
	.sheet {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 61;
		max-width: 560px;
		max-height: calc(100dvh - var(--topbar-h) - 2rem);
		overflow-y: auto;
		margin: 0 auto;
		padding: 0.6rem 1rem calc(1.2rem + env(safe-area-inset-bottom));
		background: var(--surface);
		border: 1px solid var(--border);
		border-bottom: none;
		border-radius: var(--radius) var(--radius) 0 0;
		box-shadow: 0 -18px 44px -12px rgba(0, 0, 0, 0.8);
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}
	.grab {
		width: 36px;
		height: 4px;
		border-radius: 2px;
		background: var(--border);
		margin: 0 auto 0.2rem;
	}
	.shead {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.shead h2 {
		font-size: 1.25rem;
	}
	.spacer {
		flex: 1;
	}
	.lead {
		font-size: 0.85rem;
		margin: 0 0 0.3rem;
	}
	.pintoggle {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		height: 32px;
		padding: 0 0.85rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		font: inherit;
		font-weight: 700;
		font-size: 0.82rem;
		cursor: pointer;
	}
	.pintoggle.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.gh {
		margin: 0.55rem 0.15rem 0;
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.group {
		display: flex;
		flex-direction: column;
		background: var(--surface-2);
		border-radius: var(--radius-sm);
		overflow: hidden;
	}
	.row {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		min-height: 46px;
		padding: 0.45rem 0.85rem;
		border: none;
		border-top: 1px solid var(--border);
		background: none;
		color: var(--text);
		font: inherit;
		text-align: left;
		cursor: pointer;
	}
	.row:first-child {
		border-top: none;
	}
	.row:hover:not(:disabled) {
		background: color-mix(in srgb, var(--accent) 6%, transparent);
	}
	.row:disabled {
		cursor: default;
	}
	.row.on .name {
		color: var(--accent);
	}
	.logo {
		width: 22px;
		height: 22px;
		flex: none;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}
	.logo img {
		max-width: 100%;
		max-height: 100%;
	}
	.name {
		flex: 1;
		min-width: 0;
		font-weight: 600;
		font-size: 0.92rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.sub {
		font-size: 0.78rem;
	}
	.row :global(.ck) {
		color: var(--accent);
		flex: none;
	}
	.pill,
	.pinpill {
		flex: none;
		font-size: 0.72rem;
		font-weight: 700;
		padding: 0.2rem 0.6rem;
		border-radius: var(--radius-pill);
	}
	.pill {
		background: var(--surface);
		color: var(--muted);
	}
	.pinpill {
		border: 1px solid var(--border);
		color: var(--muted);
	}
	.pinpill.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.all {
		display: inline-flex;
		align-items: center;
		gap: 0.15rem;
		align-self: center;
		margin-top: 0.6rem;
		font-size: 0.85rem;
		font-weight: 600;
	}
	@media (min-width: 900px) {
		.sheet {
			inset: 50% auto auto 50%;
			transform: translate(-50%, -50%);
			width: min(520px, calc(100vw - 2rem));
			max-height: calc(100dvh - 4rem);
			padding: 1.25rem 1.4rem 1.4rem;
			border-bottom: 1px solid var(--border);
			border-radius: var(--radius);
			box-shadow: var(--shadow-pop);
		}
		.grab {
			display: none;
		}
	}
</style>
