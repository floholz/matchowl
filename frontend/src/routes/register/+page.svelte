<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import GoogleButton from '$lib/components/GoogleButton.svelte';
	import { appConfig } from '$lib/appconfig.svelte';
	import { api } from '$lib/api';
	import { onMount } from 'svelte';

	onMount(() => appConfig.load());

	// One-time registration link (closed test): /register?token=… — the
	// server lets exactly one account through per token. Checked up front so
	// a used or expired link says so before anyone types a password.
	let token = $derived($page.url.searchParams.get('token') ?? '');
	let link = $state<'none' | 'checking' | 'ok' | 'used' | 'expired' | 'invalid'>('none');
	let linkLabel = $state('');
	$effect(() => {
		const t = token;
		if (!t) {
			link = 'none';
			return;
		}
		link = 'checking';
		let cancelled = false;
		api.checkSignupLink(t)
			.then((r) => {
				if (cancelled) return;
				if (r.ok) {
					link = 'ok';
					linkLabel = r.label ?? '';
				} else if (r.code === 'signup_link_used') link = 'used';
				else if (r.code === 'signup_link_expired') link = 'expired';
				else link = 'invalid';
			})
			.catch(() => {
				if (!cancelled) link = 'invalid';
			});
		return () => {
			cancelled = true;
		};
	});
	// The form shows while sign-up is open, or when the link checks out.
	let canRegister = $derived(appConfig.registrationOpen || link === 'ok');

	// After registering, resume an invite if one was carried in the URL.
	let invite = $derived($page.url.searchParams.get('invite'));
	function dest() {
		return invite ? `/join/${invite}` : '/';
	}
	let loginHref = $derived(
		invite ? `/login?invite=${encodeURIComponent(invite)}` : '/login'
	);

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let terms = $state(false);
	let error = $state('');
	let busy = $state(false);

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		if (password.length < 8) {
			error = 'Password must be at least 8 characters.';
			return;
		}
		if (!terms) {
			error = 'Please accept the terms to create an account.';
			return;
		}
		busy = true;
		try {
			await auth.register(name, email, password, token);
			goto(dest());
		} catch (err: unknown) {
			error = failMessage(err, 'Could not create account.');
		} finally {
			busy = false;
		}
	}

	// The sign-up gate answers 403 + {error, code}; name the link problem
	// when it is one (someone else used it meanwhile), else pass the
	// server's message through.
	function failMessage(err: unknown, fallback: string): string {
		const e = err as { message?: string; response?: { error?: string; code?: string } };
		const code = e.response?.code ?? '';
		if (code === 'signup_link_used') return 'This registration link has already been used.';
		if (code === 'signup_link_expired') return 'This registration link has expired.';
		if (code === 'signup_link_invalid') return 'This registration link is not valid.';
		if (code === 'registration_closed') return 'Sign-up is closed right now.';
		return e.response?.error ?? e.message ?? fallback;
	}

	// Google creates the account or signs into the one with the same email.
	// The terms get accepted on the first visit afterwards (the layout sends
	// every account without a current acceptance to /accept-terms).
	async function google() {
		error = '';
		busy = true;
		try {
			await auth.loginGoogle(token);
			goto(dest());
		} catch (e: unknown) {
			error = failMessage(e, 'Google sign-in failed.');
		} finally {
			busy = false;
		}
	}
</script>

<AuthShell title="Create account" lead="Tip the matches. Play the competitions. Beat your friends.">
	{#if !canRegister}
		<div class="card">
			{#if link === 'checking'}
				<p class="muted">Checking your registration link…</p>
			{:else if link === 'used'}
				<p class="kicker">Link already used</p>
				<p>
					This registration link has been used already — each one creates a
					single account. If that was you, sign in. Otherwise ask floholz for
					a fresh link.
				</p>
				<a class="btn" href={loginHref}>Sign in</a>
			{:else if link === 'expired'}
				<p class="kicker">Link expired</p>
				<p>This registration link has expired. Ask floholz for a fresh one.</p>
				<a class="btn secondary" href={loginHref}>Sign in</a>
			{:else if link === 'invalid'}
				<p class="kicker">Link not valid</p>
				<p>
					This registration link isn't valid — check that the whole address
					was copied, or ask floholz for a fresh one.
				</p>
				<a class="btn secondary" href={loginHref}>Sign in</a>
			{:else}
				<p class="kicker">Private testing</p>
				<p>
					Matchowl is in a closed test right now, so no new accounts can be
					created. If you have one already, sign in. Otherwise ask floholz for
					an invite.
				</p>
				<a class="btn" href={loginHref}>Sign in</a>
			{/if}
		</div>
	{:else}
	{#if link === 'ok'}
		<p class="linknote">
			<span class="kicker">You're invited</span>
			{#if linkLabel}<span>Registration link for <strong>{linkLabel}</strong>. </span>{/if}
			<span>It creates one account — yours.</span>
		</p>
	{/if}
	<form class="card" onsubmit={submit}>
		<div class="field">
			<label for="nm">Display name</label>
			<input id="nm" class="input" bind:value={name} autocomplete="nickname" required />
		</div>
		<div class="field">
			<label for="em">Email</label>
			<input
				id="em"
				class="input"
				type="email"
				bind:value={email}
				autocomplete="email"
				required
			/>
		</div>
		<div class="field">
			<label for="pw">Password</label>
			<input
				id="pw"
				class="input"
				type="password"
				bind:value={password}
				autocomplete="new-password"
				minlength="8"
				required
			/>
		</div>
		<label class="terms">
			<input type="checkbox" bind:checked={terms} required />
			<span>
				I accept the <a href="/legal/terms" target="_blank" rel="noopener">terms of use</a>
				and have read the <a href="/legal/privacy" target="_blank" rel="noopener">privacy notice</a>.
			</span>
		</label>
		{#if error}<p class="error">{error}</p>{/if}
		<button class="btn" disabled={busy}>{busy ? 'Creating…' : 'Create account'}</button>
		<div class="sep"><span>or</span></div>
		<GoogleButton disabled={busy} onclick={google} />
		<p class="muted switch">
			Already have an account? <a href={loginHref}>Sign in</a>
		</p>
	</form>
	<p class="muted foot">
		You can tip and play right away. Friends, pools and email need a verified
		address — we send the link as soon as you sign up.
	</p>
	{/if}
</AuthShell>

<style>
	.linknote {
		margin: 0 0 0.9rem;
		font-size: 0.9rem;
		color: var(--muted);
		text-align: center;
	}
	.linknote .kicker {
		display: block;
		margin-bottom: 0.15rem;
	}
	.switch {
		text-align: center;
		margin: 1rem 0 0;
	}
	.foot {
		font-size: 0.8rem;
		margin: 1.25rem 0 0;
		text-align: center;
	}
	.terms {
		display: flex;
		align-items: flex-start;
		gap: 0.6rem;
		margin: 0.2rem 0 1rem;
		font-size: 0.85rem;
		line-height: 1.35;
		color: var(--muted);
		cursor: pointer;
	}
	.terms input {
		margin-top: 0.2rem;
		accent-color: var(--accent);
		flex: none;
	}
	.terms a {
		color: var(--text);
		text-decoration: underline;
		text-underline-offset: 2px;
	}
	.sep {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin: 0.9rem 0;
		color: var(--muted);
		font-size: 0.8rem;
		text-transform: uppercase;
		letter-spacing: 0.1em;
	}
	.sep::before,
	.sep::after {
		content: '';
		flex: 1;
		height: 1px;
		background: var(--border);
	}
</style>
