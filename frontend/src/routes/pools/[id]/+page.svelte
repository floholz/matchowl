<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { api, type LeaderboardRow, type BotSummary, type PoolSeason } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import { pb } from '$lib/pb';
	import { tournamentStore, defaultSeason, seasonLabel } from '$lib/tournament.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { pageChrome } from '$lib/shell.svelte';
	import {
		Eye,
		EyeOff,
		Copy,
		Share2,
		ChevronDown,
		ChevronRight,
		Telescope,
		Settings,
		Check,
		X,
		Lock,
		RefreshCw,
		UserMinus,
		UserPlus,
		Bot,
		ShieldCheck,
		Crown,
		MessageSquare
	} from '@lucide/svelte';

	interface Cfg {
		match: {
			tendency: number;
			exact: number;
			totalGoals: number;
			goalDiff: number;
		};
		forecast: {
			groupPosition: number;
			perfectGroupBonus: number;
			advance: number;
			round: Record<string, number>;
		};
		tiebreakers: string[];
	}
	let cfg = $state<Cfg | null>(null);

	const tbLabel: Record<string, string> = {
		points: 'Total points',
		exactScores: 'Most exact scores',
		correctWinners: 'Most correct winners',
		goalDiffDeviation: 'Smallest goal-difference error vs. results',
		fewestTips: 'Fewest tips submitted',
		earliestEdit: 'Earliest last edit (submitted first)'
	};
	// The forecast-scored knockout rounds, from the tournament structure
	// (consolation stages like the third-place play-off are not scored per
	// round), plus the synthetic CHAMPION entry.
	/** Which tournament the board scores: '' = the server's current pick;
	 *  the response tells us which that was. */
	let tslug = $state('');
	let boardSlug = $state('');
	let boardTournament = $derived(tournamentStore.list.find((t) => t.slug === boardSlug));
	/** The pool's bound seasons (from the leaderboard response); Global has none. */
	let bound = $state<PoolSeason[]>([]);
	let isPool = $derived(bound.length > 0);
	/** Seasons the board can be filtered to: the bound ones, or (Global) every visible one. */
	let tournamentOptions = $derived(
		isPool
			? tournamentStore.list.filter((t) => bound.some((b) => b.id === t.id))
			: tournamentStore.list
					.filter((t) => t.status !== 'draft')
					.sort((a, b) => (a.startsAt < b.startsAt ? 1 : -1))
	);
	let summed = $derived(isPool && boardSlug === '');
	/** Competitions (one per key, first season in the list = newest). */
	let competitions = $derived.by(() => {
		const seen = new Map<string, (typeof tournamentOptions)[number]['competition']>();
		for (const t of tournamentOptions) if (t.competition && !seen.has(t.competition.key)) seen.set(t.competition.key, t.competition);
		return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name));
	});
	let boardSeasons = $derived(
		boardTournament ? tournamentOptions.filter((t) => t.competition?.key === boardTournament!.competition?.key) : []
	);
	function pickCompetition(key: string) {
		const s = defaultSeason(tournamentOptions.filter((t) => t.competition?.key === key));
		if (s) pickTournament(s.slug);
	}
	let fcRounds = $derived(
		(boardTournament?.structure.stages ?? [])
			.filter((s) => s.kind === 'knockout' && !s.consolation)
	);
	let roundLabel = $derived.by(() => {
		const out: Record<string, string> = {};
		for (const s of fcRounds) out[s.code] = s.name;
		out.CHAMPION = 'Champion';
		return out;
	});

	let revealed = $state(false);
	let openRow = $state<string | null>(null);
	/** Page tabs: the leaderboard, or the members (invite, roles, management). */
	let view = $state<'board' | 'members'>('board');

	let id = $derived($page.params.id ?? '');
	let league = $state<{ id: string; name: string } | null>(null);
	let rows = $state<LeaderboardRow[]>([]);
	let invite = $state('');
	let loaded = $state(false);
	let error = $state('');
	let tab = $state<'total' | 'tipsPoints' | 'forecastPoints'>('total');

	// Chat (private leagues only).
	let canChat = $state(false);
	let chatUnread = $state(0);

	// Owner-only management.
	let isOwner = $state(false);
	let isPrivate = $state(false);
	let editing = $state(false);
	let nameDraft = $state('');
	let confirmRegen = $state(false);
	let mgmtBusy = $state(false);
	let mgmtError = $state('');
	let availableBots = $state<BotSummary[]>([]);
	let botBusy = $state<string | null>(null);

	$effect(() => {
		const lid = id;
		loaded = false;
		cfg = null;
		editing = false;
		confirmRegen = false;
		mgmtError = '';
		availableBots = [];
		// The leaderboard's forecast columns come from the tournament structure.
		tournamentStore.ready().catch(() => {});
		Promise.all([api.leaderboard(lid, tslug), api.myPools()])
			.then(([lb, mine]) => {
				league = lb.pool;
				rows = lb.rows;
				boardSlug = lb.tournament ?? '';
				bound = lb.tournaments ?? [];
				cfg = (lb.scoring as Cfg | undefined) ?? null;
				const me = mine.pools.find((l) => l.id === lid);
				invite = me?.inviteCode ?? '';
				isOwner = me?.role === 'owner';
				isPrivate = me?.private ?? false;
				canChat = !!me && me.inviteCode !== 'GLOBAL';
				if (canChat) {
					api
						.chatUnread()
						.then((r) => (chatUnread = r.unread[lid] ?? 0))
						.catch(() => {});
				}
			})
			.catch(() => (error = 'Could not load this pool.'))
			.finally(() => (loaded = true));
	});

	// Keep the chat unread badge live: bump it when a new message lands in this
	// league while we're on the league page (the chat itself marks read on open,
	// and re-mounting the page re-syncs the baseline count).
	$effect(() => {
		if (!canChat || !id) return;
		const lid = id;
		let unsub: (() => void) | null = null;
		pb.collection('league_messages')
			.subscribe(
				'*',
				(e) => {
					if (e.action === 'create' && (e.record as { user?: string }).user !== auth.user?.id) {
						chatUnread += 1;
					}
				},
				{ filter: `league="${lid}"` }
			)
			.then((u) => (unsub = u))
			.catch(() => {});
		return () => unsub?.();
	});

	function enterEdit() {
		nameDraft = league?.name ?? '';
		mgmtError = '';
		confirmRegen = false;
		editing = true;
		loadBots();
	}

	async function loadBots() {
		if (!isOwner) return;
		try {
			availableBots = (await api.availableBots(id)).bots;
		} catch {
			availableBots = [];
		}
	}

	async function refreshRows() {
		try {
			const lb = await api.leaderboard(id, tslug);
			rows = lb.rows;
			boardSlug = lb.tournament ?? '';
			bound = lb.tournaments ?? [];
		} catch {
			/* keep current rows on a transient error */
		}
	}
	function pickTournament(slug: string) {
		tslug = slug;
		refreshRows();
	}
	// ---- owner: bound seasons; set up next season ----
	let seasonChoices = $derived(
		tournamentStore.list
			.filter((t) => t.status === 'active' || t.status === 'upcoming')
			.sort((a, b) => (a.status === 'active' ? -1 : 1) - (b.status === 'active' ? -1 : 1) || (a.startsAt < b.startsAt ? 1 : -1))
	);
	let draftSeasons = $state<Set<string>>(new Set());
	$effect(() => {
		draftSeasons = new Set(bound.map((t) => t.slug));
	});
	function toggleDraftSeason(slug: string) {
		const next = new Set(draftSeasons);
		if (next.has(slug)) next.delete(slug);
		else next.add(slug);
		draftSeasons = next;
	}
	async function saveSeasons() {
		if (!league) return;
		mgmtBusy = true;
		mgmtError = '';
		try {
			const r = await api.setPoolSeasons(league.id, [...draftSeasons]);
			bound = r.tournaments;
			tslug = '';
			await refreshRows();
		} catch {
			mgmtError = 'Could not save the seasons.';
		} finally {
			mgmtBusy = false;
		}
	}
	let cloning = $state(false);
	let cloneName = $state('');
	let cloneSeasons = $state<Set<string>>(new Set());
	function startClone() {
		cloneName = league?.name ?? '';
		cloneSeasons = new Set();
		cloning = true;
	}
	function toggleCloneSeason(slug: string) {
		const next = new Set(cloneSeasons);
		if (next.has(slug)) next.delete(slug);
		else next.add(slug);
		cloneSeasons = next;
	}
	async function doClone() {
		if (!league) return;
		mgmtBusy = true;
		mgmtError = '';
		try {
			const r = await api.clonePool(league.id, cloneName, [...cloneSeasons]);
			goto(`/pools/${r.id}`);
		} catch {
			mgmtError = 'Could not set up the new pool.';
		} finally {
			mgmtBusy = false;
		}
	}

	async function addBot(b: BotSummary) {
		if (!league) return;
		botBusy = b.userId;
		mgmtError = '';
		try {
			await api.addBot(league.id, b.userId);
			availableBots = availableBots.filter((x) => x.userId !== b.userId);
			await refreshRows();
		} catch {
			mgmtError = `Could not add ${b.name}.`;
		} finally {
			botBusy = null;
		}
	}
	function exitEdit() {
		editing = false;
		confirmRegen = false;
	}
	async function saveName() {
		const name = nameDraft.trim();
		if (!league) return;
		if (!name || name === league.name) {
			exitEdit();
			return;
		}
		mgmtBusy = true;
		mgmtError = '';
		try {
			await api.renamePool(league.id, name);
			league = { ...league, name };
			exitEdit();
		} catch {
			mgmtError = 'Could not rename the pool.';
		} finally {
			mgmtBusy = false;
		}
	}
	async function setPrivacy(next: boolean) {
		if (!league || next === isPrivate) return;
		mgmtBusy = true;
		mgmtError = '';
		try {
			await api.setCodePrivacy(league.id, next);
			isPrivate = next;
		} catch {
			mgmtError = 'Could not update visibility.';
		} finally {
			mgmtBusy = false;
		}
	}
	async function regenerate() {
		if (!league) return;
		mgmtBusy = true;
		mgmtError = '';
		try {
			const r = await api.regenerateCode(league.id);
			invite = r.inviteCode;
			confirmRegen = false;
			revealed = true;
		} catch {
			mgmtError = 'Could not regenerate the code.';
		} finally {
			mgmtBusy = false;
		}
	}
	// Two-step removal: clicking the button opens a confirm dialog; confirming
	// runs the actual call. removeTarget holds the pending member (and drives
	// the dialog's open state + message).
	let removeTarget = $state<{ userId: string; name: string } | null>(null);

	function requestRemove(userId: string, name: string) {
		removeTarget = { userId, name };
	}
	async function confirmRemove() {
		if (!league || !removeTarget) return;
		const { userId } = removeTarget;
		mgmtBusy = true;
		mgmtError = '';
		try {
			await api.removeMember(league.id, userId);
			rows = rows.filter((r) => r.userId !== userId);
			// A removed bot becomes available to add again.
			await loadBots();
		} catch {
			mgmtError = 'Could not remove the member.';
		} finally {
			mgmtBusy = false;
			removeTarget = null;
		}
	}

	let sorted = $derived(
		[...rows].sort((a, b) => b[tab] - a[tab])
	);
	let memberLine = $derived(
		`${rows.length} ${rows.length === 1 ? 'member' : 'members'}` +
			(bound.length
				? ` · ${bound.map((t) => `${t.competition?.shortName || t.competition?.name || ''} ${seasonLabel(t)}`.trim()).join(' · ')}`
				: isOwner
					? ' · you own this pool'
					: '')
	);
	pageChrome(() => ({ back: '/friends', title: league?.name ?? 'Pool', context: loaded ? memberLine : '' }));
	let fcView = $derived(tab === 'forecastPoints');

	// Build the avatar URL the same way auth.svelte does — a users.avatar file
	// resolves to /api/files/users/{id}/{filename} (same origin).
	// Pin your row to the bottom edge while it is scrolled out of view.
	let meVisible = $state(true);
	$effect(() => {
		if (!loaded || view !== 'board') return;
		void sorted;
		const el = document.querySelector<HTMLElement>('tr.main.lead');
		if (!el) return;
		const io = new IntersectionObserver(([e]) => (meVisible = e.isIntersecting), { rootMargin: '-60px 0px -70px 0px' });
		io.observe(el);
		return () => io.disconnect();
	});
	let meRow = $derived(sorted.find((r) => r.userId === auth.user?.id));
	let meIndex = $derived(sorted.findIndex((r) => r.userId === auth.user?.id));
	function avatarUrl(userId: string, avatar?: string | null): string | null {
		return avatar
			? pb.files.getURL({ id: userId, collectionName: 'users' }, avatar)
			: null;
	}

	function copyInvite() {
		navigator.clipboard?.writeText(invite);
	}

	let linkCopied = $state(false);
	let copyTimer: ReturnType<typeof setTimeout>;
	function shareInvite() {
		const url = `${window.location.origin}/join/${invite}`;
		navigator.clipboard?.writeText(url);
		linkCopied = true;
		clearTimeout(copyTimer);
		copyTimer = setTimeout(() => (linkCopied = false), 1800);
	}
