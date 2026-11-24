<!-- Search anything: competitions and teams (later maybe friends and
     pools), from /api/search. A star follows a team right from the list.
     The empty field shows your recent searches (this device). -->
<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import { teamsStore, crestUrl, type SearchResult, type TeamIdentity } from '$lib/teams.svelte';
	import { countryName } from '$lib/countries';
	import { feedStore } from '$lib/feed.svelte';
	import PageChrome from '$lib/components/PageChrome.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import { Search, X, Star, ChevronRight, History } from '@lucide/svelte';

	let q = $state($page.url.searchParams.get('q') ?? '');
	let res = $state<SearchResult | null>(null);
	let busy = $state(false);
	let inputEl = $state<HTMLInputElement | null>(null);

	$effect(() => {
		inputEl?.focus();
	});
	$effect(() => {
		if (auth.isAuthed && !teamsStore.loaded) teamsStore.load().catch(() => {});
	});

	// Debounced query; the newest answer wins.
	let seq = 0;
	$effect(() => {
		const s = q.trim();
		const u = new URL($page.url);
		if (s) u.searchParams.set('q', s);
		else u.searchParams.delete('q');
		if (u.search !== $page.url.search) goto(`${u.pathname}${u.search}`, { replaceState: true, noScroll: true, keepFocus: true });
		if (s.length < 2) {
			res = null;
			return;
		}
		const my = ++seq;
		busy = true;
		const t = setTimeout(() => {
			teamsStore
				.search(s)
				.then((r) => {
					if (my === seq) res = r;
				})
				.catch(() => {})
				.finally(() => {
					if (my === seq) busy = false;
				});
		}, 200);
		return () => clearTimeout(t);
	});

	function keep() {
		teamsStore.remember(q);
	}
	async function star(t: TeamIdentity) {
		keep();
		await teamsStore.setFollowing(t.key, !teamsStore.isFollowing(t.key)).catch(() => {});
		if (feedStore.scope === 'mine' && feedStore.loaded) feedStore.load().catch(() => {});
	}
	function teamMeta(t: TeamIdentity): string {
		if (t.national) return 'National team';
		return [t.country ? countryName(t.country) : '', 'club'].filter(Boolean).join(' · ');
	}
	let empty = $derived(!!res && !res.competitions.length && !res.teams.length);
</script>

<PageChrome title="Search" back="/matches" />

