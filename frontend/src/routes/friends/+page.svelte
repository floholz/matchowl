<!-- Friends: two tabs. Pools — the invite-code groups with chat and owner
     tools, each bound to seasons — plus the everyone board. Friends — the
     mutual graph: a board of you and your friends for one season, requests,
     and finding people. -->
<script lang="ts">
	import { api, type LeagueSummary, type LeaderboardRow, type Person } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import { pb } from '$lib/pb';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { tournamentStore, defaultSeason, seasonLabel } from '$lib/tournament.svelte';
	import { pageChrome } from '$lib/shell.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import { Globe, ChevronRight, ChevronDown, MessageSquare, Check, X, UserPlus, Search } from '@lucide/svelte';

	pageChrome(() => ({ title: 'Friends' }));

	let view = $derived($page.url.searchParams.get('tab') === 'friends' ? 'friends' : 'pools');
	function setView(v: 'pools' | 'friends') {
		goto(v === 'friends' ? '/friends?tab=friends' : '/friends', { replaceState: true, noScroll: true, keepFocus: true });
	}

	// ---- pools ----
	type Rank = { rank: number; total: number; points: number; leader: string; leaderPoints: number };
	let leagues = $state<LeagueSummary[]>([]);
	let ranks = $state<Record<string, Rank | null>>({});
	let unread = $state<Record<string, number>>({});
	let loaded = $state(false);
	let newName = $state('');
	let newSeasons = $state<Set<string>>(new Set());
	let joinCode = $state('');
	let error = $state('');
	let busy = $state(false);
	const isGlobal = (l: LeagueSummary) => l.inviteCode === 'GLOBAL';
	let pools = $derived(leagues.filter((l) => !isGlobal(l)));
	let global = $derived(leagues.find(isGlobal));
	/** Seasons a new pool can count: anything visible, running first. */
	let seasonChoices = $derived(
		tournamentStore.list
			.filter((t) => t.status !== 'draft')
			.sort((a, b) => (a.status === 'active' ? -1 : 1) - (b.status === 'active' ? -1 : 1) || (a.startsAt < b.startsAt ? 1 : -1))
	);

	async function load() {
		try {
			leagues = (await api.myLeagues()).leagues;
			leagues.forEach((l) => loadRank(l.id));
			api.chatUnread().then((r) => (unread = r.unread)).catch(() => {});
		} catch {
			/* ignore */
		} finally {
			loaded = true;
		}
	}
	$effect(() => {
		load();
		tournamentStore.ready().catch(() => {});
	});
	function loadRank(id: string) {
		api
			.leaderboard(id)
			.then(({ rows }) => {
				const i = rows.findIndex((r) => r.userId === auth.user?.id);
				ranks[id] =
					i >= 0
						? { rank: i + 1, total: rows.length, points: rows[i].total, leader: rows[0]?.name ?? '', leaderPoints: rows[0]?.total ?? 0 }
						: null;
			})
			.catch(() => (ranks[id] = null));
	}
	function toggleSeason(slug: string) {
		const next = new Set(newSeasons);
		if (next.has(slug)) next.delete(slug);
		else next.add(slug);
		newSeasons = next;
	}
	async function create(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			const r = await api.createLeague(newName, [...newSeasons]);
			newName = '';
			newSeasons = new Set();
			goto(`/pools/${r.id}`);
		} catch {
			error = 'Could not create the pool.';
		} finally {
			busy = false;
		}
	}
	async function join(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			const r = await api.joinLeague(joinCode);
			joinCode = '';
			goto(`/pools/${r.id}`);
		} catch {
			error = 'Invalid invite code.';
		} finally {
			busy = false;
		}
	}
	const seasonsLine = (l: LeagueSummary) =>
		l.tournaments.length
			? l.tournaments.map((t) => `${t.competition?.shortName || t.competition?.name || ''} ${seasonLabel(t)}`.trim()).join(' · ')
			: 'no season yet';

	// ---- friends ----
	let friends = $state<Person[]>([]);
	let incoming = $state<Person[]>([]);
	let outgoing = $state<Person[]>([]);
	let fLoaded = $state(false);
	let q = $state('');
	let results = $state<Person[]>([]);
	let searching = $state(false);
	let boardSlug = $state('');
	let boardRows = $state<LeaderboardRow[]>([]);
	let boardLoading = $state(false);
	let boardTournament = $derived(tournamentStore.list.find((t) => t.slug === boardSlug));
	let competitions = $derived.by(() => {
		const seen = new Map<string, { key: string; name: string; shortName: string }>();
		for (const t of seasonChoices) if (t.competition && !seen.has(t.competition.key)) seen.set(t.competition.key, t.competition);
		return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name));
	});
	let boardSeasons = $derived(
		boardTournament ? seasonChoices.filter((t) => t.competition?.key === boardTournament!.competition?.key) : []
	);
	async function loadFriends() {
		try {
			const r = await api.friends();
			friends = r.friends;
			incoming = r.incoming;
			outgoing = r.outgoing;
		} catch {
			/* ignore */
		} finally {
			fLoaded = true;
		}
	}
	async function loadBoard(slug = boardSlug) {
		boardLoading = true;
		try {
			const r = await api.friendsBoard(slug);
			boardSlug = r.tournament;
			boardRows = r.rows;
		} catch {
			boardRows = [];
		} finally {
			boardLoading = false;
		}
	}
	let friendsStarted = false;
	$effect(() => {
		if (view !== 'friends' || friendsStarted) return;
		friendsStarted = true;
		loadFriends();
		loadBoard('');
	});
	function pickCompetition(key: string) {
		const s = defaultSeason(seasonChoices.filter((t) => t.competition?.key === key));
		if (s) loadBoard(s.slug);
	}
	let searchTimer: ReturnType<typeof setTimeout>;
	function onSearch() {
		clearTimeout(searchTimer);
		const term = q.trim();
		if (term.length < 2) {
			results = [];
			return;
		}
		searchTimer = setTimeout(async () => {
			searching = true;
			try {
				results = (await api.searchPeople(term)).users;
			} finally {
				searching = false;
			}
		}, 250);
	}
	async function request(p: Person) {
		await api.requestFriend(p.userId).catch(() => {});
		await loadFriends();
		onSearch();
		loadBoard();
	}
	async function accept(p: Person) {
		await api.acceptFriend(p.userId).catch(() => {});
		await loadFriends();
		loadBoard();
	}
	async function remove(p: Person) {
		await api.removeFriend(p.userId).catch(() => {});
		await loadFriends();
		onSearch();
		loadBoard();
	}
	function avatarUrl(userId: string, avatar?: string | null): string | null {
		return avatar ? pb.files.getURL({ id: userId, collectionName: 'users' }, avatar) : null;
	}
