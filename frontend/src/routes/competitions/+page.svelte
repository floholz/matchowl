<!-- Competitions catalog: a search field in the subbar, a couple of "your
     pool mates play this" suggestions (the rest behind "show more"), then
     every competition in three sections — the ones you play, the ones you
     could, the finished ones — each row with its badge, country · kind,
     the current season with its status, and Play / Playing right there.
     Tapping the row opens the hub. -->
<script lang="ts">
	import { pb } from '$lib/pb';
	import { auth } from '$lib/auth.svelte';
	import { feedStore } from '$lib/feed.svelte';
	import {
		tournamentStore,
		defaultSeason,
		competitionLogoUrl,
		seasonLabel,
		type Tournament,
		type Competition
	} from '$lib/tournament.svelte';
	import { pageChrome } from '$lib/shell.svelte';
	import { Check, ChevronRight, Plus, Search, X } from '@lucide/svelte';

	pageChrome(() => ({ title: 'Competitions' }));

	interface Suggestion {
		id: string;
		slug: string;
		name: string;
		poolMates: number;
	}
	let suggestions = $state<Suggestion[]>([]);
	const SUGGEST_SHOWN = 2;
	let moreSuggestions = $state(false);
	let shownSuggestions = $derived(
		moreSuggestions ? suggestions : suggestions.slice(0, SUGGEST_SHOWN)
	);

	let ready = $state(false);
	let playing = $state<Set<string>>(new Set());
	let busy = $state<Set<string>>(new Set());
	let q = $state('');

	$effect(() => {
		tournamentStore.ready().then(() => (ready = true));
		if (auth.isAuthed) {
			pb.send('/api/me/tournaments', { method: 'GET' })
				.then((r) => (playing = new Set((r.tournaments ?? []).map((t: Tournament) => t.id))))
				.catch(() => {});
			pb.send('/api/tournaments/suggestions', { method: 'GET' })
				.then((r) => (suggestions = (r.suggestions ?? []).filter((x: Suggestion) => x.poolMates > 0)))
				.catch(() => {});
		}
	});

	/** Play or leave a season; Home and Matches read the feed, so refresh it. */
	async function setPlaying(t: Tournament, on: boolean) {
		if (busy.has(t.id)) return;
		busy = new Set([...busy, t.id]);
		try {
			await pb.send(`/api/tournaments/${t.slug}/play`, { method: on ? 'POST' : 'DELETE' });
			const next = new Set(playing);
			if (on) next.add(t.id);
			else next.delete(t.id);
			playing = next;
			suggestions = suggestions.filter((s) => s.id !== t.id);
			if (feedStore.loaded) feedStore.load().catch(() => {});
		} catch {
			/* leave the button as it was */
		} finally {
			const b = new Set(busy);
			b.delete(t.id);
			busy = b;
		}
	}
	async function playSuggestion(s: Suggestion) {
		const t = tournamentStore.list.find((x) => x.id === s.id);
		if (t) await setPlaying(t, true);
		else {
			await pb.send(`/api/tournaments/${s.slug}/play`, { method: 'POST' }).catch(() => {});
			playing = new Set([...playing, s.id]);
			suggestions = suggestions.filter((x) => x.id !== s.id);
		}
	}

	interface Row {
		competition: Competition;
		seasons: Tournament[];
		current: Tournament;
		playing: boolean;
	}

	/** One row per competition, keyed by its current season. */
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
		return out.sort((a, b) => a.competition.name.localeCompare(b.competition.name));
	});

	function matches(r: Row, term: string): boolean {
		if (!term) return true;
		const hay = `${r.competition.name} ${r.competition.shortName ?? ''} ${r.competition.country ?? ''} ${r.current.name}`.toLowerCase();
		return term
			.toLowerCase()
			.split(/\s+/)
			.filter(Boolean)
			.every((w) => hay.includes(w));
	}
	let filtered = $derived(rows.filter((r) => matches(r, q.trim())));
	const over = (r: Row) => r.current.status === 'finished' || r.current.status === 'archived';
	let mine = $derived(filtered.filter((r) => r.playing && !over(r)));
	let open = $derived(filtered.filter((r) => !r.playing && !over(r)));
	let done = $derived(filtered.filter(over));

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
		upcoming: 'soon',
		finished: 'finished',
		archived: 'archived'
	};

	function kindLabel(c: Competition): string {
		const kind = c.teamKind === 'club' ? 'Clubs' : 'National teams';
		return c.country && c.country !== 'World' ? `${c.country} · ${kind}` : kind;
	}
</script>

