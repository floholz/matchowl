<script lang="ts">
	import { pb } from '$lib/pb';
	import { onMount } from 'svelte';
	import { Check, Plus } from '@lucide/svelte';

	interface Comp {
		id: string;
		key: string;
		name: string;
		shortName: string;
		logo: string;
	}
	interface T {
		id: string;
		slug: string;
		name: string;
		shortName: string;
		status: string;
		startsAt: string;
		endsAt: string;
		competition?: Comp;
	}

	let list = $state<T[]>([]);
	let played = $state<Set<string>>(new Set());
	let loaded = $state(false);
	let busy = $state('');

	onMount(async () => {
		const [r, mine] = await Promise.all([
			pb.send('/api/tournaments', { method: 'GET' }),
			pb.send('/api/me/tournaments', { method: 'GET' }).catch(() => ({ tournaments: [] }))
		]);
		list = r.tournaments ?? [];
		played = new Set((mine.tournaments ?? []).map((t: T) => t.id));
		loaded = true;
	});

	/** Group by competition (standalone tournaments get their own group). */
	let groups = $derived.by(() => {
		const by = new Map<string, { name: string; logo: string; items: T[] }>();
		for (const t of list) {
			const key = t.competition?.key ?? `solo-${t.slug}`;
			if (!by.has(key))
				by.set(key, {
					name: t.competition?.name ?? t.name,
					logo: t.competition?.logo ?? '',
					items: []
				});
			by.get(key)!.items.push(t);
		}
		return [...by.values()];
	});

	async function toggle(t: T) {
		busy = t.id;
		try {
			if (played.has(t.id)) {
				await pb.send(`/api/tournaments/${t.slug}/play`, { method: 'DELETE' });
				played.delete(t.id);
			} else {
				await pb.send(`/api/tournaments/${t.slug}/play`, { method: 'POST' });
				played.add(t.id);
			}
			played = new Set(played);
		} finally {
			busy = '';
		}
	}

	function dates(t: T): string {
		if (!t.startsAt) return '';
		const f = (s: string) =>
			new Date(s).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });
		return t.endsAt ? `${f(t.startsAt)} – ${f(t.endsAt)}` : f(t.startsAt);
	}
</script>

<div class="cat stagger">
	<header class="chead">
		<p class="kicker">Pick your battles</p>
		<h1>Tournaments</h1>
	</header>

	{#if loaded && groups.length === 0}
		<div class="card empty muted">No tournaments yet — check back soon.</div>
	{/if}

	{#each groups as g (g.name)}
		<section class="comp">
			<h2 class="comp-h">{g.name}</h2>
			{#each g.items as t (t.id)}
				<div class="card trow" class:archived={t.status === 'archived'}>
					<a class="tmain" href={`/tournaments/${t.slug}`}>
						<b class="tname">{t.name}</b>
						<span class="muted tsub">
							{dates(t)}
							<span class="pill" class:live={t.status === 'active'}>{t.status}</span>
						</span>
					</a>
					{#if t.status !== 'archived' || played.has(t.id)}
						<button
							class="btn slim"
							class:secondary={played.has(t.id)}
							disabled={busy === t.id}
							onclick={() => toggle(t)}
						>
							{#if played.has(t.id)}<Check size={16} /> Playing{:else}<Plus size={16} /> Play{/if}
						</button>
					{/if}
				</div>
			{/each}
		</section>
	{/each}
</div>

<style>
	.chead {
		margin: 0.4rem 0 1.1rem;
	}
	.comp {
		margin-top: 1.4rem;
	}
	.comp-h {
		font-size: 1.05rem;
		color: var(--muted);
		margin-bottom: 0.6rem;
	}
	.trow {
		display: flex;
		align-items: center;
		gap: 0.9rem;
	}
	.trow.archived {
		opacity: 0.75;
	}
	.tmain {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		color: var(--text);
		flex: 1;
		min-width: 0;
	}
	.tsub {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.82rem;
	}
	.btn.slim {
		width: auto;
		padding: 0.5rem 1rem;
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		flex: none;
	}
	.empty {
		text-align: center;
		padding: 2rem;
	}
</style>
