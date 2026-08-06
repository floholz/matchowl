<script lang="ts">
	import { pb } from '$lib/pb';
	import { page } from '$app/stores';
	import { Check, Plus, Volleyball, Telescope, Network } from '@lucide/svelte';

	interface T {
		id: string;
		slug: string;
		name: string;
		shortName: string;
		status: string;
		startsAt: string;
		endsAt: string;
		competition?: { key: string; name: string };
	}

	let t = $state<T | null>(null);
	let playing = $state(false);
	let busy = $state(false);
	let missing = $state(false);

	$effect(() => {
		const slug = $page.params.slug;
		if (slug) load(slug);
	});

	async function load(slug: string) {
		missing = false;
		t = null;
		try {
			const [r, mine] = await Promise.all([
				pb.send(`/api/tournaments/${slug}`, { method: 'GET' }),
				pb.send('/api/me/tournaments', { method: 'GET' }).catch(() => ({ tournaments: [] }))
			]);
			t = r;
			playing = (mine.tournaments ?? []).some((x: T) => x.id === r.id);
		} catch {
			missing = true;
		}
	}

	async function toggle() {
		if (!t) return;
		busy = true;
		try {
			if (playing) {
				await pb.send(`/api/tournaments/${t.slug}/play`, { method: 'DELETE' });
				playing = false;
			} else {
				await pb.send(`/api/tournaments/${t.slug}/play`, { method: 'POST' });
				playing = true;
			}
		} finally {
			busy = false;
		}
	}

	function dates(x: T): string {
		if (!x.startsAt) return '';
		const f = (s: string) =>
			new Date(s).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });
		return x.endsAt ? `${f(x.startsAt)} – ${f(x.endsAt)}` : f(x.startsAt);
	}
</script>

{#if missing}
	<div class="card muted" style="text-align:center; padding:2rem;">
		This tournament doesn't exist. <a href="/tournaments">Back to tournaments</a>
	</div>
{:else if t}
	<div class="detail stagger">
		<header class="dhead">
			{#if t.competition}<p class="kicker">{t.competition.name}</p>{/if}
			<h1>{t.name}</h1>
			<p class="muted sub">
				{dates(t)}
				<span class="pill" class:live={t.status === 'active'}>{t.status}</span>
			</p>
		</header>

		{#if t.status !== 'archived' || playing}
			<button class="btn play" class:secondary={playing} disabled={busy} onclick={toggle}>
				{#if playing}<Check size={18} /> Playing — matches in your feed{:else}<Plus
						size={18}
					/> Play this tournament{/if}
			</button>
		{/if}

		<nav class="views">
			<a class="card view" href={`/tips?t=${t.slug}`}>
				<Volleyball size={22} />
				<b>Matches & tips</b>
				<span class="muted">The full schedule, tip every match</span>
			</a>
			<a class="card view" href={`/forecast?t=${t.slug}`}>
				<Telescope size={22} />
				<b>Forecast</b>
				<span class="muted">Your one-shot call before kickoff</span>
			</a>
			<a class="card view" href={`/tournament?t=${t.slug}`}>
				<Network size={22} />
				<b>Tables & bracket</b>
				<span class="muted">How it's actually going</span>
			</a>
		</nav>
	</div>
{/if}

<style>
	.dhead {
		margin: 0.4rem 0 1.1rem;
	}
	.sub {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-top: 0.3rem;
	}
	.btn.play {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		margin-bottom: 1.2rem;
	}
	.views {
		display: grid;
		gap: 0.85rem;
	}
	@media (min-width: 700px) {
		.views {
			grid-template-columns: repeat(3, 1fr);
		}
	}
	.view {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
		color: var(--text);
	}
	.view :global(svg) {
		color: var(--accent);
		margin-bottom: 0.2rem;
	}
	.view .muted {
		font-size: 0.82rem;
	}
	.view:hover {
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
	}
</style>
