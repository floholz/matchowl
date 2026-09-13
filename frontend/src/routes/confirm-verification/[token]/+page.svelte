<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { page } from '$app/stores';
	import AuthShell from '$lib/components/AuthShell.svelte';

	let token = $derived($page.params.token ?? '');
	let status = $state<'working' | 'done' | 'failed'>('working');
	let error = $state('');

	// Confirm immediately on landing — the token in the URL is the whole input,
	// there's nothing for the user to fill in.
	$effect(() => {
		if (!token) {
			status = 'failed';
			error = 'This verification link is incomplete.';
			return;
		}
		auth.confirmVerification(token)
			.then(() => (status = 'done'))
			.catch((err: unknown) => {
				status = 'failed';
				error =
					(err as { message?: string })?.message ??
					'This verification link is invalid or has expired.';
			});
	});
</script>

<AuthShell title="Email verification">
	{#if status === 'working'}
		<div class="card"><p class="muted wait">Verifying your email…</p></div>
	{:else if status === 'done'}
		<div class="card">
			<p class="ok">Your email is verified ✓</p>
			<p class="muted note">
				Friends and pools are open to you now, and the notifications you
				enable will reach this address — fine-tune them anytime in Settings.
			</p>
			<a class="btn" href={auth.isAuthed ? '/' : '/login'}>
				{auth.isAuthed ? 'Back to the app' : 'Sign in'}
			</a>
		</div>
	{:else}
		<div class="card">
			<p class="error">{error}</p>
			<p class="muted note">
				Links expire after a while. You can request a fresh one from Settings.
			</p>
			<a class="btn secondary" href={auth.isAuthed ? '/settings' : '/login'}>
				{auth.isAuthed ? 'Open Settings' : 'Sign in'}
			</a>
		</div>
	{/if}
</AuthShell>

<style>
	.wait {
		margin: 0;
	}
	.ok {
		color: var(--success);
		font-size: 0.95rem;
		margin: 0;
	}
	.note {
		margin: 0.5rem 0 1rem;
		font-size: 0.9rem;
	}
	.btn {
		display: inline-block;
		text-decoration: none;
	}
</style>
