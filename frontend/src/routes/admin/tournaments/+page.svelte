<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import {
		api,
		type AdminTournament,
		type AdminTournamentPayload,
		type AdminCompetition,
		type TournamentStatus
	} from '$lib/api';
	import TournamentForm from '$lib/components/admin/TournamentForm.svelte';
	import ImportWizard from '$lib/components/admin/ImportWizard.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { describeSeason } from '$lib/describe';
	import type { Tournament } from '$lib/tournament.svelte';
	import {
		Plus,
		Download,
		Pencil,
		Trash2,
		Users,
		CalendarDays,
		Database,
		ChevronLeft,
		ImageDown
	} from '@lucide/svelte';

	$effect(() => {
		if (!auth.isAdmin) goto('/');
	});

	let list = $state<AdminTournament[]>([]);
	let competitions = $state<AdminCompetition[]>([]);
	let loaded = $state(false);
	let loadError = $state('');

	async function load() {
		try {
			const [t, c] = await Promise.all([api.adminTournaments(), api.adminCompetitions()]);
			list = t.tournaments ?? [];
			competitions = c.competitions ?? [];
			compDraft = Object.fromEntries(
				competitions.map((x) => [x.id, { name: x.name, description: x.description ?? '' }])
			);
			loadError = '';
		} catch (e) {
			loadError = msg(e);
		} finally {
			loaded = true;
		}
	}
	$effect(() => {
		if (auth.isAdmin && !loaded) load();
	});

	function msg(err: unknown): string {
		const r = (err as { response?: { error?: string; message?: string; data?: unknown } })
			?.response;
		return r?.error || r?.message || (err instanceof Error ? err.message : String(err));
	}

	const compName = $derived(
		(id: string) => competitions.find((c) => c.id === id)?.name ?? ''
	);
	const compKey = $derived(
		(id: string) => competitions.find((c) => c.id === id)?.key ?? ''
	);

	// ---- competitions: inline edit of name / description, logo fetch ----
	let compDraft = $state<Record<string, { name: string; description: string }>>({});
	let compBusy = $state('');
	let compError = $state('');
	function compDirty(c: AdminCompetition): boolean {
		const d = compDraft[c.id];
		return !!d && (d.name !== c.name || d.description !== (c.description ?? ''));
	}
	async function saveComp(c: AdminCompetition) {
		const d = compDraft[c.id];
		if (!d) return;
		compBusy = c.id;
		compError = '';
		try {
			const r = await api.adminCompetitionUpdate(c.id, { name: d.name, description: d.description });
			competitions = competitions.map((x) => (x.id === c.id ? { ...x, ...r } : x));
			compDraft[c.id] = { name: r.name, description: r.description ?? '' };
			flash = `Saved ${r.name}.`;
		} catch (e) {
			compError = msg(e);
		} finally {
			compBusy = '';
		}
	}
	async function fetchCompLogo(c: AdminCompetition) {
		compBusy = c.id;
		compError = '';
		try {
			const r = await api.adminCompetitionLogo(c.id);
			competitions = competitions.map((x) => (x.id === c.id ? { ...x, ...r } : x));
		} catch (e) {
			compError = msg(e);
		} finally {
			compBusy = '';
		}
	}
	/** Placeholder for an empty description: what the hub derives from
	 *  the competition's most recent season. */
	function derivedBlurb(c: AdminCompetition): string {
		const t = list
			.filter((x) => x.competition === c.id)
			.sort((a, b) => (a.startsAt < b.startsAt ? 1 : -1))[0];
		if (!t) return 'No seasons yet.';
		return describeSeason(
			{ ...t, competition: c } as unknown as Tournament,
			t.teams
		);
	}
	function compLogo(c: AdminCompetition): string {
		return c.logo ? `/api/files/competitions/${c.id}/${c.logo}` : '';
	}

	// ---- panels ----
	type Panel = { kind: 'none' } | { kind: 'create' } | { kind: 'edit'; t: AdminTournament } | { kind: 'seed'; t: AdminTournament };
	let panel = $state<Panel>({ kind: 'none' });
	let wizardOpen = $state(false);
	let saving = $state(false);
	let saveError = $state('');
	let flash = $state('');

	function openEdit(t: AdminTournament) {
		saveError = '';
		panel = { kind: 'edit', t };
		queueMicrotask(() => document.getElementById('panel')?.scrollIntoView({ behavior: 'smooth' }));
	}

	async function save(payload: AdminTournamentPayload) {
		saving = true;
		saveError = '';
		try {
			if (panel.kind === 'edit') {
				await api.adminTournamentUpdate(panel.t.id, payload);
				flash = `Saved ${payload.name ?? payload.slug}.`;
			} else {
				const r = await api.adminTournamentCreate(payload);
				flash = `Created ${r.slug} as draft — seed it next.`;
			}
			panel = { kind: 'none' };
			await load();
		} catch (e) {
			saveError = msg(e);
		} finally {
			saving = false;
		}
	}

	// Quick status change straight from the list.
	const STATUSES: TournamentStatus[] = ['draft', 'upcoming', 'active', 'finished', 'archived'];
	let statusBusy = $state<string>('');
	async function setStatus(t: AdminTournament, status: TournamentStatus) {
		if (status === t.status) return;
		statusBusy = t.id;
		try {
			await api.adminTournamentUpdate(t.id, { status });
			flash = `${t.name}: ${t.status} → ${status}.`;
			await load();
		} catch (e) {
			flash = msg(e);
		} finally {
			statusBusy = '';
		}
	}

	// Delete (drafts only — the API refuses otherwise).
	let confirmDelete = $state<AdminTournament | null>(null);
	let deleting = $state(false);
	async function doDelete() {
		if (!confirmDelete) return;
		deleting = true;
		try {
			await api.adminTournamentDelete(confirmDelete.id);
			flash = `Deleted ${confirmDelete.slug}.`;
			confirmDelete = null;
			await load();
		} catch (e) {
			flash = msg(e);
			confirmDelete = null;
		} finally {
			deleting = false;
		}
	}

	// Seed from openfootball JSON (fallback when no API key / catalog miss).
	let seedTeams = $state('');
	let seedFixtures = $state('');
	let seedError = $state('');
	async function doSeed() {
		if (panel.kind !== 'seed') return;
		seedError = '';
		let teams: unknown, fixtures: unknown;
		try {
			teams = JSON.parse(seedTeams);
			fixtures = JSON.parse(seedFixtures);
		} catch (e) {
			seedError = `JSON: ${msg(e)}`;
			return;
		}
		saving = true;
		try {
			const r = await api.adminTournamentSeed(panel.t.id, teams, fixtures);
			flash = `Seeded ${r.teams} teams and ${r.matches} matches.`;
			panel = { kind: 'none' };
			seedTeams = seedFixtures = '';
			await load();
		} catch (e) {
			seedError = msg(e);
		} finally {
			saving = false;
		}
	}

	// Crests for an already-imported tournament (matched by team name).
	let logosBusy = $state('');
	async function fetchLogos(t: AdminTournament) {
		logosBusy = t.id;
		try {
			const r = await api.adminTournamentLogos(t.id);
			flash = `${t.name}: ${r.logos} crest${r.logos === 1 ? '' : 's'} fetched.`;
		} catch (e) {
			flash = msg(e);
		} finally {
			logosBusy = '';
		}
	}
	const hasApiLeague = (t: AdminTournament) =>
		!!(t.sync as { apiFootballLeague?: number } | null)?.apiFootballLeague;

	function fmtDate(v: string): string {
		if (!v) return '—';
		const d = new Date(v.replace(' ', 'T'));
		return isNaN(d.getTime()) ? '—' : d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
	}
	const syncLabel = $derived((t: AdminTournament) => {
		const p = (t.sync as { provider?: string } | null)?.provider;
		return p && p !== 'manual' ? p : 'manual';
	});
