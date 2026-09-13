<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import AuthShell from '$lib/components/AuthShell.svelte';

	let token = $derived($page.params.token ?? '');
	let password = $state('');
	let confirm = $state('');
	let busy = $state(false);
	let error = $state('');
	let done = $state(false);

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		if (password.length < 8) {
			error = 'Password must be at least 8 characters.';
			return;
		}
		if (password !== confirm) {
			error = 'Passwords do not match.';
			return;
		}
		busy = true;
		try {
			await auth.confirmPasswordReset(token, password, confirm);
			done = true;
			// PocketBase invalidates the session after a reset — make sure
			// nothing stale lingers, then send the user to sign in fresh.
			auth.logout();
			setTimeout(() => goto('/login'), 1200);
		} catch (err: unknown) {
			error =
				(err as { message?: string })?.message ??
				'This reset link is invalid or has expired.';
		} finally {
			busy = false;
		}
	}
</script>

<AuthShell title="Choose a new password" lead="Enter and confirm your new password.">
	{#if done}
		<div class="card">
			<p class="ok">Password updated — taking you to sign in…</p>
		</div>
	{:else}
		<form class="card" onsubmit={submit}>
			<div class="field">
				<label for="pw">New password</label>
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
			<div class="field">
				<label for="pw2">Confirm new password</label>
				<input
					id="pw2"
					class="input"
					type="password"
					bind:value={confirm}
					autocomplete="new-password"
					minlength="8"
					required
				/>
			</div>
			{#if error}<p class="error">{error}</p>{/if}
			<button class="btn" disabled={busy || !token}>
				{busy ? 'Updating…' : 'Update password'}
			</button>
			<p class="muted switch"><a href="/login">Back to sign in</a></p>
		</form>
	{/if}
</AuthShell>

<style>
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
