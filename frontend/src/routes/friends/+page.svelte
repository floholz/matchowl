<script lang="ts">
	import { api, type LeagueSummary } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';
	import { pageChrome } from '$lib/shell.svelte';
	import { Globe, ChevronRight, MessageSquare } from '@lucide/svelte';

	pageChrome(() => ({ title: 'Friends' }));

	type Rank = { rank: number; total: number; points: number; leader: string; leaderPoints: number };

	let leagues = $state<LeagueSummary[]>([]);
	let ranks = $state<Record<string, Rank | null>>({});
	let unread = $state<Record<string, number>>({});
	let loaded = $state(false);
	let newName = $state('');
	let joinCode = $state('');
	let error = $state('');
	let busy = $state(false);

	const isGlobal = (l: LeagueSummary) => l.inviteCode === 'GLOBAL';

	// Global is the everyone-league — always pin it to the top. Other leagues
	// keep the server order (sort is stable).
	let ordered = $derived(
		[...leagues].sort((a, b) => Number(isGlobal(b)) - Number(isGlobal(a)))
	);

	async function load() {
		try {
			leagues = (await api.myLeagues()).leagues;
			leagues.forEach((l) => loadRank(l.id));
			api
				.chatUnread()
				.then((r) => (unread = r.unread))
				.catch(() => {});
		} catch {
			/* ignore */
		} finally {
			loaded = true;
		}
	}
	$effect(() => {
		load();
	});

	// My placement in a league: rank by total points (mirrors the Overall tab).
	// rows.length is the number of ranked players, i.e. the league's size.
	function loadRank(id: string) {
		api
			.leaderboard(id)
			.then(({ rows }) => {
				const i = rows.findIndex((r) => r.userId === auth.user?.id);
				ranks[id] =
					i >= 0
						? {
								rank: i + 1,
								total: rows.length,
								points: rows[i].total,
								leader: rows[0]?.name ?? '',
								leaderPoints: rows[0]?.total ?? 0
							}
						: null;
			})
			.catch(() => (ranks[id] = null));
	}
	let privateLeagues = $derived(ordered.filter((l) => !isGlobal(l)));
	let global = $derived(ordered.find(isGlobal));

	async function create(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			const r = await api.createLeague(newName);
			newName = '';
			goto(`/friends/${r.id}`);
		} catch {
			error = 'Could not create league.';
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
			goto(`/friends/${r.id}`);
		} catch {
			error = 'Invalid invite code.';
		} finally {
			busy = false;
		}
	}
</script>

{#if !loaded}
	<p class="muted pad">Loading…</p>
{:else if leagues.length === 0}
	<div class="card quiet muted">No leagues yet — create one or join with a code below.</div>
{:else}
	{#each privateLeagues as l (l.id)}
		{@const r = ranks[l.id]}
		<a class="card league" href={`/friends/${l.id}`}>
			<span class="lhead">
				<b class="lname">{l.name}</b>
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
	{#if global}
		{@const r = ranks[global.id]}
		<a class="card grow" href={`/friends/${global.id}`}>
			<span class="gico"><Globe size={18} /></span>
			<span class="gtxt"><b>Global</b><span class="muted">everyone on Matchowl</span></span>
			<span class="spacer"></span>
			<span class="digits">{r ? `#${r.rank}` : '–'}<small class="muted">/{r?.total ?? global.members}</small></span>
			<ChevronRight size={16} class="cv" />
		</a>
	{/if}
{/if}

<h2 class="sec">Create or join</h2>
<section class="card actions">
	<form class="action" onsubmit={create}>
		<input
			class="input"
			placeholder="New league name"
			bind:value={newName}
			required
		/>
		<button class="btn" disabled={busy || !newName.trim()}>Create</button>
	</form>
	<div class="orsep"><span>or join one</span></div>
	<form class="action" onsubmit={join}>
		<input
			class="input code"
			placeholder="INVITE CODE"
			bind:value={joinCode}
			required
		/>
		<button class="btn secondary" disabled={busy || !joinCode.trim()}>Join</button>
	</form>
</section>

{#if error}<p class="error">{error}</p>{/if}

<style>
	.sec {
		font-size: 1.05rem;
		margin: 1.2rem 0 0.7rem;
	}
	.pad {
		padding: 0.4rem 0.2rem;
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
	.lname {
		font-size: 1.05rem;
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
	.action .btn {
		width: auto;
		padding: 0.8rem 1.1rem;
	}
	.action .input {
		flex: 1;
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
</style>
