<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import AuthShell from '$lib/components/AuthShell.svelte';

	let email = $state('');
	let busy = $state(false);
	let sent = $state(false);
	let error = $state('');

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			await auth.requestPasswordReset(email.trim());
			sent = true;
		} catch (err: unknown) {
			error = (err as { message?: string })?.message ?? 'Could not send reset email.';
		} finally {
			busy = false;
		}
	}
</script>

<AuthShell
	title="Reset password"
	lead="Enter the email you signed up with — we'll send you a reset link."
>
	{#if sent}
		<div class="card">
			<p class="ok">If that email is registered, a reset link is on its way.</p>
			<p class="muted switch"><a href="/login">Back to sign in</a></p>
		</div>
	{:else}
		<form class="card" onsubmit={submit}>
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
			{#if error}<p class="error">{error}</p>{/if}
			<button class="btn" disabled={busy || !email.trim()}>
				{busy ? 'Sending…' : 'Send reset link'}
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
