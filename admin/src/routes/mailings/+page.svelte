<!-- Mailings: one targeted email to an audience. Write it in Markdown with
     a button, pick who gets it (the count and the names update as you
     narrow), preview the real mail, send it to yourself first, then send
     it once. Sent mailings stay as history with their result. -->
<script lang="ts">
	import { api, type Mailing, type Audience, type AdminTournament, type PoolSummary } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import MailPreview from '$lib/components/admin/MailPreview.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { Plus, Pencil, Send, Trash2, Check, FlaskConical, Copy, Users } from '@lucide/svelte';

	let items = $state<Mailing[]>([]);
	let loaded = $state(false);
	let error = $state('');
	let flash = $state('');
	let tournaments = $state<AdminTournament[]>([]);
	let pools = $state<PoolSummary[]>([]);

	// The editor.
	let editId = $state('');
	let subject = $state('');
	let body = $state('');
	let ctaText = $state('');
	let ctaUrl = $state('');
	let roles = $state<string[]>([]);
	let playing = $state('');
	let pool = $state('');
	let joinedAfter = $state('');
	let joinedBefore = $state('');
	let emails = $state('');
	let exclude = $state('');
	let saving = $state(false);
	let formError = $state('');

	let audience = $derived<Audience>({
		roles: roles.length ? roles : undefined,
		playing: playing || undefined,
		pool: pool || undefined,
		joinedAfter: joinedAfter || undefined,
		joinedBefore: joinedBefore || undefined,
		emails: emails.trim() ? emails.split(/[\s,;]+/).filter(Boolean) : undefined,
		exclude: exclude.trim() ? exclude.split(/[\s,;]+/).filter(Boolean) : undefined
	});
	// Who it reaches, refreshed as the filter changes.
	let reach = $state<{ count: number; recipients: { id: string; name: string; email: string }[] } | null>(null);
	let showReach = $state(false);
	let reachTimer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		const a = audience;
		clearTimeout(reachTimer);
		reachTimer = setTimeout(async () => {
			try {
				reach = await api.mailingAudience(a);
			} catch {
				reach = null;
			}
		}, 300);
		return () => clearTimeout(reachTimer);
	});

	async function load() {
		error = '';
		try {
			const [m, t, p] = await Promise.all([api.mailings(), api.adminTournaments(), api.myPools()]);
			items = m.mailings;
			tournaments = t.tournaments.filter((x) => x.status === 'active' || x.status === 'upcoming');
			pools = p.pools;
		} catch (e) {
			error = msg(e);
		} finally {
			loaded = true;
		}
	}
	$effect(() => {
		if (auth.isAdmin) load();
	});
	function msg(err: unknown): string {
		const r = (err as { response?: { error?: string; message?: string } })?.response;
		return r?.error || r?.message || (err instanceof Error ? err.message : String(err));
	}
	function resetForm() {
		editId = '';
		subject = '';
		body = '';
		ctaText = '';
		ctaUrl = '';
		roles = [];
		playing = '';
		pool = '';
		joinedAfter = '';
		joinedBefore = '';
		emails = '';
		exclude = '';
		formError = '';
	}
	function startEdit(m: Mailing, asCopy = false) {
		editId = asCopy ? '' : m.id;
		subject = asCopy ? m.subject : m.subject;
		body = m.body;
		ctaText = m.ctaText;
		ctaUrl = m.ctaUrl;
		roles = m.audience?.roles ?? [];
		playing = m.audience?.playing ?? '';
		pool = m.audience?.pool ?? '';
		joinedAfter = m.audience?.joinedAfter ?? '';
		joinedBefore = m.audience?.joinedBefore ?? '';
		emails = (m.audience?.emails ?? []).join('\n');
		exclude = (m.audience?.exclude ?? []).join('\n');
		formError = '';
		flash = '';
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}
	async function save(): Promise<Mailing | null> {
		formError = '';
		if (!subject.trim() || !body.trim()) {
			formError = 'Subject and message are required.';
			return null;
		}
		saving = true;
		try {
			const payload = { subject: subject.trim(), body: body.trim(), ctaText: ctaText.trim(), ctaUrl: ctaUrl.trim(), audience };
			let m: Mailing;
			if (editId) {
				m = await api.updateMailing(editId, payload);
				items = items.map((x) => (x.id === editId ? m : x));
			} else {
				m = await api.createMailing(payload);
				items = [m, ...items];
				editId = m.id;
			}
			flash = 'Draft saved.';
			return m;
		} catch (e) {
			formError = msg(e);
			return null;
		} finally {
			saving = false;
		}
	}
	async function sendTest() {
		const m = await save();
		if (!m) return;
		try {
			const r = await api.mailingTest(m.id);
			flash = `Test sent to ${r.to} via ${r.provider}.`;
		} catch (e) {
			formError = msg(e);
		}
	}
	let confirmSend = $state<Mailing | null>(null);
	let confirmDelete = $state<Mailing | null>(null);
	let dialogBusy = $state(false);
	async function askSend() {
		const m = await save();
		if (m) confirmSend = m;
	}
	async function doSend() {
		if (!confirmSend) return;
		dialogBusy = true;
		try {
			const r = await api.mailingSend(confirmSend.id);
			items = items.map((x) => (x.id === r.mailing.id ? r.mailing : x));
			flash = `Sent to ${r.recipients} recipient${r.recipients === 1 ? '' : 's'} · ${r.result.sent} delivered · ${r.result.skipped} skipped (opted out or already sent) · ${r.result.failed} failed.`;
			resetForm();
		} catch (e) {
			formError = msg(e);
		} finally {
			dialogBusy = false;
			confirmSend = null;
		}
	}
	async function doDelete() {
		if (!confirmDelete) return;
		dialogBusy = true;
		try {
			await api.deleteMailing(confirmDelete.id);
			items = items.filter((x) => x.id !== confirmDelete!.id);
			if (editId === confirmDelete.id) resetForm();
		} catch (e) {
			error = msg(e);
		} finally {
			dialogBusy = false;
			confirmDelete = null;
		}
	}
	const fmt = (iso: string) => (iso ? new Date(iso).toLocaleString() : '');
	function toggleRole(r: string) {
		roles = roles.includes(r) ? roles.filter((x) => x !== r) : [...roles, r];
	}
	const describe = (a: Audience) => {
		const parts: string[] = [];
		if (a.roles?.length) parts.push(a.roles.join(' + '));
		if (a.playing) parts.push('plays ' + (tournaments.find((t) => t.id === a.playing)?.name ?? 'a season'));
		if (a.pool) parts.push('in ' + (pools.find((p) => p.id === a.pool)?.name ?? 'a pool'));
		if (a.joinedAfter) parts.push('joined after ' + a.joinedAfter);
		if (a.joinedBefore) parts.push('joined before ' + a.joinedBefore);
		if (a.emails?.length) parts.push(`${a.emails.length} listed`);
		if (a.exclude?.length) parts.push(`${a.exclude.length} excluded`);
		return parts.length ? parts.join(' · ') : 'everyone with a verified email';
	};
