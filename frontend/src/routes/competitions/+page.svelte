<script lang="ts">
	import { pb } from '$lib/pb';
	import { auth } from '$lib/auth.svelte';
	import {
		tournamentStore,
		defaultSeason,
		competitionLogoUrl,
		type Tournament,
		type Competition
	} from '$lib/tournament.svelte';
	import { pageChrome } from '$lib/shell.svelte';
	import { Check, ChevronRight, Plus } from '@lucide/svelte';

	pageChrome(() => ({ title: 'Competitions' }));

	interface Suggestion {
		id: string;
		slug: string;
		name: string;
		leagueMates: number;
	}
	let suggestions = $state<Suggestion[]>([]);

	let ready = $state(false);
	let playing = $state<Set<string>>(new Set());

	$effect(() => {
		tournamentStore.ready().then(() => (ready = true));
		if (auth.isAuthed) {
			pb.send('/api/me/tournaments', { method: 'GET' })
				.then((r) => (playing = new Set((r.tournaments ?? []).map((t: Tournament) => t.id))))
				.catch(() => {});
			pb.send('/api/tournaments/suggestions', { method: 'GET' })
				.then((r) => (suggestions = (r.suggestions ?? []).filter((x: Suggestion) => x.leagueMates > 0)))
				.catch(() => {});
		}
	});

	async function play(s: Suggestion) {
		await pb.send(`/api/tournaments/${s.slug}/play`, { method: 'POST' }).catch(() => {});
		playing = new Set([...playing, s.id]);
		suggestions = suggestions.filter((x) => x.id !== s.id);
	}

	interface Row {
		competition: Competition;
		seasons: Tournament[];
		current: Tournament;
		playing: boolean;
	}

	/** One row per competition, keyed by its current season; competitions
	 *  with something running first, then by name. */
	let rows = $derived.by<Row[]>(() => {
		const by = new Map<string, Tournament[]>();
		for (const t of tournamentStore.list) {
			if (!t.competition || t.status === 'draft') continue;
			if (!by.has(t.competition.key)) by.set(t.competition.key, []);
			by.get(t.competition.key)!.push(t);
		}
		const out: Row[] = [];
		for (const seasons of by.values()) {
			const current = defaultSeason(seasons);
			if (!current) continue;
			out.push({
				competition: current.competition,
				seasons,
				current,
				playing: seasons.some((s) => playing.has(s.id))
			});
		}
		const rank = { active: 0, upcoming: 1, finished: 2, archived: 3, draft: 4 };
		return out.sort(
			(a, b) =>
				rank[a.current.status] - rank[b.current.status] ||
				a.competition.name.localeCompare(b.competition.name)
		);
	});

	function initials(name: string): string {
		return name
			.split(/\s+/)
			.filter((w) => /^[A-Z0-9]/.test(w))
			.map((w) => w[0])
			.join('')
			.slice(0, 3);
	}

	const statusLabel: Record<string, string> = {
		active: 'live',
		upcoming: 'upcoming',
		finished: 'finished',
		archived: 'archived'
	};

	function kindLabel(c: Competition): string {
		const kind = c.teamKind === 'club' ? 'Clubs' : 'National teams';
		return c.country && c.country !== 'World' ? `${c.country} · ${kind}` : kind;
	}
</script>

<div class="cat stagger">
	{#each suggestions as s (s.id)}
		<div class="card suggest">
			<span class="stxt">
				<b>{s.name}</b>
				<span class="muted"
					>{s.leagueMates} {s.leagueMates === 1 ? 'league mate plays' : 'league mates play'} this</span
				>
			</span>
			<span class="spacer"></span>
			<button class="btn slim" onclick={() => play(s)}><Plus size={16} /> Play</button>
		</div>
	{/each}

	{#if ready && rows.length === 0}
		<div class="card empty muted">No competitions yet — check back soon.</div>
	{/if}

	{#each rows as r (r.competition.key)}
		<a class="card crow" href={`/competitions/${r.competition.key}`}>
			<span class="logo" class:ph={!competitionLogoUrl(r.competition)}>
				{#if competitionLogoUrl(r.competition)}
					<img src={competitionLogoUrl(r.competition)} alt="" loading="lazy" />
				{:else}
					{initials(r.competition.name)}
				{/if}
			</span>
			<span class="ctxt">
				<b class="cname">{r.competition.name}</b>
				<span class="muted ckind">{kindLabel(r.competition)}</span>
				<span class="cseason">
					<span class="pill" class:live={r.current.status === 'active'}
						>{statusLabel[r.current.status] ?? r.current.status}</span
					>
					<span class="muted">{r.current.name}</span>
					{#if r.playing}<span class="pill ok"><Check size={11} /> playing</span>{/if}
				</span>
			</span>
			<ChevronRight size={18} class="chev" />
		</a>
	{/each}
</div>

<style>
	.suggest {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		margin-bottom: 0.7rem;
		padding: 0.8rem 1rem;
	}
	.stxt {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		min-width: 0;
	}
	.stxt .muted {
		font-size: 0.85rem;
	}
	.btn.slim {
		width: auto;
		padding: 0.5rem 1rem;
		gap: 0.35rem;
	}
	.crow {
		display: flex;
		align-items: center;
		gap: 0.95rem;
		padding: 0.9rem 1rem;
		color: var(--text);
		text-decoration: none;
	}
	.crow + .crow {
		margin-top: 0.7rem;
	}
	.crow:hover {
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
	}
	.logo {
		flex: none;
		width: 52px;
		height: 52px;
		display: grid;
		place-items: center;
		border-radius: 14px;
		background: #fff;
		overflow: hidden;
	}
	.logo img {
		width: 80%;
		height: 80%;
		object-fit: contain;
	}
	.logo.ph {
		background: var(--surface-2);
		color: var(--muted);
		font-family: var(--font-display);
		font-size: 1rem;
		letter-spacing: 0.04em;
	}
	.ctxt {
		display: flex;
		flex-direction: column;
		gap: 0.18rem;
		flex: 1;
		min-width: 0;
	}
	.cname {
		font-size: 1.05rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.ckind {
		font-size: 0.78rem;
	}
	.cseason {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		margin-top: 0.2rem;
		font-size: 0.82rem;
		flex-wrap: wrap;
	}
	.pill.ok {
		color: var(--success);
		border-color: var(--success);
	}
	:global(.crow .chev) {
		flex: none;
		color: var(--muted);
	}
	.empty {
		text-align: center;
		padding: 2rem;
	}
</style>
