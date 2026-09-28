<!-- Admin sign-in: the same accounts as the app (role admin or owner), on
     this origin. Email + password only; the server refuses everything
     else, so a normal member who signs in here just sees "not an admin". -->
<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';
	import { LogIn } from '@lucide/svelte';

	let email = $state('');
	let password = $state('');
	let busy = $state(false);
	let error = $state('');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			await auth.login(email.trim(), password);
			goto('/', { replaceState: true });
		} catch {
			error = 'Sign-in failed. Check the email and password.';
		} finally {
			busy = false;
		}
	}
</script>

<div class="wrap">
	<form class="card" onsubmit={submit}>
		<div class="brand">
			<img src="/favicon.svg" alt="" width="40" height="40" />
			<h1>Matchowl <b>Admin</b></h1>
		</div>
		<label>
			<span>Email</span>
			<input type="email" bind:value={email} autocomplete="username" required />
		</label>
		<label>
			<span>Password</span>
			<input type="password" bind:value={password} autocomplete="current-password" required />
		</label>
		{#if error}<p class="error">{error}</p>{/if}
		<button class="btn" type="submit" disabled={busy}><LogIn size={16} /> {busy ? 'Signing in…' : 'Sign in'}</button>
	</form>
</div>

<style>
	.wrap {
		min-height: 100dvh;
		display: grid;
		place-items: center;
		padding: 1rem;
	}
	.card {
		width: min(100%, 380px);
		display: flex;
		flex-direction: column;
		gap: 0.9rem;
	}
	.brand {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		margin-bottom: 0.4rem;
	}
	h1 {
		font-size: 1.5rem;
	}
	h1 b {
		color: var(--accent);
	}
	label {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
		font-size: 0.82rem;
		font-weight: 600;
		color: var(--muted);
	}
	input {
		padding: 0.7rem 0.8rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
	}
	input:focus {
		outline: var(--ring);
		outline-offset: 1px;
	}
</style>
