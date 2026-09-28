<!-- The teams registry: one row per club or nation across every season it
     plays in (rows are per tournament in the database), with the seasons,
     the provider ids and a fix-everywhere editor for name, code and flag. -->
<script lang="ts">
	import { api, type RegistryTeam } from '$lib/api';
	import { pb } from '$lib/pb';
	import { appConfig } from '$lib/appconfig.svelte';
	import { Search, ChevronDown, Check, TriangleAlert } from '@lucide/svelte';

	let teams = $state<RegistryTeam[]>([]);
	let loaded = $state(false);
	let error = $state('');
	let q = $state('');
	let onlyMulti = $state(false);
	let onlyUnlinked = $state(false);
	let openKey = $state('');

	async function load() {
		error = '';
		try {
			teams = (await api.adminTeams()).teams;
		} catch (e) {
			error = msg(e);
		} finally {
			loaded = true;
		}
	}
	$effect(() => {
		load();
	});
	function msg(err: unknown): string {
		const r = (err as { response?: { error?: string; message?: string } })?.response;
		return r?.error || r?.message || (err instanceof Error ? err.message : String(err));
	}
	const crest = (t: RegistryTeam) => {
		if (!t.logo) return '';
		const [id, file] = t.logo.split('/');
		return pb.files.getURL({ collectionName: 'teams', id }, file);
	};
	let shown = $derived.by(() => {
		const s = q.trim().toLowerCase();
		return teams.filter(
			(t) =>
				(!s || t.name.toLowerCase().includes(s) || t.fifaCode.toLowerCase().includes(s) || t.entries.some((e) => e.season.toLowerCase().includes(s))) &&
				(!onlyMulti || t.entries.length > 1) &&
				(!onlyUnlinked || !t.providerId)
		);
	});
	let unlinked = $derived(teams.filter((t) => !t.providerId).length);

	// Editor: one team at a time; applies to every row of the team by default.
	let edit = $state<{ teamId: string; name: string; fifaCode: string; iso2: string; all: boolean } | null>(null);
	let saving = $state(false);
	let flash = $state('');
	function startEdit(t: RegistryTeam) {
		edit = { teamId: t.entries[0].teamId, name: t.name, fifaCode: t.fifaCode, iso2: t.iso2, all: true };
		flash = '';
		error = '';
	}
	async function save() {
		if (!edit) return;
		saving = true;
		error = '';
		try {
			const r = await api.adminTeamUpdate(edit.teamId, { name: edit.name, fifaCode: edit.fifaCode, iso2: edit.iso2, applyToAll: edit.all });
			flash = `Saved on ${r.updated} row${r.updated === 1 ? '' : 's'}.`;
			edit = null;
			await load();
		} catch (e) {
			error = msg(e);
		} finally {
			saving = false;
		}
	}
	const hubHref = (slug: string, key: string) => (appConfig.appUrl ? `${appConfig.appUrl}/competitions/${key}?s=${slug}` : '');
</script>