</script>

<div class="subbar tabrow">
	<div class="utabs" role="tablist">
		<button class="utab" class:on={view === 'pools'} role="tab" aria-selected={view === 'pools'} onclick={() => setView('pools')}>Pools</button>
		<button class="utab" class:on={view === 'friends'} role="tab" aria-selected={view === 'friends'} onclick={() => setView('friends')}>Friends</button>
	</div>
</div>

{#if view === 'pools'}
	{#if !loaded}
		<p class="muted pad">Loading…</p>
	{:else if pools.length === 0}
		<div class="card quiet muted">No pools yet — start one for the season with your friends, or join one with a code.</div>
	{:else}
		{#each pools as l (l.id)}
			{@const r = ranks[l.id]}
			<a class="card league" href={`/pools/${l.id}`}>
				<span class="lhead">
					<span class="lt"><b class="lname">{l.name}</b><span class="muted lseasons">{seasonsLine(l)}</span></span>
					{#if l.role === 'owner'}<span class="pill">owner</span>{/if}
					<span class="spacer"></span>
					{#if unread[l.id]}
						<span class="pill ok"><MessageSquare size={12} /> {unread[l.id] > 99 ? '99+' : unread[l.id]}</span>
					{:else}
						<span class="muted"><MessageSquare size={16} /></span>
					{/if}
				</span>
				<span class="lgrid">
					<span class="cell">
						<span class="big digits">{r ? `#${r.rank}` : '–'}<small>/{r?.total ?? l.members}</small></span>
						<span class="muted lbl">{r && r.rank === 1 ? (r.total > 1 ? 'you lead' : 'only you') : 'your place'}</span>
					</span>
					<span class="cell">
						<span class="big digits">{r?.points ?? 0}</span>
						<span class="muted lbl">your points</span>
					</span>
					<span class="cell right">
						{#if r && r.rank > 1}
							<span class="big digits">{r.leaderPoints}</span>
							<span class="muted lbl">{r.leader} leads · {r.leaderPoints - r.points} behind</span>
						{:else}
							<span class="big digits">{l.members}</span>
							<span class="muted lbl">{l.members === 1 ? 'member' : 'members'}</span>
						{/if}
					</span>
				</span>
			</a>
		{/each}
	{/if}
	{#if global}
		{@const r = ranks[global.id]}
		<a class="card grow" href={`/pools/${global.id}`}>
			<span class="gico"><Globe size={18} /></span>
			<span class="gtxt"><b>Everyone</b><span class="muted">all of Matchowl, this season</span></span>
			<span class="spacer"></span>
			<span class="digits">{r ? `#${r.rank}` : '–'}<small class="muted">/{r?.total ?? global.members}</small></span>
			<ChevronRight size={16} class="cv" />
		</a>
	{/if}

	<h2 class="sec">Start a pool</h2>
	<section class="card actions">
		<form class="action col" onsubmit={create}>
			<input class="input" placeholder="Pool name" bind:value={newName} required />
			<div class="seasons">
				<span class="muted small">Counts these seasons</span>
				<div class="chipset">
					{#each seasonChoices as t (t.id)}
						<button type="button" class="schip" class:on={newSeasons.has(t.slug)} onclick={() => toggleSeason(t.slug)}>
							{#if newSeasons.has(t.slug)}<Check size={13} />{/if}
							{t.competition?.shortName || t.competition?.name} {seasonLabel(t)}
						</button>
					{/each}
				</div>
			</div>
			<button class="btn" disabled={busy || !newName.trim() || newSeasons.size === 0}>Create pool</button>
		</form>
		<div class="orsep"><span>or join one</span></div>
		<form class="action" onsubmit={join}>
			<input class="input code" placeholder="INVITE CODE" bind:value={joinCode} required />
			<button class="btn secondary" disabled={busy || !joinCode.trim()}>Join</button>
		</form>
	</section>
	{#if error}<p class="error">{error}</p>{/if}
{:else}
	<div class="chips">
		<label class="chip sel">
			<select value={boardTournament?.competition?.key ?? ''} onchange={(e) => pickCompetition((e.currentTarget as HTMLSelectElement).value)} aria-label="Competition">
				{#each competitions as c (c.key)}<option value={c.key}>{c.shortName || c.name}</option>{/each}
			</select>
			<span class="lbl">{boardTournament?.competition?.shortName || boardTournament?.competition?.name || 'Competition'}</span>
			<ChevronDown size={14} />
		</label>
		{#if boardSeasons.length > 1}
			<label class="chip sel">
				<select value={boardSlug} onchange={(e) => loadBoard((e.currentTarget as HTMLSelectElement).value)} aria-label="Season">
					{#each boardSeasons as t (t.id)}<option value={t.slug}>{seasonLabel(t)}</option>{/each}
				</select>
				<span class="lbl">{boardTournament ? seasonLabel(boardTournament) : ''}</span>
				<ChevronDown size={14} />
			</label>
		{/if}
	</div>
	<div class="card list" class:dim={boardLoading}>
		{#each boardRows as r, i (r.userId)}
			<div class="brow" class:me={r.userId === auth.user?.id}>
				<span class="rank digits">{i + 1}</span>
				<Avatar name={r.name} src={avatarUrl(r.userId, r.avatar)} size={28} />
				<span class="bname">{r.name}{#if r.userId === auth.user?.id}<span class="pill ok you">you</span>{/if}</span>
				<span class="spacer"></span>
				<span class="bpts digits">{r.total}</span>
			</div>
		{/each}
		{#if boardRows.length <= 1 && fLoaded}
			<div class="brow muted small">Add friends below to compare — until then it's just you.</div>
		{/if}
	</div>
	{#if global}
		<a class="more" href={`/pools/${global.id}`}>Everyone on Matchowl <ChevronRight size={14} /></a>
	{/if}

	{#if incoming.length}
		<h2 class="sec">Requests</h2>
		<div class="card list">
			{#each incoming as p (p.userId)}
				<div class="brow">
					<Avatar name={p.name} src={avatarUrl(p.userId, p.avatar)} size={28} />
					<span class="bname">{p.name}</span>
					<span class="spacer"></span>
					<button class="ibtn ok" onclick={() => accept(p)} aria-label="Accept"><Check size={16} /></button>
					<button class="ibtn" onclick={() => remove(p)} aria-label="Decline"><X size={16} /></button>
				</div>
			{/each}
		</div>
	{/if}

	<h2 class="sec">Add friends</h2>
	<label class="search">
		<Search size={16} />
		<input class="input" placeholder="Search by name" bind:value={q} oninput={onSearch} />
	</label>
	{#if results.length}
		<div class="card list">
			{#each results as p (p.userId)}
				<div class="brow">
					<Avatar name={p.name} src={avatarUrl(p.userId, p.avatar)} size={28} />
					<span class="bname">{p.name}</span>
					<span class="spacer"></span>
					{#if p.state === 'accepted'}
						<span class="pill ok">friends</span>
					{:else if p.state === 'pending'}
						<span class="pill">requested</span>
					{:else if p.state === 'incoming'}
						<button class="ibtn ok" onclick={() => accept(p)} aria-label="Accept"><Check size={16} /></button>
					{:else}
						<button class="ibtn ok" onclick={() => request(p)} aria-label="Add friend"><UserPlus size={16} /></button>
					{/if}
				</div>
			{/each}
		</div>
	{:else if q.trim().length >= 2 && !searching}
		<p class="muted small pad">No one by that name.</p>
	{/if}

	{#if friends.length || outgoing.length}
		<h2 class="sec">Your friends</h2>
		<div class="card list">
			{#each friends as p (p.userId)}
				<div class="brow">
					<Avatar name={p.name} src={avatarUrl(p.userId, p.avatar)} size={28} />
					<span class="bname">{p.name}</span>
					<span class="spacer"></span>
					<button class="ibtn" onclick={() => remove(p)} aria-label="Remove friend"><X size={16} /></button>
				</div>
			{/each}
			{#each outgoing as p (p.userId)}
				<div class="brow">
					<Avatar name={p.name} src={avatarUrl(p.userId, p.avatar)} size={28} />
					<span class="bname">{p.name}</span>
					<span class="pill">requested</span>
					<span class="spacer"></span>
					<button class="ibtn" onclick={() => remove(p)} aria-label="Withdraw"><X size={16} /></button>
				</div>
			{/each}
		</div>
	{/if}
{/if}

<style>
	.sec {
		font-size: 1.05rem;
		margin: 1.2rem 0 0.7rem;
	}
	.pad {
		padding: 0.4rem 0.2rem;
	}
	.small {
		font-size: 0.8rem;
	}
	.card.quiet {
		padding: 0.9rem 1rem;
		font-size: 0.9rem;
	}
	.card.league,
	.card.grow {
		display: flex;
		color: var(--text);
		padding: 0;
	}
	.card.league {
		flex-direction: column;
	}
	.card + .card {
		margin-top: 0.75rem;
	}
	.lhead {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.75rem 0.9rem 0.4rem;
	}
	.lt {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
	}
	.lname {
		font-size: 1.05rem;
	}
	.lseasons {
		font-size: 0.74rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.lgrid {
		display: grid;
		grid-template-columns: 1fr 1fr 1.3fr;
		gap: 0.5rem;
		padding: 0.2rem 0.9rem 0.8rem;
	}
	.cell {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
	}
	.cell.right {
		align-items: flex-end;
		text-align: right;
	}
	.big {
		font-size: 1.45rem;
	}
	.big small {
		font-size: 0.75rem;
		color: var(--muted);
		font-weight: 600;
	}
	.lbl {
		font-size: 0.74rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		max-width: 100%;
	}
	.card.grow {
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 0.9rem;
	}
	.gico {
		color: var(--muted);
		display: inline-flex;
	}
	.gtxt {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		font-size: 0.92rem;
	}
	.gtxt .muted {
		font-size: 0.78rem;
	}
	.card.grow :global(.cv) {
		color: var(--muted);
	}
	.actions {
		margin-top: 0;
	}
	.action {
		display: flex;
		gap: 0.5rem;
	}
	.action.col {
		flex-direction: column;
	}
	.action .btn {
		width: auto;
		padding: 0.8rem 1.1rem;
	}
	.action.col .btn {
		width: 100%;
	}
	.action .input {
		flex: 1;
	}
	.seasons {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
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
		cursor: pointer;
	}
	.schip.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.input.code {
		text-transform: uppercase;
		letter-spacing: 0.15em;
		font-family: var(--font-mono);
	}
	.orsep {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		margin: 0.7rem 0;
		font-size: 0.75rem;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.orsep::before,
	.orsep::after {
		content: '';
		flex: 1;
		border-top: 1px solid var(--border);
	}

	/* ---- friends tab ---- */
	.chips {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.75rem;
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
		color: var(--text);
		font: inherit;
		font-weight: 700;
		font-size: 0.8rem;
		white-space: nowrap;
		cursor: pointer;
		max-width: 60vw;
	}
	.chip .lbl {
		overflow: hidden;
		text-overflow: ellipsis;
	}
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
	.card.list {
		padding: 0;
	}
	.card.list.dim {
		opacity: 0.6;
	}
	.brow {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.55rem 0.9rem;
		border-bottom: 1px solid var(--border);
		font-size: 0.92rem;
	}
	.brow:last-child {
		border-bottom: none;
	}
	.brow.me {
		background: color-mix(in srgb, var(--accent) 8%, transparent);
		box-shadow: inset 3px 0 0 var(--accent);
	}
	.rank {
		width: 1.6rem;
		color: var(--muted);
	}
	.brow.me .rank {
		color: var(--accent);
	}
	.bname {
		font-weight: 600;
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		min-width: 0;
	}
	.pill.you {
		font-size: 0.56rem;
		padding: 0.1rem 0.4rem;
	}
	.bpts {
		font-size: 1rem;
	}
	.more {
		display: inline-flex;
		align-items: center;
		gap: 0.1rem;
		margin: 0.6rem 0.3rem 0;
		font-size: 0.85rem;
		font-weight: 600;
	}
	.ibtn {
		width: 34px;
		height: 34px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--muted);
		cursor: pointer;
	}
	.ibtn.ok {
		color: var(--accent);
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
	}
	.search {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0 0.8rem;
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		color: var(--muted);
		margin-bottom: 0.75rem;
	}
	.search .input {
		border: none;
		background: transparent;
		padding-left: 0;
	}
	.search .input:focus {
		outline: none;
	}
</style>
