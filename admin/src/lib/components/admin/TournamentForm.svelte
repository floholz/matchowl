<!-- Create / edit form for a tournament record. Identity + lifecycle fields
     are plain inputs; structure / sync / forecastSpec stay JSON (they're
     validated server-side and the shapes are open-ended). Emits a payload
     with only the editable fields — the caller decides create vs update. -->
<script lang="ts">
	import { untrack } from 'svelte';
	import type {
		AdminTournament,
		AdminTournamentPayload,
		AdminCompetition,
		TournamentStatus
	} from '$lib/api';

	let {
		initial = null,
		competitions = [],
		busy = false,
		error = '',
		submitLabel = 'Save',
		onsubmit,
		oncancel
	}: {
		initial?: AdminTournament | null;
		competitions?: AdminCompetition[];
		busy?: boolean;
		error?: string;
		submitLabel?: string;
		onsubmit: (payload: AdminTournamentPayload) => void;
		oncancel?: () => void;
	} = $props();

	const STATUSES: TournamentStatus[] = ['draft', 'upcoming', 'active', 'finished', 'archived'];

	// PocketBase returns "2026-06-11 18:00:00.000Z"; datetime-local wants
	// "YYYY-MM-DDTHH:MM" in local time. Round-trip through Date.
	function toLocalInput(v: string): string {
		if (!v) return '';
		const d = new Date(v.replace(' ', 'T'));
		if (isNaN(d.getTime())) return '';
		const pad = (n: number) => String(n).padStart(2, '0');
		return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
	}
	function fromLocalInput(v: string): string {
		if (!v) return '';
		const d = new Date(v);
		return isNaN(d.getTime()) ? '' : d.toISOString();
	}
	function pretty(v: unknown): string {
		if (v == null || v === '') return '';
		return typeof v === 'string' ? v : JSON.stringify(v, null, 2);
	}

	// Snapshot of the initial record: the form owns its draft state from here
	// on (the parent re-keys the component to edit a different tournament).
	const init = untrack(() => initial);

	let slug = $state(init?.slug ?? '');
	let name = $state(init?.name ?? '');
	let shortName = $state(init?.shortName ?? '');
	let status = $state<TournamentStatus>(init?.status ?? 'draft');
	let startsAt = $state(toLocalInput(init?.startsAt ?? ''));
	let endsAt = $state(toLocalInput(init?.endsAt ?? ''));
	let extIdPrefix = $state(init?.extIdPrefix ?? '');
	let competition = $state(init?.competition ?? '');
	let structure = $state(
		pretty(init?.structure) ||
			JSON.stringify(
				{
					stages: [
						{ code: 'group', name: 'Group stage', kind: 'group' },
						{ code: 'R16', name: 'Round of 16', kind: 'knockout' },
						{ code: 'QF', name: 'Quarter-finals', kind: 'knockout' },
						{ code: 'SF', name: 'Semi-finals', kind: 'knockout' },
						{ code: '3RD', name: 'Third place', kind: 'knockout', consolation: true },
						{ code: 'FINAL', name: 'Final', kind: 'knockout' }
					],
					groupSize: 4,
					gamesPerTeam: 3,
					directQualifiers: 2
				},
				null,
				2
			)
	);
	let sync = $state(
		pretty(init?.sync) || JSON.stringify({ provider: 'manual' }, null, 2)
	);
	let forecastSpec = $state(
		pretty(init?.forecastSpec) || JSON.stringify({ mode: 'full' }, null, 2)
	);

	let localError = $state('');

	function parseJSON(label: string, text: string): unknown {
		if (!text.trim()) return undefined;
		try {
			return JSON.parse(text);
		} catch (e) {
			throw new Error(`${label}: ${e instanceof Error ? e.message : 'invalid JSON'}`);
		}
	}

	function submit(e: SubmitEvent) {
		e.preventDefault();
		localError = '';
		try {
			const payload: AdminTournamentPayload = {
				slug: slug.trim(),
				name: name.trim(),
				shortName: shortName.trim(),
				status,
				startsAt: fromLocalInput(startsAt),
				endsAt: fromLocalInput(endsAt),
				extIdPrefix: extIdPrefix.trim(),
				competition,
				structure: parseJSON('structure', structure),
				sync: parseJSON('sync', sync),
				forecastSpec: parseJSON('forecastSpec', forecastSpec)
			};
			onsubmit(payload);
		} catch (err) {
			localError = err instanceof Error ? err.message : String(err);
		}
	}
