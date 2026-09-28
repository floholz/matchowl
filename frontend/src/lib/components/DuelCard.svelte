<!-- One head-to-head pool on Home: the duel this matchday (or the last
     one), how it stands, who is next and what is left to do — the pool at
     a glance, the biggest thing on the page. "All duels" unfolds the rest
     of the matchday. Data comes from the shared h2h store (GET /api/h2h/me). -->
<script lang="ts">
	import type { H2HMePool, H2HDuelState, H2HPerson } from '$lib/api';
	import { pb } from '$lib/pb';
	import { auth } from '$lib/auth.svelte';
	import Avatar from './Avatar.svelte';
	import { Ghost as GhostIcon, ChevronRight, ChevronDown, Swords } from '@lucide/svelte';

	let { pool }: { pool: H2HMePool } = $props();

	const avatarUrl = (p: H2HPerson | null) =>
		p?.avatar ? pb.files.getURL({ id: p.userId, collectionName: 'users' }, p.avatar) : null;
	let myName = $derived(auth.user?.name || 'You');
	let myAvatar = $derived(auth.user?.avatarUrl ?? null);
	const day = (iso: string) => (iso ? new Date(iso).toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' }) : '');
	const rivalName = (d: H2HDuelState) => (d.ghost ? 'the Ghost' : (d.rival?.name ?? ''));

	/** The duel to headline: the open or last closed round; before the
	 *  first one, the next round as a preview. */
	let cur = $derived(pool.current);
	let next = $derived(pool.next);
	let open = $derived(cur?.status === 'open');
	let verdict = $derived.by(() => {
		if (!cur || !cur.paired) return '';
		const p = cur.pts ?? 0;
		if (open) return p === 3 ? 'You are ahead' : p === 1 ? 'Level' : 'You are behind';
		return p === 3 ? 'You won this matchday · +3' : p === 1 ? 'You drew this matchday · +1' : 'You lost this matchday';
	});
	/** What is left to do in the round that still takes tips and picks. */
	function todo(d: H2HDuelState): string {
		const parts: string[] = [];
		if (d.untipped) parts.push(`${d.untipped} to tip`);
		if ((d.saveCallsLeft ?? 0) > 0) parts.push(`Save Call ${d.saveCallsLeft} left`);
		if (d.paired && !d.ghost && d.banPlaced === false) parts.push('ban open');
		return parts.join(' · ');
	}
	let todoLine = $derived(open && cur ? todo(cur) : '');
	let nextLine = $derived.by(() => {
		if (!next) return '';
		const who = next.paired ? `vs ${rivalName(next)}` : 'not paired';
		return `${next.name} · ${who} · ${day(next.firstKickoff)}`;
	});
	let nextTodo = $derived(next ? todo(next) : '');
	let showAll = $state(false);
	let others = $derived(cur?.pairs ?? []);
</script>

<div class="card dc" class:open>
	<a class="head" href={`/pools/${pool.poolId}`}>
		<span class="ic"><Swords size={16} /></span>
		<span class="htxt">
			<b>{pool.name}</b>
			{#if cur}
				<span class="muted">{cur.name} · {open ? `${cur.counted} of ${cur.matches} in` : 'final'}</span>
			{:else if next}
				<span class="muted">Kicks off with {next.name}</span>
			{/if}
		</span>
		<ChevronRight size={16} class="cv" />
	</a>

	{#if cur && cur.paired}
		<div class="vs">
			<span class="side">
				<Avatar name={myName} src={myAvatar} size={40} />
				<span class="sname">You</span>
			</span>
			<span class="score digits" class:win={cur.pts === 3} class:draw={cur.pts === 1} class:loss={cur.pts === 0 && ((cur.mine ?? 0) > 0 || (cur.theirs ?? 0) > 0)}>
				{cur.mine ?? 0}<i>–</i>{cur.theirs ?? 0}
			</span>
			<span class="side">
				{#if cur.rival}
					<Avatar name={cur.rival.name} src={avatarUrl(cur.rival)} size={40} />
					<span class="sname">{cur.rival.name}</span>
				{:else}
					<span class="ghost"><GhostIcon size={20} /></span>
					<span class="sname">The Ghost</span>
				{/if}
			</span>
		</div>
		<p class="line">
			<span>{verdict}</span>
			{#if todoLine}<span class="todo">{todoLine}</span>{/if}
		</p>
	{:else if cur}
		<p class="line muted">You are not paired this matchday.</p>
	{:else if next}
		<div class="vs">
			<span class="side">
				<Avatar name={myName} src={myAvatar} size={40} />
				<span class="sname">You</span>
			</span>
			<span class="score digits muted"><i>vs</i></span>
			<span class="side">
				{#if next.rival}
					<Avatar name={next.rival.name} src={avatarUrl(next.rival)} size={40} />
					<span class="sname">{next.rival.name}</span>
				{:else if next.ghost}
					<span class="ghost"><GhostIcon size={20} /></span>
					<span class="sname">The Ghost</span>
				{:else}
					<span class="ghost"><GhostIcon size={20} /></span>
					<span class="sname">Nobody yet</span>
				{/if}
			</span>
		</div>
	{/if}

	{#if next && (cur || nextTodo)}
		<a class="next" href={pool.competition ? `/competitions/${pool.competition}?s=${encodeURIComponent(pool.seasonSlug)}&tab=matches&round=${encodeURIComponent(next.key)}` : `/pools/${pool.poolId}`}>
			<span class="nlbl">Next</span>
			<span class="ntxt">
				<b>{nextLine}</b>
				{#if nextTodo}<span class="todo">{nextTodo}</span>{/if}
			</span>
			<ChevronRight size={14} class="cv" />
		</a>
	{/if}

	{#if others.length > 1}
		<button class="all" onclick={() => (showAll = !showAll)} aria-expanded={showAll}>
			<ChevronDown size={14} class={showAll ? 'flip' : ''} />
			{showAll ? 'Hide' : 'All duels'} · {others.length}
		</button>
		{#if showAll}
			<ul class="pairs">
				{#each others as p (p.a.userId)}
					<li>
						<span class="pn" class:won={p.ptsA === 3}>{p.a.name}</span>
						<span class="ps digits"><b class:hi={p.ptsA === 3}>{p.scoreA}</b><i>–</i><b class:hi={p.ptsB === 3}>{p.scoreB}</b></span>
						<span class="pn r" class:won={p.ptsB === 3}>{p.b ? p.b.name : 'Ghost'}</span>
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
</div>

<style>
	.dc {
		padding: 0.75rem 0.9rem 0.6rem;
		border-color: color-mix(in srgb, var(--accent) 35%, var(--border));
	}
	.head {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		color: var(--text);
	}
	.ic {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		flex: none;
		border-radius: var(--radius-sm);
		color: var(--accent);
		background: color-mix(in srgb, var(--accent) 14%, transparent);
	}
	.htxt {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
		font-size: 0.92rem;
	}
	.htxt .muted {
		font-size: 0.78rem;
	}
	.head :global(.cv),
	.next :global(.cv) {
		margin-left: auto;
		color: var(--muted);
		flex: none;
	}
	.vs {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 0.6rem;
		margin: 0.9rem 0 0.4rem;
	}
	.side {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.3rem;
		min-width: 0;
	}
	.sname {
		font-size: 0.8rem;
		font-weight: 700;
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.score {
		font-size: 2.1rem;
		font-weight: 800;
		letter-spacing: 0.02em;
		display: inline-flex;
		align-items: baseline;
		gap: 0.15rem;
	}
	.score i {
		font-style: normal;
		color: var(--muted);
		font-size: 1.2rem;
	}
	.score.win {
		color: var(--accent);
	}
	.score.loss {
		color: var(--muted);
	}
	.ghost {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		border-radius: 50%;
		border: 1px dashed var(--border);
		color: var(--muted);
	}
	.line {
		margin: 0;
		text-align: center;
		font-size: 0.85rem;
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
	}
	.todo {
		color: var(--accent);
		font-weight: 700;
		font-size: 0.8rem;
	}
	.next {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		margin: 0.7rem -0.9rem 0;
		padding: 0.55rem 0.9rem;
		border-top: 1px solid var(--border);
		color: var(--text);
		font-size: 0.82rem;
	}
	.nlbl {
		flex: none;
		font-size: 0.68rem;
		font-weight: 800;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.ntxt {
		display: flex;
		flex-direction: column;
		gap: 0.05rem;
		min-width: 0;
	}
	.ntxt b {
		font-weight: 600;
	}
	.all {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		margin: 0.4rem 0 0;
		padding: 0.25rem 0;
		border: none;
		background: transparent;
		color: var(--muted);
		font: inherit;
		font-size: 0.78rem;
		font-weight: 700;
		cursor: pointer;
	}
	.all :global(.flip) {
		transform: rotate(180deg);
	}
	.pairs {
		list-style: none;
		margin: 0.2rem 0 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.82rem;
	}
	.pairs li {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 0.5rem;
	}
	.pn {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: var(--muted);
	}
	.pn.r {
		text-align: right;
	}
	.pn.won {
		color: var(--text);
		font-weight: 700;
	}
	.ps {
		font-weight: 800;
	}
	.ps i {
		font-style: normal;
		color: var(--muted);
		padding: 0 0.1rem;
	}
	.ps .hi {
		color: var(--accent);
	}
</style>
