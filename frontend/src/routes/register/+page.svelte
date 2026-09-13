<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import GoogleButton from '$lib/components/GoogleButton.svelte';

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
			await auth.register(name, email, password);
			goto(dest());
		} catch (err: unknown) {
			error = (err as { message?: string })?.message ?? 'Could not create account.';
		} finally {
			busy = false;
		}
	}

	// Google creates the account or signs into the one with the same email.
	// The terms get accepted on the first visit afterwards (the layout sends
	// every account without a current acceptance to /accept-terms).
	async function google() {
		error = '';
		busy = true;
		try {
			await auth.loginGoogle();
			goto(dest());
		} catch (e: unknown) {
			error = (e as { message?: string })?.message ?? 'Google sign-in failed.';
		} finally {
			busy = false;
		}
	}
</script>

<AuthShell title="Create account" lead="Tip the matches. Play the competitions. Beat your friends.">
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
</AuthShell>

<style>
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