</script>

<form class="tform" onsubmit={submit}>
	<div class="grid2">
		<label class="field">
			<span>Name</span>
			<input class="input" bind:value={name} required maxlength="100" placeholder="Euro 2028" />
		</label>
		<label class="field">
			<span>Short name</span>
			<input class="input" bind:value={shortName} maxlength="40" placeholder="Euro 28" />
		</label>
		<label class="field">
			<span>Slug</span>
			<input
				class="input mono"
				bind:value={slug}
				required
				pattern="[a-z0-9][a-z0-9-]{'{'}1,31{'}'}"
				placeholder="euro-2028"
				title="2–32 chars: a-z 0-9 -"
			/>
		</label>
		<label class="field">
			<span>Ext-ID prefix</span>
			<input
				class="input mono"
				bind:value={extIdPrefix}
				required
				maxlength="16"
				placeholder="EC28"
				title="Prefix for match ids; must not change after seeding"
			/>
		</label>
		<label class="field">
			<span>Status</span>
			<select class="input" bind:value={status}>
				{#each STATUSES as s (s)}<option value={s}>{s}</option>{/each}
			</select>
		</label>
		<label class="field">
			<span>Competition</span>
			<select class="input" bind:value={competition} required>
				<option value="" disabled>— choose —</option>
				{#each competitions as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
			</select>
		</label>
		<label class="field">
			<span>Starts</span>
			<input class="input" type="datetime-local" bind:value={startsAt} />
		</label>
		<label class="field">
			<span>Ends</span>
			<input class="input" type="datetime-local" bind:value={endsAt} />
		</label>
	</div>

	<label class="field">
		<span>Structure (JSON)</span>
		<textarea class="input mono" rows="10" bind:value={structure} spellcheck="false"></textarea>
	</label>
	<div class="grid2">
		<label class="field">
			<span>Sync (JSON)</span>
			<textarea class="input mono" rows="6" bind:value={sync} spellcheck="false"></textarea>
			<small class="muted"
				>provider: auto | api-football | openfootball | manual · apiFootballLeague · season ·
				openfootballURL</small
			>
		</label>
		<label class="field">
			<span>Forecast spec (JSON)</span>
			<textarea class="input mono" rows="6" bind:value={forecastSpec} spellcheck="false"
			></textarea>
			<small class="muted">mode: full | calls | none (+ calls[] for calls mode)</small>
		</label>
	</div>

	{#if localError || error}
		<p class="error">{localError || error}</p>
	{/if}

	<div class="actions">
		{#if oncancel}
			<button type="button" class="btn secondary" onclick={oncancel} disabled={busy}>Cancel</button>
		{/if}
		<button type="submit" class="btn" disabled={busy}>{busy ? 'Saving…' : submitLabel}</button>
	</div>
</form>

<style>
	.tform .field span {
		display: block;
		font-size: 0.78rem;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--muted);
		margin-bottom: 0.4rem;
	}
	.tform .field small {
		display: block;
		margin-top: 0.3rem;
		font-size: 0.75rem;
	}
	.grid2 {
		display: grid;
		grid-template-columns: 1fr;
		gap: 0 1rem;
	}
	@media (min-width: 640px) {
		.grid2 {
			grid-template-columns: 1fr 1fr;
		}
	}
	.mono {
		font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
		font-size: 0.85rem;
	}
	textarea.input {
		resize: vertical;
		line-height: 1.4;
	}
	.actions {
		display: flex;
		gap: 0.6rem;
		justify-content: flex-end;
		margin-top: 0.4rem;
	}
	.actions .btn {
		width: auto;
	}
</style>
