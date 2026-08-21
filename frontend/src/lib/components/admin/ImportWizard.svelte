<!-- "Add from API-Football" wizard: search the league catalog → pick a season
     → review/edit the derived proposal → import (creates a seeded draft).
     A native <dialog> like ConfirmDialog, but wide. -->
<script lang="ts">
	import {
		api,
		type FootballLeague,
		type ImportProposal,
		type AdminCompetition,
		type TournamentStatus
	} from '$lib/api';
	import { Search, ChevronLeft, Check, TriangleAlert, Trophy } from '@lucide/svelte';

	let {
		open = $bindable(false),
		competitions = [],
		onimported
	}: {
		open?: boolean;
		competitions?: AdminCompetition[];
		onimported?: (r: { id: string; slug: string; teams: number; matches: number }) => void;
	} = $props();

	let dialog = $state<HTMLDialogElement | null>(null);
	$effect(() => {
		const d = dialog;
		if (!d) return;
		if (open && !d.open) d.showModal();
		else if (!open && d.open) d.close();
	});

	type Step = 'search' | 'season' | 'preview' | 'done';
	let step = $state<Step>('search');
	let busy = $state(false);
	let error = $state('');

	// Step 1 — search
	let query = $state('');
	let leagues = $state<FootballLeague[]>([]);
	let searched = $state(false);
	// Step 2 — season
	let league = $state<FootballLeague | null>(null);
	// Step 3 — preview
	let proposal = $state<ImportProposal | null>(null);
	let structureText = $state('');
	let competition = $state('');
	let initialStatus = $state<TournamentStatus>('draft');
	// Step 4 — done
	let result = $state<{ id: string; slug: string; teams: number; matches: number } | null>(null);

	function reset() {
		step = 'search';
		busy = false;
		error = '';
		query = '';
		leagues = [];
		searched = false;
		league = null;
		proposal = null;
		structureText = '';
		competition = '';
		initialStatus = 'draft';
		result = null;
	}

	function close() {
		if (busy) return;
		open = false;
		reset();
	}

	async function search(e?: Event) {
		e?.preventDefault();
		if (query.trim().length < 3) {
			error = 'Type at least 3 characters.';
			return;
		}
		busy = true;
		error = '';
		try {
			const r = await api.footballLeagues(query.trim());
			leagues = r.leagues ?? [];
			searched = true;
		} catch (err) {
			error = msg(err);
		} finally {
			busy = false;
		}
	}

	function pickLeague(l: FootballLeague) {
		league = l;
		error = '';
		step = 'season';
	}

	async function pickSeason(year: number) {
		if (!league) return;
		busy = true;
		error = '';
		try {
			proposal = await api.footballPreview(league.id, year);
			structureText = JSON.stringify(proposal.structure, null, 2);
			// Pre-select a competition that already maps to this league id.
			competition = competitions.find((c) => c.apiFootballLeague === league?.id)?.id ?? '';
			step = 'preview';
		} catch (err) {
			error = msg(err);
		} finally {
			busy = false;
		}
	}

	async function doImport() {
		if (!proposal) return;
		error = '';
		let structure: Record<string, unknown>;
		try {
			structure = JSON.parse(structureText);
		} catch (err) {
			error = `structure: ${msg(err)}`;
			return;
		}
		busy = true;
		try {
			result = await api.tournamentImport(
				{ ...proposal, structure },
				{ competition: competition || undefined, status: initialStatus }
			);
			step = 'done';
			onimported?.(result);
		} catch (err) {
			error = msg(err);
		} finally {
			busy = false;
		}
	}

	function msg(err: unknown): string {
		// PocketBase ClientResponseError carries the JSON body in .response
		const r = (err as { response?: { error?: string; message?: string } })?.response;
		return r?.error || r?.message || (err instanceof Error ? err.message : String(err));
	}

	function seasonRange(s: { start: string; end: string }): string {
		if (!s.start) return '';
		const f = (d: string) => d.slice(0, 7);
		return s.end && s.end.slice(0, 4) !== s.start.slice(0, 4)
			? `${f(s.start)} → ${f(s.end)}`
			: `${f(s.start)} → ${f(s.end)}`;
	}
	function day(iso: string): string {
		return iso ? new Date(iso).toLocaleDateString() : '—';
	}
</script>

<dialog
	bind:this={dialog}
	class="wiz"
	oncancel={(e) => {
		e.preventDefault();
		close();
	}}
	onclick={(e) => {
		if (e.target === dialog) close();
	}}
