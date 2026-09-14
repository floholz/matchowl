<!-- Save call and ban for one match, inside the tip drawer: one line per
     head-to-head pool of the caller that plays this match's season (pool
     name shown only when there are several). Loads on mount, renders
     nothing when the caller has no such pool. -->
<script lang="ts">
	import { api, type H2HMatchPool } from '$lib/api';
	import { Star, Ban } from '@lucide/svelte';

	let { matchId, size = 'sm' }: { matchId: string; size?: 'sm' | 'lg' } = $props();

	let pools = $state<H2HMatchPool[] | null>(null);
	let busy = $state('');
	let err = $state('');

	$effect(() => {
		const id = matchId;
		pools = null;
		err = '';
		api
			.h2hMatch(id)
			.then((r) => {
				if (id === matchId) pools = r.pools;
			})
			.catch(() => (pools = []));
	});

	async function toggle(p: H2HMatchPool, kind: 'save' | 'ban') {
		if (busy) return;
		busy = p.poolId + kind;
		err = '';
		try {
			await api.h2hSetPick(p.poolId, matchId, kind, kind === 'save' ? !p.saved : !p.banned);
			const r = await api.h2hMatch(matchId);
			pools = r.pools;
		} catch (e: unknown) {
			err = (e as { response?: { error?: string } })?.response?.error || 'Could not place that.';
		} finally {
			busy = '';
		}
	}
</script>

{#if pools && pools.length}
	<div class="h2h" class:lg={size === 'lg'}>
		{#each pools as p (p.poolId)}
			<div class="line">
				<span class="lbl">
					{#if pools.length > 1}<b>{p.name}</b> · {/if}
					{#if p.closed}
						{p.round} closed
					{:else if !p.paired}
						not paired on {p.round}
					{:else if p.ghost}
						vs the Ghost · no ban
					{:else if p.rival}
						vs {p.rival.name}
					{/if}
					{#if p.locked}
						{#if p.rivalSaved} · <span class="rv"><Star size={11} /> {p.rival?.name} saved this</span>{/if}
						{#if p.rivalBanned} · <span class="rv ban"><Ban size={11} /> banned for you</span>{/if}
					{/if}
				</span>
				<span class="btns">
					<button
						type="button"
						class="pk"
						class:on={p.saved}
						disabled={p.locked || p.closed || !!busy || (!p.saved && p.saveCallsLeft === 0)}
						title={p.saved ? 'Take the save call back' : `Save call: counts double for you (${p.saveCallsLeft} of ${p.saveCalls} left)`}
						aria-label={p.saved ? 'Take the save call back' : 'Save call'}
						onclick={() => toggle(p, 'save')}
					>
						<Star size={14} />
						<span>{p.saved ? 'Saved' : `${p.saveCallsLeft}/${p.saveCalls}`}</span>
					</button>
					{#if p.paired && !p.ghost}
						<button
							type="button"
							class="pk ban"
							class:on={p.banned}
							disabled={p.locked || p.closed || !!busy}
							title={p.banned ? 'Lift the ban' : p.banElsewhere ? 'Move your ban here' : `Ban: does not count for ${p.rival?.name ?? 'your rival'}`}
							aria-label={p.banned ? 'Lift the ban' : 'Ban for your rival'}
							onclick={() => toggle(p, 'ban')}
						>
							<Ban size={14} />
							<span>{p.banned ? 'Banned' : 'Ban'}</span>
						</button>
					{/if}
				</span>
			</div>
		{/each}
		{#if err}<span class="error">{err}</span>{/if}
	</div>
{/if}

<style>
	.h2h {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		padding: 0.5rem 0 0.1rem;
		border-top: 1px dashed var(--border);
	}
	.line {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.6rem;
		flex-wrap: wrap;
	}
	.lbl {
		font-size: 0.74rem;
		color: var(--muted);
		min-width: 0;
	}
	.lbl b {
		color: var(--text);
	}
	.rv {
		display: inline-flex;
		align-items: center;
		gap: 0.15rem;
		color: var(--accent);
		font-weight: 700;
	}
	.rv.ban {
		color: var(--live, #e0443e);
	}
	.btns {
		display: inline-flex;
		gap: 0.35rem;
	}
	.pk {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		height: 30px;
		padding: 0 0.65rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface-2);
		color: var(--muted);
		font: inherit;
		font-size: 0.74rem;
		font-weight: 700;
	}
	.lg .pk {
		height: 36px;
		font-size: 0.82rem;
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
	.error {
		font-size: 0.74rem;
	}
</style>