</script>

<div class="head">
	<div>
		<p class="kicker"><a href="/admin" class="crumb"><ChevronLeft size={14} /> Admin</a></p>
		<h1>Tournaments</h1>
	</div>
	<div class="headactions">
		<button class="btn secondary" onclick={() => { saveError = ''; panel = { kind: 'create' }; }}>
			<Plus size={16} /> Create manually
		</button>
		<button class="btn" onclick={() => (wizardOpen = true)}>
			<Download size={16} /> Add from API-Football
		</button>
	</div>
</div>

{#if !auth.isAdmin}
	<p class="muted">Restricted.</p>
{:else}
	{#if flash}<p class="flash" role="status">{flash}</p>{/if}

	{#if panel.kind === 'create' || panel.kind === 'edit'}
		<section class="card" id="panel">
			<h2 class="sec">
				<Pencil size={18} />
				{panel.kind === 'edit' ? `Edit ${panel.t.name}` : 'New tournament'}
			</h2>
			{#key panel.kind === 'edit' ? panel.t.id : 'new'}
				<TournamentForm
					initial={panel.kind === 'edit' ? panel.t : null}
					{competitions}
					busy={saving}
					error={saveError}
					submitLabel={panel.kind === 'edit' ? 'Save changes' : 'Create draft'}
					onsubmit={save}
					oncancel={() => (panel = { kind: 'none' })}
				/>
			{/key}
			{#if panel.kind === 'edit' && panel.t.teams > 0}
				<p class="muted small">
					Slug and ext-ID prefix are part of the seeded match ids — change them only on unseeded
					drafts.
				</p>
			{/if}
		</section>
	{:else if panel.kind === 'seed'}
		<section class="card" id="panel">
			<h2 class="sec"><Database size={18} /> Seed {panel.t.name} from openfootball JSON</h2>
			<p class="muted small">
				Paste the team meta array (name, fifa_code, flag_unicode, group, confed) and the fixtures
				document (<code>{'{"matches": [...]}'}</code>). Round labels must map onto the tournament's
				stages.
			</p>
			<div class="seedgrid">
				<label class="field">
					<span>Teams (JSON array)</span>
					<textarea class="input mono" rows="10" bind:value={seedTeams} spellcheck="false"></textarea>
				</label>
				<label class="field">
					<span>Fixtures (JSON)</span>
					<textarea class="input mono" rows="10" bind:value={seedFixtures} spellcheck="false"
					></textarea>
				</label>
			</div>
			{#if seedError}<p class="error">{seedError}</p>{/if}
			<div class="actions">
				<button class="btn secondary" onclick={() => (panel = { kind: 'none' })} disabled={saving}
					>Cancel</button
				>
				<button class="btn" onclick={doSeed} disabled={saving}>{saving ? 'Seeding…' : 'Seed'}</button>
			</div>
		</section>
	{/if}

	{#if !loaded}
		<p class="muted">Loading…</p>
	{:else if loadError}
		<p class="error">{loadError}</p>
	{:else if list.length === 0}
		<section class="card empty">
			<p class="muted">No tournaments yet. Import one from the API-Football catalog or create it manually.</p>
		</section>
	{:else}
		<div class="tlist">
			{#each list as t (t.id)}
				<section class="card trow" class:draft={t.status === 'draft'}>
					<div class="tmain">
						<div class="ttitle">
							<h3>{t.name}</h3>
							<span class="pill" class:ok={t.status === 'active'} class:live={t.status === 'active'}
								>{t.status}</span
							>
							{#if compName(t.competition)}<span class="pill">{compName(t.competition)}</span>{/if}
						</div>
						<p class="meta">
							<code>{t.slug}</code> · <code>{t.extIdPrefix}</code>
							· <CalendarDays size={13} /> {fmtDate(t.startsAt)} → {fmtDate(t.endsAt)}
							· sync: {syncLabel(t)}
						</p>
						<p class="meta">
							<Database size={13} />
							{#if t.teams === 0}
								<span class="warn">not seeded</span>
							{:else}
								{t.teams} teams · {t.matches} matches
							{/if}
							· <Users size={13} /> {t.players} players
						</p>
					</div>
					<div class="tactions">
						<select
							class="input sel"
							value={t.status}
							disabled={statusBusy === t.id}
							onchange={(e) => setStatus(t, (e.currentTarget as HTMLSelectElement).value as TournamentStatus)}
							aria-label="Status"
						>
							{#each STATUSES as s (s)}<option value={s}>{s}</option>{/each}
						</select>
						{#if t.teams === 0}
							<button class="btn secondary sm" onclick={() => { seedError = ''; panel = { kind: 'seed', t }; }}>
								<Database size={14} /> Seed JSON
							</button>
						{/if}
						{#if t.teams > 0 && hasApiLeague(t)}
							<button
								class="btn secondary sm"
								disabled={logosBusy === t.id}
								onclick={() => fetchLogos(t)}
								title="Download team crests from API-Football (teams matched by name)"
							>
								<ImageDown size={14} /> {logosBusy === t.id ? 'Fetching…' : 'Fetch logos'}
							</button>
						{/if}
						<button class="btn secondary sm" onclick={() => openEdit(t)}><Pencil size={14} /> Edit</button>
						{#if t.status === 'draft'}
							<button class="btn secondary sm danger" onclick={() => (confirmDelete = t)}>
								<Trash2 size={14} /> Delete
							</button>
						{/if}
						<a class="btn ghost sm" href={`/competitions/${compKey(t.competition)}?s=${t.slug}`}>Open</a>
					</div>
				</section>
			{/each}
		</div>
	{/if}
{/if}

{#if auth.isAdmin && loaded && competitions.length}
	<h2 class="sec comps-h">Competitions</h2>
	<p class="muted small">
		Seasons group under these on the Competitions page. The description shows on the hub;
		leave it empty to use the line derived from the latest season (shown as placeholder).
	</p>
	{#if compError}<p class="error">{compError}</p>{/if}
	<div class="clist">
		{#each competitions.filter((c) => compDraft[c.id]) as c (c.id)}
			<section class="card comp">
				<div class="clogo" class:ph={!compLogo(c)}>
					{#if compLogo(c)}<img src={compLogo(c)} alt="" />{:else}<ImageDown size={18} />{/if}
				</div>
				<div class="cmain">
					<div class="ctop">
						<input class="input cname" bind:value={compDraft[c.id].name} aria-label="Name" />
						<span class="pill">{c.teamKind}</span>
						{#if c.country}<span class="pill">{c.country}</span>{/if}
					</div>
					<p class="meta">
						<code>{c.key}</code>
						{#if c.apiFootballLeague}· API-Football league {c.apiFootballLeague}{/if}
						· {list.filter((t) => t.competition === c.id).length} seasons
					</p>
					<textarea
						class="input cdesc"
						rows="2"
						placeholder={derivedBlurb(c)}
						bind:value={compDraft[c.id].description}
						aria-label="Description"
					></textarea>
				</div>
				<div class="cactions">
					{#if c.apiFootballLeague}
						<button
							class="btn secondary sm"
							disabled={compBusy === c.id}
							onclick={() => fetchCompLogo(c)}
							title="Download the league badge from API-Football"
						>
							<ImageDown size={14} /> {c.logo ? 'Refetch logo' : 'Fetch logo'}
						</button>
					{/if}
					<button
						class="btn sm"
						disabled={compBusy === c.id || !compDirty(c)}
						onclick={() => saveComp(c)}
					>
						{compBusy === c.id ? 'Saving…' : 'Save'}
					</button>
				</div>
			</section>
		{/each}
	</div>
{/if}

<ImportWizard
	bind:open={wizardOpen}
	{competitions}
	onimported={(r) => {
		flash = `Imported ${r.slug}: ${r.teams} teams, ${r.matches} matches.`;
		load();
	}}
/>

<ConfirmDialog
	open={!!confirmDelete}
	title="Delete draft tournament?"
	message={confirmDelete ? `${confirmDelete.name} (${confirmDelete.slug}) and its seeded teams/matches will be removed.` : ''}
	confirmLabel="Delete"
	danger
	busy={deleting}
	onconfirm={doDelete}
	oncancel={() => (confirmDelete = null)}
/>

<style>
	.head {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
		margin-bottom: 1.1rem;
	}
	.crumb {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		color: inherit;
		text-decoration: none;
	}
	.headactions {
		display: flex;
		gap: 0.6rem;
		flex-wrap: wrap;
	}
	.headactions .btn {
		width: auto;
	}
	.sec {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		margin: 0 0 0.9rem;
		font-size: 0.95rem;
		font-weight: 700;
		color: var(--muted);
	}
	.flash {
		margin: 0 0 0.9rem;
		padding: 0.6rem 0.8rem;
		border: 1px solid color-mix(in srgb, var(--accent) 45%, var(--border));
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--accent) 10%, var(--surface));
		font-size: 0.9rem;
	}
	.small {
		font-size: 0.82rem;
	}
	.tlist {
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
	}
	.tlist .card + .card {
		margin-top: 0;
	}
	.trow {
		display: flex;
		gap: 1rem;
		align-items: flex-start;
		flex-wrap: wrap;
	}
	.trow.draft {
		border-style: dashed;
	}
	.tmain {
		flex: 1;
		min-width: 16rem;
	}
	.ttitle {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}
	.ttitle h3 {
		margin: 0;
		font-size: 1.05rem;
	}
	.meta {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		flex-wrap: wrap;
		margin: 0.35rem 0 0;
		font-size: 0.85rem;
		color: var(--muted);
	}
	.meta code {
		font-size: 0.8rem;
		color: var(--text);
	}
	.warn {
		color: var(--warning, #d9a441);
		font-weight: 600;
	}
	.tactions {
		display: flex;
		gap: 0.45rem;
		align-items: center;
		flex-wrap: wrap;
	}
	.tactions .btn {
		width: auto;
	}
	.btn.sm {
		padding: 0.5rem 0.7rem;
		font-size: 0.78rem;
	}
	.btn.danger {
		color: var(--danger);
	}
	.sel {
		width: auto;
		padding: 0.5rem 0.6rem;
		font-size: 0.85rem;
	}
	.seedgrid {
		display: grid;
		grid-template-columns: 1fr;
		gap: 0 1rem;
	}
	@media (min-width: 720px) {
		.seedgrid {
			grid-template-columns: 1fr 1fr;
		}
	}
	.field span {
		display: block;
		font-size: 0.78rem;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--muted);
		margin-bottom: 0.4rem;
	}
	.mono {
		font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
		font-size: 0.85rem;
	}
	textarea.input {
		resize: vertical;
	}
	.actions {
		display: flex;
		gap: 0.6rem;
		justify-content: flex-end;
	}
	.actions .btn {
		width: auto;
	}
	.empty {
		text-align: center;
	}
	/* ---- competitions editor ---- */
	.comps-h {
		margin-top: 2.2rem;
	}
	.clist {
		display: grid;
		gap: 0.7rem;
		margin-top: 0.8rem;
	}
	.comp {
		display: flex;
		gap: 0.9rem;
		align-items: flex-start;
	}
	.clogo {
		flex: none;
		width: 48px;
		height: 48px;
		display: grid;
		place-items: center;
		border-radius: 12px;
		background: #fff;
		overflow: hidden;
	}
	.clogo img {
		width: 80%;
		height: 80%;
		object-fit: contain;
	}
	.clogo.ph {
		background: var(--surface-2);
		color: var(--muted);
	}
	.cmain {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}
	.ctop {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}
	.cname {
		flex: 1;
		min-width: 12rem;
		font-weight: 700;
	}
	.cdesc {
		resize: vertical;
		font-size: 0.9rem;
	}
	.cactions {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		flex: none;
	}
	@media (max-width: 640px) {
		.comp {
			flex-wrap: wrap;
		}
		.cactions {
			flex-direction: row;
			width: 100%;
			justify-content: flex-end;
		}
	}
</style>
