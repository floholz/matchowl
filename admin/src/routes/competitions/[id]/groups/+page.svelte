<!-- Groups of one tournament: drag teams between groups, add or remove
     groups, save — every group-stage match follows its teams and the
     structure's group size follows the largest group. -->
<script lang="ts">
	import { page } from '$app/stores';
	import { api, type GroupsView, type GroupTeam, type AdminTournament } from '$lib/api';
	import { pb } from '$lib/pb';
	import GroupEditor from '$lib/components/admin/GroupEditor.svelte';
	import { ChevronLeft, Save, RotateCcw, TriangleAlert } from '@lucide/svelte';

	let id = $derived($page.params.id ?? '');
	let tournament = $state<AdminTournament | null>(null);
	let view = $state<GroupsView | null>(null);
	let groups = $state<{ letter: string; teams: GroupTeam[] }[]>([]);
	let unassigned = $state<GroupTeam[]>([]);
	let error = $state('');
	let flash = $state('');
	let busy = $state(false);
	let loaded = $state(false);

	function take(v: GroupsView) {
		view = v;
		groups = v.groups.map((g) => ({ letter: g.letter, teams: [...g.teams] }));
		unassigned = [...v.unassigned];
	}
	async function load() {
		error = '';
		try {
			const [g, ts] = await Promise.all([api.adminGroups(id), api.adminTournaments()]);
			take(g);
			tournament = ts.tournaments.find((t) => t.id === id) ?? null;
		} catch (e) {
			error = msg(e);
		} finally {
			loaded = true;
		}
	}
	$effect(() => {
		if (id) load();
	});
	async function save() {
		busy = true;
		error = '';
		flash = '';
		try {
			const v = await api.adminSaveGroups(
				id,
				groups.map((g) => ({ letter: g.letter, teams: g.teams.map((t) => t.id) }))
			);
			take(v);
			flash = `Saved · ${v.groups.length} groups · the ${v.groupMatches} group matches follow their teams.`;
		} catch (e) {
			error = msg(e);
		} finally {
			busy = false;
		}
	}
	function msg(err: unknown): string {
		const r = (err as { response?: { error?: string; message?: string } })?.response;
		return r?.error || r?.message || (err instanceof Error ? err.message : String(err));
	}
	const crest = (t: GroupTeam) => (t.logo ? pb.files.getURL({ collectionName: 'teams', id: t.id }, t.logo) : '');
	let largest = $derived(groups.reduce((n, g) => Math.max(n, g.teams.length), 0));
	let games = $derived(view?.structure?.gamesPerTeam ?? 0);
	let dirty = $derived(
		!!view &&
			JSON.stringify(groups.map((g) => ({ l: g.letter, t: g.teams.map((x) => x.id).sort() }))) !==
				JSON.stringify(view.groups.map((g) => ({ l: g.letter, t: g.teams.map((x) => x.id).sort() })))
	);
</script>

<div class="head">
	<div>
		<p class="kicker"><a href="/competitions" class="crumb"><ChevronLeft size={14} /> Competitions</a></p>
		<h1>Groups{#if tournament} · {tournament.name}{/if}</h1>
		{#if view?.structure}
			<p class="muted small">
				Group size {view.structure.groupSize} · {view.structure.gamesPerTeam} games per team · {view.groupMatches} group matches.
				{#if largest && largest !== view.structure.groupSize}Saving sets the group size to {largest}.{/if}
			</p>
		{/if}
	</div>
	<div class="hbtns">
		<button class="btn secondary" disabled={!dirty || busy} onclick={() => view && take(view)}><RotateCcw size={16} /> Reset</button>
		<button class="btn" disabled={!dirty || busy} onclick={save}><Save size={16} /> {busy ? 'Saving…' : 'Save groups'}</button>
	</div>
</div>

{#if error}<p class="error">{error}</p>{/if}
{#if flash}<p class="flash">{flash}</p>{/if}
{#if games && largest - 1 > games}
	<p class="warn"><TriangleAlert size={14} /> A group of {largest} cannot be a round robin over {games} games per team — probably merged groups. Split them up.</p>
{/if}

{#if !loaded}
	<p class="muted">Loading…</p>
{:else if view}
	{#if !view.structure?.hasGroups}
		<p class="muted small">This season's structure has no group stage; groups here only matter for a tournament that has one.</p>
	{/if}
	<GroupEditor bind:groups bind:unassigned {crest} gamesPerTeam={games} />
{/if}

<style>
	.head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
		margin-bottom: 1rem;
	}
	.crumb {
		display: inline-flex;
		align-items: center;
		gap: 0.1rem;
	}
	.hbtns {
		display: flex;
		gap: 0.5rem;
	}
	.hbtns .btn {
		width: auto;
	}
	.small {
		font-size: 0.85rem;
		margin: 0.3rem 0 0;
	}
	.flash {
		color: var(--success);
		font-size: 0.9rem;
	}
	.warn {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		color: var(--warning);
		font-size: 0.9rem;
	}
</style>
