<!-- The group editor: one column per group, team chips you drag between
     them (or into the "not in a group" bin), a new-group button, a letter
     you can rename, and empty groups you can remove. Pure: it edits the
     bound `groups` / `unassigned` and never talks to the API — the groups
     page saves a tournament's, the import wizard keeps the proposal's. -->
<script lang="ts" generics="T extends { id: string; name: string; fifaCode?: string }">
	import { Plus, X, GripVertical, TriangleAlert } from '@lucide/svelte';

	let {
		groups = $bindable(),
		unassigned = $bindable(),
		crest = () => '',
		gamesPerTeam = 0
	}: {
		groups: { letter: string; teams: T[] }[];
		unassigned: T[];
		/** Crest URL for a team, '' for none. */
		crest?: (t: T) => string;
		/** For the size warning: a round-robin group of n needs n-1 games. */
		gamesPerTeam?: number;
	} = $props();

	let dragging = $state<string | null>(null); // team id in flight
	let over = $state<string | null>(null); // column key under the pointer
	// Tap to move (touch screens, and anyone who prefers it): tap a team to
	// pick it up, tap a group (or the bin) to drop it there.
	let picked = $state<string | null>(null);
	function tapTeam(id: string) {
		picked = picked === id ? null : id;
	}
	function tapZone(to: number) {
		if (picked === null) return;
		move(picked, to);
		picked = null;
	}
	let newLetter = $state('');

	const key = (i: number) => `g${i}`;
	function find(id: string): { from: number; t: T } | null {
		for (let i = 0; i < groups.length; i++) {
			const t = groups[i].teams.find((x) => x.id === id);
			if (t) return { from: i, t };
		}
		const t = unassigned.find((x) => x.id === id);
		return t ? { from: -1, t } : null;
	}
	function move(id: string, to: number) {
		const f = find(id);
		if (!f || f.from === to) return;
		if (f.from >= 0) groups[f.from].teams = groups[f.from].teams.filter((x) => x.id !== id);
		else unassigned = unassigned.filter((x) => x.id !== id);
		if (to >= 0) groups[to].teams = [...groups[to].teams, f.t].sort((a, b) => a.name.localeCompare(b.name));
		else unassigned = [...unassigned, f.t].sort((a, b) => a.name.localeCompare(b.name));
	}
	function onDragStart(e: DragEvent, id: string) {
		dragging = id;
		e.dataTransfer?.setData('text/plain', id);
		if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
	}
	function onDrop(e: DragEvent, to: number) {
		e.preventDefault();
		const id = e.dataTransfer?.getData('text/plain') || dragging;
		if (id) move(id, to);
		dragging = null;
		over = null;
	}
	function addGroup() {
		const l = newLetter.trim().toUpperCase();
		if (!l || l.length > 2 || groups.some((g) => g.letter === l)) return;
		groups = [...groups, { letter: l, teams: [] }].sort((a, b) => a.letter.localeCompare(b.letter));
		newLetter = '';
	}
	function removeGroup(i: number) {
		if (groups[i].teams.length) return;
		groups = groups.filter((_, j) => j !== i);
	}
	function rename(i: number, v: string) {
		groups[i].letter = v.trim().toUpperCase().slice(0, 2);
	}
	/** Next free letter as the default for a new group. */
	let suggested = $derived.by(() => {
		for (const l of 'ABCDEFGHIJKLMNOPQRSTUVWXYZ') if (!groups.some((g) => g.letter === l)) return l;
		return '';
	});
	const tooBig = (n: number) => gamesPerTeam > 0 && n - 1 > gamesPerTeam;
</script>