>
	<div class="inner">
		<header class="whead">
			{#if step !== 'search' && step !== 'done'}
				<button
					class="back"
					type="button"
					aria-label="Back"
					disabled={busy}
					onclick={() => {
						error = '';
						step = step === 'preview' ? 'season' : 'search';
					}}><ChevronLeft size={18} /></button
				>
			{/if}
			<h3>
				{#if step === 'search'}Add tournament from API-Football
				{:else if step === 'season'}{league?.name} · pick a season
				{:else if step === 'preview'}Review import
				{:else}Imported{/if}
			</h3>
			<span class="steps">
				{#each ['search', 'season', 'preview'] as s, i (s)}
					<span class="dot" class:on={step === s || (step === 'done' && i === 2)}></span>
				{/each}
			</span>
		</header>

		{#if step === 'search'}
			<form class="srch" onsubmit={search}>
				<input
					class="input"
					bind:value={query}
					placeholder="League or cup name — e.g. Bundesliga, Euro, World Cup"
				/>
				<button class="btn" type="submit" disabled={busy}>
					<Search size={16} />
					{busy ? '…' : 'Search'}
				</button>
			</form>
			<p class="hint muted">
				Searches the API-Football catalog (needs <code>API_FOOTBALL_KEY</code>). Free plans can only
				fetch seasons 2021–2023.
			</p>
			{#if searched && leagues.length === 0}
				<p class="muted">No leagues match “{query}”.</p>
			{/if}
			<ul class="list">
				{#each leagues as l (l.id)}
					<li>
						<button type="button" class="item" onclick={() => pickLeague(l)}>
							{#if l.logo}<img src={l.logo} alt="" class="logo" loading="lazy" />{:else}<span
									class="logo ph"><Trophy size={16} /></span
								>{/if}
							<span class="ttl">
								<strong>{l.name}</strong>
								<small class="muted">{l.country} · {l.type} · #{l.id}</small>
							</span>
							<span class="pill">{l.seasons.length} seasons</span>
						</button>
					</li>
				{/each}
			</ul>
		{:else if step === 'season' && league}
			<ul class="list">
				{#each league.seasons as s (s.year)}
					<li>
						<button type="button" class="item" disabled={busy} onclick={() => pickSeason(s.year)}>
							<span class="ttl">
								<strong>{s.year}</strong>
								<small class="muted">{seasonRange(s)}</small>
							</span>
							{#if s.current}<span class="pill ok">current</span>{/if}
						</button>
					</li>
				{/each}
			</ul>
			{#if busy}<p class="muted">Fetching fixtures, teams and standings…</p>{/if}
		{:else if step === 'preview' && proposal}
			<div class="pv">
				<div class="pvcol">
					<label class="field">
						<span>Name</span>
						<input class="input" bind:value={proposal.name} required />
					</label>
					<div class="two">
						<label class="field">
							<span>Short name</span>
							<input class="input" bind:value={proposal.shortName} />
						</label>
						<label class="field">
							<span>Slug</span>
							<input class="input mono" bind:value={proposal.slug} />
						</label>
					</div>
					<div class="two">
						<label class="field">
							<span>Ext-ID prefix</span>
							<input class="input mono" bind:value={proposal.extIdPrefix} maxlength="16" />
						</label>
						<label class="field">
							<span>Create as</span>
							<select class="input" bind:value={initialStatus}>
								<option value="draft">draft (hidden)</option>
								<option value="upcoming">upcoming</option>
								<option value="active">active</option>
							</select>
						</label>
					</div>
					<label class="field">
						<span>Competition</span>
						<select class="input" bind:value={competition}>
							<option value="">Auto — create “{proposal?.leagueName ?? 'the league'}” if needed</option>
							{#each competitions as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
						</select>
					</label>
					<label class="field">
						<span>Structure (JSON)</span>
						<textarea class="input mono" rows="12" bind:value={structureText} spellcheck="false"
						></textarea>
					</label>
				</div>
				<div class="pvcol">
					<div class="sum">
						<div class="srow"><span class="k">Shape</span><span class="v">{proposal.shape}</span></div>
						<div class="srow">
							<span class="k">Fixtures</span><span class="v">{proposal.fixtures}</span>
						</div>
						<div class="srow"><span class="k">Teams</span><span class="v">{proposal.teams.length}</span></div>
						<div class="srow">
							<span class="k">Dates</span><span class="v">{day(proposal.startsAt)} → {day(proposal.endsAt)}</span>
						</div>
						<div class="srow">
							<span class="k">Forecast</span><span class="v">{proposal.forecastSpec?.mode ?? '—'}</span>
						</div>
					</div>
					{#if proposal.warnings?.length}
						{#each proposal.warnings as w (w)}
							<p class="warn"><TriangleAlert size={14} /> {w}</p>
						{/each}
					{/if}
					{#if proposal.groups.length}
						<h4>Groups</h4>
						<div class="groups">
							{#each proposal.groups as g (g.letter)}
								<div class="grp">
									<strong>{g.letter}</strong>
									<span class="muted">{g.teams.join(', ')}</span>
								</div>
							{/each}
						</div>
					{/if}
					<h4>Rounds</h4>
					<div class="rounds">
						{#each proposal.rounds as r (r.label)}
							<div class="rnd">
								<span class="mono">{r.stage}</span>
								<span class="lbl">{r.label}</span>
								<span class="muted">{r.matches}</span>
							</div>
						{/each}
					</div>
				</div>
			</div>
			<footer class="wfoot">
				{#if error}<span class="error">{error}</span>{/if}
				<span class="spacer"></span>
				<button class="btn secondary" type="button" onclick={close} disabled={busy}>Cancel</button>
				<button class="btn" type="button" onclick={doImport} disabled={busy}>
					{busy ? 'Importing…' : `Import ${proposal.teams.length} teams · ${proposal.fixtures} matches`}
				</button>
			</footer>
		{:else if step === 'done' && result}
			<div class="done">
				<Check size={28} />
				<p><strong>{result.slug}</strong> created as {initialStatus} — {result.teams} teams, {result.matches} matches seeded.</p>
				<button class="btn" type="button" onclick={close}>Close</button>
			</div>
		{/if}

		{#if error && step !== 'preview'}
			<p class="error">{error}</p>
		{/if}
	</div>
</dialog>

<style>
	.wiz {
		border: none;
		padding: 0;
		background: transparent;
		width: min(96vw, 60rem);
		max-width: none;
		color: inherit;
	}
	.wiz::backdrop {
		background: rgba(0, 0, 0, 0.55);
		backdrop-filter: blur(2px);
	}
	.inner {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 1.1rem 1.25rem 1.25rem;
		box-shadow: 0 18px 50px rgba(0, 0, 0, 0.45);
		max-height: 88vh;
		overflow: auto;
	}
	.whead {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		margin-bottom: 0.9rem;
	}
	.whead h3 {
		margin: 0;
		font-size: 1.05rem;
		flex: 1;
	}
	.back {
		display: inline-grid;
		place-items: center;
		width: 2rem;
		height: 2rem;
		border-radius: 50%;
		border: 1px solid var(--border);
		background: var(--surface-2);
		color: var(--text);
		cursor: pointer;
	}
	.steps {
		display: inline-flex;
		gap: 0.3rem;
	}
	.dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		background: var(--border);
	}
	.dot.on {
		background: var(--accent);
	}
	.srch {
		display: flex;
		gap: 0.6rem;
	}
	.srch .btn {
		width: auto;
	}
	.hint {
		font-size: 0.82rem;
		margin: 0.6rem 0 0.8rem;
	}
	.hint code {
		font-size: 0.8em;
	}
	.list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}
	.item {
		width: 100%;
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.6rem 0.75rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		color: var(--text);
		text-align: left;
		cursor: pointer;
		font: inherit;
	}
	.item:hover:not(:disabled) {
		border-color: var(--accent);
	}
	.item:disabled {
		opacity: 0.6;
		cursor: default;
	}
	.logo {
		width: 28px;
		height: 28px;
		object-fit: contain;
		flex: none;
	}
	.logo.ph {
		display: inline-grid;
		place-items: center;
		color: var(--muted);
	}
	.ttl {
		display: flex;
		flex-direction: column;
		flex: 1;
		min-width: 0;
	}
	.ttl small {
		font-size: 0.78rem;
	}
	.pv {
		display: grid;
		grid-template-columns: 1fr;
		gap: 1rem;
	}
	@media (min-width: 800px) {
		.pv {
			grid-template-columns: 1.1fr 1fr;
		}
	}
	.two {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0 0.75rem;
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
		line-height: 1.4;
	}
	.sum {
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		padding: 0.25rem 0.75rem;
		margin-bottom: 0.7rem;
	}
	.srow {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.4rem 0;
		border-bottom: 1px solid var(--border);
		font-size: 0.9rem;
	}
	.srow:last-child {
		border-bottom: none;
	}
	.srow .k {
		color: var(--muted);
	}
	.srow .v {
		font-weight: 600;
	}
	.warn {
		display: flex;
		gap: 0.4rem;
		align-items: flex-start;
		margin: 0.3rem 0;
		font-size: 0.85rem;
		color: var(--warning, #d9a441);
	}
	h4 {
		margin: 0.8rem 0 0.4rem;
		font-size: 0.8rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--muted);
	}
	.groups {
		display: grid;
		grid-template-columns: 1fr;
		gap: 0.3rem;
		font-size: 0.85rem;
	}
	.grp {
		display: flex;
		gap: 0.6rem;
	}
	.grp strong {
		width: 1.4rem;
		color: var(--accent);
	}
	.rounds {
		display: flex;
		flex-direction: column;
		font-size: 0.85rem;
		max-height: 14rem;
		overflow: auto;
	}
	.rnd {
		display: grid;
		grid-template-columns: 6rem 1fr auto;
		gap: 0.6rem;
		padding: 0.25rem 0;
		border-bottom: 1px solid var(--border);
	}
	.rnd:last-child {
		border-bottom: none;
	}
	.wfoot {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		margin-top: 1rem;
		flex-wrap: wrap;
	}
	.wfoot .btn {
		width: auto;
	}
	.done {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.8rem;
		padding: 1rem 0;
		text-align: center;
		color: var(--text);
	}
	.done .btn {
		width: auto;
	}
	dialog[open].wiz {
		animation: wiz-pop 0.14s ease-out;
	}
	@keyframes wiz-pop {
		from {
			opacity: 0;
			transform: translateY(6px) scale(0.98);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		dialog[open].wiz {
			animation: none;
		}
	}
</style>