<div class="head">
	<div>
		<p class="kicker">Admin</p>
		<h1>Teams</h1>
		<p class="muted small">{teams.length} teams across every season · {teams.filter((t) => t.entries.length > 1).length} in more than one · {unlinked} without a provider id{#if unlinked}, link them with "Fetch logos" on the season{/if}.</p>
	</div>
</div>

<div class="filters">
	<label class="search"><Search size={16} /><input placeholder="Name, code or season" bind:value={q} /></label>
	<label class="chk"><input type="checkbox" bind:checked={onlyMulti} /> several seasons</label>
	<label class="chk"><input type="checkbox" bind:checked={onlyUnlinked} /> no provider id</label>
</div>

{#if error}<p class="error">{error}</p>{/if}
{#if flash}<p class="flash">{flash}</p>{/if}

{#if !loaded}
	<p class="muted">Loading…</p>
{:else}
	<div class="card table">
		<table>
			<thead>
				<tr><th></th><th>Team</th><th>Code</th><th>Flag</th><th>Provider</th><th class="num">Seasons</th><th></th></tr>
			</thead>
			<tbody>
				{#each shown as t (t.key)}
					<tr class:open={openKey === t.key} onclick={() => (openKey = openKey === t.key ? '' : t.key)}>
						<td class="crest">{#if crest(t)}<img src={crest(t)} alt="" />{/if}</td>
						<td class="name"><b>{t.name}</b>{#if t.mixed.length}<span class="mix" title="Differs between seasons: {t.mixed.join(', ')}"><TriangleAlert size={12} /> {t.mixed.join(', ')}</span>{/if}</td>
						<td class="mono">{t.fifaCode}</td>
						<td class="mono">{t.iso2 || '—'}</td>
						<td class="mono muted">{t.providerId ? `${t.provider} ${t.providerId}` : t.clubKey ? `club ${t.clubKey}` : '—'}</td>
						<td class="num">{t.entries.length}</td>
						<td class="cv"><ChevronDown size={14} /></td>
					</tr>
					{#if openKey === t.key}
						<tr class="detail">
							<td colspan="7">
								<ul class="entries">
									{#each t.entries as e (e.teamId)}
										<li>
											<span class="season">{e.season}</span>
											<span class="pill">{e.status}</span>
											{#if e.group}<span class="muted">group {e.group}</span>{/if}
											<span class="mono muted">{e.fifaCode}{e.iso2 ? ` · ${e.iso2}` : ''}{e.name !== t.name ? ` · "${e.name}"` : ''}</span>
											{#if hubHref(e.slug, '')}{/if}
										</li>
									{/each}
								</ul>
								{#if edit && edit.teamId === t.entries[0].teamId}
									<form class="edit" onsubmit={(e) => { e.preventDefault(); save(); }}>
										<label><span>Name</span><input bind:value={edit.name} required /></label>
										<label><span>Code</span><input bind:value={edit.fifaCode} maxlength="3" required /></label>
										<label><span>Flag (iso2)</span><input bind:value={edit.iso2} maxlength="8" placeholder="de, gb-sct" /></label>
										<label class="chk"><input type="checkbox" bind:checked={edit.all} /> apply to all {t.entries.length} seasons</label>
										<button class="btn sm" type="submit" disabled={saving}><Check size={14} /> {saving ? 'Saving…' : 'Save'}</button>
										<button class="btn secondary sm" type="button" onclick={() => (edit = null)}>Cancel</button>
									</form>
								{:else}
									<button class="btn secondary sm" onclick={(e) => { e.stopPropagation(); startEdit(t); }}>Edit name, code, flag</button>
								{/if}
							</td>
						</tr>
					{/if}
				{/each}
			</tbody>
		</table>
		{#if !shown.length}<p class="muted empty">Nothing matches.</p>{/if}
	</div>
{/if}

<style>
	.head {
		margin-bottom: 0.8rem;
	}
	.small {
		font-size: 0.85rem;
		margin: 0.3rem 0 0;
	}
	.filters {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
		margin-bottom: 0.8rem;
	}
	.search {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.4rem 0.7rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface);
		color: var(--muted);
		min-width: 260px;
	}
	.search input {
		border: none;
		background: transparent;
		color: var(--text);
		font: inherit;
		outline: none;
		width: 100%;
	}
	.chk {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.85rem;
		color: var(--muted);
	}
	.table {
		padding: 0;
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th,
	td {
		text-align: left;
		padding: 0.55rem 0.7rem;
		border-bottom: 1px solid var(--border);
		font-size: 0.88rem;
	}
	th {
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
	}
	tbody tr:not(.detail) {
		cursor: pointer;
	}
	tbody tr:not(.detail):hover {
		background: var(--surface-2);
	}
	tr.open td {
		border-bottom: none;
	}
	.crest {
		width: 36px;
	}
	.crest img {
		width: 22px;
		height: 22px;
		object-fit: contain;
		display: block;
	}
	.name {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
	}
	.mix {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		font-size: 0.72rem;
		color: var(--warning);
	}
	.mono {
		font-family: var(--font-mono);
		font-size: 0.8rem;
	}
	.num {
		text-align: right;
	}
	.cv {
		color: var(--muted);
		width: 24px;
	}
	tr.open .cv {
		transform: rotate(180deg);
	}
	.detail td {
		background: var(--surface-2);
	}
	.entries {
		list-style: none;
		margin: 0 0 0.6rem;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
	}
	.entries li {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		flex-wrap: wrap;
	}
	.season {
		font-weight: 600;
	}
	.edit {
		display: flex;
		align-items: flex-end;
		gap: 0.6rem;
		flex-wrap: wrap;
	}
	.edit label {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		font-size: 0.75rem;
		color: var(--muted);
	}
	.edit input:not([type='checkbox']) {
		padding: 0.4rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
		color: var(--text);
		font: inherit;
	}
	.edit label.chk {
		flex-direction: row;
		align-items: center;
	}
	.btn.sm {
		width: auto;
		padding: 0.5rem 0.8rem;
		font-size: 0.8rem;
	}
	.flash {
		color: var(--success);
		font-size: 0.9rem;
	}
	.empty {
		padding: 1rem;
	}
</style>