</script>

{#if error}
	<p class="error">{error}</p>
{:else if !loaded}
	<p class="muted">Loading…</p>
{:else if league}
	<div class="subbar tabrow" class:withchips={view === 'board'}>
		<div class="utabs" role="tablist">
			<button class="utab" class:on={view === 'board'} role="tab" aria-selected={view === 'board'} onclick={() => (view = 'board')}>Leaderboard</button>
			<button class="utab" class:on={view === 'members'} role="tab" aria-selected={view === 'members'} onclick={() => (view = 'members')}>Members</button>
			{#if canChat}
				<a class="utab chat" href={`/pools/${id}/chat`}>Chat{#if chatUnread > 0}<span class="badge">{chatUnread > 99 ? '99+' : chatUnread}</span>{/if}</a>
			{/if}
		</div>
		{#if view === 'board'}
			<div class="chips">
				<label class="chip sel">
					<select value={summed ? '' : (boardTournament?.competition?.key ?? '')} onchange={(e) => { const v = (e.currentTarget as HTMLSelectElement).value; if (v) pickCompetition(v); else pickTournament(''); }} aria-label="Competition">
						{#if isPool && bound.length > 1}<option value="">All seasons</option>{/if}
						{#each competitions as c (c.key)}<option value={c.key}>{c.shortName || c.name}</option>{/each}
					</select>
					<span class="lbl">{summed ? (bound.length > 1 ? 'All seasons' : (boardTournament?.competition?.shortName || bound[0]?.competition?.shortName || 'Season')) : (boardTournament?.competition?.shortName || boardTournament?.competition?.name || 'Competition')}</span>
					<ChevronDown size={14} />
				</label>
				{#if !summed && boardSeasons.length > 1}
					<label class="chip sel">
						<select value={boardSlug} onchange={(e) => pickTournament((e.currentTarget as HTMLSelectElement).value)} aria-label="Season">
							{#each boardSeasons as t (t.id)}<option value={t.slug}>{seasonLabel(t)}</option>{/each}
						</select>
						<span class="lbl">{boardTournament ? seasonLabel(boardTournament) : ''}</span>
						<ChevronDown size={14} />
					</label>
				{/if}
				{#if invite && invite !== 'GLOBAL'}
					<button class="chip inv" onclick={() => (view = 'members')}><Share2 size={14} /> Invite</button>
				{/if}
				<div class="seg2" role="tablist">
					<button class:on={tab === 'total'} onclick={() => (tab = 'total')}>Total</button>
					<button class:on={tab === 'tipsPoints'} onclick={() => (tab = 'tipsPoints')}>Tips</button>
					<button class:on={tab === 'forecastPoints'} onclick={() => (tab = 'forecastPoints')}>Forecast</button>
				</div>
			</div>
		{/if}
	</div>

	{#if mgmtError}<p class="error">{mgmtError}</p>{/if}

	{#if view === 'members'}
	<section class="card manage">
		{#if bound.length}
			<div class="muted small seasonsline">Counts {bound.map((t) => `${t.competition?.shortName || t.competition?.name || ''} ${seasonLabel(t)}`.trim()).join(' · ')}</div>
		{/if}
		<div class="mrow">
			{#if editing}
				<input class="input nameedit" bind:value={nameDraft} maxlength="64" aria-label="Pool name" onkeydown={(e) => e.key === 'Enter' && saveName()} />
				<button class="btn secondary icon" onclick={saveName} disabled={mgmtBusy} aria-label="Save name"><Check size={18} /></button>
				<button class="btn secondary icon" onclick={exitEdit} disabled={mgmtBusy} aria-label="Done editing"><X size={18} /></button>
			{:else}
				<span class="mtxt"><b>{league.name}</b><span class="muted small">{memberLine}</span></span>
				{#if isOwner}
					<button class="btn secondary slim" onclick={enterEdit}><Settings size={16} /> Manage</button>
				{/if}
			{/if}
		</div>
	</section>
	{#if editing && invite !== 'GLOBAL'}
		<section class="card vis">
			<div class="muted small">Seasons this pool counts</div>
			<div class="chipset">
				{#each seasonChoices as t (t.id)}
					<button type="button" class="schip" class:on={draftSeasons.has(t.slug)} onclick={() => toggleDraftSeason(t.slug)}>
						{#if draftSeasons.has(t.slug)}<Check size={13} />{/if}
						{t.competition?.shortName || t.competition?.name} {seasonLabel(t)}
					</button>
				{/each}
			</div>
			<button class="btn secondary slim" onclick={saveSeasons} disabled={mgmtBusy || draftSeasons.size === 0}>Save seasons</button>
		</section>
		<section class="card vis">
			<div class="muted small">Next season</div>
			{#if cloning}
				<input class="input" bind:value={cloneName} maxlength="64" aria-label="New pool name" />
				<div class="chipset">
					{#each seasonChoices as t (t.id)}
						<button type="button" class="schip" class:on={cloneSeasons.has(t.slug)} onclick={() => toggleCloneSeason(t.slug)}>
							{#if cloneSeasons.has(t.slug)}<Check size={13} />{/if}
							{t.competition?.shortName || t.competition?.name} {seasonLabel(t)}
						</button>
					{/each}
				</div>
				<div class="regrow">
					<button class="btn" onclick={doClone} disabled={mgmtBusy || !cloneName.trim() || cloneSeasons.size === 0}>Create the new pool</button>
					<button class="btn secondary" onclick={() => (cloning = false)} disabled={mgmtBusy}>Cancel</button>
				</div>
			{:else}
				<p class="muted small hint">Set up a fresh pool with the same members and settings for the next season; this one stays as history.</p>
				<button class="btn secondary slim" onclick={startClone}>Set up next season</button>
			{/if}
		</section>
	{/if}
	{#if editing}
		<section class="card vis">
			<div class="muted small">Invite code visibility</div>
			<div class="tabs vistabs">
				<button class:active={!isPrivate} onclick={() => setPrivacy(false)} disabled={mgmtBusy}
					>Members</button
				>
				<button class:active={isPrivate} onclick={() => setPrivacy(true)} disabled={mgmtBusy}
					>Private</button
				>
			</div>
			<p class="muted small hint">
				{isPrivate
					? 'Only you can see and share the invite code.'
					: 'Everyone in the pool can see and share the invite code.'}
			</p>
		</section>
	{/if}

	{#if invite && invite !== 'GLOBAL'}
		<section class="card invite">
			<div class="irow">
				<div class="ic">
					<div class="muted small">
						Invite code
						{#if isPrivate}<span class="lockpill"><Lock size={11} /> Private</span>{/if}
					</div>
					<div class="code" class:masked={!revealed}>
						{revealed ? invite : '•'.repeat(invite.length || 6)}
					</div>
				</div>
				<div class="spacer"></div>
				<button
					class="btn secondary eye"
					aria-label={revealed ? 'Hide code' : 'Reveal code'}
					onclick={() => (revealed = !revealed)}
				>
					{#if revealed}<EyeOff size={18} />{:else}<Eye size={18} />{/if}
				</button>
				<button class="btn secondary copy" onclick={copyInvite}>
					<Copy size={16} /> Copy
				</button>
			</div>
			<button class="btn share" onclick={shareInvite}>
				<Share2 size={16} />
				{linkCopied ? 'Link copied!' : 'Share invite link'}
			</button>
			{#if editing}
				{#if confirmRegen}
					<p class="muted small hint regwarn">
						This invalidates the current code and any links already shared.
					</p>
					<div class="regrow">
						<button class="btn danger" onclick={regenerate} disabled={mgmtBusy}>
							Regenerate
						</button>
						<button
							class="btn secondary"
							onclick={() => (confirmRegen = false)}
							disabled={mgmtBusy}>Cancel</button
						>
					</div>
				{:else}
					<button class="btn ghost regenbtn" onclick={() => (confirmRegen = true)}>
						<RefreshCw size={16} /> Regenerate code
					</button>
				{/if}
			{/if}
		</section>
	{/if}

	<section class="card members">
		{#each rows as r (r.userId)}
			<div class="mem">
				<Avatar name={r.name} src={avatarUrl(r.userId, r.avatar)} size={30} />
				<span class="pname">{r.name}</span>
				{#if r.userId === auth.user?.id}<span class="pill ok">you</span>{/if}
				{#if r.role === 'bot'}
					<span class="rolepill" title="Bot player"><Bot size={11} /> Bot</span>
				{:else if r.role === 'admin'}
					<span class="rolepill admin" title="Admin"><ShieldCheck size={11} /> Admin</span>
				{:else if r.role === 'owner'}
					<span class="rolepill owner" title="Platform owner"><Crown size={11} /> Owner</span>
				{/if}
				<span class="spacer"></span>
				<a class="fclink" href={`/forecast/${r.userId}`} title="View {r.name}'s forecast"><Telescope size={16} /></a>
				{#if editing && r.userId !== auth.user?.id}
					<button class="rmbtn" title="Remove {r.name}" aria-label="Remove {r.name}" disabled={mgmtBusy} onclick={() => requestRemove(r.userId, r.name)}>
						<UserMinus size={15} />
					</button>
				{/if}
			</div>
		{/each}
		{#if editing && availableBots.length}
			<div class="botsep">Add a bot player</div>
			{#each availableBots as b (b.userId)}
				<div class="mem">
					<Avatar name={b.name} src={avatarUrl(b.userId, b.avatar)} size={30} />
					<span class="pname">{b.name}</span>
					<span class="rolepill" title="Bot player"><Bot size={11} /> {b.botKind || 'Bot'}</span>
					<span class="spacer"></span>
					<button class="addbtn" title="Add {b.name} to this league" disabled={botBusy === b.userId || mgmtBusy} onclick={() => addBot(b)}>
						<UserPlus size={15} /> Add
					</button>
				</div>
			{/each}
		{/if}
	</section>
	{/if}

	{#if view === 'board'}
	<section class="card board">

		<table class="lb">
			<thead>
				<tr>
					<th>#</th>
					<th>Player</th>
					{#if fcView}
						<th class="num ext" title="Correct group positions">Grp</th>
						<th class="num ext" title="Correct advancers (group stage)">Adv</th>
						{#each fcRounds as s (s.code)}
							<th class="num ext" title="Predicted teams that reached the {s.name}"
								>{s.code === 'FINAL' ? 'F' : s.code}</th
							>
						{/each}
						<th class="num ext" title="Champion predicted correctly">Win</th>
					{:else}
						<th class="num ext" title="Matches predicted">Pred</th>
						<th class="num ext" title="Forecast points">FC</th>
						<th class="num ext" title="Exact scores (tiebreak 1)">Exact</th>
						<th class="num ext" title="Correct winners (tiebreak 2)">Win</th>
						<th class="num ext" title="Goal-diff error (tiebreak 3, lower is better)">GD&Delta;</th>
					{/if}
					<th class="num pts">Pts</th>
				</tr>
			</thead>
			<tbody>
				{#each sorted as r, i (r.userId)}
					{@const f = r.forecast ?? {}}
					<tr
						class:lead={r.userId === auth.user?.id}
						class="main"
						class:open={openRow === r.userId}
						onclick={() =>
							(openRow = openRow === r.userId ? null : r.userId)}
					>
						<td class="rank"><span class="medal" class:g={i === 0} class:s={i === 1} class:b={i === 2}>{i + 1}</span></td>
						<td class="player">
							<div class="pwrap">
								<Avatar name={r.name} src={avatarUrl(r.userId, r.avatar)} size={28} />
								<span class="pname">{r.name}</span>
								{#if r.role === 'bot'}
									<span class="rolepill" title="Bot player"><Bot size={11} /> Bot</span>
								{:else if r.role === 'admin'}
									<span class="rolepill admin" title="Admin"><ShieldCheck size={11} /> Admin</span>
								{:else if r.role === 'owner'}
									<span class="rolepill owner" title="Platform owner"><Crown size={11} /> Owner</span>
								{/if}
								<ChevronDown size={14} class="rx" />
							</div>
						</td>
						{#if fcView}
							<td class="num ext digits">{f.groups ?? 0}</td>
							<td class="num ext digits">{f.advance ?? 0}</td>
							{#each fcRounds as s (s.code)}
								<td class="num ext digits">{f[s.code] ?? 0}</td>
							{/each}
							<td class="num ext digits">{f.champion ? '✓' : '–'}</td>
						{:else}
							<td class="num ext digits">{r.predicted}</td>
							<td class="num ext digits">{r.forecastPoints}</td>
							<td class="num ext digits">{r.exactScores}</td>
							<td class="num ext digits">{r.correctWinners}</td>
							<td class="num ext digits">{r.gdDeviation}</td>
						{/if}
						<td class="num pts digits">{r[tab]}</td>
					</tr>
					{#if openRow === r.userId}
						<tr class="detail">
							<td colspan="12">
								{#if fcView}
									<div class="stats">
										<span><i>Correct group positions</i><b>{f.groups ?? 0}</b></span>
										<span><i>Correct advancers</i><b>{f.advance ?? 0}</b></span>
										{#each fcRounds as s (s.code)}
											<span><i>Reached {s.name}</i><b>{f[s.code] ?? 0}</b></span>
										{/each}
										<span><i>Champion correct</i><b>{f.champion ? 'Yes' : 'No'}</b></span>
									</div>
								{:else}
									<div class="stats">
										<span><i>Matches predicted</i><b>{r.predicted}</b></span>
										<span><i>Tip points</i><b>{r.tipsPoints}</b></span>
										<span><i>Forecast points</i><b>{r.forecastPoints}</b></span>
										<span><i>Exact scores</i><b>{r.exactScores}</b></span>
										<span><i>Correct winners</i><b>{r.correctWinners}</b></span>
										<span><i>Goal-diff error</i><b>{r.gdDeviation}</b></span>
									</div>
								{/if}
								{#if boardTournament && boardTournament.forecastSpec?.mode !== 'none'}
									<a class="fcmore" href={`/forecast/${r.userId}?t=${boardSlug}`} onclick={(e) => e.stopPropagation()}>
										<Telescope size={14} /> {r.userId === auth.user?.id ? 'Your' : `${r.name}’s`} forecast <ChevronRight size={14} />
									</a>
								{/if}
							</td>
						</tr>
					{/if}
				{/each}

			</tbody>
		</table>
		<p class="muted small note">
			Points update automatically as results come in.
		</p>
	</section>
	{#if meRow && !meVisible}
		<button class="pin" onclick={() => document.querySelector('tr.main.lead')?.scrollIntoView({ block: 'center', behavior: 'smooth' })}>
			<span class="medal" class:g={meIndex === 0} class:s={meIndex === 1} class:b={meIndex === 2}>{meIndex + 1}</span>
			<Avatar name={meRow.name} src={avatarUrl(meRow.userId, meRow.avatar)} size={26} />
			<span class="pname">{meRow.name}</span>
			<span class="pill ok you">you</span>
			<span class="spacer"></span>
			<span class="digits ppts">{meRow[tab]}</span>
		</button>
	{/if}
	{/if}

	{#if view === 'members' && cfg}
		<details class="card legend">
			<summary>How points work</summary>

			<h4>Per match (your Tip) — max {cfg.match.tendency +
					cfg.match.exact +
					cfg.match.totalGoals +
					cfg.match.goalDiff} pt</h4>
			<ul class="leg">
				<li>
					<span>Correct result — group: 1 / X / 2; knockout: the team
						that advances</span><b>{cfg.match.tendency} pt</b>
				</li>
				<li><span>Exact score</span><b>+{cfg.match.exact} pt</b></li>
				<li><span>Correct total number of goals</span><b>+{cfg.match.totalGoals} pt</b></li>
				<li><span>Correct goal difference</span><b>+{cfg.match.goalDiff} pt</b></li>
			</ul>
			<p class="muted small">
				Knockout games have no draw — the result point is for the team
				that goes through. If a knockout game is decided in extra time,
				the score points use the after-extra-time score.
			</p>

			<h4>Tournament Forecast</h4>
			<ul class="leg">
				<li><span>Each team in its correct final group position</span><b>{cfg.forecast.groupPosition} pt</b></li>
				<li><span>Whole group ordered perfectly (bonus)</span><b>+{cfg.forecast.perfectGroupBonus} pt</b></li>
				<li>
					<span>Each team you predicted to advance (group top 2, or a
						best-third pick) that actually advances</span
					><b>{cfg.forecast.advance} pt</b>
				</li>
			</ul>
			<p class="muted small">
				Reaching a knockout round (per correctly predicted team):
			</p>
			<ul class="leg">
				{#each Object.entries(roundLabel) as [k, lbl] (k)}
					{#if cfg.forecast.round[k] != null}
						<li><span>{lbl}</span><b>{cfg.forecast.round[k]} pt</b></li>
					{/if}
				{/each}
			</ul>

			<h4>Tiebreakers (in order)</h4>
			<ol class="tiebreak">
				{#each cfg.tiebreakers as t (t)}
					<li>{tbLabel[t] ?? t}</li>
				{/each}
			</ol>
		</details>
	{/if}
{/if}

<ConfirmDialog
	open={removeTarget !== null}
	title="Remove member"
	message={removeTarget
		? `Remove ${removeTarget.name} from this league?`
		: ''}
	confirmLabel="Remove"
	cancelLabel="Cancel"
	danger
	busy={mgmtBusy}
	onconfirm={confirmRemove}
	oncancel={() => (removeTarget = null)}
/>

<style>
	.nameedit {
		font-size: 1.5rem;
		font-weight: 700;
		margin-top: 0.15rem;
	}
	.icon {
		width: auto;
		padding: 0.6rem;
	}
	/* Floating chat button, bottom-right, above the mobile tab bar. */
	/* Header "Share" toggle: reveals the invite/share card. Filled accent when
	   the card is open so the toggle state is obvious. */
	.vis {
		margin-bottom: 1rem;
	}
	.vistabs {
		margin: 0.5rem 0 0;
	}
	.hint {
		margin: 0.5rem 0 0;
	}
	.lockpill {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		margin-left: 0.35rem;
		padding: 0.05rem 0.4rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		font-size: 0.7rem;
		vertical-align: middle;
	}
	.regenbtn {
		width: auto;
		margin-top: 0.85rem;
	}
	.regwarn {
		margin-top: 0.85rem;
	}
	.btn.danger {
		width: auto;
		background: var(--danger);
		/* Intentional literal: white on danger red works on all themes. */
		color: #fff;
		border-color: transparent;
	}
	.regrow {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.4rem;
	}
	.regrow .btn.secondary {
		width: auto;
	}
	.rmbtn {
		display: inline-grid;
		place-items: center;
		flex: none;
		padding: 0.15rem;
		background: none;
		border: none;
		color: var(--muted);
		cursor: pointer;
	}
	.rmbtn:hover:not(:disabled) {
		color: var(--danger);
	}
	.rmbtn:disabled {
		opacity: 0.5;
		cursor: default;
	}
	.addbtn {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		margin-left: auto;
		flex: none;
		padding: 0.25rem 0.6rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		color: var(--accent);
		font-weight: 600;
		font-size: 0.8rem;
		cursor: pointer;
	}
	.addbtn:hover:not(:disabled) {
		border-color: color-mix(in srgb, var(--accent) 40%, var(--border));
	}
	.addbtn:disabled {
		opacity: 0.5;
		cursor: default;
	}
	.irow {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.share {
		margin-top: 0.85rem;
	}
	.ic {
		min-width: 0;
	}
	.small {
		font-size: 0.8rem;
	}
	.code {
		font-family: var(--font-mono);
		font-weight: 700;
		letter-spacing: 0.2em;
		font-size: 1.3rem;
	}
	.code.masked {
		color: var(--muted);
		letter-spacing: 0.15em;
	}
	.eye {
		width: auto;
		padding: 0.7rem;
	}
	.copy {
		width: auto;
	}
	.tabs {
		display: flex;
		gap: 0.4rem;
		margin-bottom: 0.75rem;
	}
	.tabs button {
		flex: 1;
		padding: 0.5rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		color: var(--muted);
		font-weight: 600;
	}
	.tabs button.active {
		color: var(--accent-fg);
		background: var(--accent);
		border-color: var(--accent);
	}
	.lb {
		width: 100%;
		border-collapse: collapse;
	}
	.lb th,
	.lb td {
		text-align: left;
		padding: 0.6rem 0.4rem;
		border-bottom: 1px solid var(--border);
	}
	.lb th {
		color: var(--muted);
		font-size: 0.8rem;
		font-weight: 600;
	}
	.num {
		text-align: right;
	}
	.rank {
		width: 2rem;
		color: var(--muted);
		font-family: var(--font-mono);
	}
	.subbar.withchips {
		padding-bottom: 0.6rem;
	}
	.chips {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem;
		margin-top: 0.55rem;
	}
	.chips .seg2 {
		margin-left: auto;
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
		max-width: 46vw;
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
	/* Total · Tips · Forecast: a small pill segment with a lifted thumb. */
	.seg2 {
		display: inline-flex;
		gap: 2px;
		padding: 3px;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface-2);
	}
	.seg2 button {
		padding: 0.35rem 0.7rem;
		border: none;
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--muted);
		font: inherit;
		font-weight: 700;
		font-size: 0.76rem;
		cursor: pointer;
	}
	.seg2 button.on {
		background: var(--surface);
		color: var(--text);
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.4);
	}
	.fcmore {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		margin-top: 0.5rem;
		font-size: 0.82rem;
		font-weight: 600;
	}
	.card.board {
		padding-top: 0.4rem;
	}
	.utab.chat {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		color: var(--muted);
	}
	.badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 18px;
		height: 18px;
		padding: 0 5px;
		border-radius: var(--radius-pill);
		background: var(--accent);
		color: var(--accent-fg);
		font-size: 0.66rem;
		font-weight: 800;
	}
	.card.manage {
		padding: 0.75rem 0.9rem;
	}
	.seasonsline {
		margin-bottom: 0.4rem;
	}
	.chipset {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		margin: 0.5rem 0;
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
	.mrow {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.mtxt {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		flex: 1;
		min-width: 0;
	}
	.btn.slim {
		width: auto;
		padding: 0.5rem 0.9rem;
		font-size: 0.78rem;
		gap: 0.3rem;
	}
	.card.members {
		padding: 0;
	}
	.mem {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.55rem 0.9rem;
		border-bottom: 1px solid var(--border);
		font-size: 0.92rem;
	}
	.mem:last-child {
		border-bottom: none;
	}
	.mem .pname {
		font-weight: 600;
	}
	.botsep {
		padding: 0.6rem 0.9rem 0.3rem;
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
		border-bottom: 1px solid var(--border);
	}
	.medal {
		width: 22px;
		height: 22px;
		border-radius: 50%;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-family: var(--font-mono);
		font-weight: 800;
		font-size: 0.72rem;
		color: var(--muted);
	}
	.medal.g,
	.medal.s,
	.medal.b {
		color: #1c0e00;
	}
	.medal.g {
		background: var(--gold);
	}
	.medal.s {
		background: #c9ccd3;
	}
	.medal.b {
		background: #c98a52;
	}
	.chip.inv {
		color: var(--accent);
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
	}
	.pin {
		position: fixed;
		left: 1rem;
		right: 1rem;
		bottom: calc(var(--nav-h) + 0.75rem);
		z-index: 30;
		max-width: calc(var(--maxw) - 2rem);
		margin: 0 auto;
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.55rem 0.9rem;
		border: 1px solid color-mix(in srgb, var(--accent) 50%, var(--border));
		border-radius: 14px;
		background: color-mix(in srgb, var(--accent) 8%, var(--surface));
		color: var(--text);
		font: inherit;
		font-weight: 600;
		box-shadow: var(--shadow-pop);
		cursor: pointer;
	}
	.pin .pill.you {
		font-size: 0.56rem;
		padding: 0.1rem 0.4rem;
	}
	.ppts {
		font-size: 1rem;
	}
	@media (min-width: 900px) {
		.pin {
			bottom: 1rem;
		}
	}
	tr.lead td {
		background: color-mix(in srgb, var(--accent) 9%, transparent);
	}
	tr.lead .rank {
		color: var(--accent);
		font-weight: 800;
	}
	.lb th.num,
	.lb td.num {
		text-align: right;
	}

	/* Pts is the focus — set it apart from the stat columns. */
	.lb th.pts,
	.lb td.pts {
		padding-left: 1.15rem;
		border-left: 1px solid var(--border);
		font-size: 1.02rem;
	}
	.lb th.pts {
		font-size: 0.8rem;
	}

	/* Extra tiebreaker columns: desktop only. */
	.ext {
		display: none;
	}
	.player {
		width: 100%;
		/* Auto table layout would grow this cell to fit a long name and push
		   the points columns off-screen; capping max-width makes it absorb
		   the leftover space instead, so the name truncates. */
		max-width: 0;
	}
	.pwrap {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.pname {
		/* Flex items refuse to shrink below their content without this. */
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.rolepill {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		flex: none;
		padding: 0.05rem 0.4rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		font-size: 0.7rem;
		color: var(--muted);
	}
	.rolepill.admin {
		color: var(--accent);
		border-color: color-mix(in srgb, var(--accent) 40%, var(--border));
	}
	.rolepill.owner {
		color: var(--warning);
		border-color: color-mix(in srgb, var(--warning) 40%, var(--border));
	}
	.fclink {
		display: inline-grid;
		place-items: center;
		color: var(--muted);
		flex: none;
	}
	.fclink:hover {
		color: var(--accent);
	}
	:global(.lb .rx) {
		color: var(--muted);
		transition: transform 0.15s ease;
		margin-left: auto;
	}
	tr.main.open :global(.rx) {
		transform: rotate(180deg);
	}
	tr.main {
		cursor: pointer;
	}
	.detail td {
		padding: 0 0.4rem 0.7rem;
	}
	.stats {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.4rem 1rem;
	}
	.stats span {
		display: flex;
		justify-content: space-between;
		gap: 0.6rem;
		padding: 0.35rem 0;
		border-bottom: 1px solid var(--border);
	}
	.stats i {
		color: var(--muted);
		font-style: normal;
		font-size: 0.85rem;
	}
	.stats b {
		font-family: var(--font-mono);
	}

	@media (min-width: 760px) {
		.ext {
			display: table-cell;
		}
		:global(.lb .rx) {
			display: none;
		}
		tr.main {
			cursor: default;
		}
		.detail {
			display: none;
		}
	}
	.note {
		margin: 0.75rem 0 0;
	}
	.legend summary {
		cursor: pointer;
		font-weight: 700;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		font-size: 0.85rem;
		color: var(--accent);
	}
	.legend h4 {
		margin: 1rem 0 0.5rem;
		font-size: 0.95rem;
	}
	.legend .small {
		margin: 0.4rem 0 0;
	}
	ul.leg {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	ul.leg li {
		display: flex;
		align-items: baseline;
		gap: 0.75rem;
		padding: 0.4rem 0;
		border-bottom: 1px solid var(--border);
	}
	ul.leg li span {
		flex: 1;
	}
	ul.leg li b {
		font-family: var(--font-mono);
		color: var(--accent);
		white-space: nowrap;
	}
	ol.tiebreak {
		margin: 0.5rem 0 0;
		padding-left: 1.3rem;
		line-height: 1.8;
	}
	ol.tiebreak li {
		padding-left: 0.3rem;
	}
</style>
