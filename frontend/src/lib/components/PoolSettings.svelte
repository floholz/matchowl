<!-- The three things that define a pool: the one season it plays, the mode
     (classic points table or head-to-head matchday duels) and, for h2h,
     the save-call allowance. Used by the create sheet, the owner's
     settings and "Set up next season". Picking a season resets mode and
     save calls to the server's defaults for that season's shape. -->
<script lang="ts">
	import { api, type PoolMode } from '$lib/api';
	import { seasonLabel, type Tournament } from '$lib/tournament.svelte';
	import { Check } from '@lucide/svelte';

	let {
		seasons,
		tournament = $bindable(''),
		mode = $bindable<PoolMode>('classic'),
		saveCalls = $bindable(1),
		disabled = false
	}: {
		seasons: Tournament[];
		tournament: string;
		mode: PoolMode;
		saveCalls: number;
		disabled?: boolean;
	} = $props();

	let matchesPerRound = $state(0);

	async function pick(slug: string) {
		if (disabled || slug === tournament) return;
		tournament = slug;
		try {
			const d = await api.poolDefaults(slug);
			if (tournament !== slug) return; // picked another one meanwhile
			mode = d.mode;
			saveCalls = d.saveCalls;
			matchesPerRound = d.matchesPerRound;
		} catch {
			/* keep what we have; the server fills defaults on submit anyway */
		}
	}
</script>

<div class="field">
	<span>Season</span>
	<div class="chipset">
		{#each seasons as t (t.id)}
			<button type="button" class="schip" class:on={tournament === t.slug} {disabled} onclick={() => pick(t.slug)}>
				{#if tournament === t.slug}<Check size={13} />{/if}
				{t.competition?.shortName || t.competition?.name} {seasonLabel(t)}
			</button>
		{/each}
	</div>
</div>
<div class="field">
	<span>Mode</span>
	<div class="modes" role="radiogroup">
		<button type="button" class="mode" class:on={mode === 'classic'} role="radio" aria-checked={mode === 'classic'} {disabled} onclick={() => (mode = 'classic')}>
			<b>Classic</b>
			<small>One points table for the whole season.</small>
		</button>
		<button type="button" class="mode" class:on={mode === 'h2h'} role="radio" aria-checked={mode === 'h2h'} {disabled} onclick={() => (mode = 'h2h')}>
			<b>Head-to-head</b>
			<small>A duel against a pool mate every matchday, with save calls and bans.</small>
		</button>
	</div>
</div>
{#if mode === 'h2h'}
	<div class="field">
		<span>Save calls per matchday</span>
		<div class="seg" role="radiogroup">
			<button type="button" class:on={saveCalls === 1} role="radio" aria-checked={saveCalls === 1} {disabled} onclick={() => (saveCalls = 1)}>1</button>
			<button type="button" class:on={saveCalls === 2} role="radio" aria-checked={saveCalls === 2} {disabled} onclick={() => (saveCalls = 2)}>2</button>
		</div>
		<small class="muted">Matches that count double for you{matchesPerRound ? ` — a matchday has ${matchesPerRound} matches` : ''}.</small>
	</div>
{/if}

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}
	.field > span {
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.chipset {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
	}
	.schip {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		height: 32px;
		padding: 0 0.75rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface-2);
		color: var(--muted);
		font: inherit;
		font-weight: 700;
		font-size: 0.78rem;
	}
	.schip.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.modes {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.4rem;
	}
	.mode {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.2rem;
		padding: 0.6rem 0.7rem;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		text-align: left;
	}
	.mode small {
		font-size: 0.74rem;
		color: var(--muted);
		line-height: 1.3;
	}
	.mode.on {
		border-color: var(--accent);
		box-shadow: inset 0 0 0 1px var(--accent);
	}
	.mode.on b {
		color: var(--accent);
	}
	.seg {
		display: inline-flex;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		overflow: hidden;
		width: max-content;
	}
	.seg button {
		min-width: 3rem;
		padding: 0.35rem 0.9rem;
		border: 0;
		background: var(--surface-2);
		color: var(--muted);
		font: inherit;
		font-weight: 700;
	}
	.seg button.on {
		background: var(--accent);
		color: var(--accent-fg);
	}
	button:disabled {
		opacity: 0.6;
	}
	.muted {
		color: var(--muted);
		font-size: 0.78rem;
	}
	@media (max-width: 420px) {
		.modes {
			grid-template-columns: 1fr;
		}
	}
</style>
