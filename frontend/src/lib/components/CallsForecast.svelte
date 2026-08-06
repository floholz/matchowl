<script lang="ts">
	import type { ForecastStore, ForecastCall } from '$lib/forecast.svelte';
	import Flag from './Flag.svelte';
	import { Check } from '@lucide/svelte';

	let { store, readOnly = false }: { store: ForecastStore; readOnly?: boolean } = $props();

	let calls = $derived(store.spec.calls ?? []);
	/** All teams of the tournament, name-sorted (calls pick from the whole
	 *  field, not per group). */
	let teams = $derived(
		Object.values(store.teams).sort((a, b) => a.name.localeCompare(b.name))
	);

	function picked(call: ForecastCall): number {
		const v = store.calls[call.key];
		return Array.isArray(v) ? v.length : v ? 1 : 0;
	}
</script>

{#each calls as call (call.key)}
	<section class="card call">
		<header class="call-h">
			<b>{call.name}</b>
			<span class="muted">
				{call.type === 'team'
					? 'one pick'
					: `pick ${call.count ?? '?'} — ${picked(call)} / ${call.count ?? '?'}`}
				· {call.points}&thinsp;pt{call.type === 'teamset' ? ' each' : ''}
			</span>
		</header>
		<div class="picks">
			{#each teams as t (t.id)}
				{@const on = store.inCall(call, t.id)}
				<button
					class="pick"
					class:on
					disabled={readOnly ||
						store.locked ||
						(!on && call.type === 'teamset' && picked(call) >= (call.count ?? 99))}
					onclick={() => store.toggleCall(call, t.id)}
				>
					<Flag iso2={t.iso2} code={t.fifaCode} />
					<span class="pn">{t.name}</span>
					{#if on}<Check size={14} />{/if}
				</button>
			{/each}
		</div>
	</section>
{/each}

<style>
	.call {
		margin-bottom: 0.85rem;
	}
	.call-h {
		display: flex;
		align-items: baseline;
		gap: 0.6rem;
		margin-bottom: 0.7rem;
	}
	.call-h .muted {
		font-size: 0.78rem;
	}
	.picks {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
		gap: 0.4rem;
	}
	.pick {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.45rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		font-size: 0.82rem;
		font-weight: 600;
		cursor: pointer;
		text-align: left;
		transition:
			border-color 0.12s ease,
			background 0.12s ease;
	}
	.pick .pn {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.pick.on {
		border-color: var(--accent);
		background: color-mix(in srgb, var(--accent) 14%, var(--surface-2));
		color: var(--text);
	}
	.pick.on :global(svg) {
		color: var(--accent);
	}
	.pick:disabled {
		opacity: 0.55;
		cursor: default;
	}
</style>