</script>

<div class="head">
	<div>
		<p class="kicker">Admin</p>
		<h1>Mailings</h1>
		<p class="muted small">One email to an audience. Recipients can opt out under "News from Matchowl" in their settings; unverified addresses never get mail.</p>
	</div>
</div>

{#if error}<p class="error">{error}</p>{/if}

<div class="grid">
	<section class="card form">
		<h2 class="sec">{#if editId}<Pencil size={18} /> Edit draft{:else}<Plus size={18} /> New mailing{/if}</h2>
		<label class="fld">
			<span>Subject</span>
			<input bind:value={subject} maxlength="200" placeholder="Matchowl is back for the new season" />
		</label>
		<label class="fld">
			<span>Message <small class="muted">Markdown · <code>{'{{name}}'}</code> = the recipient's first name</small></span>
			<textarea bind:value={body} rows="10" placeholder={'Hi {{name}},\n\n…'}></textarea>
		</label>
		<div class="row">
			<label class="fld grow">
				<span>Button text <small class="muted">optional</small></span>
				<input bind:value={ctaText} maxlength="60" placeholder="Open Matchowl" />
			</label>
			<label class="fld grow">
				<span>Button link</span>
				<input bind:value={ctaUrl} maxlength="500" placeholder="https://play.matchowl.app/…" />
			</label>
		</div>

		<h3 class="sub"><Users size={16} /> Audience <span class="count" class:zero={reach?.count === 0}>{reach ? `${reach.count} recipient${reach.count === 1 ? '' : 's'}` : '…'}</span></h3>
		<div class="aud">
			<div class="fld">
				<span>Role</span>
				<div class="chips">
					{#each ['member', 'admin', 'owner'] as r (r)}
						<button class="chip" class:on={roles.includes(r)} onclick={() => toggleRole(r)}>{r}</button>
					{/each}
				</div>
			</div>
			<label class="fld">
				<span>Plays</span>
				<select bind:value={playing}>
					<option value="">any season</option>
					{#each tournaments as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
				</select>
			</label>
			<label class="fld">
				<span>Member of pool</span>
				<select bind:value={pool}>
					<option value="">any</option>
					{#each pools as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
				</select>
			</label>
			<div class="row">
				<label class="fld grow"><span>Joined after</span><input type="date" bind:value={joinedAfter} /></label>
				<label class="fld grow"><span>Joined before</span><input type="date" bind:value={joinedBefore} /></label>
			</div>
			<div class="row">
				<label class="fld grow"><span>Only these emails <small class="muted">one per line</small></span><textarea bind:value={emails} rows="3"></textarea></label>
				<label class="fld grow"><span>Never these emails</span><textarea bind:value={exclude} rows="3"></textarea></label>
			</div>
			{#if reach?.count}
				<button class="linkb" onclick={() => (showReach = !showReach)}>{showReach ? 'Hide' : 'Show'} recipients</button>
				{#if showReach}
					<ul class="reach">
						{#each reach.recipients as r (r.id)}<li>{r.name} <span class="muted">{r.email}</span></li>{/each}
					</ul>
				{/if}
			{/if}
		</div>

		<MailPreview {subject} {body} {ctaText} {ctaUrl} />

		{#if formError}<p class="error">{formError}</p>{/if}
		{#if flash}<p class="flash">{flash}</p>{/if}
		<div class="actions">
			{#if editId}<button class="btn ghost" onclick={resetForm} disabled={saving}>New</button>{/if}
			<button class="btn secondary" onclick={save} disabled={saving}>{saving ? 'Saving…' : 'Save draft'}</button>
			<button class="btn secondary" onclick={sendTest} disabled={saving}><FlaskConical size={16} /> Send me a test</button>
			<button class="btn" onclick={askSend} disabled={saving || !reach?.count}><Send size={16} /> Send to {reach?.count ?? 0}</button>
		</div>
	</section>

	<section class="list">
		<h2 class="sec">History</h2>
		{#if !loaded}
			<p class="muted">Loading…</p>
		{:else if !items.length}
			<p class="muted">No mailings yet.</p>
		{:else}
			<ul>
				{#each items as m (m.id)}
					<li class="card ml" class:sent={m.status === 'sent'} class:editing={editId === m.id}>
						<div class="mtop">
							{#if m.status === 'sent'}<span class="pill sent"><Check size={12} /> sent {fmt(m.sentAt)}</span>{:else}<span class="pill">draft</span>{/if}
							<span class="when muted">{fmt(m.created)}</span>
						</div>
						<strong>{m.subject}</strong>
						<p class="muted small">{describe(m.audience ?? {})}{#if m.status === 'sent' && m.result} · {m.recipients} recipients · {m.result.sent} delivered · {m.result.skipped} skipped · {m.result.failed} failed{/if}</p>
						<div class="mactions">
							{#if m.status === 'draft'}
								<button class="ib" onclick={() => startEdit(m)}><Pencil size={14} /> Edit</button>
							{:else}
								<button class="ib" onclick={() => startEdit(m, true)}><Copy size={14} /> Duplicate</button>
							{/if}
							<button class="ib danger" onclick={() => (confirmDelete = m)}><Trash2 size={14} /></button>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>

<ConfirmDialog
	open={!!confirmSend}
	title="Send this mailing?"
	message={`"${confirmSend?.subject ?? ''}" goes to ${reach?.count ?? 0} recipients now. It can only be sent once.`}
	confirmLabel="Send"
	busy={dialogBusy}
	onconfirm={doSend}
	oncancel={() => (confirmSend = null)}
/>
<ConfirmDialog
	open={!!confirmDelete}
	title="Delete mailing?"
	message="Drafts are gone for good; a sent mailing only loses its history entry."
	confirmLabel="Delete"
	danger
	busy={dialogBusy}
	onconfirm={doDelete}
	oncancel={() => (confirmDelete = null)}
/>

<style>
	.head {
		margin-bottom: 1rem;
	}
	.small {
		font-size: 0.85rem;
		margin: 0.3rem 0 0;
	}
	.grid {
		display: grid;
		grid-template-columns: minmax(0, 1.4fr) minmax(280px, 1fr);
		gap: 1.2rem;
		align-items: start;
	}
	@media (max-width: 1000px) {
		.grid {
			grid-template-columns: 1fr;
		}
	}
	.sec {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 1.15rem;
		margin-bottom: 0.8rem;
	}
	.sub {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		margin: 1.2rem 0 0.5rem;
		font-size: 0.95rem;
	}
	.count {
		margin-left: auto;
		padding: 0.15rem 0.6rem;
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--accent) 16%, transparent);
		color: var(--accent);
		font-size: 0.78rem;
		font-weight: 800;
	}
	.count.zero {
		background: var(--surface-2);
		color: var(--muted);
	}
	.fld {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
		margin-bottom: 0.7rem;
		font-size: 0.8rem;
		font-weight: 700;
		color: var(--muted);
	}
	.fld small {
		font-weight: 400;
		margin-left: 0.3rem;
	}
	.fld input,
	.fld textarea,
	.fld select {
		padding: 0.6rem 0.7rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
	}
	.fld textarea {
		resize: vertical;
	}
	.row {
		display: flex;
		gap: 0.7rem;
	}
	.grow {
		flex: 1;
	}
	.aud {
		padding: 0.7rem 0.8rem 0.3rem;
		border: 1px dashed var(--border);
		border-radius: var(--radius-sm);
	}
	.chips {
		display: flex;
		gap: 0.4rem;
	}
	.chip {
		padding: 0.3rem 0.7rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--muted);
		font: inherit;
		font-size: 0.8rem;
		font-weight: 700;
		cursor: pointer;
	}
	.chip.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.linkb {
		border: none;
		background: transparent;
		color: var(--accent);
		font: inherit;
		font-size: 0.8rem;
		font-weight: 700;
		cursor: pointer;
		padding: 0 0 0.5rem;
	}
	.reach {
		list-style: none;
		margin: 0 0 0.6rem;
		padding: 0;
		max-height: 220px;
		overflow: auto;
		font-size: 0.82rem;
		columns: 2;
	}
	.actions {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
		justify-content: flex-end;
	}
	.actions .btn {
		width: auto;
	}
	.flash {
		color: var(--success);
		font-size: 0.9rem;
	}
	.list ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}
	.ml {
		padding: 0.8rem 0.9rem;
	}
	.ml.editing {
		border-color: var(--accent);
	}
	.mtop {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.3rem;
	}
	.when {
		margin-left: auto;
		font-size: 0.75rem;
	}
	.pill.sent {
		color: var(--success);
		border-color: var(--success);
	}
	.mactions {
		display: flex;
		gap: 0.4rem;
		margin-top: 0.4rem;
	}
	.ib {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		padding: 0.3rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--muted);
		font: inherit;
		font-size: 0.78rem;
		font-weight: 700;
		cursor: pointer;
	}
	.ib.danger {
		color: var(--danger);
	}
</style>