<div class="search">
	<label class="field">
		<Search size={18} />
		<input
			bind:this={inputEl}
			bind:value={q}
			type="search"
			placeholder="Competitions and teams"
			autocomplete="off"
			enterkeyhint="search"
			onkeydown={(e) => e.key === 'Enter' && keep()}
			aria-label="Search competitions and teams"
		/>
		{#if q}
			<button class="clear" onclick={() => { q = ''; inputEl?.focus(); }} aria-label="Clear"><X size={16} /></button>
		{/if}
	</label>

	{#if q.trim().length < 2}
		{#if teamsStore.recent.length}
			<div class="sec"><h2>Recent</h2><button class="more" onclick={() => teamsStore.clearRecent()}>Clear</button></div>
			<div class="card rows">
				{#each teamsStore.recent as r (r)}
					<button class="row recent" onclick={() => (q = r)}>
						<History size={16} class="ic" />
						<span class="txt"><b>{r}</b></span>
						<ChevronRight size={16} class="cv" />
					</button>
				{/each}
			</div>
		{:else}
			<p class="muted hint">Find a competition or a team. Star a team to follow it: its matches join My matches.</p>
		{/if}
	{:else if res}
		{#if res.competitions.length}
			<div class="sec"><h2>Competitions</h2></div>
			<div class="card rows">
				{#each res.competitions as c (c.key)}
					<a class="row" href={`/competitions/${c.key}`} onclick={keep}>
						<span class="logo">{#if c.logo}<img src={`/api/files/${c.logo}`} alt="" />{/if}</span>
						<span class="txt"><b>{c.name}</b>{#if c.country}<span class="muted">{c.country}</span>{/if}</span>
						<ChevronRight size={16} class="cv" />
					</a>
				{/each}
			</div>
		{/if}
		{#if res.teams.length}
			<div class="sec"><h2>Teams</h2></div>
			<div class="card rows">
				{#each res.teams as t (t.key)}
					{@const on = teamsStore.isFollowing(t.key)}
					<div class="row team">
						<a class="tlink" href={`/teams/${t.key}`} onclick={keep}>
						<span class="logo">
							{#if t.national && t.iso2}
								<Flag iso2={t.iso2} code={t.fifaCode} logo={crestUrl(t.crest)} size={28} />
							{:else if t.crest}
								<img src={crestUrl(t.crest)} alt="" />
							{/if}
						</span>
						<span class="txt"><b>{t.name}</b><span class="muted">{teamMeta(t)}{on ? ' · following' : ''}</span></span>
						</a>
						<button
							class="star"
							class:on
							onclick={() => star(t)}
							aria-pressed={on}
							aria-label={on ? `Unfollow ${t.name}` : `Follow ${t.name}`}
						>
							<Star size={20} fill={on ? 'currentColor' : 'none'} />
						</button>
					</div>
				{/each}
			</div>
		{/if}
		{#if empty}
			<p class="muted hint">Nothing found for “{q.trim()}”.</p>
		{/if}
	{:else if busy}
		<p class="muted hint">Searching…</p>
	{/if}
</div>

<style>
	.field {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0 0.85rem;
		height: 46px;
		border-radius: var(--radius-sm);
		background: var(--surface);
		border: 1px solid var(--border);
		color: var(--muted);
		margin-top: 0.4rem;
	}
	.field:focus-within {
		border-color: var(--accent);
	}
	.field input {
		flex: 1;
		min-width: 0;
		border: 0;
		outline: none;
		background: transparent;
		color: var(--text);
		font: inherit;
		font-size: 1rem;
	}
	.field input::-webkit-search-cancel-button {
		display: none;
	}
	.clear {
		background: none;
		border: 0;
		color: var(--muted);
		cursor: pointer;
		display: inline-flex;
		padding: 0.3rem;
	}
	.sec {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		margin: 1.2rem 0.15rem 0.55rem;
	}
	.sec h2 {
		font-size: 1.2rem;
	}
	.sec .more {
		margin-left: auto;
		background: none;
		border: 0;
		color: var(--accent);
		font-size: 0.82rem;
		font-weight: 600;
		cursor: pointer;
		padding: 0;
	}
	.card.rows {
		padding: 0;
	}
	.row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		width: 100%;
		padding: 0.7rem 0.9rem;
		border: 0;
		border-bottom: 1px solid var(--border);
		background: none;
		color: var(--text);
		font: inherit;
		text-align: left;
		cursor: pointer;
	}
	.row:last-child {
		border-bottom: none;
	}
	.row.team {
		cursor: default;
		padding-right: 0.5rem;
	}
	.tlink {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex: 1;
		min-width: 0;
		color: var(--text);
	}
	.logo {
		width: 28px;
		height: 28px;
		flex: none;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}
	.logo img {
		width: 28px;
		height: 28px;
		object-fit: contain;
	}
	.txt {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
		flex: 1;
		font-size: 0.92rem;
	}
	.txt .muted {
		font-size: 0.8rem;
	}
	.row :global(.cv),
	.row :global(.ic) {
		color: var(--muted);
		flex: none;
	}
	.star {
		background: none;
		border: 0;
		color: var(--muted);
		cursor: pointer;
		display: inline-flex;
		padding: 0.35rem;
		flex: none;
	}
	.star.on,
	.star:hover {
		color: var(--accent);
	}
	.hint {
		margin: 1.2rem 0.2rem;
		font-size: 0.9rem;
	}
</style>
