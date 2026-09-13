<script lang="ts">
	// Pool invite landing. Resolves the code, then: signed out → sign in /
	// create account (carrying the code); unverified → verify first (pools
	// are a social feature); finished pool → says so before anyone signs up
	// for nothing; otherwise joins and opens the pool. Every failure names
	// its reason — the server's, not a generic one.
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import AuthShell from '$lib/components/AuthShell.svelte';

	let code = $derived($page.params.code ?? '');
	let poolName = $state('');
	let phase = $state<
		'loading' | 'invite' | 'joining' | 'finished' | 'unverified' | 'invalid' | 'error'
	>('loading');
	let reason = $state('');

	let sent = $state(false);
	let busy = $state(false);
	let sendError = $state('');
	async function resend() {
		sendError = '';
		busy = true;
		try {
			await auth.requestVerification();
			sent = true;
		} catch (err: unknown) {
			sendError =
				(err as { message?: string })?.message ?? 'Could not send the verification email.';
		} finally {
			busy = false;
		}
	}

	type ApiErr = { status?: number; response?: { error?: string; code?: string } };

	$effect(() => {
		const c = code;
		if (!c) {
			phase = 'invalid';
			return;
		}
		let cancelled = false;
		(async () => {
			let preview: { id: string; name: string; finished: boolean };
			try {
				preview = await api.invitePreview(c);
			} catch {
				if (!cancelled) phase = 'invalid';
				return;
			}
			if (cancelled) return;
			poolName = preview.name;
			if (preview.finished) {
				phase = 'finished';
				return;
			}
			if (!auth.isAuthed) {
				phase = 'invite';
				return;
			}
			if (!auth.user?.verified) {
				phase = 'unverified';
				return;
			}
			phase = 'joining';
			try {
				const r = await api.joinPool(c);
				if (!cancelled) goto(`/pools/${r.id}`, { replaceState: true });
			} catch (err: unknown) {
				if (cancelled) return;
				const e = err as ApiErr;
				if (e.response?.code === 'unverified') phase = 'unverified';
				else if (e.status === 409) phase = 'finished';
				else if (e.status === 404) phase = 'invalid';
				else {
					phase = 'error';
					reason = e.response?.error ?? '';
				}
			}
		})();
		return () => {
			cancelled = true;
		};
	});

	const backHref = $derived(auth.isAuthed ? '/friends' : '/login');
	const backLabel = $derived(auth.isAuthed ? 'Go to Friends' : 'Sign in');
</script>

<AuthShell title="Pool invite">
	<div class="card">
		{#if phase === 'loading'}
			<p class="muted">Checking your invite…</p>
		{:else if phase === 'joining'}
			<p class="muted">Joining <strong>{poolName}</strong>…</p>
		{:else if phase === 'invite'}
			<p class="kicker">You've been invited</p>
			<h2 class="lname">{poolName}</h2>
			<p class="muted">Sign in or create an account to join this pool.</p>
			<a class="btn" href={`/register?invite=${encodeURIComponent(code)}`}>Create account</a>
			<a class="btn secondary" href={`/login?invite=${encodeURIComponent(code)}`}>Sign in</a>
		{:else if phase === 'finished'}
			<p class="kicker">Pool finished</p>
			<h2 class="lname">{poolName}</h2>
			<p class="muted">
				Every season this pool counted is over, so it takes no new members.
				Ask whoever runs it to set it up again for the next season — you'll
				get a fresh invite.
			</p>
			<a class="btn secondary" href={backHref}>{backLabel}</a>
		{:else if phase === 'unverified'}
			<p class="kicker">Verify your email first</p>
			<h2 class="lname">{poolName}</h2>
			<p class="muted">
				Pools and friends open up once <strong>{auth.user?.email}</strong> is
				confirmed. Use the link we sent you, or send a fresh one — then open
				this invite again.
			</p>
			{#if sent}
				<p class="ok">Verification email sent — check your inbox.</p>
			{:else}
				<button class="btn" onclick={resend} disabled={busy}>
					{busy ? 'Sending…' : 'Send verification email'}
				</button>
			{/if}
			{#if sendError}<p class="error">{sendError}</p>{/if}
			<a class="btn secondary" href="/">Back to the app</a>
		{:else if phase === 'error'}
			<p class="error">Couldn't join <strong>{poolName}</strong>.</p>
			{#if reason}<p class="muted small">{reason}</p>{/if}
			<button class="btn" onclick={() => location.reload()}>Try again</button>
			<a class="btn secondary" href={backHref}>{backLabel}</a>
		{:else}
			<p class="error">This invite link is invalid or has expired.</p>
			<p class="muted small">Ask for a fresh link, or join with the code under Friends.</p>
			<a class="btn secondary" href={backHref}>{backLabel}</a>
		{/if}
	</div>
</AuthShell>

<style>
	.lname {
		margin: 0.1rem 0 0.6rem;
		font-size: 1.7rem;
	}
	.card .btn + .btn,
	.card p + .btn,
	.card .btn + p {
		margin-top: 0.6rem;
	}
	.card .btn {
		display: block;
		text-align: center;
		text-decoration: none;
	}
	.ok {
		color: var(--success);
		font-size: 0.95rem;
		margin: 0;
	}
	.small {
		font-size: 0.85rem;
	}
</style>
