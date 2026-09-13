<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import GoogleButton from '$lib/components/GoogleButton.svelte';

	let identity = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);

	// After signing in, resume an invite if one was carried in the URL.
	let invite = $derived($page.url.searchParams.get('invite'));
	function dest() {
		return invite ? `/join/${invite}` : '/';
	}
	let registerHref = $derived(
		invite ? `/register?invite=${encodeURIComponent(invite)}` : '/register'
	);

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			await auth.login(identity, password);
			goto(dest());
		} catch {
			error = 'Invalid email or password.';
		} finally {
			busy = false;
		}
	}

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

<AuthShell title="Sign in" lead="Tip the matches. Play the competitions. Beat your friends.">
	<form class="card" onsubmit={submit}>
		<div class="field">
			<label for="id">Email</label>
			<input
				id="id"
				class="input"
				type="email"
				bind:value={identity}
				autocomplete="email"
				required
			/>
		</div>
		<div class="field">
			<div class="lblrow">
				<label for="pw">Password</label>
				<a class="forgot" href="/forgot-password">Forgot password?</a>
			</div>
			<input
				id="pw"
				class="input"
				type="password"
				bind:value={password}
				autocomplete="current-password"
				required
			/>
		</div>
		{#if error}<p class="error">{error}</p>{/if}
		<button class="btn" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
		<div class="sep"><span>or</span></div>
		<GoogleButton disabled={busy} onclick={google} />
		<p class="muted switch">
			No account? <a href={registerHref}>Create one</a>
		</p>
	</form>
	<p class="muted foot">
		<a href="/help">How Matchowl works</a> · <a href="/legal/privacy">Privacy</a> ·
		<a href="/legal/imprint">Imprint</a>
	</p>
</AuthShell>

<style>
	.switch {
		text-align: center;
		margin: 1rem 0 0;
	}
	.foot {
		text-align: center;
		font-size: 0.8rem;
		margin: 1.25rem 0 0;
	}
	.lblrow {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.75rem;
	}
	.forgot {
		font-size: 0.8rem;
		color: var(--muted);
	}
	.forgot:hover {
		color: var(--accent);
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