<div class="subbar catbar">
	<label class="search">
		<Search size={16} class="sico" />
		<input
			class="sinput"
			type="search"
			placeholder="Search competitions"
			bind:value={q}
			autocomplete="off"
			aria-label="Search competitions"
		/>
		{#if q}<button class="clear" aria-label="Clear" onclick={() => (q = '')}><X size={14} /></button>{/if}
	</label>
</div>

<div class="cat stagger">
	{#if suggestions.length && !q}
		<div class="sec2 first"><h2>Your pool mates play</h2></div>
		{#each shownSuggestions as s (s.id)}
			<div class="card suggest">
				<span class="stxt">
					<b>{s.name}</b>
					<span class="muted small">{s.poolMates} {s.poolMates === 1 ? 'pool mate plays' : 'pool mates play'} this</span>
				</span>
				<span class="spacer"></span>
				<button class="tbtn p" onclick={() => playSuggestion(s)}><Plus size={14} /> Play</button>
			</div>
		{/each}
		{#if suggestions.length > SUGGEST_SHOWN}
			<button class="more" onclick={() => (moreSuggestions = !moreSuggestions)}>
				{moreSuggestions ? 'Show fewer' : `Show ${suggestions.length - SUGGEST_SHOWN} more`}
			</button>
		{/if}
	{/if}

	{#if ready && rows.length === 0}
		<div class="card empty muted">No competitions yet — check back soon.</div>
	{:else if ready && filtered.length === 0}
		<div class="card empty muted">Nothing matches “{q}”.</div>
	{/if}

	{#snippet row(r: Row)}
		{@const t = r.current}
		{@const isBusy = busy.has(t.id)}
		<div class="card crow" class:on={r.playing}>
			<a class="clink" href={`/competitions/${r.competition.key}`}>
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
						<span class="pill" class:live={t.status === 'active'}>{statusLabel[t.status] ?? t.status}</span>
						<span class="muted">{seasonLabel(t)}{#if r.seasons.length > 1} · {r.seasons.length} seasons{/if}</span>
					</span>
				</span>
				<ChevronRight size={18} class="chev" />
			</a>
			{#if auth.isAuthed && (t.status !== 'archived' || r.playing)}
				<button
					class="tbtn"
					class:p={!r.playing}
					class:on={r.playing}
					disabled={isBusy}
					onclick={() => setPlaying(t, !r.playing)}
					aria-pressed={r.playing}
				>
					{#if r.playing}<Check size={14} /> Playing{:else}<Plus size={14} /> Play{/if}
				</button>
			{/if}
		</div>
	{/snippet}

	{#if mine.length}
		<div class="sec2" class:first={!suggestions.length || !!q}><h2>Playing</h2><span class="pill ok">{mine.length}</span></div>
		{#each mine as r (r.competition.key)}{@render row(r)}{/each}
	{/if}
	{#if open.length}
		<div class="sec2" class:first={!mine.length && (!suggestions.length || !!q)}>
			<h2>{mine.length ? 'More competitions' : 'Play a competition'}</h2>
		</div>
		{#each open as r (r.competition.key)}{@render row(r)}{/each}
	{/if}
	{#if done.length}
		<div class="sec2"><h2>Finished</h2></div>
		{#each done as r (r.competition.key)}{@render row(r)}{/each}
	{/if}
</div>

<style>
	.catbar {
		padding-top: 0.35rem;
		padding-bottom: 0.6rem;
	}
	.search {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		height: 40px;
		padding: 0 0.6rem 0 0.8rem;
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		color: var(--muted);
	}
	.search:focus-within {
		border-color: var(--accent);
	}
	:global(.search .sico) {
		flex: none;
	}
	.sinput {
		flex: 1;
		min-width: 0;
		background: transparent;
		border: 0;
		color: var(--text);
		font: inherit;
		font-size: 0.95rem;
		outline: none;
	}
	.sinput::-webkit-search-cancel-button {
		display: none;
	}
	.clear {
		display: grid;
		place-items: center;
		width: 26px;
		height: 26px;
		border-radius: 50%;
		border: 0;
		background: var(--surface-2);
		color: var(--muted);
		cursor: pointer;
	}
	.sec2 {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin: 1.1rem 0.15rem 0.55rem;
	}
	.sec2 h2 {
		font-size: 1.05rem;
		margin: 0;
	}
	.sec2.first {
		margin-top: 0.2rem;
	}
	.suggest {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		padding: 0.8rem 1rem;
	}
	.suggest + .suggest {
		margin-top: 0.55rem;
	}
	.stxt {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		min-width: 0;
	}
	.small {
		font-size: 0.85rem;
	}
	.more {
		display: block;
		margin: 0.5rem auto 0;
		padding: 0.35rem 0.9rem;
		border: 0;
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--accent);
		font: inherit;
		font-size: 0.85rem;
		font-weight: 700;
		cursor: pointer;
	}
	.tbtn {
		flex: none;
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		height: 32px;
		padding: 0 0.8rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: transparent;
		color: var(--muted);
		font: inherit;
		font-size: 0.82rem;
		font-weight: 700;
		cursor: pointer;
		white-space: nowrap;
	}
	.tbtn.p {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.tbtn.on {
		color: var(--success);
		border-color: color-mix(in srgb, var(--success) 55%, var(--border));
	}
	.tbtn:disabled {
		opacity: 0.6;
		cursor: default;
	}
	.crow {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.75rem 0.85rem 0.75rem 0.9rem;
	}
	.crow + .crow {
		margin-top: 0.55rem;
	}
	.crow.on {
		border-color: color-mix(in srgb, var(--success) 35%, var(--border));
	}
	.crow:hover {
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
	}
	.clink {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		flex: 1;
		min-width: 0;
		color: var(--text);
		text-decoration: none;
	}
	.logo {
		flex: none;
		width: 48px;
		height: 48px;
		display: grid;
		place-items: center;
		border-radius: 13px;
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
		font-size: 0.95rem;
		letter-spacing: 0.04em;
	}
	.ctxt {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		flex: 1;
		min-width: 0;
	}
	.cname {
		font-size: 1.02rem;
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
		margin-top: 0.15rem;
		font-size: 0.8rem;
		flex-wrap: wrap;
	}
	.pill.ok {
		color: var(--success);
		border-color: var(--success);
	}
	:global(.clink .chev) {
		flex: none;
		color: var(--muted);
	}
	.empty {
		text-align: center;
		padding: 2rem;
	}
	@media (max-width: 420px) {
		.ckind {
			display: none;
		}
	}
</style>