<div class="ge">
	<div class="cols">
		{#each groups as g, i (key(i))}
			<!-- svelte-ignore a11y_no_noninteractive_element_interactions, a11y_click_events_have_key_events -->
			<section
				class="col"
				role="group"
				aria-label="Group {g.letter}"
				class:over={over === key(i)}
				class:warn={tooBig(g.teams.length)}
				class:hasPick={picked !== null}
				ondragover={(e) => { e.preventDefault(); over = key(i); }}
				ondragleave={() => (over === key(i) ? (over = null) : null)}
				ondrop={(e) => onDrop(e, i)}
				onclick={() => tapZone(i)}
			>
				<header>
					<input class="letter" value={g.letter} maxlength="2" aria-label="Group letter" onclick={(e) => e.stopPropagation()} onchange={(e) => rename(i, (e.currentTarget as HTMLInputElement).value)} />
					<span class="n muted">{g.teams.length}</span>
					{#if tooBig(g.teams.length)}<span class="wico" title="More teams than a round robin over {gamesPerTeam} games allows"><TriangleAlert size={14} /></span>{/if}
					{#if !g.teams.length}
						<button class="rm" title="Remove empty group" aria-label="Remove group" onclick={(e) => { e.stopPropagation(); removeGroup(i); }}><X size={14} /></button>
					{/if}
				</header>
				<ul>
					{#each g.teams as t (t.id)}
						<li><button type="button" class="chip" class:lift={dragging === t.id} class:picked={picked === t.id} aria-pressed={picked === t.id} draggable="true" ondragstart={(e) => onDragStart(e, t.id)} ondragend={() => { dragging = null; over = null; }} onclick={(e) => { e.stopPropagation(); tapTeam(t.id); }}>
							<GripVertical size={12} class="grip" />
							{#if crest(t)}<img src={crest(t)} alt="" />{:else if t.fifaCode}<span class="code">{t.fifaCode}</span>{/if}
							<span class="nm">{t.name}</span>
						</button></li>
					{/each}
					{#if !g.teams.length}<li class="empty muted">Drop teams here</li>{/if}
				</ul>
			</section>
		{/each}
		<section class="col new">
			<header>
				<input class="letter" placeholder={suggested} bind:value={newLetter} maxlength="2" aria-label="New group letter" onkeydown={(e) => e.key === 'Enter' && addGroup()} />
				<button class="add" onclick={() => { if (!newLetter) newLetter = suggested; addGroup(); }} aria-label="Add group"><Plus size={14} /> Group</button>
			</header>
			<p class="muted small">Type a letter (or take the suggested one) and add a group, then drag teams in — or tap a team, then tap a group.</p>
		</section>
	</div>
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions, a11y_click_events_have_key_events -->
	<section
		class="bin"
		role="group"
		aria-label="Not in a group"
		class:over={over === 'bin'}
		class:hasPick={picked !== null}
		ondragover={(e) => { e.preventDefault(); over = 'bin'; }}
		ondragleave={() => (over === 'bin' ? (over = null) : null)}
		ondrop={(e) => onDrop(e, -1)}
		onclick={() => tapZone(-1)}
	>
		<header><b>Not in a group</b><span class="n muted">{unassigned.length}</span></header>
		<ul class="row">
			{#each unassigned as t (t.id)}
				<li><button type="button" class="chip" class:lift={dragging === t.id} class:picked={picked === t.id} aria-pressed={picked === t.id} draggable="true" ondragstart={(e) => onDragStart(e, t.id)} ondragend={() => { dragging = null; over = null; }} onclick={(e) => { e.stopPropagation(); tapTeam(t.id); }}>
					<GripVertical size={12} class="grip" />
					{#if crest(t)}<img src={crest(t)} alt="" />{:else if t.fifaCode}<span class="code">{t.fifaCode}</span>{/if}
					<span class="nm">{t.name}</span>
				</button></li>
			{/each}
			{#if !unassigned.length}<li class="empty muted">Every team is in a group. Drop one here to take it out.</li>{/if}
		</ul>
	</section>
</div>

<style>
	.ge {
		display: flex;
		flex-direction: column;
		gap: 0.8rem;
	}
	.cols {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 0.6rem;
	}
	.col,
	.bin {
		border: 1px dashed var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		padding: 0.5rem;
		min-height: 120px;
		transition: border-color 0.12s ease, background 0.12s ease;
	}
	.col.over,
	.bin.over {
		border-color: var(--accent);
		background: color-mix(in srgb, var(--accent) 10%, var(--surface-2));
	}
	.col.warn {
		border-color: var(--warning);
	}
	.col.new {
		border-style: dotted;
		background: transparent;
	}
	header {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		margin-bottom: 0.4rem;
	}
	.letter {
		width: 3rem;
		padding: 0.25rem 0.4rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
		color: var(--text);
		font: inherit;
		font-weight: 800;
		text-align: center;
		text-transform: uppercase;
	}
	.n {
		font-size: 0.75rem;
	}
	.wico {
		color: var(--warning);
		display: inline-flex;
	}
	.rm,
	.add {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.5rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--muted);
		font: inherit;
		font-size: 0.75rem;
		font-weight: 700;
		cursor: pointer;
	}
	.add {
		color: var(--accent);
		border-color: var(--accent);
	}
	ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
	}
	ul.row {
		flex-direction: row;
		flex-wrap: wrap;
	}
	.chip {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		width: 100%;
		padding: 0.35rem 0.5rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
		color: var(--text);
		font: inherit;
		font-size: 0.85rem;
		text-align: left;
		cursor: grab;
		user-select: none;
	}
	ul.row .chip {
		width: auto;
	}
	.chip.lift {
		opacity: 0.45;
	}
	.chip.picked {
		border-color: var(--accent);
		box-shadow: var(--glow);
	}
	.col.hasPick,
	.bin.hasPick {
		cursor: copy;
	}
	.chip :global(.grip) {
		color: var(--muted);
		flex: none;
	}
	.chip img {
		width: 18px;
		height: 18px;
		object-fit: contain;
	}
	.code {
		font-family: var(--font-mono);
		font-size: 0.7rem;
		font-weight: 700;
		color: var(--muted);
		width: 2rem;
	}
	.nm {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.empty {
		font-size: 0.78rem;
		padding: 0.4rem 0.2rem;
	}
	.small {
		font-size: 0.78rem;
		margin: 0;
	}
</style>
