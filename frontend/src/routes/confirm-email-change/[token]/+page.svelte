<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import AuthShell from '$lib/components/AuthShell.svelte';

	let token = $derived($page.params.token ?? '');
	// The new address rides in the token payload — decode it for display so
	// the user sees what they're confirming (purely cosmetic, the server
	// re-validates the token).
	let newEmail = $derived.by(() => {
		try {
			return JSON.parse(atob(token.split('.')[1] ?? '')).newEmail ?? '';
		} catch {
			return '';
		}
	});

	let password = $state('');
	let busy = $state(false);
	let error = $state('');
	let done = $state(false);

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			await auth.confirmEmailChange(token, password);
			done = true;
			// The email change invalidates every session — clear the stale one
			// and send the user to sign in with the new address.
			auth.logout();
			setTimeout(() => goto('/login'), 1500);
		} catch (err: unknown) {
			error =
				(err as { message?: string })?.message ??
				'This link is invalid or has expired, or the password is wrong.';
		} finally {
			busy = false;
		}
	}
</script>

<AuthShell title="Confirm new email">
	<p class="muted lead">
		{#if newEmail}
			Confirm <strong>{newEmail}</strong> as your new address by entering your
			account password.
		{:else}
			Enter your account password to confirm the email change.
		{/if}
	</p>

	{#if done}
		<div class="card">
			<p class="ok">Email changed — sign in again with the new address…</p>
		</div>
	{:else}
		<form class="card" onsubmit={submit}>
			<div class="field">
				<label for="pw">Account password</label>
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
			<button class="btn" disabled={busy || !token}>
				{busy ? 'Confirming…' : 'Confirm email change'}
			</button>
			<p class="muted switch"><a href="/login">Back to sign in</a></p>
		</form>
	{/if}
</AuthShell>

<style>
	.lead {
		margin: 0.25rem 0 0;
	}
	.ok {
		color: var(--success);
		font-size: 0.95rem;
		margin: 0;
	}
	.switch {
		text-align: center;
		margin: 1rem 0 0;
	}
</style>
