<script lang="ts">
	// One-time registration links for the closed test. Mint a link with a
	// note on who it's for, copy or share it, and see who came in through
	// which one. Links only matter while REGISTRATION_OPEN=0 (the server
	// gate); with sign-up open they are harmless and simply unnecessary.
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import { api, type SignupLink } from '$lib/api';
	import { appConfig } from '$lib/appconfig.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { onMount } from 'svelte';
	import { ChevronLeft, Plus, Copy, Share2, Trash2, Check, Ticket } from '@lucide/svelte';

	$effect(() => {
		if (!auth.isAdmin) goto('/');
	});
	onMount(() => appConfig.load());

	let links = $state<SignupLink[]>([]);
	let loaded = $state(false);
	let loadError = $state('');

	async function load() {
		try {
			links = (await api.signupLinks()).links ?? [];
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

	function msg(e: unknown): string {
		const err = e as { response?: { message?: string; error?: string }; message?: string };
		return err.response?.message ?? err.response?.error ?? err.message ?? 'Something went wrong.';
	}

	// ---- mint ----
	let label = $state('');
	let expiresInDays = $state(14);
	let creating = $state(false);
	let createError = $state('');
	let justCreated = $state<SignupLink | null>(null);

	async function create(e: Event) {
		e.preventDefault();
		createError = '';
		creating = true;
		try {
			const l = await api.createSignupLink(label.trim(), expiresInDays);
			links = [l, ...links];
			justCreated = l;
			label = '';
			copy(l);
		} catch (e) {
			createError = msg(e);
		} finally {
			creating = false;
		}
	}

	// ---- share ----
	const urlOf = (l: SignupLink) =>
		`${window.location.origin}/register?token=${encodeURIComponent(l.token)}`;
	let copiedId = $state('');
	let copyTimer: ReturnType<typeof setTimeout> | undefined;
	function copy(l: SignupLink) {
		navigator.clipboard?.writeText(urlOf(l)).then(
			() => {
				copiedId = l.id;
				clearTimeout(copyTimer);
				copyTimer = setTimeout(() => (copiedId = ''), 1800);
			},
			() => {}
		);
	}
	function share(l: SignupLink) {
		const url = urlOf(l);
		if (navigator.share) {
			navigator
				.share({ title: 'Your Matchowl invite', text: 'Create your Matchowl account:', url })
				.catch(() => {});
		} else copy(l);
	}
	const canShare = typeof navigator !== 'undefined' && !!navigator.share;

	// ---- revoke ----
	let confirmRevoke = $state<SignupLink | null>(null);
	let revoking = $state(false);
	async function doRevoke() {
		if (!confirmRevoke) return;
		revoking = true;
		try {
			await api.revokeSignupLink(confirmRevoke.id);
			links = links.filter((l) => l.id !== confirmRevoke!.id);
			if (justCreated?.id === confirmRevoke.id) justCreated = null;
			confirmRevoke = null;
		} catch (e) {
			loadError = msg(e);
		} finally {
			revoking = false;
		}
	}

	// ---- display ----
	const fmt = (iso: string) =>
		iso ? new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' }) : '';
	function statusText(l: SignupLink): string {
		if (l.status === 'used') return `used by ${l.usedBy?.name || 'someone'} · ${fmt(l.usedAt)}`;
		if (l.status === 'expired') return `expired ${fmt(l.expiresAt)}`;
		return l.expiresAt ? `valid until ${fmt(l.expiresAt)}` : 'no expiry';
	}
	let open = $derived(links.filter((l) => l.status === 'open'));
	let rest = $derived(links.filter((l) => l.status !== 'open'));
</script>

<div class="head">
	<div>
		<p class="kicker"><a href="/admin" class="crumb"><ChevronLeft size={14} /> Admin</a></p>
		<h1>Registration links</h1>
	</div>
</div>

{#if !auth.isAdmin}
	<p class="muted">Restricted.</p>
{:else}
	{#if appConfig.loaded && appConfig.registrationOpen}
		<p class="flash" role="status">
			Sign-up is open right now, so anyone can create an account without a link.
			Set <code>REGISTRATION_OPEN=0</code> to close it; these links then open the door
			one account at a time.
		</p>
	{/if}

	<section class="card">
		<h2 class="sec"><Ticket size={18} /> New link</h2>
		<form class="mint" onsubmit={create}>
			<div class="field grow">
				<label for="lbl">Who is it for?</label>
				<input
					id="lbl"
					class="input"
					bind:value={label}
					placeholder="Anna, the office group…"
					maxlength="80"
					autocomplete="off"
				/>
			</div>
			<div class="field">
				<label for="exp">Valid for</label>
				<select id="exp" class="input" bind:value={expiresInDays}>
					<option value={1}>1 day</option>
					<option value={7}>7 days</option>
					<option value={14}>14 days</option>
					<option value={30}>30 days</option>
					<option value={0}>No expiry</option>
				</select>
			</div>
			<button class="btn" disabled={creating}>
				<Plus size={16} />
				{creating ? 'Creating…' : 'Create link'}
			</button>
		</form>
		{#if createError}<p class="error">{createError}</p>{/if}
		{#if justCreated}
			<div class="fresh">
				<p class="muted small">
					{copiedId === justCreated.id ? 'Copied to the clipboard.' : 'Ready to share.'}
					Each link creates exactly one account.
				</p>
				<div class="urlrow">
					<code class="url">{urlOf(justCreated)}</code>
					<button class="ibtn" onclick={() => copy(justCreated!)} aria-label="Copy link">
						{#if copiedId === justCreated.id}<Check size={16} />{:else}<Copy size={16} />{/if}
					</button>
					{#if canShare}
						<button class="ibtn" onclick={() => share(justCreated!)} aria-label="Share link">
							<Share2 size={16} />
						</button>
					{/if}
				</div>
			</div>
		{/if}
	</section>

	{#if loadError}<p class="error">{loadError}</p>{/if}

	<section class="card">
		<h2 class="sec">Open links {#if loaded}<span class="count">{open.length}</span>{/if}</h2>
		{#if !loaded}
			<p class="muted">Loading…</p>
		{:else if open.length === 0}
			<p class="muted">No unused links. Create one above.</p>
		{:else}
			<ul class="list">
				{#each open as l (l.id)}
					<li class="row">
						<div class="who">
							<span class="lbl">{l.label || 'Unlabelled'}</span>
							<span class="meta muted small">
								created {fmt(l.created)}{#if l.createdBy?.name}{' '}by {l.createdBy.name}{/if}
								· {statusText(l)}
							</span>
						</div>
						<div class="acts">
							<button class="ibtn" onclick={() => copy(l)} aria-label="Copy link" title="Copy link">
								{#if copiedId === l.id}<Check size={16} />{:else}<Copy size={16} />{/if}
							</button>
							{#if canShare}
								<button class="ibtn" onclick={() => share(l)} aria-label="Share link" title="Share">
									<Share2 size={16} />
								</button>
							{/if}
							<button
								class="ibtn danger"
								onclick={() => (confirmRevoke = l)}
								aria-label="Revoke link"
								title="Revoke"
							>
								<Trash2 size={16} />
							</button>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	{#if rest.length > 0}
		<section class="card">
			<h2 class="sec">Used &amp; expired <span class="count">{rest.length}</span></h2>
			<ul class="list">
				{#each rest as l (l.id)}
					<li class="row">
						<div class="who">
							<span class="lbl">{l.label || 'Unlabelled'}</span>
							<span class="meta muted small">
								created {fmt(l.created)} · {statusText(l)}
								{#if l.usedBy?.email}{' '}<span class="email">({l.usedBy.email})</span>{/if}
							</span>
						</div>
						<div class="acts">
							<span class="pill" class:ok={l.status === 'used'}>{l.status}</span>
							{#if l.status === 'expired'}
								<button
									class="ibtn danger"
									onclick={() => (confirmRevoke = l)}
									aria-label="Delete link"
									title="Delete"
								>
									<Trash2 size={16} />
								</button>
							{/if}
						</div>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
{/if}

<ConfirmDialog
	open={!!confirmRevoke}
	title={confirmRevoke?.status === 'expired' ? 'Delete this link?' : 'Revoke this link?'}
	message={confirmRevoke
		? `${confirmRevoke.label || 'The unlabelled link'} will stop working — anyone who still has it can't create an account with it.`
		: ''}
	confirmLabel={confirmRevoke?.status === 'expired' ? 'Delete' : 'Revoke'}
	danger
	busy={revoking}
	onconfirm={doRevoke}
	oncancel={() => (confirmRevoke = null)}
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
	.sec {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		margin: 0 0 0.9rem;
		font-size: 0.95rem;
		font-weight: 700;
		color: var(--muted);
	}
	.count {
		font-weight: 600;
		font-size: 0.8rem;
		padding: 0.05rem 0.5rem;
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		border: 1px solid var(--border);
	}
	.flash {
		margin: 0 0 0.9rem;
		padding: 0.6rem 0.8rem;
		border: 1px solid color-mix(in srgb, var(--accent) 45%, var(--border));
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--accent) 10%, var(--surface));
		font-size: 0.9rem;
	}
	.flash code {
		font-size: 0.85em;
	}
	.small {
		font-size: 0.82rem;
	}

	/* ---- mint form ---- */
	.mint {
		display: flex;
		align-items: flex-end;
		gap: 0.7rem;
		flex-wrap: wrap;
	}
	.mint .field {
		margin: 0;
		min-width: 9rem;
	}
	.mint .grow {
		flex: 1 1 14rem;
	}
	.mint .btn {
		width: auto;
		flex: none;
	}
	.fresh {
		margin-top: 1rem;
		padding-top: 0.9rem;
		border-top: 1px solid var(--border);
	}
	.fresh p {
		margin: 0 0 0.5rem;
	}
	.urlrow {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.url {
		flex: 1 1 auto;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		padding: 0.5rem 0.65rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		font-size: 0.82rem;
	}

	/* ---- lists ---- */
	.list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
	}
	.row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.8rem;
		padding: 0.6rem 0;
		border-bottom: 1px solid var(--border);
	}
	.row:last-child {
		border-bottom: none;
	}
	.who {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
	}
	.lbl {
		font-weight: 600;
	}
	.meta {
		overflow-wrap: anywhere;
	}
	.email {
		opacity: 0.8;
	}
	.acts {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		flex: none;
	}
	.ibtn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2.1rem;
		height: 2.1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		cursor: pointer;
	}
	.ibtn:hover {
		border-color: var(--accent);
		color: var(--accent);
	}
	.ibtn.danger:hover {
		border-color: var(--danger);
		color: var(--danger);
	}
</style>
